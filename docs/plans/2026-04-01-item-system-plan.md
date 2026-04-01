# Item System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a consumable item system with 6 item types (3 attributes x base/potential) that boost tower stats via drag-and-drop from a bottom toolbar.

**Architecture:** New `internal/core/item/` package for data model (pure logic, no rendering deps). New `hud/action_bar.go` for bottom toolbar, `hud/item_panel.go` for item grid popup. Two new interactModes (`modeItemPanel`, `modeItemDrag`) in stage_types.go. Gesture system extended with item-drag state tracked in stage.go.

**Tech Stack:** Go / Ebitengine, existing draw/theme/hud packages.

---

### Task 1: Item Data Model

**Files:**
- Create: `internal/core/item/item.go`
- Test: `internal/core/item/item_test.go`

**Step 1: Write the failing test**

```go
// internal/core/item/item_test.go
package item

import "testing"

func TestInventoryInitial(t *testing.T) {
	inv := NewInventory(5)
	for _, k := range AllKinds {
		if inv.Count(k) != 5 {
			t.Errorf("kind %d: got %d, want 5", k, inv.Count(k))
		}
	}
}

func TestUseDecrementsAndFails(t *testing.T) {
	inv := NewInventory(1)
	if !inv.Use(KindBaseDamage) {
		t.Fatal("first use should succeed")
	}
	if inv.Count(KindBaseDamage) != 0 {
		t.Fatal("count should be 0 after use")
	}
	if inv.Use(KindBaseDamage) {
		t.Fatal("use on empty should fail")
	}
}

func TestItemDefs(t *testing.T) {
	for _, k := range AllKinds {
		d := Defs[k]
		if d.Name == "" {
			t.Errorf("kind %d has empty name", k)
		}
		if d.BoostVal == 0 {
			t.Errorf("kind %d has zero boost", k)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go test ./internal/core/item/ -v -count=1`
Expected: FAIL (package does not exist)

**Step 3: Write implementation**

```go
// internal/core/item/item.go
package item

import "image/color"

// Kind identifies an item type.
type Kind int

const (
	KindBaseDamage      Kind = iota // 攻击磨石 — +BaseDamage
	KindPotentialDamage             // 攻击秘卷 — +PotentialDamage
	KindBaseSpeed                   // 速射齿轮 — +BaseSpeed
	KindPotentialSpeed              // 速射秘卷 — +PotentialSpeed
	KindBaseRange                   // 瞄准镜片 — +BaseRange
	KindPotentialRange              // 瞄准秘卷 — +PotentialRange
	KindCount                       // sentinel
)

// AllKinds for iteration.
var AllKinds = [...]Kind{
	KindBaseDamage, KindPotentialDamage,
	KindBaseSpeed, KindPotentialSpeed,
	KindBaseRange, KindPotentialRange,
}

// Def describes a single item type.
type Def struct {
	Kind     Kind
	Name     string     // 中文名
	BoostVal float64    // 绝对加成值
	Color    color.RGBA // 图标色
}

// Defs is the item definition table indexed by Kind.
var Defs = [KindCount]Def{
	KindBaseDamage:      {KindBaseDamage, "攻击磨石", 2, color.RGBA{R: 239, G: 68, B: 68, A: 255}},
	KindPotentialDamage: {KindPotentialDamage, "攻击秘卷", 3, color.RGBA{R: 185, G: 28, B: 28, A: 255}},
	KindBaseSpeed:       {KindBaseSpeed, "速射齿轮", 0.15, color.RGBA{R: 250, G: 204, B: 21, A: 255}},
	KindPotentialSpeed:  {KindPotentialSpeed, "速射秘卷", 0.2, color.RGBA{R: 202, G: 138, B: 4, A: 255}},
	KindBaseRange:       {KindBaseRange, "瞄准镜片", 12, color.RGBA{R: 59, G: 130, B: 246, A: 255}},
	KindPotentialRange:  {KindPotentialRange, "瞄准秘卷", 18, color.RGBA{R: 30, G: 64, B: 175, A: 255}},
}

// Inventory tracks item counts.
type Inventory struct {
	counts [KindCount]int
}

// NewInventory creates an inventory with `n` of each item.
func NewInventory(n int) *Inventory {
	inv := &Inventory{}
	for i := range inv.counts {
		inv.counts[i] = n
	}
	return inv
}

// Count returns current quantity.
func (inv *Inventory) Count(k Kind) int { return inv.counts[k] }

// Use consumes one item. Returns false if none left.
func (inv *Inventory) Use(k Kind) bool {
	if inv.counts[k] <= 0 {
		return false
	}
	inv.counts[k]--
	return true
}

// TotalCount returns sum of all items.
func (inv *Inventory) TotalCount() int {
	total := 0
	for _, c := range inv.counts {
		total += c
	}
	return total
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go test ./internal/core/item/ -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/item/
git commit -m "feat: add item data model with 6 item types and inventory"
```

