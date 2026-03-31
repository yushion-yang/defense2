// sim.go — Lightweight pure-logic game simulator for regression testing.
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

	// SpawnedEnemies tracks enemy pointers in spawn order (set by builder).
	SpawnedEnemies []*enemy.Enemy
}

// Step advances the simulation by one tick.
func (s *Sim) Step() {
	s.Tick++
	dt := s.DT

	// 1. Enemy movement + dying cleanup
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

		// Update tower angle toward target
		tw.Angle = math.Atan2(target.Y-tw.Y, target.X-tw.X)

		switch tw.AttackStyleID {
		case tower.StyleSpinAoE, tower.StyleAuraDot:
			// AoE damage: hit all enemies in range
			s.Enemies.Each(func(e *enemy.Enemy) {
				if e.IsDying() {
					return
				}
				dist := math.Hypot(e.X-tw.X, e.Y-tw.Y)
				if dist > tw.Range {
					return
				}
				result := combat.ProcessDamage(combat.DamageInput{
					Target:    e,
					RawDamage: tw.Damage,
				})
				if result.Killed {
					s.Kills++
					s.Gold += e.Reward
					s.Enemies.Kill(e)
				}
			})
		case tower.StyleLaser, tower.StyleWideBeam:
			// Direct single-target damage
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
			// Projectile-based
			speed := tw.ProjectileSpeed
			if speed == 0 {
				speed = 300
			}
			s.Projectiles.Fire(tw.X, tw.Y, target.X, target.Y,
				tw.Damage, speed, 4, target, tw.InstanceKey)
		}
	}

	// 3. Projectile update (tracking + movement)
	s.Projectiles.Update(dt)

	// 4. Projectile hit detection
	s.Projectiles.Each(func(p *projectile.Projectile) {
		s.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() || !p.Active {
				return
			}
			// Standard tracking: only hit locked target
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
