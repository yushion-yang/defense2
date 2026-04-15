// descriptor_budget_test.go — 预算计算系统测试。
//
// 验证 CalcBudget / CalcBuildCost 的计算逻辑。
// 所有 abilityCosts 通过参数注入，不依赖全局配置。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// testBudgetRules 返回测试用的标准预算规则（与 budget-rules.json 一致）。
func testBudgetRules() descriptor.BudgetRules {
	return descriptor.BudgetRules{
		BaseCap:  50,
		MaxSlots: 6,
		AttackStyleCosts: map[string]int{
			"projectile": 0,
			"scatter":    12,
			"wideBeam":   14,
			"spin_aoe":   10,
			"radial":     11,
			"barrage":    13,
		},
		TierCosts: map[string]int{
			"S": 4,
			"B": 2,
			"D": 0,
		},
		SpecialtyCost: 2,
		BaseBuildCost: 40,
		CostPerPoint:  0.5,
	}
}

// testAbilityCosts 返回测试用的能力费用表。
func testAbilityCosts() map[string]int {
	return map[string]int{
		"splash":    8,
		"crit":      7,
		"bounce":    6,
		"slowPower": 5,
		"burn":      9,
		"poison":    4,
		"stunChance": 10,
	}
}

func TestCalcBudget_Basic(t *testing.T) {
	// scatter(12) + S+B+D(4+2+0=6) + specialty(2) + splash+crit(8+7=15) = 35
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "test_basic",
		Name:        "Test Tower",
		AttackStyle: "scatter",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "B", "range": "D"},
		Specialty:   "damage",
		Abilities:   []string{"splash", "crit"},
	}
	costs := testAbilityCosts()

	result := descriptor.CalcBudget(bp, rules, costs)

	if result.Cap != 50 {
		t.Errorf("cap: got %d, want 50", result.Cap)
	}
	if result.Used != 35 {
		t.Errorf("used: got %d, want 35", result.Used)
	}
	// 检查分项
	if result.Breakdown["attackStyle"] != 12 {
		t.Errorf("breakdown[attackStyle]: got %d, want 12", result.Breakdown["attackStyle"])
	}
	if result.Breakdown["tiers"] != 6 {
		t.Errorf("breakdown[tiers]: got %d, want 6", result.Breakdown["tiers"])
	}
	if result.Breakdown["specialty"] != 2 {
		t.Errorf("breakdown[specialty]: got %d, want 2", result.Breakdown["specialty"])
	}
	if result.Breakdown["abilities"] != 15 {
		t.Errorf("breakdown[abilities]: got %d, want 15", result.Breakdown["abilities"])
	}
}

func TestCalcBudget_OverBudget(t *testing.T) {
	// wideBeam(14) + SSS(12) + specialty(2) + stunChance+burn+splash(10+9+8=27) = 55 > 50
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "over",
		Name:        "Over Budget",
		AttackStyle: "wideBeam",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "S", "range": "S"},
		Specialty:   "range",
		Abilities:   []string{"stunChance", "burn", "splash"},
	}
	costs := testAbilityCosts()

	result := descriptor.CalcBudget(bp, rules, costs)

	if result.Used != 55 {
		t.Errorf("used: got %d, want 55", result.Used)
	}
	if result.Used <= result.Cap {
		t.Error("expected over budget")
	}
}

func TestCalcBudget_Zero(t *testing.T) {
	// projectile(0) + DDD(0) + no specialty + no abilities = 0
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "zero",
		Name:        "Zero Cost",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "",
		Abilities:   nil,
	}

	result := descriptor.CalcBudget(bp, rules, nil)

	if result.Used != 0 {
		t.Errorf("used: got %d, want 0", result.Used)
	}
}

func TestCalcBudget_ExactlyAtCap(t *testing.T) {
	// wideBeam(14) + SSS(4+4+4=12) + specialty(2) + splash+crit+abilityX(8+7+7=22) = 50
	rules := testBudgetRules()
	costs := map[string]int{"splash": 8, "crit": 7, "abilityX": 7}
	bp := descriptor.TowerBlueprint{
		ID:          "exact",
		Name:        "Exact Budget",
		AttackStyle: "wideBeam",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "S", "range": "S"},
		Specialty:   "damage",
		Abilities:   []string{"splash", "crit", "abilityX"},
	}

	result := descriptor.CalcBudget(bp, rules, costs)

	if result.Used != 50 {
		t.Errorf("used: got %d, want 50", result.Used)
	}
	if result.Used != result.Cap {
		t.Error("expected exactly at cap")
	}
}

func TestCalcBuildCost(t *testing.T) {
	rules := testBudgetRules()

	tests := []struct {
		name       string
		usedBudget int
		want       int
	}{
		{"zero budget", 0, 40},                   // 40 + 0*0.5 = 40
		{"used 6", 6, 43},                        // 40 + int(6*0.5) = 40+3 = 43
		{"used 35", 35, 57},                      // 40 + int(35*0.5) = 40+17 = 57
		{"used 50 max", 50, 65},                  // 40 + int(50*0.5) = 40+25 = 65
		{"odd used 7", 7, 43},                    // 40 + int(7*0.5) = 40+3 = 43
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := descriptor.CalcBuildCost(tt.usedBudget, rules)
			if got != tt.want {
				t.Errorf("CalcBuildCost(%d): got %d, want %d", tt.usedBudget, got, tt.want)
			}
		})
	}
}

func TestCalcBudget_NoSpecialty(t *testing.T) {
	// 无专精时 specialty 费用为 0
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "no_spec",
		Name:        "No Specialty",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "B", "atkSpeed": "B", "range": "B"},
		Specialty:   "",
		Abilities:   nil,
	}

	result := descriptor.CalcBudget(bp, rules, nil)

	// projectile(0) + BBB(2+2+2=6) + no specialty(0) + no abilities(0) = 6
	if result.Used != 6 {
		t.Errorf("used: got %d, want 6", result.Used)
	}
	if result.Breakdown["specialty"] != 0 {
		t.Errorf("breakdown[specialty]: got %d, want 0", result.Breakdown["specialty"])
	}
}

func TestCalcBudget_UnknownAbilityCostIsZero(t *testing.T) {
	// 能力费用表中不存在的能力，费用按 0 算（不报错，由 Validate 捕获）
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "unknown_ab",
		Name:        "Unknown Ability",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "",
		Abilities:   []string{"nonexistent"},
	}

	result := descriptor.CalcBudget(bp, rules, testAbilityCosts())

	if result.Used != 0 {
		t.Errorf("used: got %d, want 0", result.Used)
	}
}
