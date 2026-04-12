// crowd_control.go — 控制效果（CC）韧性系统。
// 统一处理眩晕、减速等控制效果的施加，支持韧性减免和免疫检查。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/i18n"
)

// MinSpeedRatio 返回全局减速下限（从 balance.json 实时读取，不再冻结于 init 时刻）。
func MinSpeedRatio() float64 { return config.GlobalBalance().Combat.MinSpeedRatio }

// ApplyStun 对敌人施加眩晕效果。
// 检查免疫状态，应用韧性减免后设置眩晕计时器。
// 返回 true 表示成功施加。
func ApplyStun(e *enemy.Enemy, duration float64, source string) bool {
	// 控制免疫检查（archetype flags + BuffList）
	if e.IsControlImmune || e.IsStunImmune || e.HasControlImmunity() {
		e.SetFloatText(i18n.T("combat.immune"), 220, 60, 60)
		return false
	}

	// 韧性减免：实际持续时间 = 原始时间 * (1 - 韧性)
	actualDuration := duration * (1 - e.Tenacity)
	if actualDuration <= 0 {
		return false
	}

	// 通过 BuffList 施加眩晕（Override 模式，后来居上）
	wasStunned := e.Buffs.Has(buff.IDStun)
	e.Buffs.Add(buff.Buff{
		ID:        buff.IDStun,
		Category:  buff.CatCC,
		Source:    source,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
	if !wasStunned {
		e.SetFloatText(i18n.T("combat.stun"), 255, 220, 60)
	}
	tel.T.Record("cc", "stun")
	return true
}

// ApplySlow 对敌人施加减速效果。
// 检查免疫状态，应用韧性减免，速度不低于 BaseSpeed * MinSpeedRatio。
// 返回 true 表示成功施加。
func ApplySlow(e *enemy.Enemy, factor, duration float64, source string) bool {
	// 控制免疫检查（archetype flags + BuffList）
	if e.IsControlImmune || e.IsSlowImmune || e.HasControlImmunity() {
		if e.IsControlImmune || e.HasControlImmunity() {
			e.SetFloatText(i18n.T("combat.immune"), 220, 60, 60)
		} else {
			e.SetFloatText(i18n.T("combat.immune"), 60, 180, 200)
		}
		return false
	}

	// 韧性减免持续时间
	actualDuration := duration * (1 - e.Tenacity)
	if actualDuration <= 0 {
		return false
	}

	// 减速倍率下限限制（速度不低于初始速度的 MinSpeedRatio）
	bal := config.GlobalBalance()
	if factor < bal.Combat.MinSpeedRatio {
		factor = bal.Combat.MinSpeedRatio
	}

	// 手动比较减速强度（lower factor = stronger slow，Strongest 模式比较 Value 不适用）
	existing, hasExisting := e.Buffs.Get(buff.IDSlow)
	if hasExisting && factor >= existing.Value && actualDuration <= existing.Remaining {
		// 已有减速更强或相同，保持不变
		return true
	}
	e.Buffs.RemoveByID(buff.IDSlow) // 清除旧减速，施加新的
	e.Buffs.Add(buff.Buff{
		ID:        buff.IDSlow,
		Category:  buff.CatCC,
		Source:    source,
		Value:     factor,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
	e.Speed = e.BaseSpeed * factor
	if !hasExisting {
		e.SetFloatText(i18n.T("combat.slow"), 60, 180, 255)
	}
	tel.T.Record("cc", "slow")
	return true
}
