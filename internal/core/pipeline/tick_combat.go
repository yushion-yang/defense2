// tick_combat.go — 战斗管线：塔→弹射物→敌人的完整战斗流程。
//
// 包含三个子管线（按执行顺序）：
//  1. TickTowerCombat        — 塔的索敌→冷却→开火
//  2. TickProjectileHits     — 弹射物的碰撞检测→命中处理→击杀
//  3. TickEnemyStatusEffects — 敌人状态效果 tick（DoT 结算/HP 归零击杀）
//
// 关系：
//   - 被 stage.go 通过 Func() 包装为 TickSystem 注册到 Orchestrator
//   - 依赖 combat.ApplyHit 做命中伤害结算（含暴击/溅射/弹射/CC）
//   - 依赖 physics.SpatialGrid 加速弹射物碰撞查询（从 O(P×E) 降至 O(P×k)）
package pipeline

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/physics"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

const (
	fireAnimDuration     = 0.4   // 开火动画时长(秒)，用于渲染层的射击闪光/后坐动画
	defaultStrengthValue = 100.0 // 多目标能力计算的默认 strength 基准（无 Strength 时的兜底值）
	_                    = 0     // maxShotsPerFrame 已移除：每塔每帧最多射击 1 次（防止高倍速下爆发）
)

// TickTowerCombat 塔战斗子管线：遍历所有塔，按攻击方式分发射击逻辑。
//
// 流程概览（每座塔）：
//  1. 跳过正在出售的塔
//  2. 衰减射击动画计时器（纯视觉，不影响逻辑）
//  3. 解析攻击方式（ResolveAttackStyle：受能力改变，如 scatter/barrage）
//  4. 自管理攻击方式（barrage 等）走独立 tick 路径，不受标准冷却控制
//  5. 标准流程：冷却递减 → 索敌 → 转向 → 开火 → 多目标额外射击 → 重置冷却
//
// beams 可为 nil（无 beam 渲染支持时），所有回调均可为 nil。
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

		// 自管理攻击方式（如 barrage/连击）：不走标准"冷却→索敌→开火"流程，
		// 而是由 handler 自行管理 burst 节奏（如每 0.08s 一颗追踪弹）。
		if combat.IsSelfManaged(style) {
			combat.TickSelfManaged(t, ctx)
			return
		}

		// 标准冷却流程：每帧最多射击 1 次。
		// 关键：FireTimer 钳制为 0 而非允许负值累积。
		// 原因：波间空闲期塔无目标，若允许负值累积，开波瞬间会因
		// "欠了很多冷却"而连续爆发多次射击，破坏游戏平衡。
		t.FireTimer -= dt
		if t.FireTimer > 0 {
			return
		}
		t.FireTimer = 0 // 钳制，不累积

		handler := combat.Get(style)
		if handler == nil {
			return
		}

		target := tower.AcquireTarget(t, enemies)
		if target == nil {
			return
		}

		t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)
		handler.Fire(t, target, ctx)

		// 多目标攻击（multiTarget 能力）：主目标已射击，额外目标各追加一颗弹。
		// 额外目标由 FindExtraTargets 从射程内排除主目标后选取。
		if extra := multiTargetCount(t); extra > 0 {
			targets := tower.FindExtraTargets(t, enemies, extra, target)
			for _, et := range targets {
				handler.Fire(t, et, ctx)
			}
		}

		t.FireTimer = 1.0 / t.AttackSpeed // 重置冷却
		t.FireAnim = fireAnimDuration
		if onFire != nil {
			onFire(t, string(style))
		}
	})
}

// HitCallback 弹射物命中回调（用于生成飘字、音效等）。
type HitCallback = combat.HitCallback

