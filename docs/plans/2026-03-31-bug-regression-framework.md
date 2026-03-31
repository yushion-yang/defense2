# Bug Regression Framework Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a pure-logic game simulator (`sim`) enabling multi-tick regression tests for any core game bug without Ebitengine.

**Architecture:** `tests/regression/sim/` contains a lightweight Sim engine that wires `enemy.Pool`, `projectile.Pool`, `tower.Tower`, and `combat.ProcessDamage` into a tick loop. A fluent Builder API constructs test scenarios. Domain-specific assertion helpers make writing regression cases trivial.

**Tech Stack:** Go standard `testing`, existing `internal/core/*` packages only.

---

### Task 1: Sim engine core

**Files:**
- Create: `tests/regression/sim/sim.go`

**Step 1: Write the Sim struct and tick loop**

```go
// sim.go - Lightweight pure-logic game simulator for regression testing.
// No Ebitengine dependency. Wires core subsystems into a deterministic tick loop.
package sim

import (
	"math"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// Sim is a headless game simulator for regression testing.
type Sim struct {
	Enemies     *enemy.Pool
	Towers      []*tower.Tower
	Projectiles *projectile.Pool
	EventBus    *event.Bus
	Waypoints   []gamemap.Point
	Gold        int
	Lives       int
	Tick        int
	DT          float64
	Kills       int
	Leaked      int
}

// Step advances the simulation by one tick.
func (s *Sim) Step() {
	s.Tick++
	dt := s.DT

	// 1. Enemy movement
	s.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			e.DyingTimer -= dt
			if e.DyingTimer <= 0 {
				s.Enemies.FinishDying(e)
			}
			return
		}
		reached := enemy.MoveAlongPath(e, s.Waypoints, dt)
		if reached {
			s.Lives--
			s.Leaked++
			s.Enemies.KillImmediate(e)
		}
	})

	// 2. Tower targeting + firing
	for _, tw := range s.Towers {
		if !tw.Active {
			continue
		}
		tw.FireTimer -= dt
		if tw.FireTimer > 0 {
			continue
		}
		target := tower.FindNearestEnemy(tw, s.Enemies)
		if target == nil {
			continue
		}
		tw.FireTimer = 1.0 / tw.AttackSpeed

		switch tw.AttackStyleID {
		case tower.StyleLaser, tower.StyleWideBeam, tower.StyleSpinAoE, tower.StyleAuraDot:
			// Direct damage styles: apply damage immediately
			result := combat.ProcessDamage(combat.DamageInput{
				Target:    target,
				RawDamage: tw.Damage,
			})
			if result.Killed {
				s.Kills++
				s.Gold += target.Reward
				s.Enemies.Kill(target)
			}
		default:
			// Projectile-based styles
			speed := tw.ProjectileSpeed
			if speed == 0 {
				speed = 300
			}
			s.Projectiles.Fire(tw.X, tw.Y, target.X, target.Y,
				tw.Damage, speed, 4, target, tw.InstanceKey)
		}
	}

	// 3. Projectile update
	s.Projectiles.Update(dt)

	// 4. Projectile hit detection
	s.Projectiles.Each(func(p *projectile.Projectile) {
		s.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() {
				return
			}
			// Standard tracking projectile: only hit its target
			if p.Target != nil && p.Target != e {
				return
			}
			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			hitDist := p.Radius + e.Radius
			if hitDist < 8 {
				hitDist = 8
			}
			if dist > hitDist {
				return
			}
			result := combat.ProcessDamage(combat.DamageInput{
				Target:    e,
				RawDamage: p.Damage,
			})
			if result.Killed {
				s.Kills++
				s.Gold += e.Reward
				s.Enemies.Kill(e)
			}
			s.Projectiles.Release(p)
		})
	})

	// 5. Status effects
	s.Enemies.Each(func(e *enemy.Enemy) {
		if !e.IsDying() {
			enemy.TickStatusEffects(e, dt)
		}
	})
}

// RunTicks advances the simulation by n ticks and returns self for chaining.
func (s *Sim) RunTicks(n int) *Sim {
	for i := 0; i < n; i++ {
		s.Step()
	}
	return s
}

// RunUntil advances until pred returns true or maxTicks is reached.
// Returns true if pred was satisfied.
func (s *Sim) RunUntil(pred func(*Sim) bool, maxTicks int) bool {
	for i := 0; i < maxTicks; i++ {
		if pred(s) {
			return true
		}
		s.Step()
	}
	return pred(s)
}

// EnemyAt returns a pointer to the i-th spawned enemy (by pool slot order).
// Panics if index is out of range.
func (s *Sim) EnemyAt(idx int) *enemy.Enemy {
	count := 0
	var result *enemy.Enemy
	for i := range s.enemySlice() {
		e := &s.enemySlice()[i]
		if e.Active || e.ID > 0 { // include dying enemies
			if count == idx {
				result = e
				break
			}
			count++
		}
	}
	if result == nil {
		panic("sim.EnemyAt: index out of range")
	}
	return result
}

// enemySlice exposes the internal slice for indexed access.
// This is a testing helper only.
func (s *Sim) enemySlice() []enemy.Enemy {
	// We need access to the underlying slice. Since Pool doesn't expose it,
	// we track spawned enemies in the builder instead.
	// Fallback: iterate via Each and collect.
	return nil // placeholder - see builder
}
```

