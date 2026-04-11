# Mascot Guide System — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a visual anime mascot assistant that appears in all scenes, showing speech-bubble dialogs triggered by game events, scene transitions, and idle timeouts.

**Architecture:** Core logic in `internal/core/mascot/` (Guide state machine, JSON-driven dialogs). Rendering in `internal/render/hud/` (MascotVM + speech bubble + sprite). Integration at Game struct level — no per-scene modification needed.

**Tech Stack:** Go, Ebitengine v2.9.9, JSON config, existing `anim.Animator` + `draw.*` + HUD ViewModel pattern.

---

### Task 1: Data Types — `internal/core/mascot/dialog.go`

**Files:**
- Create: `internal/core/mascot/dialog.go`
- Test: `internal/core/mascot/dialog_test.go`

**Step 1: Write the test**

```go
// dialog_test.go
package mascot

import "testing"

func TestDialogHasLines(t *testing.T) {
    d := Dialog{
        ID:    "test_hello",
        Scene: "title",
        Lines: []Line{
            {Text: "Hello!", Expression: "happy", AutoAdvance: 3},
        },
        Once: true,
    }
    if d.ID != "test_hello" {
        t.Fatalf("unexpected ID: %s", d.ID)
    }
    if len(d.Lines) != 1 {
        t.Fatalf("expected 1 line, got %d", len(d.Lines))
    }
    if d.Lines[0].Expression != "happy" {
        t.Fatalf("unexpected expression: %s", d.Lines[0].Expression)
    }
}
```