---

### Task 2: ApplyItem — Wire Items to Tower Stats

**Files:**
- Create: `internal/core/item/apply.go`
- Test: `internal/core/item/apply_test.go`

**Step 1: Write the failing test**

```go
// internal/core/item/apply_test.go
package item

import (
	"testing"

	"defense2/internal/core/tower"
)

func TestApplyItem(t *testing.T) {
	tw := &tower.Tower{
		BaseDamage: 10, PotentialDamage: 15,
		BaseSpeed: 1.0, PotentialSpeed: 0.5,
		BaseRange: 100, PotentialRange: 50,
	}

	tests := []struct {
		kind  Kind
		field string
		want  float64
	}{
		{KindBaseDamage, "BaseDamage", 12},
		{KindPotentialDamage, "PotentialDamage", 18},
		{KindBaseSpeed, "BaseSpeed", 1.15},
		{KindPotentialSpeed, "PotentialSpeed", 0.7},
		{KindBaseRange, "BaseRange", 112},
		{KindPotentialRange, "PotentialRange", 68},
	}

	for _, tt := range tests {
		ApplyItem(tw, tt.kind)
	}

	if tw.BaseDamage != 12 {
		t.Errorf("BaseDamage = %v, want 12", tw.BaseDamage)
	}
	if tw.PotentialDamage != 18 {
		t.Errorf("PotentialDamage = %v, want 18", tw.PotentialDamage)
	}
	if tw.BaseRange != 112 {
		t.Errorf("BaseRange = %v, want 112", tw.BaseRange)
	}
	if tw.PotentialRange != 68 {
		t.Errorf("PotentialRange = %v, want 68", tw.PotentialRange)
	}
	// float comparison with tolerance for speed
	if diff := tw.BaseSpeed - 1.15; diff > 0.001 || diff < -0.001 {
		t.Errorf("BaseSpeed = %v, want ~1.15", tw.BaseSpeed)
	}
	if diff := tw.PotentialSpeed - 0.7; diff > 0.001 || diff < -0.001 {
		t.Errorf("PotentialSpeed = %v, want ~0.7", tw.PotentialSpeed)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go test ./internal/core/item/ -v -count=1 -run TestApplyItem`
Expected: FAIL

**Step 3: Write implementation**

```go
// internal/core/item/apply.go
package item

import "defense2/internal/core/tower"

// ApplyItem adds the item's boost value to the tower's base/potential field
// and recalculates effective stats.
func ApplyItem(t *tower.Tower, k Kind) {
	v := Defs[k].BoostVal
	switch k {
	case KindBaseDamage:
		t.BaseDamage += v
	case KindPotentialDamage:
		t.PotentialDamage += v
	case KindBaseSpeed:
		t.BaseSpeed += v
	case KindPotentialSpeed:
		t.PotentialSpeed += v
	case KindBaseRange:
		t.BaseRange += v
	case KindPotentialRange:
		t.PotentialRange += v
	}
	t.RecalcStats()
}
```

**Step 4: Run test**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go test ./internal/core/item/ -v -count=1`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/item/apply.go internal/core/item/apply_test.go
git commit -m "feat: add ApplyItem to boost tower base/potential stats"
```

---

### Task 3: Add interactModes + Inventory to StageScene

**Files:**
- Modify: `internal/scene/stage_types.go` (add `modeItemPanel`, `modeItemDrag`)
- Modify: `internal/scene/stage.go` (add `inventory`, `dragItem*` fields, init inventory in constructor)

**Step 1: Add modes to stage_types.go**

