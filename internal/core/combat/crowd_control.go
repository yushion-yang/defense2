// crowd_control.go — 控制效果（CC）韧性系统。
// 统一处理眩晕、减速等控制效果的施加，支持韧性减免和免疫检查。
//
// 本文件被 apply_hit.go 的 applyHitEffectsUnified() 调用，
// 是能力系统产出 CC 效果的唯一执行路径。
//
// 韧性（Tenacity）机制：
//   - 敌人的 Tenacity 字段范围 0~1，表示 CC 持续时间减免比例
//   - 实际持续时间 = 原始持续时间 × (1 - Tenacity)
//   - Tenacity=0.5 的 Boss 受到 2 秒眩晕只会被晕 1 秒
//   - Tenacity=1.0 完全免疫所有 CC（等效于 IsControlImmune）
//
// 免疫层次（从高到低）：
//  1. IsControlImmune / HasControlImmunity() — 免疫所有 CC
//  2. IsStunImmune / IsSlowImmune — 免疫特定 CC 类型
//  3. Tenacity — 按比例缩减持续时间
//
// CC 效果通过 BuffList 管理，眩晕用 Override 模式（后来居上），
// 减速用手动比较（lower factor = stronger slow，不适合 Strongest 模式的 Value 排序）。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/i18n"
)

// MinSpeedRatio 返回全局减速下限（从 balance.json 实时读取，不再冻结于 init 时刻）。
// 这个下限防止敌人被减速到完全静止（体验上"极慢但仍在动"优于"冻住不动"）。
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

	// 手动比较减速强度（lower factor = stronger slow）。
	// 不能使用 BuffList 的 Strongest 堆叠模式，因为 Strongest 按 Value 大小比较，
	// 但减速的 factor 越小=越强（0.3 比 0.7 更强），语义方向相反。
	// 所以这里手动比较：新减速比旧的更强（factor更低）或持续更久才替换。
	existing, hasExisting := e.Buffs.Get(buff.IDSlow)
	if hasExisting && factor >= existing.Value && actualDuration <= existing.Remaining {
		return true
	}
	e.Buffs.RemoveByID(buff.IDSlow)
	e.Buffs.Add(buff.Buff{
		ID:        buff.IDSlow,
		Category:  buff.CatCC,
		Source:    source,
		Value:     factor,
		Duration:  actualDuration,
		Remaining: actualDuration,
	})
	// 立即更新实际速度：Speed = BaseSpeed × factor。
	// 这里直接设置 Speed 是因为减速通过 BuffList Tick 自动恢复——
	// buff 到期时 movement.go 会重置 Speed = BaseSpeed。
	e.Speed = e.BaseSpeed * factor
	if !hasExisting {
		e.SetFloatText(i18n.T("combat.slow"), 60, 180, 255)
	}
	tel.T.Record("cc", "slow")
	return true
}