**Step 2: Run test — expect FAIL** (package doesn't exist yet)

```bash
go test ./internal/core/mascot/ -run TestDialogHasLines -v
```

**Step 3: Write minimal implementation**

```go
// dialog.go
package mascot

// Line is one speech bubble message within a dialog.
type Line struct {
    Text        string  `json:"text"`
    Expression  string  `json:"expression"`  // "idle"/"talk"/"happy"/"surprised"
    AutoAdvance float64 `json:"autoAdvance"` // seconds; 0 = click to advance
}

// Dialog is a triggered conversation sequence.
type Dialog struct {
    ID       string `json:"id"`       // unique identifier
    Scene    string `json:"scene"`    // "title"/"stage"/"select"/"*" (wildcard)
    Trigger  string `json:"trigger"`  // event name; "scene_enter" = on scene switch
    Lines    []Line `json:"lines"`
    Once     bool   `json:"once"`     // show only once ever (persisted)
    Priority int    `json:"priority"` // higher wins when multiple match
}
```

**Step 4: Run test — expect PASS**

```bash
go test ./internal/core/mascot/ -run TestDialogHasLines -v
```

**Step 5: Commit**

```bash
git add internal/core/mascot/
git commit -m "feat(mascot): add dialog data types"
```

---

### Task 2: JSON Loader — `internal/core/mascot/loader.go`

**Files:**
- Create: `internal/core/mascot/loader.go`
- Create: `config/mascot/dialogs-title.json` (starter content)
- Test: `internal/core/mascot/loader_test.go`

**Step 1: Create starter JSON**

```json
// config/mascot/dialogs-title.json
[
  {
    "id": "title_welcome",
    "scene": "title",
    "trigger": "scene_enter",
    "lines": [
      {"text": "欢迎回来，指挥官！", "expression": "happy", "autoAdvance": 4},
      {"text": "准备好迎接新的挑战了吗？", "expression": "talk", "autoAdvance": 3}
    ],
    "once": false,
    "priority": 10
  }
]
```

**Step 2: Write the test**

```go
// loader_test.go
package mascot

import (
    "testing"
)

const testJSON = `[
  {
    "id": "test_dialog",
    "scene": "title",
    "trigger": "scene_enter",
    "lines": [
      {"text": "Hello!", "expression": "happy", "autoAdvance": 3}
    ],
    "once": true,
    "priority": 5
  }
]`

func TestLoadDialogsFromJSON(t *testing.T) {
    dialogs, err := ParseDialogs([]byte(testJSON))
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }
    if len(dialogs) != 1 {
        t.Fatalf("expected 1 dialog, got %d", len(dialogs))
    }
    d := dialogs[0]
    if d.ID != "test_dialog" {
        t.Errorf("unexpected ID: %s", d.ID)
    }
    if d.Scene != "title" {
        t.Errorf("unexpected scene: %s", d.Scene)
    }
    if !d.Once {
        t.Error("expected Once=true")
    }
    if len(d.Lines) != 1 || d.Lines[0].Text != "Hello!" {
        t.Error("unexpected lines")
    }
}
```

**Step 3: Run test — expect FAIL**

**Step 4: Implement loader**

```go
// loader.go
package mascot

import (
    "encoding/json"
    "fmt"
)

// ParseDialogs decodes a JSON array of Dialog objects.
func ParseDialogs(data []byte) ([]Dialog, error) {
    var dialogs []Dialog
    if err := json.Unmarshal(data, &dialogs); err != nil {
        return nil, fmt.Errorf("parse mascot dialogs: %w", err)
    }
    return dialogs, nil
}

// AssetReader reads embedded files.
type AssetReader interface {
    ReadFile(name string) ([]byte, error)
}

// LoadAllDialogs loads all dialog JSON files from config/mascot/.
func LoadAllDialogs(fs AssetReader) ([]Dialog, error) {
    files := []string{
        "config/mascot/dialogs-title.json",
        "config/mascot/dialogs-stage.json",
        "config/mascot/dialogs-select.json",
        "config/mascot/dialogs-common.json",
    }
    var all []Dialog
    for _, f := range files {
        data, err := fs.ReadFile(f)
        if err != nil {
            continue // file not found is OK (not all scenes need dialogs yet)
        }
        dialogs, err := ParseDialogs(data)
        if err != nil {
            return nil, fmt.Errorf("load %s: %w", f, err)
        }
        all = append(all, dialogs...)
    }
    return all, nil
}
```

**Step 5: Run test — expect PASS**

**Step 6: Commit**

```bash
git add internal/core/mascot/ config/mascot/
git commit -m "feat(mascot): add JSON dialog loader"
```

---

### Task 3: Guide State Machine — `internal/core/mascot/mascot.go`

**Files:**
- Create: `internal/core/mascot/mascot.go`
- Test: `internal/core/mascot/mascot_test.go`

**Step 1: Write tests**

```go
// mascot_test.go
package mascot

import "testing"

func makeTestGuide() *Guide {
    dialogs := []Dialog{
        {
            ID: "enter_title", Scene: "title", Trigger: "scene_enter",
            Lines:    []Line{{Text: "Welcome!", Expression: "happy", AutoAdvance: 2}},
            Priority: 10,
        },
        {
            ID: "wave_tip", Scene: "stage", Trigger: "wave_start",
            Lines: []Line{
                {Text: "Wave incoming!", Expression: "surprised"},
                {Text: "Good luck!", Expression: "talk", AutoAdvance: 3},
            },
            Once:     true,
            Priority: 5,
        },
    }
    return NewGuide(dialogs, nil)
}

func TestSetSceneTriggersEnterDialog(t *testing.T) {
    g := makeTestGuide()
    g.SetScene("title")
    if g.current == nil {
        t.Fatal("expected dialog to start on scene_enter")
    }
    if g.current.ID != "enter_title" {
        t.Fatalf("unexpected dialog: %s", g.current.ID)
    }
    vm := g.VM()
    if vm.Text != "Welcome!" {
        t.Fatalf("unexpected text: %s", vm.Text)
    }
}

func TestTriggerMatchesEvent(t *testing.T) {
    g := makeTestGuide()
    g.SetScene("stage")
    g.Trigger("wave_start")
    if g.current == nil || g.current.ID != "wave_tip" {
        t.Fatal("expected wave_tip dialog")
    }
}

func TestClickAdvance(t *testing.T) {
    g := makeTestGuide()
    g.SetScene("stage")
    g.Trigger("wave_start")
    // Line 0: "Wave incoming!" (no autoAdvance → click to advance)
    g.ClickAdvance()
    vm := g.VM()
    if vm.Text != "Good luck!" {
        t.Fatalf("expected line 2, got: %s", vm.Text)
    }
}

func TestAutoAdvance(t *testing.T) {
    g := makeTestGuide()
    g.SetScene("title")
    // AutoAdvance = 2s
    g.Tick(1.0)
    if g.current == nil {
        t.Fatal("should still be active at 1s")
    }
    g.Tick(1.5) // total 2.5s > 2s threshold
    if g.current != nil {
        t.Fatal("should have auto-advanced past last line")
    }
}

func TestOnceDialogNotRepeated(t *testing.T) {
    g := makeTestGuide()
    g.SetScene("stage")
    g.Trigger("wave_start")
    // Advance through all lines
    g.ClickAdvance()         // line 0 → line 1
    g.Tick(4.0)              // line 1 auto-advance → done
    // Try triggering again
    g.Trigger("wave_start")
    if g.current != nil {
        t.Fatal("once dialog should not repeat")
    }
}

func TestVMWhenIdle(t *testing.T) {
    g := makeTestGuide()
    vm := g.VM()
    if vm.HasDialog {
        t.Fatal("should have no dialog initially")
    }
    if !vm.Visible {
        t.Fatal("mascot should be visible")
    }
}
```

**Step 2: Run tests — expect FAIL**

**Step 3: Implement Guide**

```go
// mascot.go
package mascot

// Guide is the mascot assistant state machine.
type Guide struct {
    dialogs  []Dialog
    current  *Dialog
    lineIdx  int
    timer    float64
    shown    map[string]bool // dialog IDs already shown (for Once dialogs)
    scene    string
    visible  bool
}

// MascotVM is the view-model snapshot for rendering.
type MascotVM struct {
    Visible    bool
    HasDialog  bool
    Text       string
    Expression string
    CanClick   bool   // true if current line has no autoAdvance (click to continue)
}

// NewGuide creates a Guide with preloaded dialogs.
// shownIDs is the set of Once dialog IDs already shown (from persistence).
func NewGuide(dialogs []Dialog, shownIDs map[string]bool) *Guide {
    if shownIDs == nil {
        shownIDs = make(map[string]bool)
    }
    return &Guide{
        dialogs: dialogs,
        shown:   shownIDs,
        visible: true,
    }
}

// SetScene updates the current scene and triggers any scene_enter dialogs.
func (g *Guide) SetScene(name string) {
    if g.scene == name {
        return
    }
    g.scene = name
    g.Trigger("scene_enter")
}

// Trigger attempts to start a dialog matching the given event in the current scene.
func (g *Guide) Trigger(event string) {
    if g.current != nil {
        return // don't interrupt active dialog
    }
    var best *Dialog
    for i := range g.dialogs {
        d := &g.dialogs[i]
        if d.Trigger != event {
            continue
        }
        if d.Scene != g.scene && d.Scene != "*" {
            continue
        }
        if d.Once && g.shown[d.ID] {
            continue
        }
        if best == nil || d.Priority > best.Priority {
            best = d
        }
    }
    if best != nil {
        g.startDialog(best)
    }
}

// Tick advances auto-advance timers. dt is seconds.
func (g *Guide) Tick(dt float64) {
    if g.current == nil {
        return
    }
    line := g.currentLine()
    if line == nil {
        return
    }
    if line.AutoAdvance <= 0 {
        return
    }
    g.timer += dt
    if g.timer >= line.AutoAdvance {
        g.advanceLine()
    }
}

// ClickAdvance advances the dialog when the player clicks.
// Only works if the current line has no autoAdvance (i.e., click-to-continue).
func (g *Guide) ClickAdvance() {
    if g.current == nil {
        return
    }
    line := g.currentLine()
    if line == nil {
        return
    }
    if line.AutoAdvance > 0 {
        // Also allow click to skip auto-advance lines
    }
    g.advanceLine()
}

// VM returns a snapshot for the renderer.
func (g *Guide) VM() MascotVM {
    vm := MascotVM{
        Visible:    g.visible,
        Expression: "idle",
    }
    line := g.currentLine()
    if line != nil {
        vm.HasDialog = true
        vm.Text = line.Text
        vm.Expression = line.Expression
        vm.CanClick = line.AutoAdvance <= 0
    }
    return vm
}

// ShownIDs returns the set of dialog IDs that have been shown (for persistence).
func (g *Guide) ShownIDs() map[string]bool {
    cp := make(map[string]bool, len(g.shown))
    for k, v := range g.shown {
        cp[k] = v
    }
    return cp
}

// --- internal ---

func (g *Guide) startDialog(d *Dialog) {
    g.current = d
    g.lineIdx = 0
    g.timer = 0
}

func (g *Guide) currentLine() *Line {
    if g.current == nil || g.lineIdx >= len(g.current.Lines) {
        return nil
    }
    return &g.current.Lines[g.lineIdx]
}

func (g *Guide) advanceLine() {
    g.lineIdx++
    g.timer = 0
    if g.lineIdx >= len(g.current.Lines) {
        g.finishDialog()
    }
}

func (g *Guide) finishDialog() {
    if g.current != nil && g.current.Once {
        g.shown[g.current.ID] = true
    }
    g.current = nil
    g.lineIdx = 0
    g.timer = 0
}
```

**Step 4: Run tests — expect PASS**

```bash
go test ./internal/core/mascot/ -v -race
```

**Step 5: Commit**

```bash
git add internal/core/mascot/
git commit -m "feat(mascot): implement Guide state machine with trigger/advance/once"
```

---

### Task 4: Speech Bubble Component — `internal/render/hud/speech_bubble.go`

**Files:**
- Create: `internal/render/hud/speech_bubble.go`

**Note:** No unit test for pure render functions — verified visually. Follow existing HUD pattern (e.g., `tutorial_overlay.go`).

**Step 1: Implement speech bubble**

```go
// speech_bubble.go — Reusable speech bubble with tail triangle.
package hud

import (
    "image/color"

    "defense2/internal/render"
    "defense2/internal/render/draw"
    "defense2/internal/render/theme"

    "github.com/hajimehoshi/ebiten/v2"
)

// SpeechBubbleVM data for rendering a speech bubble.
type SpeechBubbleVM struct {
    Text    string
    AnchorX float32 // tail points to this X (mascot center)
    AnchorY float32 // tail points to this Y (mascot top)
    MaxW    float32 // max bubble width
}

// DrawSpeechBubble renders a rounded speech bubble with tail.
func DrawSpeechBubble(screen *ebiten.Image, vm SpeechBubbleVM) {
    fm := render.GlobalFont()
    if fm == nil || vm.Text == "" {
        return
    }

    const (
        padX    float32 = 14
        padY    float32 = 10
        radius  float32 = 10
        tailH   float32 = 10
        lineH   float64 = 16 // line height for wrapped text
    )

    // Wrap text.
    lines := wrapText(fm, vm.Text, float64(vm.MaxW-padX*2), theme.FontBody)
    if len(lines) == 0 {
        return
    }

    // Calculate bubble size.
    maxLineW := float32(0)
    for _, line := range lines {
        w := float32(fm.MeasureText(line, theme.FontBody))
        if w > maxLineW {
            maxLineW = w
        }
    }
    bubbleW := maxLineW + padX*2
    bubbleH := float32(len(lines))*float32(lineH) + padY*2

    // Position bubble above anchor point.
    bubbleX := vm.AnchorX - bubbleW/2
    bubbleY := vm.AnchorY - tailH - bubbleH

    // Clamp to screen.
    if bubbleX < 4 {
        bubbleX = 4
    }
    if bubbleX+bubbleW > float32(theme.CanvasW)-4 {
        bubbleX = float32(theme.CanvasW) - 4 - bubbleW
    }
    if bubbleY < 4 {
        bubbleY = 4
    }

    // Background.
    bgClr := color.RGBA{R: 255, G: 255, B: 255, A: 235}
    draw.RoundRect(screen, bubbleX, bubbleY, bubbleW, bubbleH, radius, bgClr)
    // Border.
    draw.StrokeRoundRect(screen, bubbleX, bubbleY, bubbleW, bubbleH, radius, 1.2,
        color.RGBA{R: 160, G: 140, B: 200, A: 180})

    // Tail triangle (pointing down to mascot).
    tailX := vm.AnchorX
    tailTopY := bubbleY + bubbleH
    draw.Triangle(screen,
        float32(tailX)-6, tailTopY-1,
        float32(tailX)+6, tailTopY-1,
        float32(tailX), tailTopY+tailH,
        bgClr)

    // Draw text lines.
    textClr := color.RGBA{R: 40, G: 35, B: 55, A: 255}
    textX := float64(bubbleX + padX)
    textY := float64(bubbleY+padY) + theme.FontBody/2
    for _, line := range lines {
        fm.DrawText(screen, line, textX, textY, theme.FontBody, textClr)
        textY += lineH
    }
}
```

**Important:** Check if `draw.Triangle` exists. If not, build the tail from two `draw.ThickLine` calls or a `draw.FilledRect` rotated. The simplest fallback is to skip the tail for now and just use a rounded rect — the bubble still works without a tail pointer.

**Step 2: Commit**

```bash
git add internal/render/hud/speech_bubble.go
git commit -m "feat(mascot): add speech bubble HUD component"
```

---

### Task 5: Mascot Overlay — `internal/render/hud/mascot_overlay.go`

**Files:**
- Create: `internal/render/hud/mascot_overlay.go`

**Step 1: Implement overlay renderer**

This renders the mascot sprite + speech bubble using the MascotVM from `internal/core/mascot/`.

```go
// mascot_overlay.go — Mascot assistant HUD overlay.
package hud

import (
    "math"

    "defense2/internal/core/game"
    "defense2/internal/render/draw"
    "defense2/internal/render/theme"

    "github.com/hajimehoshi/ebiten/v2"
)

// MascotOverlayVM is the view-model for the mascot overlay.
type MascotOverlayVM struct {
    Visible    bool
    HasDialog  bool
    Text       string
    Expression string
    CanClick   bool
    Sprite     *ebiten.Image // current expression sprite
    AnimTime   float64       // for idle bob animation
}

// MascotHitTest returns true if (x,y) is within the mascot clickable region.
func MascotHitTest(x, y float64) bool {
    bx, by, bw, bh := mascotBounds()
    return x >= float64(bx) && x <= float64(bx+bw) &&
        y >= float64(by) && y <= float64(by+bh)
}

// mascotBounds returns the bounding rect of the mascot sprite area.
func mascotBounds() (x, y, w, h float32) {
    const spriteW float32 = 80
    const spriteH float32 = 160
    x = float32(game.ScreenWidth) - spriteW - 12
    y = float32(game.ScreenHeight) - float32(theme.BottomMargin) - spriteH - 8
    return x, y, spriteW, spriteH
}

// DrawMascotOverlay renders the mascot character and optional speech bubble.
func DrawMascotOverlay(screen *ebiten.Image, vm MascotOverlayVM) {
    if !vm.Visible {
        return
    }

    mx, my, mw, mh := mascotBounds()
    cx := mx + mw/2
    cy := my + mh/2

    // Idle bob animation.
    bobY := float32(math.Sin(vm.AnimTime*2.5) * 2.0)

    // Draw sprite.
    if vm.Sprite != nil {
        draw.SpriteScaled(screen, vm.Sprite, float64(cx), float64(cy+bobY), float64(mh)/float64(vm.Sprite.Bounds().Dy()))
    } else {
        // Placeholder: simple silhouette when no sprite loaded.
        draw.RoundRect(screen, mx, my+bobY, mw, mh, 12,
            theme.SurfaceDim)
    }

    // Speech bubble.
    if vm.HasDialog && vm.Text != "" {
        DrawSpeechBubble(screen, SpeechBubbleVM{
            Text:    vm.Text,
            AnchorX: cx,
            AnchorY: my + bobY,
            MaxW:    260,
        })
    }
}
```

**Step 2: Commit**

```bash
git add internal/render/hud/mascot_overlay.go
git commit -m "feat(mascot): add mascot overlay renderer with sprite + bubble"
```

---

### Task 6: Sprite Export + Loading

**Files:**
- Create: `assets/mascot/mascot-idle-0.png` (export from SVG)
- Create: `internal/render/mascot_sprite.go` (sprite loader)

**Step 1: Export PNG from SVG**

Use any method to convert `assets/mascot/src/mascot-idle-0-bishoujo.svg` to PNG. If no conversion tool is available, the system will use the placeholder rect from Task 5 until PNGs are added.

For programmatic conversion (optional helper script):
```bash
# If rsvg-convert is available:
rsvg-convert -w 140 -h 280 assets/mascot/src/mascot-idle-0-bishoujo.svg -o assets/mascot/mascot-idle-0.png
# Or use Inkscape:
# inkscape --export-type=png --export-width=140 assets/mascot/src/mascot-idle-0-bishoujo.svg -o assets/mascot/mascot-idle-0.png
```

**Step 2: Create sprite loader**

```go
// mascot_sprite.go — Load mascot PNG sprites.
package render

import (
    "log"

    "defense2/internal/render/anim"
    "defense2/internal/render/sprite"
)

// AssetReader reads embedded files.
type AssetReader interface {
    ReadFile(name string) ([]byte, error)
}

// LoadMascotSprites loads mascot expression sprites into an Animator.
func LoadMascotSprites(fs AssetReader) *anim.Animator {
    a := anim.NewAnimator()

    expressions := []string{"idle", "talk", "happy", "surprised"}
    for _, expr := range expressions {
        path := "assets/mascot/mascot-" + expr + "-0.png"
        data, err := fs.ReadFile(path)
        if err != nil {
            continue // expression not available yet
        }
        img := sprite.Parse(data, 0, 0) // 0,0 = use native size
        if img != nil {
            a.AddAnim(expr, []*ebiten.Image{img}, 1, true)
        }
    }

    // Fallback: if no idle, warn.
    if !a.HasAnim("idle") {
        log.Println("[mascot] warning: no idle sprite found, using placeholder")
    }

    return a
}
```

**Note:** Check `sprite.Parse` signature — it may require width/height. If it only accepts explicit dimensions, use `image/png` decode directly (like warden loader does). Adjust based on actual `sprite.Parse` API.

**Step 3: Commit**

```bash
git add internal/render/mascot_sprite.go assets/mascot/
git commit -m "feat(mascot): add sprite loader + initial PNG assets"
```

---

### Task 7: Game Struct Integration

**Files:**
- Modify: `internal/scene/game.go`
- Modify: `internal/scene/scene.go` (add scene name awareness)

**Step 1: Add scene name tracking to Game**

The Game struct needs to know the current scene name for `Guide.SetScene()`. Add a `sceneName` string field, set it when switching scenes.

In `game.go`:
- Add field: `mascot *mascot.Guide`, `sceneName string`, `mascotAnim *anim.Animator`, `mascotTime float64`
- In `NewGame()`: load mascot dialogs + create Guide instance
- In `Update()`: call `g.mascot.SetScene(g.sceneName)` and `g.mascot.Tick(dt)`
- In `Draw()`: build `MascotOverlayVM` from `g.mascot.VM()` + `g.mascotAnim` and call `DrawMascotOverlay()`
- In `SwitchScene()`: detect scene name from type switch or add a `Namer` interface

**Scene name detection approach** — add method to scene interface or use type switch:

```go
func (g *Game) currentSceneName() string {
    switch g.current.(type) {
    case *TitleScene:
        return "title"
    case *SelectScene:
        return "select"
    case *CampaignSelectScene:
        return "campaign_select"
    case *StageScene:
        return "stage"
    case *ResultScene:
        return "result"
    case *SettingsScene:
        return "settings"
    default:
        return "unknown"
    }
}
```

**Step 2: Wire up Update**

```go
// In Game.Update(), after scene update:
const dt = 1.0 / 60.0
g.mascotTime += dt
g.mascot.SetScene(g.currentSceneName())
g.mascot.Tick(dt)
```

**Step 3: Wire up Draw**

```go
// In Game.Draw(), after scene draw, before transition overlay:
if !HeadlessMode {
    coreVM := g.mascot.VM()
    if g.mascotAnim != nil && g.mascotAnim.HasAnim(coreVM.Expression) {
        g.mascotAnim.Play(coreVM.Expression)
    }
    overlayVM := hud.MascotOverlayVM{
        Visible:    coreVM.Visible,
        HasDialog:  coreVM.HasDialog,
        Text:       coreVM.Text,
        Expression: coreVM.Expression,
        CanClick:   coreVM.CanClick,
        Sprite:     nil, // set below
        AnimTime:   g.mascotTime,
    }
    if g.mascotAnim != nil {
        overlayVM.Sprite = g.mascotAnim.CurrentImage()
    }
    hud.DrawMascotOverlay(screen, overlayVM)
}
```

**Step 4: Wire up click handling**

```go
// In Game.Update(), after mascot.Tick:
mx, my := draw.CursorPos()
if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
    if hud.MascotHitTest(float64(mx), float64(my)) {
        g.mascot.ClickAdvance()
    }
}
```

**Step 5: Run `make check-all`**

```bash
make check-all
```

**Step 6: Commit**

```bash
git add internal/scene/game.go
git commit -m "feat(mascot): integrate Guide into Game struct update/draw loop"
```

---

### Task 8: Starter Dialog Content

**Files:**
- Create: `config/mascot/dialogs-stage.json`
- Create: `config/mascot/dialogs-select.json`
- Create: `config/mascot/dialogs-common.json`
- Update: `config/mascot/dialogs-title.json` (more dialogs)

**Step 1: Write dialog content**

`dialogs-title.json`:
```json
[
  {
    "id": "title_welcome",
    "scene": "title",
    "trigger": "scene_enter",
    "lines": [
      {"text": "欢迎回来，指挥官！", "expression": "happy", "autoAdvance": 4},
      {"text": "准备好迎接新的挑战了吗？", "expression": "talk", "autoAdvance": 3}
    ],
    "once": false,
    "priority": 10
  }
]
```

`dialogs-select.json`:
```json
[
  {
    "id": "select_first_visit",
    "scene": "select",
    "trigger": "scene_enter",
    "lines": [
      {"text": "选择一张地图开始战斗吧！", "expression": "talk", "autoAdvance": 4},
      {"text": "建议先从左边的简单地图开始哦~", "expression": "happy", "autoAdvance": 4}
    ],
    "once": true,
    "priority": 10
  }
]
```

`dialogs-stage.json`:
```json
[
  {
    "id": "stage_first_enter",
    "scene": "stage",
    "trigger": "scene_enter",
    "lines": [
      {"text": "战场准备就绪！先建一座防御塔吧。", "expression": "talk", "autoAdvance": 5}
    ],
    "once": true,
    "priority": 10
  },
  {
    "id": "stage_wave_start",
    "scene": "stage",
    "trigger": "wave_start",
    "lines": [
      {"text": "敌人来了！注意防线！", "expression": "surprised", "autoAdvance": 3}
    ],
    "once": true,
    "priority": 5
  },
  {
    "id": "stage_first_upgrade",
    "scene": "stage",
    "trigger": "upgrade",
    "lines": [
      {"text": "升级成功！塔变强了呢~", "expression": "happy", "autoAdvance": 3}
    ],
    "once": true,
    "priority": 5
  }
]
```

`dialogs-common.json`:
```json
[
  {
    "id": "idle_tip_1",
    "scene": "*",
    "trigger": "idle_timeout",
    "lines": [
      {"text": "有什么不明白的，点我就好~", "expression": "talk", "autoAdvance": 4}
    ],
    "once": false,
    "priority": 1
  }
]
```

**Step 2: Verify JSON parses correctly**

```bash
go test ./internal/core/mascot/ -v -race
```

**Step 3: Commit**

```bash
git add config/mascot/
git commit -m "feat(mascot): add starter dialog content for all scenes"
```

---

### Task 9: Persistence for Once Dialogs

**Files:**
- Modify: `internal/core/persistence/progress.go` — add `MascotShown` field
- Modify: `internal/scene/game.go` — load/save shown IDs

**Step 1: Add field to Progress**

In `progress.go`, add to `Progress` struct:

```go
MascotShown map[string]bool `json:"mascotShown"` // mascot Once dialog IDs already shown
```

In `NewProgress()`, initialize:

```go
MascotShown: make(map[string]bool),
```

**Step 2: Add ProgressManager methods**

```go
func (pm *ProgressManager) MascotShownIDs() map[string]bool {
    if pm.progress.MascotShown == nil {
        pm.progress.MascotShown = make(map[string]bool)
    }
    return pm.progress.MascotShown
}

func (pm *ProgressManager) SaveMascotShown(ids map[string]bool) {
    pm.progress.MascotShown = ids
    pm.save()
}
```

**Step 3: Wire into Game**

In `NewGame()`, pass `pm.MascotShownIDs()` to `mascot.NewGuide()`.
On scene transitions or periodic save, call `pm.SaveMascotShown(g.mascot.ShownIDs())`.

**Step 4: Run tests**

```bash
make check-all
```

**Step 5: Commit**

```bash
git add internal/core/persistence/progress.go internal/scene/game.go
git commit -m "feat(mascot): persist Once dialog shown state"
```

---

### Task 10: Visual Verification + Polish

**Step 1: Run the game**

```bash
make run
```

**Step 2: Verify**

- [ ] Mascot appears on title screen (bottom-right)
- [ ] Speech bubble shows "欢迎回来，指挥官！" on title enter
- [ ] Dialog auto-advances after 4 seconds
- [ ] Mascot appears on select screen with first-visit tip
- [ ] Mascot appears in stage with build tip
- [ ] Clicking mascot/bubble advances dialog
- [ ] Placeholder rect shows if no PNG sprite yet
- [ ] No crash in headless/autoplay mode (mascot skipped)

**Step 3: Fix any rendering issues**

- Adjust bubble position, size, font colors
- Ensure bubble doesn't overlap ActionBar
- Ensure mascot doesn't block tower placement areas

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(mascot): visual polish and position adjustments"
```

---

## Execution Dependencies

```
Task 1 (types) ──→ Task 2 (loader) ──→ Task 3 (state machine) ──→ Task 7 (integration)
                                                                  ↗
Task 4 (bubble) ──→ Task 5 (overlay) ──→ Task 6 (sprites) ──────
                                                                  ↘
                                         Task 8 (content) ──→ Task 9 (persistence) ──→ Task 10 (verify)
```

Tasks 1-3 and Tasks 4-6 can run **in parallel**.
Task 7 requires both chains complete.
Tasks 8-10 are sequential after Task 7.
