# Bug Regression Framework Design

Date: 2026-03-31

## Goal

Build a lightweight, pure-logic game simulator (`sim`) that can replay multi-tick game scenarios without Ebitengine/GUI. Each regression test case reconstructs the exact conditions of a historical bug and asserts the fix holds.

## Architecture

```
tests/regression/
├── sim/
│   ├── sim.go          # Sim engine: tick loop, wire core objects
│   ├── builder.go      # Fluent API for scene construction
│   └── assert.go       # Domain-specific assertion helpers
├── combat_test.go      # Combat bug regressions
├── enemy_test.go       # Enemy behavior regressions
├── economy_test.go     # Economy system regressions
├── tower_test.go       # Tower system regressions
└── event_test.go       # Event bus regressions
```

## Sim Engine (`sim.go`)

### Core Struct

```go
type Sim struct {
    Enemies    *enemy.Pool
    Towers     []*tower.Tower
    Projectiles *projectile.Pool
    EventBus   *event.Bus
    Waypoints  []gamemap.Point
    Gold       int
    Lives      int
    Tick       int
    DT         float64 // fixed 1/60
}
```

### Tick Loop

Each `Sim.Step()` executes in order:

1. **Enemy movement** — `enemy.MoveAlongPath()` for each active enemy
2. **Tower targeting** — `tower.FindNearestEnemy()` per tower
3. **Tower combat** — fire projectiles or apply direct damage (simplified: skip AttackHandler registry, directly call `ProcessDamage` for non-projectile styles)
4. **Projectile update** — `projectile.Pool.Update(dt)`
5. **Projectile hit detection** — distance check, `combat.ProcessDamage()`
6. **Status effects** — `enemy.TickStatusEffects(dt)`
7. **Dying cleanup** — finish dying enemies whose timer expired
8. **Leak check** — enemies with `ReachedEnd` lose lives

Not simulated (intentionally): rendering, audio, HUD, input, warden, skills (can be added later).

### Public API

```go
func (s *Sim) Step()                    // advance 1 tick
func (s *Sim) RunTicks(n int) *Sim      // advance N ticks, return self
func (s *Sim) RunUntil(pred func(*Sim) bool, maxTicks int) *Sim
```

## Builder (`builder.go`)

Fluent API for constructing test scenarios:

```go
sim.New().
    WithPath(points...).                 // custom waypoints
    WithStraightPath(length).            // shorthand: (0,0) -> (length,0)
    WithEnemy(archetype, hp, speed).     // spawn one enemy
    WithEnemyAt(x, y, hp, speed).        // spawn at specific position
    WithTower(key, x, y, dmg, rng).      // place a tower
    WithGold(amount).
    WithLives(amount).
    ApplySlow(enemyIdx, factor, dur).    // pre-apply status effect
    ApplyStun(enemyIdx, dur).
    ApplyBurn(enemyIdx, dps, dur).
    Build() *Sim
```

Key design choices:
- `Build()` returns `*Sim` for direct state inspection
- Builder stores pending operations; `Build()` materializes them
- Enemy pool size auto-calculated from enemy count (+ headroom)
- Default DT = 1.0/60.0

## Assertions (`assert.go`)

Domain-aware assertion helpers on `*Sim`:

```go
// Enemy state
func (s *Sim) AssertEnemyHP(t, idx, op, expected)
func (s *Sim) AssertEnemySpeed(t, idx, op, expected)
func (s *Sim) AssertEnemyAlive(t, idx)
func (s *Sim) AssertEnemyDead(t, idx)
func (s *Sim) AssertEnemyDying(t, idx)
func (s *Sim) AssertEnemyReachedEnd(t, idx)
func (s *Sim) AssertEnemyHasStatus(t, idx, status)

// Pool state
func (s *Sim) AssertEnemyCount(t, op, expected)
func (s *Sim) AssertProjectileCount(t, op, expected)

// Economy
func (s *Sim) AssertGold(t, op, expected)
func (s *Sim) AssertLives(t, op, expected)

// Generic
func (s *Sim) AssertField(t, fieldName, op, expected)
```

`op` is a comparison string: `"=="`, `">"`, `">="`, `"<"`, `"<="`, `"!="`.

## Regression Test Convention

Each test:
1. Names: `TestRegression_<System>_<BugDescription>`
2. Comment: `// BUG: <one-line description of the original bug>`
3. Uses builder to reconstruct the exact failing scenario
4. Asserts the fix holds

Example:

```go
// BUG: SlowFactor=0 caused enemy speed to drop to 0 (should clamp to MinSpeedRatio=0.2)
func TestRegression_CC_SlowMinSpeedClamp(t *testing.T) {
    s := sim.New().
        WithStraightPath(500).
        WithEnemy("normal", 100, 100).
        Build()
    combat.ApplySlow(&s.EnemyAt(0), 0.0, 5.0, "test")
    s.RunTicks(1)
    speed := s.EnemyAt(0).Speed
    if speed < 100*enemy.MinSpeedRatio {
        t.Fatalf("speed %.1f below min ratio, expected >= %.1f", speed, 100*enemy.MinSpeedRatio)
    }
}
```

## Seed Regression Cases from MEMORY.md

Initial regression cases derived from documented bugs:

| ID | System | Bug | Test |
|----|--------|-----|------|
| 1 | CC | SlowFactor=0 → speed=0 | Clamp to MinSpeedRatio |
| 2 | Enemy | All archetypes "normal" | SpawnConfig applies HpScale/SpeedScale |
| 3 | Projectile | Fire-and-forget miss | Target tracking updates VX/VY |
| 4 | Pipeline | Shield absorb + shieldIgnore | Pure damage bypasses shield |
| 5 | Pipeline | DamageCap + Silenced | Silenced disables DamageCap |
| 6 | Combat | Stun probability | 12% chance, tenacity reduces duration |
| 7 | Enemy | Dying enemy still targeted | IsDying() skips targeting |
| 8 | Pool | Kill→Count decrement timing | Count drops on Kill, Active on FinishDying |
| 9 | Buff | Burn independent of bleed | Separate timers, both tick |
| 10 | Economy | Leak loses lives | ReachedEnd → lives-- |

## Non-Goals (v1)

- Warden simulation
- Skill system simulation
- Multi-path maps
- Config loading from JSON (tests construct objects directly)
- Performance benchmarking

These can be added incrementally.

## Dependencies

Only `internal/core/*` packages:
- `enemy`, `tower`, `projectile`, `combat`, `event`, `gamemap`, `buff`
- No `scene`, `render`, `audio`, `config`, `autoplay`
- Build tag: none needed (pure Go, no Ebitengine imports)
