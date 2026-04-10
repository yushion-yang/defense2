// builder.go — Fluent API for constructing regression test scenarios.
package sim

import (
	"fmt"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/event"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// Pt is a shorthand for creating a gamemap.Point.
func Pt(x, y float64) gamemap.Point {
	return gamemap.Point{X: x, Y: y}
}

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
	key             string
	x, y, dmg, rng  float64
	attackSpeed     float64
	style           string
	projectileSpeed float64
}

type effectSpec struct {
	enemyIdx int
	kind     string // "slow", "stun", "burn", "bleed", "root"
	factor   float64
	duration float64
	dps      float64
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
// Default: attackSpeed=1, style=projectile, projectileSpeed=300.
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
	if len(b.waypoints) == 0 {
		b.waypoints = []gamemap.Point{
			{X: 0, Y: 100},
			{X: 600, Y: 100},
		}
	}

	poolSize := len(b.enemies) + 16
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
			// Clear spawn animation for test-spawned enemies so they are
			// immediately targetable and damageable in regression tests.
			e.SpawnTimer = 0
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
			FireTimer:       0,
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
		}
	}

	return s
}
