# Visual Polish Sprint 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add core visual feedback (enemy death animation, hit reaction, muzzle flash particles) and scene transitions (fade to black).

**Architecture:** Death/hit animations live on Enemy struct fields, rendered by DrawEnemies. Scene transitions managed by Game struct with a fade overlay. All changes use existing draw primitives and particle system — no new shaders or packages needed.

**Tech Stack:** Ebitengine v2.9.9, existing particle system, existing anim.Animator, existing draw.* primitives.

---

## Overview

| Task | Feature | Key Files |
|------|---------|-----------|
| 1 | Enemy death animation (fade+shrink) | enemy.go, pool.go, draw_enemy.go |
| 2 | Enemy hit reaction (hit frames + flash) | draw_enemy.go, stage.go |
| 3 | Tower muzzle flash particles | stage.go (combat callbacks) |
| 4 | Scene fade transition | game.go |

---

### Task 1: Enemy Death Animation

Add a 0.3s dying state where the enemy sprite shrinks, fades, and floats up instead of instantly vanishing.

**Files:**
- Modify: `internal/core/enemy/enemy.go` — add DyingTimer/DyingDuration fields
- Modify: `internal/core/enemy/pool.go` — Kill sets DyingTimer instead of immediate deactivation
- Modify: `internal/render/draw_enemy.go` — apply scale/alpha/offset for dying enemies
- Modify: `internal/scene/stage.go` — tick dying enemies, skip dying enemies from gameplay logic
- Test: `tests/core/enemy_death_anim_test.go`

**Enemy struct additions** (enemy.go):
```go
DyingTimer    float64 // >0 means dying animation in progress
DyingDuration float64 // total dying time (set on kill)
```

**Pool.Kill change** (pool.go):
```go
func (p *Pool) Kill(e *Enemy) {
    if e.Active && e.DyingTimer <= 0 {
        e.DyingTimer = 0.3    // start dying
        e.DyingDuration = 0.3
        if e.Boss {
            e.DyingTimer = 0.5
            e.DyingDuration = 0.5
        }
    }
}
```

Add `FinishDying` for actual deactivation:
```go
func (p *Pool) FinishDying(e *Enemy) {
    if e.Active {
        e.Active = false
        e.DyingTimer = 0
        p.Count--
    }
}
```

**Pool.Each must still include dying enemies for rendering** but gameplay systems (targeting, collision) should skip them. Add a helper:
```go
func (e *Enemy) IsDying() bool { return e.DyingTimer > 0 }
```

Targeting and collision code already uses `e.Active` — dying enemies are still Active, so add `IsDying()` checks in:
- `targeting.go` FindNearestEnemy: skip `e.IsDying()`
- `tick_combat.go` TickProjectileHits: skip `e.IsDying()`

**stage.go Update**: Add dying tick after enemy movement:
```go
s.enemies.Each(func(e *enemy.Enemy) {
    if e.IsDying() {
        e.DyingTimer -= gameDT
        if e.DyingTimer <= 0 {
            s.enemies.FinishDying(e)
        }
    }
})
```

**DrawEnemies change** (draw_enemy.go):
When `e.IsDying()`, compute animation progress and apply transforms:
```go
if e.IsDying() {
    progress := 1.0 - e.DyingTimer/e.DyingDuration // 0→1
    scale := 1.0 - progress                         // 1→0
    alpha := 1.0 - progress                          // 1→0
    offsetY := -progress * 8                         // float up 8px
    // Apply to sprite rendering: multiply display size by scale,
    // shift cy by offsetY, apply alpha via ColorScale
    // Skip HP bar, status dots, etc. for dying enemies
}
```

**Tests:**
```go
TestEnemyDyingState — Kill sets DyingTimer, IsDying returns true
TestEnemyDyingFinish — After DyingTimer expires, FinishDying deactivates
TestDyingEnemySkippedByTargeting — Dying enemies not targeted
```

