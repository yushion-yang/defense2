// tick_combat.go — 战斗管线。
// 包含塔索敌射击、弹射物碰撞检测+能力触发、敌人状态效果处理三个子管线。
package pipeline

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

// TickTowerCombat 塔战斗子管线：按攻击方式分发射击逻辑。
// beams 可为 nil（无 beam 渲染支持时），onFire/onHit 可为 nil。
func TickTowerCombat(towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, beams *combat.BeamPool, dt float64, onFire func(*tower.Tower, string), onHit combat.HitCallback, onCC combat.CCCallback) {
	ctx := &combat.AttackContext{
		Enemies:     enemies,
		Projectiles: projectiles,
		Beams:       beams,
		OnFire:      onFire,
		OnHit:       onHit,
		OnCC:        onCC,
		DT:          dt,
		// OnAbilityHit 已废弃：所有 handler 通过 ApplyHit 统一处理
	}

	towers.Each(func(t *tower.Tower) {
		// 正在出售的塔跳过战斗
		if t.Selling {
			return
		}

		// 射击动画衰减
		if t.FireAnim > 0 {
			t.FireAnim -= dt
		}

		style := t.ResolveAttackStyle()
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

			// 多目标攻击：对额外目标各发射一颗弹
			if extra := multiTargetCount(t); extra > 0 {
				targets := tower.FindExtraTargets(t, enemies, extra, target)
				for _, et := range targets {
					handler.Fire(t, et, ctx)
				}
			}
		}
		t.FireTimer = 1.0 / t.AttackSpeed
		t.FireAnim = 0.15
		if onFire != nil {
			onFire(t, string(style))
		}
	})
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = combat.HitCallback

// scatterHit 散射弹命中记录（同组同敌人合并）。
type scatterHit struct {
	enemy    *enemy.Enemy
	towerKey string
	count    int     // 命中弹丸数
	damage   float64 // 单颗伤害
}

// TickProjectileHits 弹射物碰撞子管线：检测碰撞 → 触发能力 → 扣血 → 击杀。
// 返回本帧击杀数。onHit 可为 nil。
//
// 碰撞规则（塔防模型）：
//   - 追踪弹（Target != nil）：只和锁定目标碰撞，穿过其他敌人
//   - 穿透弹（Penetrate=true）：对路径上所有敌人碰撞，命中后继续飞行
//   - 散射弹（ScatterGroup>0）：路径碰撞，同组命中同敌人合并为一次伤害
//   - 散射视觉弹（ScatterVisual）：不参与碰撞（旧版兼容）
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool, onHit HitCallback, onCC combat.CCCallback) int {
	kills := 0

	// 散射命中收集（key = groupID<<32|enemyID）
	scatterHits := map[int64]*scatterHit{}

	projectiles.Each(func(p *projectile.Projectile) {
		// 散射视觉弹不参与碰撞检测（旧版兼容）
		if p.ScatterVisual {
			return
		}

		enemies.Each(func(e *enemy.Enemy) {
			if !p.Active {
				return
			}
			if e.IsDying() {
				return
			}

			// 追踪弹只和锁定目标碰撞（穿透弹和散射弹除外）
			if p.Target != nil && !p.Penetrate && p.ScatterGroup == 0 && e != p.Target {
				return
			}

			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return
			}

			// 穿透弹/散射弹：跳过已命中的敌人
			if p.Penetrate || p.ScatterGroup > 0 {
				for _, hitID := range p.PenHitIDs {
					if hitID == e.ID {
						return
					}
				}
			}

			// ── 散射弹（穿透）：记录命中，延迟合并处理 ──
			if p.ScatterGroup > 0 {
				// 弹幕盾：吸收散射弹丸（不记录命中，释放弹丸）
				if e.ProjectileBlockChance > 0 && !e.AbilitySilenced {
					e.BlockFlash = 0.25
					projectiles.Release(p)
					return
				}
				p.PenHitIDs = append(p.PenHitIDs, e.ID)

				key := int64(p.ScatterGroup)<<32 | int64(e.ID)
				if sh, ok := scatterHits[key]; ok {
					sh.count++
				} else {
					scatterHits[key] = &scatterHit{
						enemy:    e,
						towerKey: p.SourceTowerKey,
						count:    1,
						damage:   p.Damage,
					}
				}
				// 不 Release：弹丸继续飞行穿透后续敌人，到 MaxRange 自然消亡
				return
			}

			// ── 普通弹/追踪弹/穿透弹：统一命中处理 ──
			var srcTower *tower.Tower
			if p.SourceTowerKey != "" {
				towers.Each(func(t *tower.Tower) {
					if srcTower == nil && t.InstanceKey == p.SourceTowerKey {
						srcTower = t
					}
				})
			}

			hitStyle := ""
			if srcTower != nil {
				hitStyle = string(srcTower.AttackStyleID)
			}
			out := combat.ApplyHit(combat.HitInput{
				Tower: srcTower, Target: e, BaseDamage: p.Damage, Style: hitStyle,
				Enemies: enemies, Projectiles: projectiles, Projectile: p, OnCC: onCC,
			}, onHit)

			if out.Killed {
				kills += 1 + out.ExtraKills
			}

			if p.Penetrate {
				p.PenHitIDs = append(p.PenHitIDs, e.ID)
				// 弹幕盾：阻止穿透弹继续飞行
				if e.ProjectileBlockChance > 0 && !e.AbilitySilenced {
					projectiles.Release(p)
				}
			} else {
				projectiles.Release(p)
			}
		})
	})

	// ── 散射命中合并处理 ──
	for _, sh := range scatterHits {
		e := sh.enemy
		if !e.Active || e.IsDying() {
			continue
		}
		mergedDamage := sh.damage * float64(sh.count)

		var srcTower *tower.Tower
		if sh.towerKey != "" {
			towers.Each(func(t *tower.Tower) {
				if srcTower == nil && t.InstanceKey == sh.towerKey {
					srcTower = t
				}
			})
		}

		out := combat.ApplyHit(combat.HitInput{
			Tower: srcTower, Target: e, BaseDamage: mergedDamage, Style: "scatter",
			Enemies: enemies, Projectiles: projectiles, OnCC: onCC,
		}, onHit)

		if out.Killed {
			kills += 1 + out.ExtraKills
		}
	}

	return kills
}


// multiTargetCount 返回塔的多目标额外目标数（不含主目标）。
// 公式: targets = floor(base + potential * (strength/100)) - 1（减去主目标）。
// 无 multiTarget 能力时返回 0。
func multiTargetCount(t *tower.Tower) int {
	for _, aName := range t.Abilities {
		if aName == "multiTarget" {
			abTable := config.GlobalAbilityTable()
			if abTable == nil {
				return 1
			}
			def, ok := abTable["multiTarget"]
			if !ok {
				return 1
			}
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			total := int(def.CalcScale(str)) // 总目标数（含主目标）
			if total < 1 {
				total = 1
			}
			return total - 1 // 额外目标数
		}
	}
	return 0
}

// TickEnemyStatusEffects 敌人状态效果子管线：处理所有敌人的减速/流血，击杀血量归零的敌人。
func TickEnemyStatusEffects(enemies *enemy.Pool, dt float64, onDotDmg func(e *enemy.Enemy, dmg float64)) {
	enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		enemy.TickStatusEffects(e, dt)
		// DoT tick 触发时弹浮字
		if e.LastDotDmg > 0 && onDotDmg != nil {
			onDotDmg(e, e.LastDotDmg)
			e.LastDotDmg = 0
		}
		if e.HP <= 0 && e.Active {
			enemies.Kill(e)
		}
	})
}