// TickProjectileHits 弹射物碰撞子管线：检测碰撞 → 触发能力 → 扣血 → 击杀。
// 返回本帧击杀数。onHit 可为 nil。
//
// 碰撞模型（塔防特有，不同于物理引擎的通用碰撞）：
//   - 追踪弹（Target != nil, Penetrate=false）：只和锁定目标做距离判定，
//     飞行途中穿过其他敌人不触发碰撞。这是最常见的类型。
//     优势：无需空间查询，O(1) 单目标检测。
//   - 穿透弹（Penetrate=true，如散射/环射）：对路径上所有敌人碰撞，
//     命中后不消失继续飞行，通过 PenHitIDs 防止重复命中同一目标。
//   - 非追踪非穿透弹（如 mortar 落点）：空间查询最近敌人，命中后消失。
//
// 空间查询策略：
//   - grid != nil → 使用 SpatialGrid 查询（64px cell，O(k) 候选）
//   - grid == nil → 回退全量遍历（仅测试/debug 场景）
func TickProjectileHits(projectiles *projectile.Pool, enemies *enemy.Pool, towers *tower.Pool, grid *physics.SpatialGrid, onHit HitCallback, onCC combat.CCCallback, onSplashVFX func(x, y, radius float64)) int {
	kills := 0

	projectiles.Each(func(p *projectile.Projectile) {
		if !p.Active {
			return
		}

		// ── 路径 A：追踪弹 ── 只和锁定目标碰撞，O(1) 检测
		if p.Target != nil && !p.Penetrate {
			e := p.Target
			// ID 校验防止目标被回收后池槽位复用导致打错敌人
			if !e.Active || e.IsDying() || e.IsSpawning() || e.ID != p.TargetID {
				return
			}
			dx := p.X - e.X
			dy := p.Y - e.Y
			if math.Hypot(dx, dy) > p.Radius+e.Radius {
				return
			}
			kills += processHit(p, e, towers, enemies, projectiles, onHit, onCC, onSplashVFX)
			return
		}

		// ── 路径 B：非追踪弹/穿透弹 ── 需要空间查询检测附近所有敌人
		checkEnemy := func(e *enemy.Enemy) {
			if !p.Active || !e.Active || e.IsDying() || e.IsSpawning() {
				return
			}
			dx := p.X - e.X
			dy := p.Y - e.Y
			if math.Hypot(dx, dy) > p.Radius+e.Radius {
				return
			}
			// 穿透弹防重复命中：已命中过的目标 ID 记录在 PenHitIDs 中
			if p.Penetrate {
				for _, hitID := range p.PenHitIDs {
					if hitID == e.ID {
						return
					}
				}
			}
			kills += processHit(p, e, towers, enemies, projectiles, onHit, onCC, onSplashVFX)
		}

		if grid != nil {
			// +32 是搜索半径余量：确保边缘敌人（最大半径约 24px）不被遗漏
			for _, idx := range grid.Query(p.X, p.Y, p.Radius+32) {
				if !p.Active {
					break
				}
				checkEnemy(enemies.ByIndex(idx))
			}
		} else {
			enemies.Each(func(e *enemy.Enemy) {
				checkEnemy(e)
			})
		}
	})

	return kills
}

// processHit 统一命中处理（追踪弹和空间查询两条路径共用）。
//
// 流程：
//  1. 通过 SourceTowerKey 反查发射塔（可能已被卖掉，此时 srcTower=nil）
//  2. 确定 hitStyle（弹射命中覆写为 "bounce"，因为弹射弹的伤害/VFX 有独立逻辑）
//  3. 委托 combat.ApplyHit 做完整伤害结算（暴击/溅射/弹射/CC/击杀）
//  4. 穿透弹记录已命中 ID 继续飞行；普通弹命中后立即回收
//  5. 返回击杀数（含溅射/弹射等连锁击杀）
func processHit(p *projectile.Projectile, e *enemy.Enemy, towers *tower.Pool, enemies *enemy.Pool, projectiles *projectile.Pool, onHit HitCallback, onCC combat.CCCallback, onSplashVFX func(x, y, radius float64)) int {
	var srcTower *tower.Tower
	if p.SourceTowerKey != "" {
		srcTower = towers.ByInstanceKey(p.SourceTowerKey)
	}

	hitStyle := ""
	if srcTower != nil {
		hitStyle = string(srcTower.AttackStyleID)
	}
	// 弹射弹（BounceCount > 0）覆写 style 为 "bounce"，
	// 使 ApplyHit 走弹射专用伤害衰减逻辑而非原始攻击方式。
	if p.BounceCount > 0 {
		hitStyle = "bounce"
	}

	out := combat.ApplyHit(combat.HitInput{
		Tower: srcTower, Target: e, BaseDamage: p.Damage, Style: hitStyle,
		Enemies: enemies, Projectiles: projectiles, Projectile: p, OnCC: onCC,
		OnSplashVFX: onSplashVFX,
	}, onHit)

	killed := 0
	if out.Killed {
		killed = 1
	}

	// 穿透弹：记录命中 ID 后继续飞行，除非被能力阻断（如 damageCap 挡住）
	// 普通弹：命中即回收
	if p.Penetrate {
		p.PenHitIDs = append(p.PenHitIDs, e.ID)
		if out.ProjectileBlocked {
			projectiles.Release(p)
		}
	} else {
		projectiles.Release(p)
	}

	return killed
}

// multiTargetCount 返回塔的多目标额外目标数（不含主目标）。
// 公式: totalTargets = floor(CalcScale(strength)) — 由能力配置定义 base+potential 缩放。
// 返回值 = totalTargets - 1（减去已射击的主目标）。
// 无 multiTarget 能力时返回 0，主流程不会调用 FindExtraTargets。
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

// TickEnemyStatusEffects 敌人状态效果子管线。
//
// 流程：
//  1. 跳过正在死亡/出生的敌人
//  2. TickStatusEffects: buff 时间递减、CC 过期移除、DoT 伤害累计到 LastDotDmg
//  3. 若有 DoT 伤害，走 ApplyDamage 完整管线（含虚弱增伤/坚韧减伤/无敌免疫等）
//  4. HP ≤ 0 的敌人调用 Kill（触发击杀奖励 + 死亡效果如分裂）
//
// DoT 被归类为 DmgMagic（魔法伤害），受虚弱增伤但不穿无敌。
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
