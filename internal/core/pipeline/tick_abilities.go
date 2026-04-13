// tick_abilities.go — 塔能力 tick 管线：每帧重建 buff → 执行能力逻辑 → 收集经济产出。
//
// 核心职责：管理塔的 buff 生命周期和能力执行。
// 必须在 TickTowerCombat 之前执行，因为能力可能改变塔的攻击属性（范围/伤害/攻速）。
//
// Phase 编号系统（执行顺序）：
//
//	Phase 1    — 清除临时战力 + 重置塔属性（上帧光环 buff 清零）
//	Phase 1.05 — 链网络重建（聚能战灵专属，塔间战力传递）
//	Phase 1.1  — tick 塔 BuffList（递减时间，移除过期 buff）
//	Phase 1.5  — 重置敌人每帧临时状态（沉默等，由后续能力重新设置）
//	Phase 2    — 执行所有 Ticker 能力（光环/区域/经济等）
//
// 编号用小数而非整数是因为这些 Phase 在后期插入时不想重编已有的 Phase 1/2，
// 类似 BASIC 行号的编排思路。
//
// 关系：
//   - 被 stage.go 通过 Func() 包装注册到 Orchestrator
//   - 依赖 tower.Ticker 接口（能力实现各自的 OnTick）
//   - 依赖 strength.RebuildChainNetwork（聚能战灵的链网络）
package pipeline

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

// chainTowersBuf 包级预分配缓冲区，避免每帧为链网络重建分配 slice。
// Phase 1.05 每帧 reset length=0 后复用底层数组。
var chainTowersBuf []strength.ChainTower

// TickTowerAbilities 执行所有塔的 Ticker 能力，返回本帧总金币收入。
//
// 完整流程（5 个 Phase）：
//
//	Phase 1:    清除临时战力 ClearTransient() + 重置塔属性 RecalcStats()
//	Phase 1.05: 链网络重建（仅 chainEnabled=true 时，聚能战灵激活链传递）
//	Phase 1.1:  tick 塔 BuffList（光环 buff 过期移除，0.15s 短时效）
//	Phase 1.5:  重置敌人每帧标记（Silenced/AbilitySilenced）
//	Phase 2:    遍历所有塔的所有能力，执行 Ticker.OnTick()
//
// chainEnabled 用 variadic 是为了向后兼容——大多数调用方不关心链网络。
func TickTowerAbilities(towers *tower.Pool, enemies *enemy.Pool, dt float64, chainEnabled ...bool) int {
	// --- Phase 1: 清除临时战力 + 重置塔属性 ---
	// 光环/链网络的临时加成每帧重算，必须先清零再由后续 Phase 重建。
	// ClearTransient 使用 clear() 复用 map 内存（不 make 新 map），零分配。
	// StatsDirty=false 复位：后续 Phase 1.05 的链网络或 Phase 2 的光环能力
	// 若设置了 SetTemp，会标记 dirty=true，触发按需 RecalcStats。
	towers.Each(func(t *tower.Tower) {
		if t.Strength != nil {
			t.Strength.ClearTransient()
		}
		t.StatsDirty = false
		resetTowerStats(t)
	})

	// --- Phase 1.05: 链网络重建（仅聚能战灵 forge 启用时生效）---
	// 链网络让相邻塔之间传递战力加成（类似 Factorio 的电网）。
	// RebuildChainNetwork 会对符合条件的塔 pair 调用 SetTemp，
	// 被影响的塔标记 StatsDirty=true，在 Phase 2 结束后由 RecalcStats 消费。
	if len(chainEnabled) > 0 && chainEnabled[0] {
		chainTowersBuf = chainTowersBuf[:0] // 复用底层数组
		idx := 0
		towers.Each(func(t *tower.Tower) {
			chainTowersBuf = append(chainTowersBuf, strength.ChainTower{
				Index: idx, X: t.X, Y: t.Y, Strength: t.Strength,
			})
			idx++
		})
		strength.RebuildChainNetwork(chainTowersBuf)
		towers.Each(func(t *tower.Tower) {
			if t.Strength != nil && len(t.Strength.Temp) > 0 {
				t.StatsDirty = true
			}
		})
	}

	// --- Phase 1.1: tick 塔 BuffList（递减时间，移除过期 buff）---
	// 光环类 buff 时效极短（0.15s），若光环能力本帧未续期则自动过期。
	// 这实现了"光环范围离开即失效"的效果，无需显式移除。
	towers.Each(func(t *tower.Tower) {
		if t.Buffs != nil {
			t.Buffs.Tick(dt)
		}
	})

	// --- Phase 1.5: 重置敌人每帧临时标记 ---
	// Silenced / AbilitySilenced 是每帧由区域能力重新设置的标记，
	// 不走 BuffList（因为是布尔开关而非有时效的 buff）。
	// 每帧先清零，Phase 2 中的沉默区域能力会对范围内敌人重新标记。
	// 虚弱（Weaken）已迁入 BuffList，用 0.2s 短 buff 实现，无需手动重置。
	enemies.EachActive(func(e *enemy.Enemy) {
		e.Silenced = false
		e.AbilitySilenced = false
	})

	// --- Phase 2: 执行所有 Ticker 能力 ---
	// 遍历每座塔的能力列表，对实现了 tower.Ticker 接口的能力调用 OnTick。
	// 非 Ticker 能力（如被动加属性的能力）在此跳过——它们在 RecalcStats 中生效。
	// Ticker 能力包括：光环（给周围塔/敌人施加 buff）、区域（沉默/虚弱）、经济（产金）。
	goldEarned := 0
	ctx := &tower.TickContext{
		Enemies: enemies,
		Towers:  towers,
		DT:      dt,
	}
	towers.Each(func(t *tower.Tower) {
		if t.Selling {
			return // 出售中的塔跳过能力执行
		}
		for _, aName := range t.Abilities {
			ab, ok := tower.Lookup(aName)
			if !ok {
				continue
			}
			ticker, ok := ab.(tower.Ticker)
			if !ok {
				continue // 非 Ticker 能力——被动属性，不需要每帧 tick
			}
			result := ticker.OnTick(t, ctx)
			if result != nil {
				goldEarned += result.GoldEarned
			}
		}
	})

	return goldEarned
}

// resetTowerStats 根据战力系统重算塔的 Damage/Range/AttackSpeed/CritBonus/DamageAmp。
// 公式: attr = Base + Potential * (Strength / 100)
// 光环 buff 通过 BuffList.Tick() 自然过期（0.15s），RecalcStats 从 BuffList 聚合当前有效 buff。
// 此函数是单行包装，保留是为了语义清晰 + 未来可能添加 Phase 1 专属逻辑。
func resetTowerStats(t *tower.Tower) {
	t.RecalcStats()
}
