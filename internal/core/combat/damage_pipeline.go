// damage_pipeline.go — 8步伤害管线（战斗系统的最终伤害计算层）。
// 所有伤害（弹射物、光束、技能、DOT）最终都应通过 ApplyDamage 处理，
// 确保免疫/减免/阈值等机制统一生效。
//
// 本文件是伤害数值的唯一出口：apply_hit.go 和其他伤害来源都调用 ApplyDamage()，
// 不允许直接修改 enemy.HP。这保证了所有防御机制（免疫/减免/上限）不会被绕过。
//
// 8步管线:
//
//  1. 免疫检查（不可选中/无敌/伤害免疫，pure 穿透所有）
//  2. Boss百分比HP上限（%HP伤害对Boss额外限制，防止百分比伤害秒杀 Boss）
//  3. 攻击者增伤buff（damageUp 乘数，true/pure 跳过）
//  4. 目标减伤buff（damageDown 乘数，true/pure 跳过）
//     4.1 减伤比例（damageReduce，来自 BuffList 的 buff 模板）
//     4.25 虚弱增伤（weaken/weakenZone，上限50%，防止叠加过高）
//     4.5 伤害上限（damageCap/damageCapPercent，沉默时失效——这是对抗 damageCap 的方式之一）
//  5. HP扣减（最低保底 1 点，确保任何命中都有效果）
//  6. 阈值触发（HP比例触发器，用于 Boss 阶段转换等机制）
//  7. 死亡检查
//
// 伤害类型体系（定义在 damage_type.go）：
//   - physical/magic：受所有增减伤影响（步骤 3/4/4.1）
//   - true：忽略增减伤，但不穿透免疫（步骤 1 仍生效）
//   - pure：穿透一切，包括无敌状态（仅步骤 5-7 生效）
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
	"defense2/internal/i18n"
)

const damageCapFlashDuration = 0.3 // 伤害上限触发视觉时长(秒)
const weakenAmplifyMax = 0.5       // 虚弱增伤上限（防止多塔叠加导致倍率失控）

// DamageInput 伤害管线输入参数。
// 调用者负责填充攻击者/目标信息，管线本身不查询外部状态。
// AttackerDamageUp/TargetDamageDown 由外部 buff 系统预解算后传入，
// 保持管线纯函数特性（相同输入 → 相同输出）。
type DamageInput struct {
	Target      *enemy.Enemy // 受击目标
	RawDamage   float64      // 原始伤害值
	DamageType  string       // 伤害类型（physical/magic/true/pure）
	IsPercentHP bool         // 是否为百分比最大生命伤害
	PercentCap  float64      // Boss百分比伤害上限（默认0.05=5%maxHP）
	SourceLabel string       // 来源描述（调试用）

	// 攻击者增伤倍率（由外部 buff 系统解算后传入，0 或 1 表示无加成）
	AttackerDamageUp float64
	// 目标减伤倍率（由外部 buff 系统解算后传入，0 或 1 表示无减免）
	TargetDamageDown float64
}

// DamageResult 伤害管线输出结果。
// 保留每个步骤的中间值（AfterAttackerMod / AfterTargetMod / AfterDamageCap），
// 便于调试和遥测分析各环节的伤害衰减情况。
type DamageResult struct {
	Blocked       bool   // 伤害是否被完全阻挡
	BlockedReason string // 阻挡原因（"invincible"/"damageImmune"/"untargetable"）

	RawDamage        float64 // 步骤1: 原始伤害
	AfterAttackerMod float64 // 步骤3: 攻击者增伤后
	AfterTargetMod   float64 // 步骤4: 目标减伤后
	AfterDamageCap   float64 // 步骤4.5: 伤害上限后
	HPDamage         float64 // 步骤5: 实际扣减HP量
	FinalDamage      float64 // 总伤害
	Killed           bool    // 步骤7: 是否死亡

	Thresholds []enemy.Threshold // 步骤6: 本次触发的阈值列表
}

