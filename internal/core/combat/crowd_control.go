// crowd_control.go — 控制效果（CC）韧性系统。
// 统一处理眩晕、减速等控制效果的施加，支持韧性减免和免疫检查。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
)

// MinSpeedRatio 返回全局减速下限（从 balance.json 实时读取，不再冻结于 init 时刻）。
func MinSpeedRatio() float64 { return config.GlobalBalance().Combat.MinSpeedRatio }

// ApplyStun 对敌人施加眩晕效果。
// 检查免疫状态，应用韧性减免后设置眩晕计时器。
// 返回 true 表示成功施加。
func ApplyStun(e *enemy.Enemy, duration float64, source string) bool {
	// 控制免疫检查（archetype flags + BuffList）
	if e.IsControlImmune || e.IsStunImmune || e.HasControlImmunity() {
		e.SetFloatText("免疫", 220, 60, 60)
		return false
	}

	// 韧性减免：实际持续时间 = 原始时间 * (1 - 韧性)
	actualDuration := duration * (1 - e.Tenacity)
	if actualDuration <= 0 {
		return false
	}

	// 通过 BuffList 施加眩晕（Override 模式，后来居上）
	e.Buffs.Add(buff.Buff{
		ID:        "stun",
		Category:  buff.CatCC,
		Source:    source,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
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
			e.SetFloatText("免疫", 220, 60, 60)
		} else {
			e.SetFloatText("免疫", 60, 180, 200)
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
	existing, hasExisting := e.Buffs.Get("slow")
	if hasExisting && factor >= existing.Value && actualDuration <= existing.Remaining {
		// 已有减速更强或相同，保持不变
		return true
	}
	e.Buffs.RemoveByID("slow") // 清除旧减速，施加新的
	e.Buffs.Add(buff.Buff{
		ID:        "slow",
		Category:  buff.CatCC,
		Source:    source,
		Value:     factor,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
	e.Speed = e.BaseSpeed * factor
	tel.T.Record("cc", "slow")
	return true
}

// ApplyRoot 对敌人施加定身效果（禁止移动）。
// 检查免疫状态，应用韧性减免后设置定身计时器。
// 返回 true 表示成功施加。
func ApplyRoot(e *enemy.Enemy, duration float64, source string) bool {
	// 控制免疫检查（archetype flags + BuffList）
	if e.IsControlImmune || e.IsRootImmune || e.HasControlImmunity() {
		e.SetFloatText("免疫", 220, 60, 60)
		return false
	}

	// 韧性减免：实际持续时间 = 原始时间 * (1 - 韧性)
	actualDuration := duration * (1 - e.Tenacity)
	if actualDuration <= 0 {
		return false
	}

	// 通过 BuffList 施加定身（Override 模式，后来居上）
	e.Buffs.Add(buff.Buff{
		ID:        "root",
		Category:  buff.CatCC,
		Source:    source,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
	tel.T.Record("cc", "root")
	return true
}

// ApplyControlImmunity 给予敌人一段时间的控制免疫。
// duration > 0 时为限时免疫（由 TickStatusEffects 倒计时清除），
// duration <= 0 时为永久免疫。
func ApplyControlImmunity(e *enemy.Enemy, duration float64) {
	// 清除 BuffList 中所有 CC
	e.Buffs.ClearByCategory(buff.CatCC)

	// 施加控制免疫 buff
	e.Buffs.Add(buff.Buff{
		ID:        "controlImmune",
		Category:  buff.CatDefense,
		Source:    "purge",
		Duration:  duration,
		Remaining: duration,
	})

	// Set archetype immunity flags (temporary, cleared when buff expires in TickStatusEffects)
	e.IsControlImmune = true
	e.IsStunImmune = true
	e.IsSlowImmune = true
	e.IsRootImmune = true

	// Restore speed (CC cleared by ClearByCategory above)
	e.Speed = e.BaseSpeed
}
