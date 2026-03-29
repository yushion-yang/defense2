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
func TickTowerCombat(towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, beams *combat.BeamPool, dt float64, onFire func(style string), onHit combat.HitCallback) {
	ctx := &combat.AttackContext{
		Enemies:     enemies,
		Projectiles: projectiles,
		Beams:       beams,
		OnFire:      onFire,
		OnHit:       onHit,
		DT:          dt,
		// 能力触发回调：供 spin_aoe 等非弹射物攻击方式使用
		OnAbilityHit: func(t *tower.Tower, e *enemy.Enemy, damage float64) float64 {
			return applyTowerAbilities(t, e, damage, enemies, projectiles)
		},
	}

	towers.Each(func(t *tower.Tower) {
		// 射击动画衰减
		if t.FireAnim > 0 {
			t.FireAnim -= dt
		}

		// 技能压制普攻时跳过射击
		if t.SkillSuppressFire {
			return
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
			onFire(string(style))
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
//   - 穿刺弹（Pierce=true）：对路径上所有敌人碰撞，命中后继续飞行
//   - 散射弹（ScatterGroup>0）：路径碰撞，同组命中同敌人合并为一次伤害
//   - 散射视觉弹（ScatterVisual）：不参与碰撞（旧版兼容）
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool, onHit HitCallback) int {
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

			// 追踪弹只和锁定目标碰撞（穿刺弹和散射弹除外）
			if p.Target != nil && !p.Pierce && p.ScatterGroup == 0 && e != p.Target {
				return
			}

			dx := p.X - e.X
			dy := p.Y - e.Y
			dist := math.Hypot(dx, dy)
			if dist > p.Radius+e.Radius {
				return
			}

			// 穿刺弹：跳过已命中的敌人
			if p.Pierce {
				for _, hitID := range p.PierceHitIDs {
					if hitID == e.ID {
						return
					}
				}
			}

			// ── 散射弹（穿透）：跳过已命中敌人，记录命中，延迟合并处理 ──
			if p.ScatterGroup > 0 {
				// 穿透：跳过已命中的敌人
				for _, hitID := range p.PierceHitIDs {
					if hitID == e.ID {
						return
					}
				}
				p.PierceHitIDs = append(p.PierceHitIDs, e.ID)

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

			// ── 普通弹/追踪弹/穿刺弹：即时处理 ──
			totalDamage := p.Damage

			var srcTower *tower.Tower
			if p.SourceTowerKey != "" {
				towers.Each(func(t *tower.Tower) {
					if srcTower == nil && t.InstanceKey == p.SourceTowerKey {
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

			// Shield 吸收
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
				hitStyle := ""
				if srcTower != nil {
					hitStyle = string(srcTower.AttackStyleID)
				}
				onHit(e, totalDamage, killed, hitStyle)
			}

			if killed {
				// 死亡爆炸：检查来源塔是否有 deathMark 能力
				if srcTower != nil {
					kills += applyDeathExplosion(srcTower, e, enemies, onHit)
				}
				enemies.Kill(e)
				kills++
			}

			if p.Pierce {
				p.PierceHitIDs = append(p.PierceHitIDs, e.ID)
				p.PierceCount++
				p.Damage *= p.PierceDecay
				if p.PierceCount >= p.PierceMax {
					projectiles.Release(p)
				} else {
					retargetPierce(p, e, enemies)
				}
			} else {
				projectiles.Release(p)
			}
		})
	})

	// ── 散射命中合并处理 ──
	for _, sh := range scatterHits {
		e := sh.enemy
		if !e.Active {
			continue
		}
		totalDamage := sh.damage * float64(sh.count)

		// 查找来源塔，触发 OnHit 能力（以合并伤害为基准）
		var srcTower *tower.Tower
		if sh.towerKey != "" {
			towers.Each(func(t *tower.Tower) {
				if srcTower == nil && t.InstanceKey == sh.towerKey {
					srcTower = t
				}
			})
		}
		if srcTower != nil {
			totalDamage += applyTowerAbilities(srcTower, e, totalDamage, enemies, projectiles)
		}

		// Shield 吸收
		if e.ShieldHP > 0 {
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
			onHit(e, totalDamage, killed, "scatter")
		}
		if killed {
			enemies.Kill(e)
			kills++
		}
	}

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
	// 减速（走 crowd_control 统一逻辑：免疫检查 + 韧性减免 + 减速下限）
	if r.Slow != nil {
		combat.ApplySlow(target, r.Slow.Factor, r.Slow.Duration, p.SourceTowerKey)
	}
	// 眩晕（走 crowd_control 统一逻辑：免疫检查 + 韧性减免）
	if r.Stun != nil {
		combat.ApplyStun(target, r.Stun.Duration, p.SourceTowerKey)
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
	// 弹射：向附近最近的未命中敌人发射衰减弹射物
	if r.Bounce != nil && p.BounceCount < r.Bounce.MaxBounces {
		// 构建已命中列表（当前目标 + 历史命中）
		hitIDs := append([]int{}, p.BounceHitIDs...)
		hitIDs = append(hitIDs, target.ID)

		var best *enemy.Enemy
		bestDist := r.Bounce.Range
		enemies.Each(func(e2 *enemy.Enemy) {
			// 排除所有已命中的敌人（不弹回）
			for _, id := range hitIDs {
				if e2.ID == id {
					return
				}
			}
			d := math.Hypot(e2.X-target.X, e2.Y-target.Y)
			if d < bestDist {
				bestDist = d
				best = e2
			}
		})
		if best != nil && projectiles != nil {
			// 弹射伤害 = 塔原始伤害 * 固定比例（不递减）
			bounceDmg := r.Bounce.SrcDamage * r.Bounce.DamageRatio
			projectiles.FireBounce(target.X, target.Y, best, bounceDmg, p.Speed, p.Radius, p.SourceTowerKey, p.BounceCount+1, hitIDs)
		}
	}
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

// applyDeathExplosion 检查塔是否有 deathMark 能力，若有则对被杀敌人周围造成 AoE 爆炸。
// 返回爆炸击杀数。
func applyDeathExplosion(t *tower.Tower, killed *enemy.Enemy, enemies *enemy.Pool, onHit combat.HitCallback) int {
	for _, aName := range t.Abilities {
		if aName != "deathMark" {
			continue
		}
		abTable := config.GlobalAbilityTable()
		if abTable == nil {
			return 0
		}
		def, ok := abTable["deathMark"]
		if !ok {
			return 0
		}
		str := 100.0
		if t.Strength != nil {
			str = t.Strength.Effective()
		}
		explodeDmg := def.CalcScale(str)
		explodeR := def.Param
		extraKills := 0
		enemies.Each(func(e2 *enemy.Enemy) {
			if e2 == killed {
				return
			}
			if math.Hypot(e2.X-killed.X, e2.Y-killed.Y) <= explodeR {
				e2.HP -= explodeDmg
				if onHit != nil {
					onHit(e2, explodeDmg, e2.HP <= 0, "explosion")
				}
				if e2.HP <= 0 {
					enemies.Kill(e2)
					extraKills++
				}
			}
		})
		return extraKills
	}
	return 0
}

// applyTowerAbilities 触发塔的所有 OnHit 能力，返回额外伤害并应用效果。
// 用于散射合并和 spin_aoe 等非标准弹射物路径。
func applyTowerAbilities(t *tower.Tower, e *enemy.Enemy, hitDamage float64, enemies *enemy.Pool, projectiles *projectile.Pool) float64 {
	synth := &projectile.Projectile{
		Damage:         hitDamage,
		SourceTowerKey: t.InstanceKey,
	}
	bonus := 0.0
	for _, aName := range t.Abilities {
		ab, ok := tower.Registry[aName]
		if !ok {
			continue
		}
		result := ab.OnHit(t, synth, e)
		if result == nil {
			continue
		}
		bonus += result.BonusDamage
		applyHitEffects(result, e, synth, enemies, projectiles)
	}
	return bonus
}

// TickEnemyStatusEffects 敌人状态效果子管线：处理所有敌人的减速/流血，击杀血量归零的敌人。
func TickEnemyStatusEffects(enemies *enemy.Pool, dt float64, onDotDmg func(e *enemy.Enemy, dmg float64)) {
	enemies.Each(func(e *enemy.Enemy) {
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
