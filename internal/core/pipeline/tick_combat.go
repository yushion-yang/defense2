// tick_combat.go — 战斗管线。
// 包含塔索敌射击、弹射物碰撞检测+能力触发、敌人状态效果处理三个子管线。
package pipeline

import (
	"math"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// TickTowerCombat 塔战斗子管线：按攻击方式分发射击逻辑。
// beams 可为 nil（无 beam 渲染支持时），onFire/onHit 可为 nil。
func TickTowerCombat(towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, beams *combat.BeamPool, dt float64, onFire func(style string), onHit combat.HitCallback) {
	ctx := &combat.AttackContext{
		Enemies:     enemies,
		Projectiles: projectiles,
		Beams:       beams,
		OnFire:      onFire,
		OnHit:       onHit,
		DT:          dt,
	}

	towers.Each(func(t *tower.Tower) {
		// 射击动画衰减
		if t.FireAnim > 0 {
			t.FireAnim -= dt
		}

		style := t.AttackStyleID
		if style == "" {
			style = tower.StyleProjectile
		}
		ctx.Style = string(style)

		// 自管理攻击方式：每帧 tick，不走标准冷却
		if combat.IsSelfManaged(style) {
			combat.TickSelfManaged(t, ctx)
			return
		}

		// 标准冷却流程
		t.FireTimer -= dt
		if t.FireTimer > 0 {
			return
		}

		target := tower.AcquireTarget(t, enemies)
		if target == nil {
			return
		}

		t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)

		handler := combat.Get(style)
		if handler != nil {
			handler.Fire(t, target, ctx)
		}
		t.FireTimer = 1.0 / t.AttackSpeed
		t.FireAnim = 0.15
		if onFire != nil {
			onFire(string(style))
		}
	})
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = combat.HitCallback

// TickProjectileHits 弹射物碰撞子管线：检测碰撞 → 触发能力 → 扣血 → 击杀。
// 返回本帧击杀数。onHit 可为 nil。
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool, onHit HitCallback) int {
	kills := 0
	projectiles.Each(func(p *projectile.Projectile) {
		// 散射视觉弹不参与碰撞检测
		if p.ScatterVisual {
			return
		}

		enemies.Each(func(e *enemy.Enemy) {
			if !p.Active {
				return // 已被前一个碰撞消耗
			}
			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return // 未碰撞
			}

			// 穿刺弹：跳过已命中的敌人
			if p.Pierce {
				for _, hitID := range p.PierceHitIDs {
					if hitID == e.ID {
						return
					}
				}
			}

			totalDamage := p.Damage

			// 查找发射该弹射物的来源塔（按 Key 匹配）
			var srcTower *tower.Tower
			if p.SourceTowerKey != "" {
				towers.Each(func(t *tower.Tower) {
					if srcTower == nil && t.Key == p.SourceTowerKey {
						srcTower = t
					}
				})
			}

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
					applyHitEffects(result, e, p, enemies, projectiles)
				}
			}

			// Shield 吸收（检查 shieldIgnore）
			hasShieldIgnore := false
			if srcTower != nil {
				for _, aName := range srcTower.Abilities {
					if aName == "shieldIgnore" {
						hasShieldIgnore = true
						break
					}
				}
			}
			if e.ShieldHP > 0 && !hasShieldIgnore {
				if totalDamage <= e.ShieldHP {
					e.ShieldHP -= totalDamage
					totalDamage = 0
				} else {
					totalDamage -= e.ShieldHP
					e.ShieldHP = 0
				}
			}

			e.HP -= totalDamage
			killed := e.HP <= 0
			if onHit != nil {
				// 弹射物命中时使用来源塔的攻击方式
				hitStyle := ""
				if srcTower != nil {
					hitStyle = string(srcTower.AttackStyleID)
				}
				onHit(e, totalDamage, killed, hitStyle)
			}

			if killed {
				enemies.Kill(e)
				kills++
			}

			// 穿刺弹：命中后继续飞行
			if p.Pierce {
				p.PierceHitIDs = append(p.PierceHitIDs, e.ID)
				p.PierceCount++
				p.Damage *= p.PierceDecay
				if p.PierceCount >= p.PierceMax {
					projectiles.Release(p)
				} else {
					// 寻找下一个最近的未命中敌人重定向
					retargetPierce(p, e, enemies)
				}
			} else {
				projectiles.Release(p)
			}
		})
	})
	return kills
}

// retargetPierce 穿刺弹命中后寻找下一个最近目标。
func retargetPierce(p *projectile.Projectile, justHit *enemy.Enemy, enemies *enemy.Pool) {
	var best *enemy.Enemy
	bestDist := 300.0 // 穿刺搜索范围
	enemies.Each(func(e *enemy.Enemy) {
		if e == justHit {
			return
		}
		for _, id := range p.PierceHitIDs {
			if id == e.ID {
				return
			}
		}
		d := math.Hypot(e.X-justHit.X, e.Y-justHit.Y)
		if d < bestDist {
			bestDist = d
			best = e
		}
	})
	if best != nil {
		p.Target = best
	} else {
		p.Target = nil // 无目标，直线飞行至消亡
	}
}

// applyHitEffects 将能力效果（减速、眩晕、流血、灼烧、溅射、弹射）施加到目标及周围敌人。
func applyHitEffects(r *tower.HitResult, target *enemy.Enemy, p *projectile.Projectile, enemies *enemy.Pool, projectiles *projectile.Pool) {
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
	// 灼烧（独立于流血）
	if r.Burn != nil {
		target.BurnTimer = r.Burn.Duration
		target.BurnDPS = r.Burn.DPS
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
	// 弹射：向附近最近敌人发射衰减弹射物
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces {
		var best *enemy.Enemy
		bestDist := r.Bounce.Range
		enemies.Each(func(e2 *enemy.Enemy) {
			if e2 == target {
				return
			}
			d := math.Hypot(e2.X-target.X, e2.Y-target.Y)
			if d < bestDist {
				bestDist = d
				best = e2
			}
		})
		if best != nil && projectiles != nil {
			bounceDmg := p.Damage * r.Bounce.DamageDecay
			projectiles.FireBounce(target.X, target.Y, best, bounceDmg, p.Speed, p.Radius, p.SourceTowerKey, p.BounceCount+1)
		}
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