Add after `modeUpgrade`:
```go
	modeItemPanel  // 道具面板打开
	modeItemDrag   // 拖拽道具中
```

**Step 2: Add fields to StageScene struct in stage.go**

Add after `wardenCfg` field area:
```go
	// 道具系统
	inventory     *item.Inventory // 道具背包
	dragItemKind  item.Kind       // 当前拖拽的道具类型
	dragItemActive bool           // 是否正在拖拽道具
	itemPanelOpen  bool           // 道具面板是否打开
```

**Step 3: Initialize inventory in NewStageScene (or NewStageSceneWithOpts)**

Find where `gold` is set and add nearby:
```go
s.inventory = item.NewInventory(5)
```

Add import: `"defense2/internal/core/item"`

**Step 4: Verify build**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./...`
Expected: BUILD OK

**Step 5: Commit**

```bash
git add internal/scene/stage_types.go internal/scene/stage.go
git commit -m "feat: add item inventory and interaction modes to StageScene"
```

---

### Task 4: ActionBar HUD Component

**Files:**
- Create: `internal/render/hud/action_bar.go`
- Modify: `internal/render/theme/layout.go` (add ActionBar constants)
- Modify: `internal/render/theme/colors.go` (add ActionBar colors)

**Step 1: Add layout constants to theme/layout.go**

```go
// ---------------------------------------------------------------------------
// ActionBar (bottom center toolbar)
// ---------------------------------------------------------------------------

const (
	ActionBarH      = 40
	ActionBarBtnW   = 80
	ActionBarBtnH   = 34
	ActionBarBtnGap = 8
	ActionBarBtnR   = 12
)
```

**Step 2: Add colors to theme/colors.go**

```go
// ---------------------------------------------------------------------------
// ActionBar
// ---------------------------------------------------------------------------

var (
	ActionBarBg     = rgba(15, 23, 42, 200)
	ActionBarBorder = rgba(255, 255, 255, 20)
)
```

**Step 3: Create action_bar.go**

```go
// action_bar.go — Bottom center action toolbar.
// Displays [Build] [Items] buttons. Future-expandable.
package hud

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ActionBarData holds runtime data for the action bar.
type ActionBarData struct {
	BuildActive bool // build mode highlighted
	ItemActive  bool // item panel highlighted
	ItemTotal   int  // total items remaining (badge)
}

var lastActionBarBtnRects []ui.Rect
var lastActionBarBtnNames []string

// DrawActionBar renders the bottom center toolbar.
func DrawActionBar(screen *ebiten.Image, d ActionBarData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		btnH   = float32(theme.ActionBarBtnH)
		btnGap = float32(theme.ActionBarBtnGap)
		btnR   = float32(theme.ActionBarBtnR)
		barH   = float32(theme.ActionBarH)
	)

	type btnDef struct {
		name  string
		label string
		clr   color.RGBA
	}

	buildClr := theme.ToneSecondary
	if d.BuildActive {
		buildClr = theme.TonePrimary
	}
	itemClr := theme.ToneSecondary
	if d.ItemActive {
		itemClr = theme.ToneAccent
	}

	btns := []btnDef{
		{"build", "造塔", buildClr},
		{"items", "道具", itemClr},
	}

	items := make([]ui.ButtonRowItem, len(btns))
	names := make([]string, len(btns))
	for i, b := range btns {
		items[i] = ui.ButtonRowItem{Label: b.label, Color: b.clr}
		names[i] = b.name
	}

	// Calculate total width
	const btnPadX float32 = 16
	totalW := float32(0)
	for _, it := range items {
		tw := float32(fm.MeasureText(it.Label, theme.FontH2))
		w := tw + btnPadX*2
		if w < 60 {
			w = 60
		}
		totalW += w
	}
	totalW += float32(len(items)-1) * btnGap

	// Pill background
	pillW := totalW + 24 // 12px padding each side
	pillH := barH
	pillX := (float32(theme.CanvasW) - pillW) / 2
	pillY := float32(theme.CanvasH) - float32(theme.BottomMargin) - pillH

	draw.RoundRect(screen, pillX, pillY, pillW, pillH, 16, theme.ActionBarBg)
	draw.StrokeRoundRect(screen, pillX, pillY, pillW, pillH, 16, 1, theme.ActionBarBorder)

	// Buttons centered inside pill
	btnArea := ui.Rect{
		X: pillX + 12,
		Y: pillY + (pillH-btnH)/2,
		W: totalW,
		H: btnH,
	}

	result := ui.DrawButtonRowAutoWidth(screen, btnArea, items, ui.ButtonRowStyle{
		Height:   btnH,
		Gap:      btnGap,
		Radius:   btnR,
		FontSize: theme.FontH2,
	})
	lastActionBarBtnRects = result.Rects
	lastActionBarBtnNames = names

	// Item count badge on the items button
	if d.ItemTotal > 0 && len(result.Rects) > 1 {
		// 在道具按钮右上角画 badge 数字可以后续加，先不画
	}
}

