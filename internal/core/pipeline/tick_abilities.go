// tick_abilities.go — 能力 tick 管线。
// 负责每帧重置塔临时 buff、执行所有 Ticker 能力（光环/区域/经济）、
// 并收集经济产出。
package pipeline

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// TickTowerAbilities 执行所有塔的 Ticker 能力，返回本帧总金币收入。
//
// 调用流程：
//  1. 重置所有塔属性为等级基准值（清除上帧光环 buff）
//  2. 遍历每座塔的每个能力，对实现 Ticker 接口的能力调用 OnTick
//  3. 累计金币收入并返回
func TickTowerAbilities(towers *tower.Pool, enemies *enemy.Pool, dt float64) int {
	// --- Phase 1: 重置塔属性为等级基准值 ---
	// 光环 buff 是每帧临时叠加的，必须先归位再重新计算。
	towers.Each(func(t *tower.Tower) {
		resetTowerStats(t)
	})

	// --- Phase 2: 执行所有 Ticker 能力 ---
	goldEarned := 0
	ctx := &tower.TickContext{
		Enemies: enemies,
		Towers:  towers,
		DT:      dt,
	}
	towers.Each(func(t *tower.Tower) {
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

// resetTowerStats 将塔的 Damage/Range/AttackSpeed 重置为基础值。
// 属性缩放由战力系统（StrengthConfig.CalcAttribute）驱动，此处仅恢复 base。
func resetTowerStats(t *tower.Tower) {
	t.Damage = t.BaseDamage
	t.Range = t.BaseRange
	t.AttackSpeed = t.BaseSpeed
}