Note: `EnemyAt` needs access to pool internals. We'll use a tracked slice approach in the builder instead. Let me revise - we'll store spawned enemy pointers in Sim.

**Revised approach for EnemyAt:**

```go
// SpawnedEnemies tracks enemy pointers in spawn order (set by builder).
SpawnedEnemies []*enemy.Enemy
```

**Step 2: Verify it compiles**

Run: `cd tests/regression && go build ./sim/`
Expected: PASS (no errors)

**Step 3: Commit**

```
git add tests/regression/sim/sim.go
git commit -m "feat: add regression sim engine core with tick loop"
```

---

### Task 2: Fluent Builder

**Files:**
- Create: `tests/regression/sim/builder.go`

**Step 1: Write the Builder**

```go
// builder.go - Fluent API for constructing regression test scenarios.
package sim

import (
	"fmt"

	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// Builder constructs a Sim instance using a fluent API.
type Builder struct {
	waypoints []gamemap.Point
	enemies   []enemySpec
	towers    []towerSpec
	gold      int
	lives     int
	effects   []effectSpec
}

type enemySpec struct {
	x, y, hp, speed float64
	archetype       string
	cfg             *enemy.SpawnConfig
}

type towerSpec struct {
	key                  string
	x, y, dmg, rng      float64
	attackSpeed          float64
	style                string
	projectileSpeed      float64
}

type effectSpec struct {
	enemyIdx int
	kind     string // "slow", "stun", "burn", "bleed", "root"
	// slow
	factor float64
	// stun/root/slow duration
	duration float64
	// burn/bleed
	dps float64
}

// New creates a new Builder with sensible defaults.
func New() *Builder {
	return &Builder{
		gold:  999,
		lives: 20,
	}
}

// WithPath sets custom waypoints.
func (b *Builder) WithPath(pts ...gamemap.Point) *Builder {
	b.waypoints = pts
	return b
}

// WithStraightPath creates a horizontal path from (0,100) to (length,100).
func (b *Builder) WithStraightPath(length float64) *Builder {
	b.waypoints = []gamemap.Point{
		{X: 0, Y: 100},
		{X: length, Y: 100},
	}
	return b
}

// WithEnemy adds an enemy at the path start.
func (b *Builder) WithEnemy(archetype string, hp, speed float64) *Builder {
	b.enemies = append(b.enemies, enemySpec{
		archetype: archetype, hp: hp, speed: speed,
	})
	return b
}

// WithEnemyAt adds an enemy at a specific position.
func (b *Builder) WithEnemyAt(x, y, hp, speed float64) *Builder {
	b.enemies = append(b.enemies, enemySpec{
		x: x, y: y, hp: hp, speed: speed, archetype: "normal",
	})
	return b
}

// WithEnemyCfg adds an enemy with full SpawnConfig.
func (b *Builder) WithEnemyCfg(archetype string, baseHP, baseSpeed float64, cfg *enemy.SpawnConfig) *Builder {
	b.enemies = append(b.enemies, enemySpec{
		archetype: archetype, hp: baseHP, speed: baseSpeed, cfg: cfg,
	})
	return b
}

// WithTower places a tower at (x, y) with given damage and range.
func (b *Builder) WithTower(key string, x, y, dmg, rng float64) *Builder {
	b.towers = append(b.towers, towerSpec{
		key: key, x: x, y: y, dmg: dmg, rng: rng,
		attackSpeed: 1.0, style: "projectile", projectileSpeed: 300,
	})
	return b
}

// WithTowerFull places a tower with all parameters.
func (b *Builder) WithTowerFull(key, style string, x, y, dmg, rng, atkSpd, projSpd float64) *Builder {
	b.towers = append(b.towers, towerSpec{
		key: key, x: x, y: y, dmg: dmg, rng: rng,
		attackSpeed: atkSpd, style: style, projectileSpeed: projSpd,
	})
	return b
}

// WithGold sets starting gold.
func (b *Builder) WithGold(g int) *Builder {
	b.gold = g
	return b
}

// WithLives sets starting lives.
func (b *Builder) WithLives(l int) *Builder {
	b.lives = l
	return b
}

// ApplySlow queues a slow effect on enemy at index.
func (b *Builder) ApplySlow(enemyIdx int, factor, duration float64) *Builder {
	b.effects = append(b.effects, effectSpec{
		enemyIdx: enemyIdx, kind: "slow", factor: factor, duration: duration,
	})
	return b
}

// ApplyStun queues a stun effect on enemy at index.
func (b *Builder) ApplyStun(enemyIdx int, duration float64) *Builder {
	b.effects = append(b.effects, effectSpec{
		enemyIdx: enemyIdx, kind: "stun", duration: duration,
	})
	return b
}

// ApplyBurn queues a burn effect on enemy at index.
func (b *Builder) ApplyBurn(enemyIdx int, dps, duration float64) *Builder {
	b.effects = append(b.effects, effectSpec{
		enemyIdx: enemyIdx, kind: "burn", dps: dps, duration: duration,
	})
	return b
}

// ApplyBleed queues a bleed effect on enemy at index.
func (b *Builder) ApplyBleed(enemyIdx int, dps, duration float64) *Builder {
	b.effects = append(b.effects, effectSpec{
		enemyIdx: enemyIdx, kind: "bleed", dps: dps, duration: duration,
	})
	return b
}

// Build materializes the scenario into a Sim.
func (b *Builder) Build() *Sim {
	// Default path if none set
	if len(b.waypoints) == 0 {
		b.waypoints = []gamemap.Point{
			{X: 0, Y: 100},
			{X: 600, Y: 100},
		}
	}

	poolSize := len(b.enemies) + 16 // headroom
	if poolSize < 32 {
		poolSize = 32
	}

	s := &Sim{
		Enemies:     enemy.NewPool(poolSize),
		Projectiles: projectile.NewPool(64),
		EventBus:    event.NewBus(),
		Waypoints:   b.waypoints,
		Gold:        b.gold,
		Lives:       b.lives,
		DT:          1.0 / 60.0,
	}

	// Spawn enemies
	for _, spec := range b.enemies {
		x, y := spec.x, spec.y
		if x == 0 && y == 0 {
			x = b.waypoints[0].X
			y = b.waypoints[0].Y
		}
		e := s.Enemies.Spawn(x, y, spec.hp, spec.speed, 1, spec.archetype, spec.cfg)
		if e != nil {
			s.SpawnedEnemies = append(s.SpawnedEnemies, e)
		}
	}

	// Place towers
	for i, spec := range b.towers {
		tw := &tower.Tower{
			X:               spec.x,
			Y:               spec.y,
			Damage:          spec.dmg,
			Range:           spec.rng,
			AttackSpeed:     spec.attackSpeed,
			FireTimer:       0, // ready to fire immediately
			Key:             spec.key,
			InstanceKey:     fmt.Sprintf("%s_%d", spec.key, i),
			Active:          true,
			AttackStyleID:   spec.style,
			ProjectileSpeed: spec.projectileSpeed,
		}
		s.Towers = append(s.Towers, tw)
	}

	// Apply pre-set status effects
	for _, eff := range b.effects {
		if eff.enemyIdx >= len(s.SpawnedEnemies) {
			continue
		}
		e := s.SpawnedEnemies[eff.enemyIdx]
		switch eff.kind {
		case "slow":
			combat.ApplySlow(e, eff.factor, eff.duration, "test")
		case "stun":
			combat.ApplyStun(e, eff.duration, "test")
		case "burn":
			e.BurnTimer = eff.duration
			e.BurnDPS = eff.dps
		case "bleed":
			e.BleedTimer = eff.duration
			e.BleedDPS = eff.dps
		case "root":
			combat.ApplyRoot(e, eff.duration, "test")
		}
	}

	return s
}
```

