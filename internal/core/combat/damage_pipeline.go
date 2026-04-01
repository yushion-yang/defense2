// damage_pipeline.go — 7步伤害管线。
// 所有伤害（弹射物、光束、技能、DOT）最终都应通过 ProcessDamage 处理，
// 确保免疫/减免/阈值等机制统一生效。
//
// 7步管线:
//
//  1. 免疫检查（不可选中/无敌/伤害免疫，pure穿透所有）
//  2. Boss百分比HP上限（%HP伤害对Boss额外限制）
//  3. 攻击者增伤buff（damageUp乘数，true/pure跳过）
//  4. 目标减伤buff（damageDown乘数，true/pure跳过）
//     4.25 虚弱增伤（weaken/weakenZone，上限50%）
//     4.5 伤害上限（damageCap/damageCapPercent，沉默时失效）
//  5. HP扣减
//  6. 阈值触发（HP比例触发器）
//  7. 死亡检查
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	tel "defense2/internal/core/telemetry"
)

// MaxDamageAmplify 虚弱增伤上限（0.5 = 最多 +50% 受伤）。
const MaxDamageAmplify = 0.5

// DamageInput 伤害管线输入参数。
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

// ProcessDamage 执行7步伤害管线。
// 这是所有伤害的统一入口，确保免疫/减免/阈值等机制一致生效。
func ProcessDamage(input DamageInput) DamageResult {
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
			tel.T.Record("pipeline", "immunity_block_untargetable")
			return result
		}
		if e.IsInvincible {
			result.Blocked = true
			result.BlockedReason = "invincible"
			tel.T.Record("pipeline", "immunity_block_invincible")
			return result
		}
		if e.IsDamageImmune {
			result.Blocked = true
			result.BlockedReason = "damageImmune"
			tel.T.Record("pipeline", "immunity_block_immune")
			return result
		}
	}

	// ── 步骤2: Boss百分比HP上限 ──
	tel.T.Record("pipeline", "boss_hp_cap")
	if input.IsPercentHP && e.Boss {
		cap := input.PercentCap
		if cap <= 0 {
			cap = 0.05 // 默认5%maxHP
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

	// ── 步骤4.1: 减伤比例（damageReduce buff 模板） ──
	if !IgnoresReduction(dmgType) && e.DamageReduceRatio > 0 {
		damage *= (1 - e.DamageReduceRatio)
	}

	// ── 步骤4.25: 虚弱增伤（weaken/weakenZone） ──
	tel.T.Record("pipeline", "damage_amplify")
	if e.DamageAmplify > 0 {
		amp := e.DamageAmplify
		if amp > MaxDamageAmplify {
			amp = MaxDamageAmplify
		}
		damage *= 1 + amp
	}

	// ── 步骤4.5: 伤害上限 ──
	tel.T.Record("pipeline", "damage_cap")
	if !e.Silenced {
		if e.DamageCap > 0 && damage > e.DamageCap {
			damage = e.DamageCap
		}
		if e.DamageCapPercent > 0 {
			pctCap := e.MaxHP * e.DamageCapPercent
			if damage > pctCap {
				damage = pctCap
			}
		}
	}
	result.AfterDamageCap = damage

	// 最低伤害保底 1 点
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

// QuickDamage 快捷伤害函数（不需要完整管线时使用）。
// 直接走管线但使用默认参数，返回最终伤害和击杀标记。
func QuickDamage(target *enemy.Enemy, rawDamage float64, damageType string) (finalDamage float64, killed bool) {
	r := ProcessDamage(DamageInput{
		Target:     target,
		RawDamage:  rawDamage,
		DamageType: damageType,
	})
	return r.FinalDamage, r.Killed
}

// ApplyDamageUp 计算攻击者增伤系数的辅助函数。
// buffValue: 从 BuffList.GetEffective("damageUp") 获得的乘法堆叠值。
// 返回值直接赋给 DamageInput.AttackerDamageUp。
func ApplyDamageUp(buffValue float64) float64 {
	if buffValue <= 0 {
		return 0 // 无 buff
	}
	return buffValue
}

// ApplyDamageDown 计算目标减伤系数的辅助函数。
func ApplyDamageDown(buffValue float64) float64 {
	if buffValue <= 0 {
		return 0 // 无 buff
	}
	return math.Max(buffValue, 0.2) // 最低减到 20%
}
