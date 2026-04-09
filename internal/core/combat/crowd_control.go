// crowd_control.go — 控制效果（CC）韧性系统。
// 统一处理眩晕、减速、定身等控制效果的施加，支持韧性减免和免疫检查。
package combat

import (
	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
)

// MinSpeedRatio 全局减速下限：速度不低于初始速度的 20%。
const MinSpeedRatio = 0.2

// ApplyStun 对敌人施加眩晕效果。
// 检查免疫状态，应用韧性减免后设置眩晕计时器。
// 返回 true 表示成功施加。
func ApplyStun(e *enemy.Enemy, duration float64, source string) bool {
	// 控制免疫检查
	if e.IsControlImmune || e.IsStunImmune {
		e.SetFloatText("免疫", 220, 60, 60)
		return false
	}

	// 韧性减免：实际持续时间 = 原始时间 * (1 - 韧性)
	actualDuration := duration * (1 - e.Tenacity)
	if actualDuration <= 0 {
		return false
	}

	// 取较长的眩晕时间（不叠加，只刷新）
	if actualDuration > e.StunTimer {
		e.StunTimer = actualDuration
	}
	tel.T.Record("cc", "stun")
	return true
}

// ApplySlow 对敌人施加减速效果。
// 检查免疫状态，应用韧性减免，速度不低于 BaseSpeed * MinSpeedRatio。
// 返回 true 表示成功施加。
func ApplySlow(e *enemy.Enemy, factor, duration float64, source string) bool {
	// 控制免疫检查
	if e.IsControlImmune || e.IsSlowImmune {
		if e.IsControlImmune {
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
	if factor < MinSpeedRatio {
		factor = MinSpeedRatio
	}

	// 取更强的减速效果（更低的 factor = 更慢）
	if actualDuration > e.SlowTimer || factor < e.SlowFactor {
		e.SlowTimer = actualDuration
		e.SlowFactor = factor
		e.Speed = e.BaseSpeed * factor
	}
	tel.T.Record("cc", "slow")
	return true
}

// ApplyControlImmunity 给予敌人一段时间的控制免疫。
// duration > 0 时为限时免疫（由 TickStatusEffects 倒计时清除），
// duration <= 0 时为永久免疫。
func ApplyControlImmunity(e *enemy.Enemy, duration float64) {
	e.ControlImmuneTimer = duration
	e.IsControlImmune = true
	e.IsStunImmune = true
	e.IsSlowImmune = true

	// 清除当前正在生效的控制效果
	e.StunTimer = 0
	e.SlowTimer = 0
	e.SlowFactor = 1
	e.Speed = e.BaseSpeed
	e.RootTimer = 0
}