**Commit:**
```bash
git commit -m "feat: add enemy death animation (0.3s fade+shrink+float)"
```

---

### Task 2: Enemy Hit Reaction (Hit Frames + White Flash)

Use the existing hit animation frames from assets and the existing HitFlash field.

**Files:**
- Modify: `internal/render/draw_enemy.go` — getEnemyFrame to handle hit state, apply white flash ColorScale
- Modify: `internal/render/anim/anim.go` — verify PlayOnce exists or add it
- Modify: `internal/scene/stage.go` — trigger hit animation on damage

**Current state:**
- Enemy already has `HitFlash float64` field (set to 0.12 on hit in stage.go)
- `draw_enemy.go` line 75: `er.getEnemyFrame(e, 1.0/60.0)` — currently only uses "walk" animation
- Assets have `{archetype}-hit-0.png` and `{archetype}-hit-1.png` per enemy type
- `anim.Loader` auto-detects multi-frame PNGs for "hit" state

**getEnemyFrame change:**
Currently always plays "walk". Change to:
```go
func (er *EnemyRenderer) getEnemyFrame(e *enemy.Enemy, dt float64) *ebiten.Image {
    animator := er.getOrCreateAnimator(e.Archetype)
    if animator == nil { return nil }

    // If HitFlash > 0 and hit animation exists, play it
    if e.HitFlash > 0 && animator.HasAnimation("hit") {
        if animator.Current() != "hit" {
            animator.PlayOnce("hit", "walk") // play hit then revert to walk
        }
    }

    animator.Update(dt)
    return animator.Frame()
}
```

Check if `anim.Animator` has `PlayOnce(name, revertTo string)` — if not, add it. Also add `HasAnimation(name)` and `Current()`.

**White flash rendering** (draw_enemy.go body rendering section):
After drawing the sprite, if `e.HitFlash > 0`, draw a white overlay:
```go
if e.HitFlash > 0 {
    // Draw the same sprite again with additive white tint
    flashAlpha := float32(e.HitFlash / 0.12) * 0.5 // peak 50% white
    // Use ColorScale to make sprite white-ish
    // Ebitengine: op.ColorScale.Scale(1+flashAlpha, 1+flashAlpha, 1+flashAlpha, 1)
}
```

Actually simpler: draw a white-filled circle over the enemy with alpha:
```go
if e.HitFlash > 0 {
    flashAlpha := uint8(float64(e.HitFlash/0.12) * 120) // peak alpha 120/255
    draw.FilledCircle(screen, cx, cy, r, color.RGBA{255, 255, 255, flashAlpha})
}
```

**Commit:**
```bash
git commit -m "feat: add enemy hit reaction (hit frames + white flash overlay)"
```

---

### Task 3: Tower Muzzle Flash Particles

Hook existing `EmitMuzzleFlash` into all tower fire events.

**Files:**
- Modify: `internal/scene/stage.go` — add muzzle flash to onFire callback

**Current state:**
- `particle.EmitMuzzleFlash(pool, x, y, angle)` already exists
- Tower fire callback (line ~1302): `func(style string)` — has tower info available
- Need tower position and firing angle

**The onFire callback** in TickTowerCombat currently only receives `style string`. We need the tower's position and target angle. Two options:

Option A: Capture tower ref in closure (already done — the callback is inside towers.Each)
Option B: Extend callback signature

Go with Option A — the tower is already available in the surrounding scope. Check how TickTowerCombat is called:

```go
pipeline.TickTowerCombat(s.towers, s.enemies, s.projectiles, s.beams, gameDT,
    func(style string) {  // onFire
        s.audioMgr.PlayThrottled(...)
    },
    func(e *enemy.Enemy, damage float64, killed bool, _ string) { // onHit
        ...
    })
```

The problem: onFire doesn't know which tower fired. We need to either:
1. Change TickTowerCombat's onFire signature to `func(t *tower.Tower, style string)`
2. Or use a different hook point

Check `tick_combat.go` to see if tower is available at the onFire call site. If yes, extend the callback signature.