**Step 2: Verify it compiles**

Run: `cd tests/regression && go build ./sim/`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/sim/builder.go
git commit -m "feat: add fluent builder for regression sim scenarios"
```

---

### Task 3: Assertion helpers

**Files:**
- Create: `tests/regression/sim/assert.go`

**Step 1: Write assertion helpers**

```go
// assert.go - Domain-specific test assertion helpers for Sim.
package sim

import (
	"fmt"
	"testing"
)

// cmp evaluates "actual op expected" for float64.
func cmp(actual float64, op string, expected float64) bool {
	switch op {
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	case ">":
		return actual > expected
	case ">=":
		return actual >= expected
	case "<":
		return actual < expected
	case "<=":
		return actual <= expected
	default:
		panic("unknown operator: " + op)
	}
}

// cmpInt evaluates "actual op expected" for int.
func cmpInt(actual int, op string, expected int) bool {
	return cmp(float64(actual), op, float64(expected))
}

func failMsg(t *testing.T, label string, actual interface{}, op string, expected interface{}) {
	t.Helper()
	t.Fatalf("%s: got %v, expected %s %v", label, actual, op, expected)
}

// AssertEnemyHP checks enemy HP at index.
func (s *Sim) AssertEnemyHP(t *testing.T, idx int, op string, expected float64) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !cmp(e.HP, op, expected) {
		failMsg(t, fmt.Sprintf("enemy[%d].HP", idx), e.HP, op, expected)
	}
}

