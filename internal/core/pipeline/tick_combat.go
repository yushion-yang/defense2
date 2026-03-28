// tick_combat.go — 战斗管线。
// 包含塔索敌射击、弹射物碰撞检测+能力触发、敌人状态效果处理三个子管线。
package pipeline

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

const projectileSpeed = 300.0 // 弹射物飞行速度（像素/秒）
const projectileRadius = 4.0  // 弹射物碰撞半径（像素）

// TickTowerCombat 塔战斗子管线：索敌 → 冷却检查 → 发射弹射物。
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
		t.FireTimer = 1.0 / t.AttackSpeed // 冷却时间 = 1 / 攻速
	})
}

// TickProjectileHits 弹射物碰撞子管线：检测碰撞 → 触发能力 → 扣血 → 击杀。
// 返回本帧击杀数。
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool) int {
	kills := 0
	projectiles.Each(func(p *projectile.Projectile) {
		enemies.Each(func(e *enemy.Enemy) {
			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return // 未碰撞
			}

			totalDamage := p.Damage

			// 查找发射该弹射物的塔并触发能力
			// TODO: 后续用弹射物标记来源塔 ID，目前简化处理
			var srcTower *tower.Tower
			towers.Each(func(t *tower.Tower) {
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

// applyHitEffects 将能力效果（减速、眩晕、流血、溅射）施加到目标及周围敌人。
func applyHitEffects(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, enemies *enemy.Pool) {
	// 减速
	if r.Slow != nil {
		target.SlowTimer = r.Slow.Duration
		target.SlowFactor = r.Slow.Factor
		target.Speed = target.BaseSpeed * r.Slow.Factor
	}
	// 眩晕
	if r.Stun != nil {
		target.StunTimer = r.Stun.Duration
	}
	// 流血
	if r.Bleed != nil {
		target.BleedTimer = r.Bleed.Duration
		target.BleedDPS = r.Bleed.DPS
	}
	// 溅射：对目标周围敌人造成比例伤害
	if r.Splash != nil {
		splashDamage := p.Damage * r.Splash.Ratio
		enemies.Each(func(e *enemy.Enemy) {
			if e == target {
				return // 跳过已命中的目标
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

// TickEnemyStatusEffects 敌人状态效果子管线：处理所有敌人的减速/流血，击杀血量归零的敌人。
func TickEnemyStatusEffects(enemies *enemy.Pool, dt float64) {
	enemies.Each(func(e *enemy.Enemy) {
		enemy.TickStatusEffects(e, dt)
		if e.HP <= 0 && e.Active {
			enemies.Kill(e)
		}
	})
}