// ActionBarHitTest returns the button name at (px,py), or "".
func ActionBarHitTest(px, py float32) string {
	idx := ui.HitTestButtonRow(lastActionBarBtnRects, float64(px), float64(py))
	if idx >= 0 && idx < len(lastActionBarBtnNames) {
		return lastActionBarBtnNames[idx]
	}
	return ""
}

// ActionBarRect returns the bounding rect of the action bar (for UI region detection).
func ActionBarRect() (x, y, w, h float32) {
	if len(lastActionBarBtnRects) == 0 {
		return 0, 0, 0, 0
	}
	first := lastActionBarBtnRects[0]
	last := lastActionBarBtnRects[len(lastActionBarBtnRects)-1]
	ax := first.X - 12
	ay := first.Y - 3
	aw := (last.X + last.W) - first.X + 24
	ah := float32(theme.ActionBarH)
	return ax, ay, aw, ah
}
```

**Step 4: Verify build**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./...`
Expected: BUILD OK

**Step 5: Commit**

```bash
git add internal/render/hud/action_bar.go internal/render/theme/layout.go internal/render/theme/colors.go
git commit -m "feat: add ActionBar HUD component (bottom toolbar)"
```

---

### Task 5: Item Panel HUD Component

**Files:**
- Create: `internal/render/hud/item_panel.go`

**Step 1: Create item_panel.go**