// AssertEnemySpeed checks enemy Speed at index.
func (s *Sim) AssertEnemySpeed(t *testing.T, idx int, op string, expected float64) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !cmp(e.Speed, op, expected) {
		failMsg(t, fmt.Sprintf("enemy[%d].Speed", idx), e.Speed, op, expected)
	}
}

// AssertEnemyAlive checks enemy is still active and not dying.
func (s *Sim) AssertEnemyAlive(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.Active || e.IsDying() {
		t.Fatalf("enemy[%d]: expected alive, Active=%v Dying=%v", idx, e.Active, e.IsDying())
	}
}

// AssertEnemyDead checks enemy is dead (inactive or dying).
func (s *Sim) AssertEnemyDead(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if e.Active && !e.IsDying() {
		t.Fatalf("enemy[%d]: expected dead, Active=%v HP=%.1f", idx, e.Active, e.HP)
	}
}

// AssertEnemyDying checks enemy is in dying animation.
func (s *Sim) AssertEnemyDying(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.IsDying() {
		t.Fatalf("enemy[%d]: expected dying, DyingTimer=%.3f", idx, e.DyingTimer)
	}
}

// AssertEnemyReachedEnd checks enemy reached path end.
func (s *Sim) AssertEnemyReachedEnd(t *testing.T, idx int) {
	t.Helper()
	e := s.SpawnedEnemies[idx]
	if !e.ReachedEnd {
		t.Fatalf("enemy[%d]: expected ReachedEnd, X=%.1f Y=%.1f", idx, e.X, e.Y)
	}
}

