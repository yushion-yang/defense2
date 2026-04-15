// adapter.go — EffectResult → HitResult/TickResult 适配层。
//
// 描述符引擎的 Interpreter 产出 []EffectResult，
// 而现有战斗管线 (combat/apply_hit.go) 消费 *tower.HitResult，
// tick 系统 (pipeline/tick_abilities.go) 消费 *tower.TickResult。
// 本文件负责两者之间的桥接转换。
//
// 映射策略：
//   - Flat/HpPercent 伤害 → SeparateDamage（独立走 damageCap 管线）
//   - Ratio 伤害 → BonusDamage（已由 Effect.Apply 预乘 TowerDamage）
//   - 同类效果累加（多个 damage 求和）
//   - CC/DoT 取最后一个（last wins）
//   - Buff/SelfBuff/Gold/ModifyStat/Weaken/Silence 不进入 HitResult
package descriptor

import "defense2/internal/core/tower"

// AdaptToHitResult 将描述符引擎的效果列表转为战斗管线的 HitResult。
// 多个同类效果合并（如多个 damage 效果累加）。
// 返回 nil 如果没有任何 onHit 相关效果。
func AdaptToHitResult(results []EffectResult) *tower.HitResult {
	if len(results) == 0 {
		return nil
	}

	var hr tower.HitResult
	hasEffect := false

	for i := range results {
		r := &results[i]
		switch r.Type {
		case EffTypeDamage:
			hasEffect = true
			switch r.DamageMode {
			case DmgFlat, DmgHpPercent:
				hr.SeparateDamage += r.Damage
			case DmgRatio:
				hr.BonusDamage += r.Damage
			}
			if r.IsCrit {
				hr.IsCrit = true
			}

		case EffTypeSlow:
			hasEffect = true
			hr.Slow = &tower.SlowEffect{
				Factor:   r.SlowFactor,
				Duration: r.Duration,
			}

		case EffTypeStun:
			hasEffect = true
			hr.Stun = &tower.StunEffect{
				Duration: r.StunDur,
			}

		case EffTypeDot:
			eff := &tower.BleedEffect{
				DPS:      r.DotValue,
				Duration: r.DotDuration,
			}
			switch r.DotSubtype {
			case "bleed":
				hasEffect = true
				hr.Bleed = eff
			case "burn":
				hasEffect = true
				hr.Burn = eff
			// poison 等其他 DoT 子类型暂无 HitResult 对应字段，跳过
			}

		case EffTypeCrit:
			hasEffect = true
			hr.IsCrit = true

		// Purge 暂时不进入 HitResult（需要战斗管线后续支持），
		// 但标记 hasEffect 以确保结果不被丢弃。
		case EffTypePurge:
			hasEffect = true

		// 以下类型不进入 HitResult，由战斗管线或 tick 系统单独处理
		case EffTypeWeaken, EffTypeSilence:
		case EffTypeRoot:
		case EffTypeBuff, EffTypeSelfBuff:
		case EffTypeGold:
		case EffTypeModifyStat:
		}
	}

	if !hasEffect {
		return nil
	}
	return &hr
}

// AdaptToTickResult 将描述符引擎的效果列表转为 tick 管线的 TickResult。
// 当前仅提取 Gold 效果，其他 tick 效果（buff/zone）由管线直接应用。
// 返回 nil 如果没有任何 tick 相关效果。
func AdaptToTickResult(results []EffectResult) *tower.TickResult {
	if len(results) == 0 {
		return nil
	}

	var tr tower.TickResult
	hasEffect := false

	for i := range results {
		r := &results[i]
		if r.Type == EffTypeGold {
			hasEffect = true
			tr.GoldEarned += int(r.GoldAmount)
		}
	}

	if !hasEffect {
		return nil
	}
	return &tr
}