If extending is too invasive, simpler approach: emit muzzle flash in the onHit callback at the source tower position (we already look up sourceTower there). This fires on projectile impact, not tower fire, but visually it maps to the same events with a slight delay.

Actually simplest: just emit from tower position on the attack VFX state change. In `draw_tower.go`, when `t.FireAnim > 0`, emit particles. But draw functions shouldn't have side effects.

**Recommended**: Extend onFire to `func(t *tower.Tower, style string)`. The change in tick_combat.go is a 1-line signature change + 1-line call change.

```go
// In stage.go onFire callback:
func(t *tower.Tower, style string) {
    s.audioMgr.PlayThrottled(gameAudio.FireSFXForStyle(style), 100)
    if t.Target != nil {
        angle := math.Atan2(t.Target.Y-t.Y, t.Target.X-t.X)
        particle.EmitMuzzleFlash(s.particlePool, t.X, t.Y, angle)
    }
}
```

**Commit:**
```bash
git commit -m "feat: add muzzle flash particles on tower fire"
```

---

### Task 4: Scene Fade Transition

Add a simple fade-to-black transition between all scene switches.

**Files:**
- Modify: `internal/scene/game.go` — add transition state machine + fade overlay

**Design:**

```go
type transitionState int
const (
    transIdle transitionState = iota
    transFadeOut  // current scene fading to black
    transFadeIn   // new scene fading from black
)

// Add to Game struct:
transState  transitionState
transAlpha  float64      // 0.0 (transparent) to 1.0 (black)
transSpeed  float64      // alpha change per tick (default: 1.0/18 = ~0.3s at 60fps)
pendingNext Scene        // scene to switch to after fade-out completes
```

**SwitchScene change:**
```go
func (g *Game) SwitchScene(next Scene) {
    if g.transState != transIdle {
        return // ignore during transition
    }
    g.pendingNext = next
    g.transState = transFadeOut
    g.transAlpha = 0
    g.transSpeed = 1.0 / 18.0 // 0.3s fade
}
```

**Update change:**
```go
func (g *Game) Update() error {
    switch g.transState {
    case transFadeOut:
        g.transAlpha += g.transSpeed
        if g.transAlpha >= 1.0 {
            g.transAlpha = 1.0
            g.current = g.pendingNext
            g.pendingNext = nil
            g.transState = transFadeIn
        }
        // Still update current scene during fade-out (keeps it alive)
        return g.current.Update()
    case transFadeIn:
        g.transAlpha -= g.transSpeed
        if g.transAlpha <= 0 {
            g.transAlpha = 0
            g.transState = transIdle
        }
        return g.current.Update()
    default:
        return g.current.Update()
    }
}
```

**Draw change:**
```go
func (g *Game) Draw(screen *ebiten.Image) {
    g.current.Draw(screen)
    if g.transAlpha > 0 {
        // Draw full-screen black overlay with transAlpha
        a := uint8(g.transAlpha * 255)
        draw.FilledRect(screen, 0, 0, float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy()),
            color.RGBA{0, 0, 0, a}, false)
    }
}
```

Note: `draw.FilledRect` uses physical pixel dimensions from screen.Bounds(), not logical coords (since screen IS at physical resolution). The fade overlay must cover the full physical screen.

**Tests:**
```go
TestTransitionFadeOut — SwitchScene starts fadeout, transAlpha increases
TestTransitionFadeIn — After fadeout completes, scene switches, transAlpha decreases
TestTransitionBlocksDuringActive — SwitchScene ignored during active transition
```

**Commit:**
```bash
git commit -m "feat: add fade-to-black scene transition"
```

---

## Verification

After all tasks:
```bash
make test    # all tests pass
make run     # visual verification:
             # - kill an enemy: fade+shrink death animation
             # - hit an enemy: white flash + hit frame
             # - tower fires: muzzle flash particles
             # - switch scenes: smooth fade to black
```