// AssertEnemyCount checks active enemy count.
func (s *Sim) AssertEnemyCount(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Enemies.Count, op, expected) {
		failMsg(t, "enemyCount", s.Enemies.Count, op, expected)
	}
}

// AssertProjectileCount checks active projectile count.
func (s *Sim) AssertProjectileCount(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Projectiles.Count, op, expected) {
		failMsg(t, "projectileCount", s.Projectiles.Count, op, expected)
	}
}

// AssertGold checks current gold.
func (s *Sim) AssertGold(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Gold, op, expected) {
		failMsg(t, "gold", s.Gold, op, expected)
	}
}

// AssertLives checks current lives.
func (s *Sim) AssertLives(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Lives, op, expected) {
		failMsg(t, "lives", s.Lives, op, expected)
	}
}

// AssertKills checks total kills.
func (s *Sim) AssertKills(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Kills, op, expected) {
		failMsg(t, "kills", s.Kills, op, expected)
	}
}

// AssertLeaked checks total leaked enemies.
func (s *Sim) AssertLeaked(t *testing.T, op string, expected int) {
	t.Helper()
	if !cmpInt(s.Leaked, op, expected) {
		failMsg(t, "leaked", s.Leaked, op, expected)
	}
}
```

**Step 2: Verify it compiles**

Run: `cd tests/regression && go build ./sim/`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/sim/assert.go
git commit -m "feat: add domain-specific assertion helpers for regression sim"
```

---

### Task 4: First regression test — slow min speed clamp

**Files:**
- Create: `tests/regression/combat_test.go`

**Step 1: Write the failing test (RED)**

```go
package regression_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// BUG: SlowFactor=0 caused enemy speed to drop to 0.
// Fix: combat.ApplySlow clamps factor to MinSpeedRatio (0.2).
func TestRegression_CC_SlowMinSpeedClamp(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 100, 100).
		Build()

	// Apply slow with factor=0 (should be clamped)
	combat.ApplySlow(s.SpawnedEnemies[0], 0.0, 5.0, "test")
	s.RunTicks(1)

	s.AssertEnemySpeed(t, 0, ">=", 100*enemy.MinSpeedRatio)
	s.AssertEnemySpeed(t, 0, ">", 0)
}

// BUG: Stun did not prevent movement.
// Fix: MoveAlongPath checks StunTimer > 0 and skips movement.
func TestRegression_CC_StunPreventsMovement(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 100, 100).
		Build()

	startX := s.SpawnedEnemies[0].X
	combat.ApplyStun(s.SpawnedEnemies[0], 2.0, "test")
	s.RunTicks(60) // 1 second

	// Enemy should not have moved
	if s.SpawnedEnemies[0].X != startX {
		t.Fatalf("stunned enemy moved: startX=%.1f nowX=%.1f", startX, s.SpawnedEnemies[0].X)
	}
}
```

**Step 2: Run test to verify it passes (GREEN — these bugs are already fixed)**

Run: `go test ./tests/regression/ -v -run TestRegression_CC`
Expected: PASS (confirms the fixes hold)

**Step 3: Commit**

```
git add tests/regression/combat_test.go
git commit -m "test: add CC regression cases (slow clamp, stun movement)"
```