```go
// item_panel.go — Item inventory popup panel.
// 2-column x 3-row grid of item cards. Rendered above ActionBar.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ItemCardVM represents one item card in the panel.
type ItemCardVM struct {
	Name  string
	Count int
	Color color.RGBA // item color
	Kind  int        // item.Kind as int (avoid importing item in hud)
}

// ItemPanelData holds data for the item panel.
type ItemPanelData struct {
	Cards   []ItemCardVM
	Visible bool
}

// Item panel layout constants.
const (
	ipCols   = 2
	ipRows   = 3
	ipCardW  = float32(110)
	ipCardH  = float32(50)
	ipCardGap = float32(8)
	ipCardR  = float32(8)
	ipPadX   = float32(14)
	ipPadY   = float32(12)
	ipTitleH = float32(28)
)

type itemPanelMetrics struct {
	panelX, panelY, panelW, panelH float32
	gridX, gridY                    float32
}

func calcItemPanelMetrics() itemPanelMetrics {
	gridW := float32(ipCols)*ipCardW + float32(ipCols-1)*ipCardGap
	gridH := float32(ipRows)*ipCardH + float32(ipRows-1)*ipCardGap

	panelW := gridW + ipPadX*2
	panelH := ipTitleH + gridH + ipPadY*2

	panelX := (float32(theme.CanvasW) - panelW) / 2
	// Position above ActionBar
	panelY := float32(theme.CanvasH) - float32(theme.BottomMargin) - float32(theme.ActionBarH) - panelH - 4

	return itemPanelMetrics{
		panelX: panelX, panelY: panelY,
		panelW: panelW, panelH: panelH,
		gridX: panelX + ipPadX,
		gridY: panelY + ipTitleH + ipPadY,
	}
}

// DrawItemPanel renders the item panel popup.
func DrawItemPanel(screen *ebiten.Image, d ItemPanelData) {
	if !d.Visible || len(d.Cards) == 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	m := calcItemPanelMetrics()

	// Panel background
	draw.RoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.PanelRadius), theme.PanelBg)
	draw.StrokeRoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.PanelRadius), 1, theme.PanelBorder)

	// Title
	fm.DrawBoldText(screen, "道具", float64(m.panelX)+float64(ipPadX), float64(m.panelY)+6, theme.FontLG, theme.TextTitle)

	// Cards
	for i, card := range d.Cards {
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)

		// Card background
		cardBg := color.RGBA{R: 30, G: 40, B: 60, A: 200}
		if card.Count <= 0 {
			cardBg = color.RGBA{R: 25, G: 30, B: 40, A: 120} // dimmed
		}
		draw.RoundRect(screen, cx, cy, ipCardW, ipCardH, ipCardR, cardBg)

		// Color indicator circle
		circleX := float32(float64(cx) + 16)
		circleY := cy + ipCardH/2
		circleClr := card.Color
		if card.Count <= 0 {
			circleClr.A = 80
		}
		draw.FilledCircle(screen, circleX, circleY, 8, circleClr)

		// Name
		nameClr := theme.TextBody
		if card.Count <= 0 {
			nameClr = theme.TextLocked
		}
		fm.DrawText(screen, card.Name, float64(cx)+30, float64(cy)+8, theme.FontSM, nameClr)

		// Count
		countTxt := fmt.Sprintf("x%d", card.Count)
		countClr := theme.TextMuted
		if card.Count <= 0 {
			countClr = theme.TextLocked
		}
		fm.DrawText(screen, countTxt, float64(cx)+30, float64(cy)+26, theme.FontXS, countClr)
	}
}

// ItemPanelHitTest returns the card index at (px,py), or -1.
// Only returns cards with count > 0.
func ItemPanelHitTest(px, py float32, cards []ItemCardVM) int {
	if len(cards) == 0 {
		return -1
	}
	m := calcItemPanelMetrics()

	// Quick bounds check
	if px < m.panelX || px > m.panelX+m.panelW || py < m.panelY || py > m.panelY+m.panelH {
		return -1
	}

	for i, card := range cards {
		if card.Count <= 0 {
			continue
		}
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)
		if px >= cx && px <= cx+ipCardW && py >= cy && py <= cy+ipCardH {
			return i
		}
	}
	return -1
}

// ItemPanelContains returns true if (px,py) is inside the item panel.
func ItemPanelContains(px, py float32) bool {
	m := calcItemPanelMetrics()
	return px >= m.panelX && px <= m.panelX+m.panelW &&
		py >= m.panelY && py <= m.panelY+m.panelH
}

// DrawDragItem renders a floating item icon at the cursor position during drag.
func DrawDragItem(screen *ebiten.Image, x, y float32, clr color.RGBA, name string) {
	// Semi-transparent circle + name
	draw.FilledCircle(screen, x, y, 14, clr)
	draw.CircleOutline(screen, x, y, 14, 2, color.RGBA{R: 255, G: 255, B: 255, A: 180})
	if fm := render.GlobalFont(); fm != nil {
		fm.DrawCenteredText(screen, name, float64(x), float64(y)+20, theme.FontXS, color.White)
	}
}
```

**Step 2: Verify build**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./...`
Expected: BUILD OK

**Step 3: Commit**

```bash
git add internal/render/hud/item_panel.go
git commit -m "feat: add ItemPanel HUD component for item grid popup"
```

---

### Task 6: Wire ActionBar + ItemPanel into Stage Draw

**Files:**
- Modify: `internal/scene/stage.go` (Draw section: add ActionBar + ItemPanel + DragItem rendering)

**Step 1: In drawScene(), add ActionBar draw call**

After `DrawTopBar` call (~line 1449), add:
```go
	// HUD：底部动作栏
	hud.DrawActionBar(screen, hud.ActionBarData{
		BuildActive: s.imode == modeBuildMenu || s.imode == modeBuildPlace,
		ItemActive:  s.imode == modeItemPanel || s.imode == modeItemDrag,
		ItemTotal:   s.inventory.TotalCount(),
	})
```

**Step 2: Add ItemPanel draw after BuildMenu draw**

After `DrawBuildMenu` call (~line 1452), add:
```go
	// HUD：道具面板
	hud.DrawItemPanel(screen, s.buildItemPanelData())
