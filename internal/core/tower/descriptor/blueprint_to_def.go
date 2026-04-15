// blueprint_to_def.go — 蓝图→塔定义转换。
//
// 将 TowerBlueprint 转换为 tower.TowerDef，使自定义塔可通过 Pool.Place() 放置。
// 转换逻辑参考 stage.go 的 loadClassicTowerDefs()：
//   - tiers 字典查 tier-presets 获取 Base/Potential
//   - specialty 为对应属性的 Potential 追加 basePotential 加成
//   - 建造费用由预算系统计算
//   - FixedTiers=true 确保 Place 跳过随机 roll
package descriptor

import (
	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// defaultProjectileSpeed 弹道默认速度（与 towers.json 一致）。
const defaultProjectileSpeed = 300

// attackStyleSpriteMap 攻击方式→默认精灵标识映射。
// 与 pool.go spriteKeyForStyle 逻辑一致，但额外补充 radial→nova。
var attackStyleSpriteMap = map[string]string{
	"scatter":  "shotgun",
	"wideBeam": "prism",
	"spin_aoe": "cyclone",
	"radial":   "nova",
	"barrage":  "gatling",
}

// BlueprintToTowerDef 将蓝图转换为 TowerDef，使其可被 Pool.Place() 使用。
//
// 转换步骤：
//  1. 从 tierPresets 查表获取三属性的 Base/Potential
//  2. 专精属性追加 basePotential 加成
//  3. 计算预算并推算建造费用
//  4. 映射攻击方式和精灵标识
//  5. 设置 FixedTiers/PresetAbilities 等扩展字段
func BlueprintToTowerDef(bp *TowerBlueprint, tierPresets *config.TierPresets, budgetRules *BudgetRules, abilityCosts map[string]int) tower.TowerDef {
	// 从 tier-presets 查表获取 Base/Potential
	dmgTier := tierPresets.Damage.Tiers[bp.Tiers["damage"]]
	spdTier := tierPresets.AttackSpeed.Tiers[bp.Tiers["atkSpeed"]]
	rngTier := tierPresets.Range.Tiers[bp.Tiers["range"]]

	// 解析专精
	specialtyID := parseSpecialtyStr(bp.Specialty)

	// 计算预算和建造费用
	budget := CalcBudget(*bp, *budgetRules, abilityCosts)
	buildCost := CalcBuildCost(budget.Used, *budgetRules)

	// 如果蓝图自带 BuildCost 则优先使用（向后兼容）
	if bp.BuildCost > 0 {
		buildCost = bp.BuildCost
	}

	def := tower.TowerDef{
		// 身份
		Key:   bp.ID,
		Label: bp.Name,

		// 攻击方式
		AttackStyleID:   tower.AttackStyle(bp.AttackStyle),
		ProjectileSpeed: defaultProjectileSpeed,

		// 从 tier 查表设置 Base/Potential
		CfgBaseDamage:   dmgTier.Base,
		PotentialDamage: dmgTier.Potential,
		CfgBaseSpeed:    spdTier.Base,
		PotentialSpeed:  spdTier.Potential,
		CfgBaseRange:    rngTier.Base,
		PotentialRange:  rngTier.Potential,

		// 经济
		Cost: buildCost,

		// 强度升级
		StrengthCost:    bp.Strength.Cost,
		StrengthAmount:  bp.Strength.Amount,
		MaxStrengthBuys: bp.Strength.MaxPurchases,

		// 行为规则
		AbilityAcquireMode: "preset",

		// 扩展字段
		FixedTiers:        true,
		FixedSpecialty:    specialtyID,
		PresetAbilities:   bp.Abilities,
		SpriteKeyOverride: resolveSpriteKey(bp.SpriteKey, bp.AttackStyle),
	}

	// 专精加成：对应属性的 Potential 追加 basePotential
	switch specialtyID {
	case 0:
		def.PotentialDamage += tierPresets.Damage.BasePotential
	case 1:
		def.PotentialSpeed += tierPresets.AttackSpeed.BasePotential
	case 2:
		def.PotentialRange += tierPresets.Range.BasePotential
	}

	return def
}

// parseSpecialtyStr 将专精字符串转为索引。
// 0=damage, 1=atkSpeed, 2=range, -1=无专精。
func parseSpecialtyStr(s string) int {
	switch s {
	case "damage":
		return 0
	case "atkSpeed":
		return 1
	case "range":
		return 2
	default:
		return -1
	}
}

// resolveSpriteKey 确定精灵标识。
// 优先使用蓝图显式指定的 spriteKey，否则从攻击方式推导。
func resolveSpriteKey(spriteKey, attackStyle string) string {
	if spriteKey != "" {
		return spriteKey
	}
	if key, ok := attackStyleSpriteMap[attackStyle]; ok {
		return key
	}
	return "sentinel"
}