---

### Task 5: Damage pipeline regression tests

**Files:**
- Create: `tests/regression/pipeline_test.go`

**Step 1: Write tests**

```go
package regression_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// BUG: Shield absorb was not applied, damage went straight to HP.
// Fix: ProcessDamage step 5 consumes ShieldHP before HP.
func TestRegression_Pipeline_ShieldAbsorb(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("shielded", 100, 60, &enemy.SpawnConfig{
			HpScale: 1, SpeedScale: 1, Radius: 8, ShieldScale: 0.5,
		}).
		Build()

	e := s.SpawnedEnemies[0]
	// Enemy has 100 HP + 50 shield (100 * 0.5)
	result := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 30,
	})

	if result.ShieldAbsorbed != 30 {
		t.Fatalf("shield should absorb 30, absorbed %.1f", result.ShieldAbsorbed)
	}
	if e.HP != 100 {
		t.Fatalf("HP should be untouched at 100, got %.1f", e.HP)
	}
}

// BUG: Pure damage did not bypass shield.
// Fix: ProcessDamage step 5 skips shield for pure damage type.
func TestRegression_Pipeline_PureBypassesShield(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("shielded", 100, 60, &enemy.SpawnConfig{
			HpScale: 1, SpeedScale: 1, Radius: 8, ShieldScale: 0.5,
		}).
		Build()

	e := s.SpawnedEnemies[0]
	result := combat.ProcessDamage(combat.DamageInput{
		Target:     e,
		RawDamage:  30,
		DamageType: combat.DmgPure,
	})

	if result.ShieldAbsorbed != 0 {
		t.Fatalf("pure should bypass shield, absorbed %.1f", result.ShieldAbsorbed)
	}
	if e.HP != 70 {
		t.Fatalf("HP should be 70, got %.1f", e.HP)
	}
}

// BUG: DamageCap not disabled when enemy is silenced.
// Fix: ProcessDamage step 4.5 skips damageCap when Silenced=true.
func TestRegression_Pipeline_SilenceDisablesDamageCap(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 60).
		Build()

	e := s.SpawnedEnemies[0]
	e.DamageCap = 50

	// Without silence: damage capped at 50
	r1 := combat.ProcessDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r1.FinalDamage > 50 {
		t.Fatalf("capped damage should be <=50, got %.1f", r1.FinalDamage)
	}

	// With silence: damage cap disabled
	e.HP = 1000 // reset
	e.Silenced = true
	r2 := combat.ProcessDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r2.FinalDamage != 200 {
		t.Fatalf("silenced should disable cap, got %.1f", r2.FinalDamage)
	}
}
```

**Step 2: Run tests**

Run: `go test ./tests/regression/ -v -run TestRegression_Pipeline`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/pipeline_test.go
git commit -m "test: add damage pipeline regression cases (shield, pure, silence)"
```

---

### Task 6: Enemy behavior regression tests

**Files:**
- Create: `tests/regression/enemy_test.go`

**Step 1: Write tests**

```go
package regression_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// BUG: All enemy archetypes were "normal" — SpawnConfig scales were not applied.
// Fix: Pool.Spawn applies cfg.HpScale/SpeedScale/Radius to base values.
func TestRegression_Enemy_ArchetypeScaling(t *testing.T) {
	cfg := &enemy.SpawnConfig{
		HpScale:    2.0,
		SpeedScale: 0.5,
		Radius:     12,
	}
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("tank", 100, 60, cfg).
		Build()

	e := s.SpawnedEnemies[0]
	if e.HP != 200 { // 100 * 2.0
		t.Fatalf("HP should be 200, got %.1f", e.HP)
	}
	if e.Speed != 30 { // 60 * 0.5
		t.Fatalf("Speed should be 30, got %.1f", e.Speed)
	}
	if e.Radius != 12 {
		t.Fatalf("Radius should be 12, got %.1f", e.Radius)
	}
	if e.Archetype != "tank" {
		t.Fatalf("Archetype should be 'tank', got '%s'", e.Archetype)
	}
}

