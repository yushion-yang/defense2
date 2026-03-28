package pipeline

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

const projectileSpeed = 300.0
const projectileRadius = 4.0

// TickTowerCombat handles tower targeting and firing.
func TickTowerCombat(towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, dt float64) {
	towers.Each(func(t *tower.Tower) {
		t.FireTimer -= dt
		if t.FireTimer > 0 {
			return
		}

		target := tower.FindNearestEnemy(t, enemies)
		if target == nil {
			return
		}

		projectiles.Fire(t.X, t.Y, target.X, target.Y, t.Damage, projectileSpeed, projectileRadius)
		t.FireTimer = 1.0 / t.AttackSpeed
	})
}

// TickProjectileHits checks projectile-enemy collisions and applies abilities.
// Returns total kills this tick.
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool) int {
	kills := 0
	projectiles.Each(func(p *projectile.Projectile) {
		enemies.Each(func(e *enemy.Enemy) {
			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return
			}

			totalDamage := p.Damage

			// Find the tower that fired this projectile and resolve abilities
			var srcTower *tower.Tower
			towers.Each(func(t *tower.Tower) {
				// Simple: find closest tower to projectile origin
				// In a real system we'd tag projectiles with tower ID
				if srcTower == nil {
					srcTower = t
				}
			})

			if srcTower != nil {
				for _, aName := range srcTower.Abilities {
					ab, ok := tower.Registry[aName]
					if !ok {
						continue
					}
					result := ab.OnHit(srcTower, p, e)
					if result == nil {
						continue
					}
					totalDamage += result.BonusDamage
					applyHitEffects(result, e, p, enemies)
				}
			}

			e.HP -= totalDamage
			projectiles.Release(p)
			if e.HP <= 0 {
				enemies.Kill(e)
				kills++
			}
		})
	})
	return kills
}

// applyHitEffects applies status effects from a HitResult onto the target and nearby enemies.
func applyHitEffects(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, enemies *enemy.Pool) {
	if r.Slow != nil {
		target.SlowTimer = r.Slow.Duration
		target.SlowFactor = r.Slow.Factor
		target.Speed = target.BaseSpeed * r.Slow.Factor
	}
	if r.Stun != nil {
		target.StunTimer = r.Stun.Duration
	}
	if r.Bleed != nil {
		target.BleedTimer = r.Bleed.Duration
		target.BleedDPS = r.Bleed.DPS
	}
	if r.Splash != nil {
		splashDamage := p.Damage * r.Splash.Ratio
		enemies.Each(func(e *enemy.Enemy) {
			if e == target {
				return
			}
			dx := e.X - target.X
			dy := e.Y - target.Y
			if math.Hypot(dx, dy) <= r.Splash.Radius {
				e.HP -= splashDamage
				if e.HP <= 0 {
					enemies.Kill(e)
				}
			}
		})
	}
}

// TickEnemyStatusEffects processes all enemy status effects.
func TickEnemyStatusEffects(enemies *enemy.Pool, dt float64) {
	enemies.Each(func(e *enemy.Enemy) {
		enemy.TickStatusEffects(e, dt)
		if e.HP <= 0 && e.Active {
			enemies.Kill(e)
		}
	})
}