```

**Step 3: Add drag indicator draw before Toast**

Before `hud.DrawToast(screen)` (~line 1528), add:
```go
	// 道具拖拽指示器
	if s.dragItemActive {
		mx, my := s.gesture.CursorPos()
		def := item.Defs[s.dragItemKind]
		hud.DrawDragItem(screen, float32(mx), float32(my), def.Color, def.Name)
	}
```

**Step 4: Add buildItemPanelData helper**

Add in stage.go:
```go
func (s *StageScene) buildItemPanelData() hud.ItemPanelData {
	if s.imode != modeItemPanel && s.imode != modeItemDrag {
		return hud.ItemPanelData{Visible: false}
	}
	cards := make([]hud.ItemCardVM, item.KindCount)
	for _, k := range item.AllKinds {
		cards[k] = hud.ItemCardVM{
			Name:  item.Defs[k].Name,
			Count: s.inventory.Count(k),
			Color: item.Defs[k].Color,
			Kind:  int(k),
		}
	}
	return hud.ItemPanelData{Cards: cards, Visible: true}
}
```

**Step 5: Verify build**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./...`
Expected: BUILD OK

**Step 6: Commit**

```bash
git add internal/scene/stage.go
git commit -m "feat: wire ActionBar and ItemPanel into stage rendering"
```

---

### Task 7: Wire ActionBar + ItemPanel Input Handling

**Files:**
- Modify: `internal/scene/stage_input.go` (add ActionBar hit tests, item panel tap, drag logic)

**Step 1: Move "build" button handling from TopBar to ActionBar**

In `handleInput()`, after TopBar button checks, add ActionBar check:

```go
	// ActionBar 按钮
	actionBtn := hud.ActionBarHitTest(ftx, fty)
	switch actionBtn {
	case "build":
		if s.imode == modeBuildMenu {
			s.imode = modeIdle
		} else {
			s.imode = modeBuildMenu
			s.selectedTower = nil
			s.wardenPanelOpen = false
			s.itemPanelOpen = false
		}
		return
	case "items":
		if s.imode == modeItemPanel {
			s.imode = modeIdle
			s.itemPanelOpen = false
		} else {
			s.imode = modeItemPanel
			s.itemPanelOpen = true
			s.selectedTower = nil
			s.wardenPanelOpen = false
		}
		return
	}
```

**Step 2: Add modeItemPanel tap handling in switch**

In the mode dispatch switch, add:
```go
	case modeItemPanel:
		cards := s.buildItemPanelCards()
		idx := hud.ItemPanelHitTest(ftx, fty, cards)
		if idx >= 0 {
			// Start drag
			s.dragItemKind = item.Kind(idx)
			s.dragItemActive = true
			s.imode = modeItemDrag
		} else if !hud.ItemPanelContains(ftx, fty) {
			s.imode = modeIdle
			s.itemPanelOpen = false
		}
```

**Step 3: Handle drag in gesture Update section**

For `modeItemDrag`, the gesture system needs to track "drag from item". Since the current Gesture only handles camera drag, we handle item drag separately:

At the top of `handleInput()`, add a special path for `modeItemDrag`:
```go
	if s.imode == modeItemDrag {
		g.DragEnabled = false
		g.Update()
		if g.JustTapped() {
			// Released — check if on a tower
			tapX, tapY := g.TapPos()
			wtx, wty := s.screenToWorld(tapX, tapY)
			target := s.towerAtPixel(wtx, wty)
			if target != nil {
				item.ApplyItem(target, s.dragItemKind)
				s.inventory.Use(s.dragItemKind)
				hud.ShowToast(item.Defs[s.dragItemKind].Name + " → " + target.Label)
			}
			s.dragItemActive = false
			s.imode = modeItemPanel
		}
		// ESC cancel
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.dragItemActive = false
			s.imode = modeItemPanel
		}
		return
	}
```

Wait — the Gesture treats mouse-up as a Tap only if the cursor didn't move. For drag, we need a different approach. The item drag should track mouse-down on the card, follow the cursor while held, and apply on release over a tower.

**Revised approach:** Since the current gesture system only produces Tap on short press, we need to handle item drag as a custom state:

- In `modeItemPanel`: when JustTapped on a card, switch to `modeItemDrag` and set `dragItemActive=true`
- In `modeItemDrag`: skip normal gesture, directly read mouse/touch position. On next JustTapped (mouse release), apply to tower. This is simpler: click to pick up → click tower to apply → ESC to cancel. This is actually the "click-to-select + click-to-apply" pattern which is simpler and maps well to the existing gesture system.