// BUG: Dying enemy was still targeted by towers.
// Fix: FindNearestEnemy skips enemies where IsDying() is true.
func TestRegression_Enemy_DyingNotTargeted(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1, 60).   // will die from 1 hit
		WithEnemy("normal", 100, 60). // backup target
		WithTower("basic", 250, 100, 10, 300).
		Build()

	// Run enough ticks for tower to fire and kill first enemy
	s.RunTicks(120) // 2 seconds

	// First enemy should be dead, second should be targeted
	s.AssertEnemyDead(t, 0)
}

// BUG: Kill→Count was out of sync — Count decremented on FinishDying instead of Kill.
// Fix: Kill decrements Count immediately; FinishDying only clears Active.
func TestRegression_Enemy_KillCountTiming(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if pool.Count != 1 {
		t.Fatalf("count after spawn should be 1, got %d", pool.Count)
	}

	pool.Kill(e)
	if pool.Count != 0 {
		t.Fatalf("count after Kill should be 0, got %d", pool.Count)
	}
	if !e.Active {
		t.Fatal("enemy should still be Active during dying animation")
	}

	pool.FinishDying(e)
	if e.Active {
		t.Fatal("enemy should be inactive after FinishDying")
	}
}

// BUG: Enemy leaking didn't decrement lives.
// Fix: Sim.Step checks ReachedEnd and decrements Lives.
func TestRegression_Enemy_LeakLosesLife(t *testing.T) {
	s := sim.New().
		WithPath(
			sim.Pt(0, 100),
			sim.Pt(100, 100), // short path
		).
		WithEnemy("runner", 100, 1000). // very fast
		WithLives(20).
		Build()

	s.RunTicks(60) // 1 second, fast enemy should reach end

	s.AssertLives(t, "<", 20)
	s.AssertLeaked(t, ">", 0)
}
```

Note: We need a `Pt` helper in the sim package:

Add to `builder.go`:
```go
// Pt is a shorthand for creating a gamemap.Point.
func Pt(x, y float64) gamemap.Point {
	return gamemap.Point{X: x, Y: y}
}
```

**Step 2: Run tests**

Run: `go test ./tests/regression/ -v -run TestRegression_Enemy`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/enemy_test.go tests/regression/sim/builder.go
git commit -m "test: add enemy behavior regression cases (archetype, dying, count, leak)"
```

---

### Task 7: Projectile tracking regression test

**Files:**
- Modify: `tests/regression/combat_test.go` (append)

**Step 1: Write test**

```go
// BUG: Projectile was fire-and-forget — fixed VX/VY caused all shots to miss when enemies turned.
// Fix: Projectile.Update() recalculates VX/VY toward target each frame while target is alive.
func TestRegression_Projectile_TrackingTarget(t *testing.T) {
	// L-shaped path: enemy goes right then down
	s := sim.New().
		WithPath(
			sim.Pt(0, 0),
			sim.Pt(200, 0),   // right
			sim.Pt(200, 200), // then down
		).
		WithEnemy("normal", 50, 100). // moderate speed
		WithTower("archer", 100, 0, 50, 300). // tower on the path
		Build()

	// Run until enemy is dead or 5 seconds elapsed
	dead := s.RunUntil(func(s *sim.Sim) bool {
		return s.SpawnedEnemies[0].IsDying() || !s.SpawnedEnemies[0].Active
	}, 300)

	if !dead {
		t.Fatalf("projectile should have tracked and killed enemy, HP=%.1f",
			s.SpawnedEnemies[0].HP)
	}
}
```

**Step 2: Run test**