// ApplyDamage 执行8步伤害管线。
// 这是所有伤害的统一入口，确保免疫/减免/阈值等机制一致生效。
func ApplyDamage(input DamageInput) DamageResult {
	e := input.Target
	dmgType := input.DamageType
	if dmgType == "" {
		dmgType = DmgPhysical
	}

	// 遥测：记录伤害类型
	tel.T.Record("damage_type", dmgType)

	result := DamageResult{
		RawDamage: input.RawDamage,
	}
	damage := input.RawDamage

	// ── 步骤1: 免疫检查 ──
	tel.T.Record("pipeline", "immunity_check")
	if !IgnoresInvincible(dmgType) {
		if e.IsUntargetable {
			result.Blocked = true
			result.BlockedReason = "untargetable"
			e.SetFloatText(i18n.T("combat.damage_immune"), 160, 80, 255)
			tel.T.Record("pipeline", "immunity_block_untargetable")
			return result
		}
		if e.IsInvincible {
			result.Blocked = true
			result.BlockedReason = "invincible"
			e.SetFloatText(i18n.T("combat.damage_immune"), 160, 80, 255)
			tel.T.Record("pipeline", "immunity_block_invincible")
			return result
		}
		if e.IsDamageImmune {
			result.Blocked = true
			result.BlockedReason = "damageImmune"
			e.SetFloatText(i18n.T("combat.damage_immune"), 160, 80, 255)
			tel.T.Record("pipeline", "immunity_block_immune")
			return result
		}
	}

	// ── 步骤2: Boss百分比HP上限 ──
	tel.T.Record("pipeline", "boss_hp_cap")
	if input.IsPercentHP && e.Boss {
		cap := input.PercentCap
		if cap <= 0 {
			cap = config.GlobalSpawnerConfig().Boss.PercentHpCap
		}
		maxDmg := e.MaxHP * cap
		if maxDmg < 1 {
			maxDmg = 1
		}
		if damage > maxDmg {
			damage = maxDmg
		}
	}

	// ── 步骤3: 攻击者增伤buff ──
	tel.T.Record("pipeline", "attacker_buff")
	if !IgnoresReduction(dmgType) && input.AttackerDamageUp > 0 {
		damage *= input.AttackerDamageUp
	}
	result.AfterAttackerMod = damage

	// ── 步骤4: 目标减伤buff ──
	tel.T.Record("pipeline", "target_debuff")
	if !IgnoresReduction(dmgType) && input.TargetDamageDown > 0 {
		damage *= input.TargetDamageDown
	}
	result.AfterTargetMod = damage

	// ── 步骤4.1: 减伤比例（damageReduce via BuffList） ──
	// 来自 buff 模板系统（如 berserk 的减伤效果），与步骤 4 的外部 debuff 独立乘算。
	// 沉默时失效（与 evasion/armorPlating 保持一致，abilities.json silenceable=true）。
	if dr := e.GetDamageReduce(); !IgnoresReduction(dmgType) && !e.AbilitySilenced && dr > 0 {
		damage *= (1 - dr)
	}

	// ── 步骤4.25: 虚弱增伤（weaken/weakenZone） ──
	// weaken 由塔能力在 ApplyHit 中提前施加（OnHit 阶段），所以本次命中就能受益。
	// 硬上限 50%：防止多塔叠加 weaken 导致伤害倍率失控。
	tel.T.Record("pipeline", "damage_amplify")
	amp := e.GetWeakenAmplify()
	if amp > 0 {
		if amp > weakenAmplifyMax {
			amp = weakenAmplifyMax
		}
		damage *= 1 + amp
	}

	// ── 步骤4.5: 伤害上限 ──
	// damageCap 是敌人的被动防御能力：单次伤害不超过固定值或最大 HP 百分比。
	// 沉默（Silenced）时失效——沉默是玩家对抗 damageCap 的主要手段之一。
	// barrage 连射是另一种对策：每颗子弹独立走管线，各自受 cap 限制，
	// 但总伤害 = cap × 弹数，高攻速高弹数可突破 cap。
	tel.T.Record("pipeline", "damage_cap")
	if !e.Silenced {
		capped := false
		if e.DamageCap > 0 && damage > e.DamageCap {
			damage = e.DamageCap
			capped = true
		}
		if e.DamageCapPercent > 0 {
			pctCap := e.MaxHP * e.DamageCapPercent
			if damage > pctCap {
				damage = pctCap
				capped = true
			}
		}
		if capped {
			e.DamageCapHit = damageCapFlashDuration
			e.SetFloatText(i18n.T("combat.damage_cap"), 255, 180, 40)
		}
	}
	result.AfterDamageCap = damage

	// 最低伤害保底 1 点：确保任何非零伤害命中都有效果，
	// 避免高减伤敌人完全免疫普通塔的攻击（玩家应感受到"打了但很少"而非"无效"）。
	if damage < 1 && input.RawDamage > 0 {
		damage = 1
	}

	// ── 步骤5: HP扣减 ──
	tel.T.Record("pipeline", "hp_deduct")
	result.HPDamage = damage
	e.HP -= damage
	if e.HP < 0 {
		e.HP = 0
	}
	result.FinalDamage = result.HPDamage

	// ── 步骤6: 阈值触发 ──
	tel.T.Record("pipeline", "threshold")
	result.Thresholds = enemy.CheckThresholds(e)

	// ── 步骤7: 死亡检查 ──
	tel.T.Record("pipeline", "death_check")
	result.Killed = e.HP <= 0
	if result.Killed {
		tel.T.Record("pipeline", "death")
	}

	// 最低伤害保底（即使经过减免，结果伤害不能为负）
	if result.FinalDamage < 0 {
		result.FinalDamage = 0
	}

	return result
}

// QuickDamage 快捷伤害函数（DoT、战灵技能等简单伤害源使用）。
// 省略了 AttackerDamageUp/TargetDamageDown 等外部 buff 参数，
// 但仍走完整 8 步管线确保免疫/减免/上限等机制生效。
func QuickDamage(target *enemy.Enemy, rawDamage float64, damageType string) (finalDamage float64, killed bool) {
	r := ApplyDamage(DamageInput{
		Target:     target,
		RawDamage:  rawDamage,
		DamageType: damageType,
	})
	return r.FinalDamage, r.Killed
}