Since the user chose "drag release", but the current gesture system doesn't support drag-from-UI (UI taps cancel on drag), we'll implement it as: **press card → enter drag mode → release anywhere = apply or cancel**. We need to bypass the gesture's anti-drag-on-UI behavior for this mode.

**Simplest workable approach**: Use the raw `ebiten.IsMouseButtonPressed` / `inpututil.IsMouseButtonJustReleased` in `modeItemDrag` to detect mouse-up directly:

```go
	if s.imode == modeItemDrag && s.dragItemActive {
		// Skip normal gesture — track raw input for drag release
		g.DragEnabled = false
		g.Update() // still update cursor pos

		// Check for mouse/touch release
		released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
		if !released {
			// Check touch release
			for _, id := range inpututil.AppendJustReleasedTouchIDs(nil) {
				_ = id
				released = true
				break
			}
		}

		if released {
			mx, my := g.CursorPos()
			wtx, wty := s.screenToWorld(mx, my)
			target := s.towerAtPixel(wtx, wty)
			if target != nil {
				item.ApplyItem(target, s.dragItemKind)
				s.inventory.Use(s.dragItemKind)
				hud.ShowToast(item.Defs[s.dragItemKind].Name + " → " + target.Label)
			}
			s.dragItemActive = false
			s.imode = modeItemPanel
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.dragItemActive = false
			s.imode = modeItemPanel
		}
		return
	}
```

And for entering drag from `modeItemPanel`, replace tap detection with press detection:
```go
	// In modeItemPanel: detect press-down on a card to start drag
	if s.imode == modeItemPanel {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
			mx, my := g.CursorPos()
			cards := s.buildItemPanelCards()
			idx := hud.ItemPanelHitTest(float32(mx), float32(my), cards)
			if idx >= 0 {
				s.dragItemKind = item.Kind(idx)
				s.dragItemActive = true
				s.imode = modeItemDrag
				return
			}
		}
	}
```

**Step 4: Add ESC handling for modeItemPanel/modeItemDrag**

In the ESC switch:
```go
	case modeItemPanel:
		s.imode = modeIdle
		s.itemPanelOpen = false
		s.dragItemActive = false
	case modeItemDrag:
		s.dragItemActive = false
		s.imode = modeItemPanel
```

**Step 5: Add helper method**

```go
func (s *StageScene) buildItemPanelCards() []hud.ItemCardVM {
	cards := make([]hud.ItemCardVM, item.KindCount)
	for _, k := range item.AllKinds {
		cards[k] = hud.ItemCardVM{
			Name:  item.Defs[k].Name,
			Count: s.inventory.Count(k),
			Color: item.Defs[k].Color,
			Kind:  int(k),
		}
	}
	return cards
}
```

**Step 6: Update IsOnUI to include ActionBar region**

In `newStageGesture()`:
```go
	// Add ActionBar region
	abx, aby, abw, abh := hud.ActionBarRect()
	if abw > 0 && fx >= abx && fx <= abx+abw && fy >= aby && fy <= aby+abh {
		return true
	}
```

**Step 7: Remove "build" from TopBar**

In `top_bar.go`, remove the `btns = append(btns, btnDef{"build", "造塔", buildTone})` line. The build button now lives in ActionBar.

**Step 8: Verify build + manual test**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./...`
Expected: BUILD OK

**Step 9: Commit**

```bash
git add internal/scene/stage_input.go internal/scene/stage.go internal/render/hud/top_bar.go
git commit -m "feat: wire item drag-and-drop interaction into stage input"
```

---

### Task 8: Integration Test + Polish

**Files:**
- Run full test suite
- Fix any build/test issues

**Step 1: Run all tests**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go test ./internal/core/item/ -v -count=1 -race`
Expected: PASS

**Step 2: Run build**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && go build ./cmd/game/`
Expected: BUILD OK

**Step 3: Run lint if available**

Run: `cd /Users/yushion/Games/defense2/.claude/worktrees/radiant-sniffing-anchor && make lint 2>/dev/null || echo "lint not configured"`

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete item system with drag-and-drop"
```