Run: `go test ./tests/regression/ -v -run TestRegression_Projectile`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/combat_test.go
git commit -m "test: add projectile tracking regression case"
```

---

### Task 8: DoT (burn/bleed) independent regression

**Files:**
- Modify: `tests/regression/combat_test.go` (append)

**Step 1: Write test**

```go
// BUG: Burn was treated as bleed (shared timer).
// Fix: Burn has independent BurnTimer/BurnDPS fields.
func TestRegression_DoT_BurnIndependentOfBleed(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 0). // stationary for easy observation
		ApplyBurn(0, 100, 3.0).       // 100 DPS burn for 3s
		ApplyBleed(0, 50, 3.0).       // 50 DPS bleed for 3s
		Build()

	e := s.SpawnedEnemies[0]
	if e.BurnTimer <= 0 || e.BleedTimer <= 0 {
		t.Fatal("both burn and bleed should be active")
	}
	if e.BurnDPS != 100 || e.BleedDPS != 50 {
		t.Fatalf("DPS values wrong: burn=%.0f bleed=%.0f", e.BurnDPS, e.BleedDPS)
	}

	// Run 1 DoT tick cycle (0.5s = 30 ticks)
	s.RunTicks(30)

	// Combined DoT: (100+50) * 0.5 = 75 damage per tick
	expectedHP := 1000.0 - 75.0
	if e.HP > expectedHP+1 || e.HP < expectedHP-10 {
		t.Fatalf("HP after 1 DoT tick should be ~%.0f, got %.1f", expectedHP, e.HP)
	}
}
```

**Step 2: Run test**

Run: `go test ./tests/regression/ -v -run TestRegression_DoT`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/combat_test.go
git commit -m "test: add DoT burn/bleed independence regression case"
```

---

### Task 9: Tenacity regression test

**Files:**
- Modify: `tests/regression/combat_test.go` (append)

**Step 1: Write test**

```go
// BUG: Tenacity was not reducing CC duration.
// Fix: ApplyStun/ApplySlow multiply duration by (1 - Tenacity).
func TestRegression_CC_TenacityReducesDuration(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("boss", 1000, 100).
		Build()

	e := s.SpawnedEnemies[0]
	e.Tenacity = 0.5 // 50% CC reduction

	combat.ApplyStun(e, 2.0, "test")
	if e.StunTimer != 1.0 { // 2.0 * (1 - 0.5)
		t.Fatalf("stun with 50%% tenacity should be 1.0s, got %.2f", e.StunTimer)
	}

	combat.ApplySlow(e, 0.5, 4.0, "test")
	if e.SlowTimer != 2.0 { // 4.0 * (1 - 0.5)
		t.Fatalf("slow with 50%% tenacity should be 2.0s, got %.2f", e.SlowTimer)
	}
}

// BUG: Full tenacity (1.0) should make enemy CC immune.
func TestRegression_CC_FullTenacityImmune(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("boss", 1000, 100).
		Build()

	e := s.SpawnedEnemies[0]
	e.Tenacity = 1.0

	ok := combat.ApplyStun(e, 2.0, "test")
	if ok {
		t.Fatal("stun should fail with tenacity=1.0")
	}
	if e.StunTimer != 0 {
		t.Fatalf("stun timer should be 0, got %.2f", e.StunTimer)
	}
}
```

**Step 2: Run test**

Run: `go test ./tests/regression/ -v -run TestRegression_CC_Tenacity`
Expected: PASS

**Step 3: Commit**

```
git add tests/regression/combat_test.go
git commit -m "test: add tenacity CC reduction regression cases"
```

---

### Task 10: Verify full suite and final commit

**Step 1: Run entire regression suite**

Run: `go test ./tests/regression/... -v -count=1`
Expected: ALL PASS

**Step 2: Run existing tests to ensure no breakage**

Run: `go test ./tests/... -v -count=1`
Expected: ALL PASS

**Step 3: Final commit with all files**

```
git add tests/regression/
git commit -m "feat: add bug regression framework with sim engine and 10 initial cases"
```
