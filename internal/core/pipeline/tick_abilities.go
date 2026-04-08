// tick_abilities.go — 能力 tick 管线。
// 负责每帧重置塔临时 buff、执行所有 Ticker 能力（光环/区域/经济）、
// 并收集经济产出。
package pipeline

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

// TickTowerAbilities 执行所有塔的 Ticker 能力，返回本帧总金币收入。
//
// 调用流程：
//  1. 重置所有塔属性为等级基准值（清除上帧光环 buff）
//  2. 遍历每座塔的每个能力，对实现 Ticker 接口的能力调用 OnTick
//  3. 累计金币收入并返回
func TickTowerAbilities(towers *tower.Pool, enemies *enemy.Pool, dt float64, chainEnabled ...bool) int {
	// --- Phase 1: 清除临时战力 + 重置塔属性 ---
	// 光环/链网络的临时加成每帧重算，先清除再重建。
	towers.Each(func(t *tower.Tower) {
		if t.Strength != nil {
			t.Strength.ClearTransient()
		}
		resetTowerStats(t)
	})

	// --- Phase 1.05: 链网络（仅聚能战灵启用时生效）---
	if len(chainEnabled) > 0 && chainEnabled[0] {
		var chainTowers []strength.ChainTower
		idx := 0
		towers.Each(func(t *tower.Tower) {
			chainTowers = append(chainTowers, strength.ChainTower{
				Index: idx, X: t.X, Y: t.Y, Strength: t.Strength,
			})
			idx++
		})
		strength.RebuildChainNetwork(chainTowers)
	}

	// --- Phase 1.1: tick 塔 buff（递减时间，移除过期 buff）---
	towers.Each(func(t *tower.Tower) {
		t.TickBuffs(dt)
	})

	// --- Phase 1.5: 重置敌人每帧临时状态（沉默/区域虚弱等，由区域能力重新设置） ---
	enemies.Each(func(e *enemy.Enemy) {
		e.Silenced = false
		e.AbilitySilenced = false
		// zone 型虚弱每帧由 weakenZone 重新设置；
		// OnHit 型虚弱(DamageAmplifyTimer>0)不在此清零，由 TickStatusEffects 倒计时管理。
		if e.DamageAmplifyTimer <= 0 {
			e.DamageAmplify = 0
		}
	})

	// --- Phase 2: 执行所有 Ticker 能力 ---
	goldEarned := 0
	ctx := &tower.TickContext{
		Enemies: enemies,
		Towers:  towers,
		DT:      dt,
	}
	towers.Each(func(t *tower.Tower) {
		if t.Selling {
			return // selling towers skip abilities
		}
		for _, aName := range t.Abilities {
			ab, ok := tower.Registry[aName]
			if !ok {
				continue
			}
			ticker, ok := ab.(tower.Ticker)
			if !ok {
				continue
			}
			result := ticker.OnTick(t, ctx)
			if result != nil {
				goldEarned += result.GoldEarned
			}
		}
	})

	return goldEarned
}

// resetTowerStats 根据战力系统重算塔的 Damage/Range/AttackSpeed。
// 公式: attr = base + potential * (strength / 100)
func resetTowerStats(t *tower.Tower) {
	t.Mods = tower.AttrMods{} // 清零临时修饰
	t.RecalcStats()
}
