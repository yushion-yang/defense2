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

		// Fire projectile
		projectiles.Fire(t.X, t.Y, target.X, target.Y, t.Damage, projectileSpeed, projectileRadius)
		t.FireTimer = 1.0 / t.AttackSpeed
	})
}

// TickProjectileHits checks projectile-enemy collisions.
// Returns total kills this tick.
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool) int {
	kills := 0
	projectiles.Each(func(p *projectile.Projectile) {
		enemies.Each(func(e *enemy.Enemy) {
			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist <= p.Radius+e.Radius {
				e.HP -= p.Damage
				projectiles.Release(p)
				if e.HP <= 0 {
					enemies.Kill(e)
					kills++
				}
			}
		})
	})
	return kills
}
