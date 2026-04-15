// budget.go — 预算计算系统。
//
// 计算塔蓝图的预算消耗（攻击方式/属性档位/专精/能力费用），
// 以及根据预算使用量推算建造费用。
// 由 blueprint.go 的 ValidateBlueprint 调用进行预算合规检查。
package descriptor

// BudgetRules 预算规则配置。
type BudgetRules struct {
	BaseCap          int            `json:"baseCap"`          // 预算上限
	MaxSlots         int            `json:"maxSlots"`         // 最大能力槽数
	AttackStyleCosts map[string]int `json:"attackStyleCosts"` // 各攻击方式的预算费用
	TierCosts        map[string]int `json:"tierCosts"`        // 各档位的预算费用
	SpecialtyCost    int            `json:"specialtyCost"`    // 专精的预算费用
	BaseBuildCost    int            `json:"baseBuildCost"`    // 基础建造金币花费
	CostPerPoint     float64        `json:"costPerPoint"`     // 每预算点的额外金币花费
}

// BudgetResult 预算计算结果。
type BudgetResult struct {
	Cap       int            // 预算上限
	Used      int            // 已使用预算
	Breakdown map[string]int // 各项预算明细
}

// CalcBudget 计算蓝图的预算使用量。
func CalcBudget(bp TowerBlueprint, rules BudgetRules, abilityCosts map[string]int) BudgetResult {
	breakdown := map[string]int{}

	// 攻击方式费用
	breakdown["attackStyle"] = rules.AttackStyleCosts[bp.AttackStyle]

	// 档位费用
	tierTotal := 0
	for _, tier := range bp.Tiers {
		tierTotal += rules.TierCosts[tier]
	}
	breakdown["tiers"] = tierTotal

	// 专精费用
	if bp.Specialty != "" {
		breakdown["specialty"] = rules.SpecialtyCost
	} else {
		breakdown["specialty"] = 0
	}

	// 能力费用
	abilityTotal := 0
	for _, ab := range bp.Abilities {
		abilityTotal += abilityCosts[ab]
	}
	breakdown["abilities"] = abilityTotal

	used := 0
	for _, v := range breakdown {
		used += v
	}

	return BudgetResult{
		Cap:       rules.BaseCap,
		Used:      used,
		Breakdown: breakdown,
	}
}

// CalcBuildCost 根据已使用预算计算建造金币花费。
func CalcBuildCost(usedBudget int, rules BudgetRules) int {
	return rules.BaseBuildCost + int(float64(usedBudget)*rules.CostPerPoint)
}
