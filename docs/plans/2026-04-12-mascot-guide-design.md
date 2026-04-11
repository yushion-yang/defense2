# Mascot Guide System Design

Date: 2026-04-12

## Goal

Add a visual anime-style mascot assistant that appears across all game scenes,
providing newbie guidance, tutorial tips, and easter egg interactions via speech
bubbles.

## Architecture

### Package Layout

```
config/mascot/                  # Dialog content (JSON, per-scene)
  dialogs-title.json
  dialogs-stage.json
  dialogs-select.json
  dialogs-common.json           # Shared idle/easter-egg lines

internal/core/mascot/           # Core logic (zero render dependency)
  mascot.go                     # Guide state machine
  dialog.go                     # Dialog/Line data types
  loader.go                     # JSON loader

internal/render/hud/
  mascot_overlay.go             # MascotVM + DrawMascotOverlay()
  speech_bubble.go              # Reusable speech bubble component

assets/mascot/                  # PNG sprite frames
  src/                          # SVG source files
  mascot-idle-0.png             # Bishoujo style, ~140x280 source
  mascot-talk-0.png
  mascot-happy-0.png
  mascot-surprised-0.png
```

### Data Structures

```go
// Dialog represents a triggered conversation (1+ lines).
type Dialog struct {
    ID       string  `json:"id"`        // e.g. "title_welcome"
    Scene    string  `json:"scene"`     // "title"/"stage"/"select"/"*"
    Trigger  string  `json:"trigger"`   // event name or "scene_enter"
    Lines    []Line  `json:"lines"`
    Once     bool    `json:"once"`      // show only once (persisted)
    Priority int     `json:"priority"`  // higher wins on conflict
}

type Line struct {
    Text        string  `json:"text"`
    Expression  string  `json:"expression"`    // "idle"/"happy"/"talk"/"surprised"
    AutoAdvance float64 `json:"autoAdvance"`   // seconds; 0 = click to advance
}
```

### Guide State Machine

```go
type Guide struct {
    dialogs    map[string][]Dialog   // scene -> dialogs
    current    *Dialog
    lineIdx    int
    timer      float64
    shown      map[string]bool       // persisted once-shown set
    scene      string
    visible    bool
    idleTimer  float64               // triggers random idle chat
}

// Key methods:
// SetScene(name)        — match scene_enter dialogs
// Trigger(event)        — match event-triggered dialogs
// Tick(dt)              — auto-advance timer + idle timer
// ClickAdvance()        — user click/tap to advance line
// VM() MascotVM         — snapshot for rendering
// ShownIDs() []string   — for persistence
```

### Integration Point (Game struct)

Guide lives on the Game struct, above all scenes. No per-scene modification
needed for basic operation. Scenes can call `Guide.Trigger()` for context-
specific events via the Switcher interface (new optional method) or via
EventBus subscription.

```go
// game.go
type Game struct {
    // ... existing fields ...
    mascot *mascot.Guide
}

func (g *Game) Update() error {
    g.currentScene.Update()
    g.mascot.SetScene(g.currentSceneName())
    g.mascot.Tick(dt)
    // Handle mascot click (if tapped in mascot region)
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.currentScene.Draw(screen)
    hud.DrawMascotOverlay(screen, g.mascot.VM())
}
```

### Trigger Types

| Type          | Example              | Mechanism                         |
|---------------|----------------------|-----------------------------------|
| Scene enter   | `scene_enter`        | Guide.SetScene() auto-match       |
| Game event    | `wave_start`         | EventBus -> Guide.Trigger()       |
| Idle timeout  | 30s no interaction   | idleTimer in Guide.Tick()         |
| Manual click  | Tap mascot           | Random tip or easter egg          |
| Conditional   | First clear / low hp | Scene calls Guide.Trigger()       |

### Rendering

- Character: right side of screen, bottom-aligned above ActionBar (stage) or
  bottom-right corner (other scenes)
- Speech bubble: above character, RoundRect + triangle tail + wrapped text
- Click mascot or bubble = advance dialog
- When no dialog: small idle animation (bob + blink)
- Slide-in/out animation on dialog start/end using easing functions

### Persistence

Extend `persistence/progress.go` with `MascotShown map[string]bool` to track
which `Once` dialogs have been displayed.

### Sprite Pipeline

1. SVG source in `assets/mascot/src/`
2. Export to PNG at appropriate resolution
3. Load via existing sprite cache + anim.Animator
4. States: idle, talk, happy, surprised (2-4 frames each)
5. For MVP: single frame per state, placeholder if needed

## Scope

### Phase 1 (MVP)
- Core Guide state machine + JSON loader
- MascotVM + DrawMascotOverlay with speech bubble
- Single PNG per expression (idle/talk/happy)
- Title + Stage scene integration
- 5-10 starter dialogs

### Phase 2
- Full animation frames (idle bob, blink, talk mouth)
- All scenes covered with dialogs
- Idle random chat + easter eggs
- Click interaction for tips
- Persistence for Once dialogs

### Phase 3
- Rich expressions and reactions
- Context-aware tips (e.g. "your tower is low health!")
- Achievement celebration animations
- User preference to hide/show mascot
