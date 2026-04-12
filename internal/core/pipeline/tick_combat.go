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

const (
	fireAnimDuration     = 0.4   // 开火动画时长(秒)
	defaultStrengthValue = 100.0 // 默认强度基准值
	maxShotsPerFrame     = 20    // 单塔单帧最大射击次数（防止极端攻速卡死）
)

// TickTowerCombat 塔战斗子管线：按攻击方式分发射击逻辑。
// beams 可为 nil（无 beam 渲染支持时），onFire/onHit 可为 nil。
func TickTowerCombat(towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, beams *combat.BeamPool, dt float64, onFire func(*tower.Tower, string), onHit combat.HitCallback, onCC combat.CCCallback, onSplashVFX func(x, y, radius float64)) {
	ctx := &combat.AttackContext{
		Enemies:     enemies,
		Projectiles: projectiles,
		Beams:       beams,
		OnFire:      onFire,
		OnHit:       onHit,
		OnCC:        onCC,
		OnSplashVFX: onSplashVFX,
		DT:          dt,
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

		// 标准冷却流程：循环消耗 timer，允许超高攻速每帧多次射击。
		// maxShotsPerFrame 防止极端攻速导致单帧卡死。
		t.FireTimer -= dt
		if t.FireTimer > 0 {
			return
		}

		handler := combat.Get(style)
		if handler == nil {
			return
		}

		shotInterval := 1.0 / t.AttackSpeed
		shots := 0
		for t.FireTimer <= 0 && shots < maxShotsPerFrame {
			target := tower.AcquireTarget(t, enemies)
			if target == nil {
				break
			}

			t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)
			handler.Fire(t, target, ctx)

			// 多目标攻击：对额外目标各发射一颗弹
			if extra := multiTargetCount(t); extra > 0 {
				targets := tower.FindExtraTargets(t, enemies, extra, target)
				for _, et := range targets {
					handler.Fire(t, et, ctx)
				}
			}

			t.FireTimer += shotInterval
			shots++
		}

		if shots > 0 {
			t.FireAnim = fireAnimDuration
			if onFire != nil {
				onFire(t, string(style))
			}
		}
	})
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = combat.HitCallback

// TickProjectileHits 弹射物碰撞子管线：检测碰撞 → 触发能力 → 扣血 → 击杀。
// 返回本帧击杀数。onHit 可为 nil。
//
// 碰撞规则（塔防模型）：
//   - 追踪弹（Target != nil）：只和锁定目标碰撞，穿过其他敌人
//   - 穿透弹（Penetrate=true, 含散射/环射）：对路径上所有敌人碰撞，命中后继续飞行
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool, onHit HitCallback, onCC combat.CCCallback, onSplashVFX func(x, y, radius float64)) int {
	kills := 0

	// Tower lookup via ByInstanceKey (O(1) grid index, zero allocation)

	projectiles.Each(func(p *projectile.Projectile) {
		enemies.Each(func(e *enemy.Enemy) {
			if !p.Active || e.IsDying() || e.IsSpawning() {
				return
			}

			// 追踪弹只和锁定目标碰撞（穿透弹除外）
			// ID 校验防止敌人槽位复用后误命中新敌人（ABA 问题）
			if p.Target != nil && !p.Penetrate {
				if e != p.Target || p.Target.ID != p.TargetID {
					return
				}
			}

			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return
			}

			// 穿透弹：跳过已命中的敌人
			if p.Penetrate {
				for _, hitID := range p.PenHitIDs {
					if hitID == e.ID {
						return
					}
				}
			}

			// ── 统一命中处理 ──
			var srcTower *tower.Tower
			if p.SourceTowerKey != "" {
				srcTower = towers.ByInstanceKey(p.SourceTowerKey)
			}

			hitStyle := ""
			if srcTower != nil {
				hitStyle = string(srcTower.AttackStyleID)
			}
			// 弹射弹强制标记为 bounce（不依赖塔的 AttackStyleID）
			if p.BounceCount > 0 {
				hitStyle = "bounce"
			}
			out := combat.ApplyHit(combat.HitInput{
				Tower: srcTower, Target: e, BaseDamage: p.Damage, Style: hitStyle,
				Enemies: enemies, Projectiles: projectiles, Projectile: p, OnCC: onCC,
				OnSplashVFX: onSplashVFX,
			}, onHit)

			if out.Killed {
				kills += 1 + out.ExtraKills
			}

			if p.Penetrate {
				p.PenHitIDs = append(p.PenHitIDs, e.ID)
				// 弹幕盾：阻止穿透弹继续飞行（通过 ApplyHit 返回的统一标记）
				if out.ProjectileBlocked {
					projectiles.Release(p)
				}
			} else {
				projectiles.Release(p)
			}
		})
	})

	return kills
}

// multiTargetCount 返回塔的多目标额外目标数（不含主目标）。
// 公式: targets = floor(base + potential * (strength/100)) - 1（减去主目标）。
// 无 multiTarget 能力时返回 0。
func multiTargetCount(t *tower.Tower) int {
	for _, aName := range t.Abilities {
		if aName == tower.AbilityMultiTarget {
			abTable := config.GlobalAbilityTable()
			if abTable == nil {
				return 1
			}
			def, ok := abTable[tower.AbilityMultiTarget]
			if !ok {
				return 1
			}
			str := defaultStrengthValue
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
// DoT 伤害通过 ApplyDamage 管线结算（走虚弱/坚韧/免疫/减伤等完整流程）。
func TickEnemyStatusEffects(enemies *enemy.Pool, dt float64, onDotDmg func(e *enemy.Enemy, dmg float64)) {
	enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		enemy.TickStatusEffects(e, dt)
		// DoT tick 触发时走伤害管线
		if e.LastDotDmg > 0 {
			result := combat.ApplyDamage(combat.DamageInput{
				Target:      e,
				RawDamage:   e.LastDotDmg,
				DamageType:  combat.DmgMagic, // DoT 为魔法伤害（受虚弱/坚韧影响，不穿无敌）
				SourceLabel: "dot",
			})
			if onDotDmg != nil && result.FinalDamage > 0 {
				onDotDmg(e, result.FinalDamage)
			}
			e.LastDotDmg = 0
		}
		if e.HP <= 0 && e.Active {
			enemies.Kill(e)
		}
	})
}
