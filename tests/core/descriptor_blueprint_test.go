// descriptor_blueprint_test.go — TowerBlueprint 校验逻辑测试。
//
// 验证 ValidateBlueprint 对各种非法蓝图的检测能力。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// hasValidationError 检查错误列表中是否包含指定 field 的错误。
func hasValidationError(errs []descriptor.ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func TestValidateBlueprint_Valid(t *testing.T) {
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "valid_tower",
		Name:        "Valid Tower",
		AttackStyle: "scatter",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "B", "range": "D"},
		Specialty:   "damage",
		Abilities:   []string{"splash", "crit"},
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateBlueprint_OverBudget(t *testing.T) {
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "over",
		Name:        "Over Budget",
		AttackStyle: "wideBeam",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "S", "range": "S"},
		Specialty:   "range",
		Abilities:   []string{"stunChance", "burn", "splash"},
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if !hasValidationError(errs, "budget") {
		t.Error("expected budget error")
	}
}

func TestValidateBlueprint_TooManyAbilities(t *testing.T) {
	rules := testBudgetRules()
	// 用低费用能力，确保不超预算但数量超标
	costs := map[string]int{
		"a1": 1, "a2": 1, "a3": 1, "a4": 1,
		"a5": 1, "a6": 1, "a7": 1,
	}
	bp := descriptor.TowerBlueprint{
		ID:          "many_ab",
		Name:        "Too Many",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "",
		Abilities:   []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7"}, // 7 > maxSlots(6)
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if !hasValidationError(errs, "abilities") {
		t.Error("expected abilities count error")
	}
}

func TestValidateBlueprint_DuplicateAbilities(t *testing.T) {
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "dup_ab",
		Name:        "Dup Abilities",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "",
		Abilities:   []string{"splash", "splash"},
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if !hasValidationError(errs, "abilities") {
		t.Error("expected duplicate abilities error")
	}
}

func TestValidateBlueprint_InvalidTier(t *testing.T) {
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "bad_tier",
		Name:        "Bad Tier",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "X", "atkSpeed": "B", "range": "D"},
		Specialty:   "",
		Abilities:   nil,
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if !hasValidationError(errs, "tiers") {
		t.Error("expected tiers error for invalid tier 'X'")
	}
}

func TestValidateBlueprint_InvalidSpecialty(t *testing.T) {
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "bad_spec",
		Name:        "Bad Specialty",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "power",
		Abilities:   nil,
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if !hasValidationError(errs, "specialty") {
		t.Error("expected specialty error for 'power'")
	}
}

func TestValidateBlueprint_MultipleErrors(t *testing.T) {
	// 同时触发多个校验错误
	rules := testBudgetRules()
	costs := testAbilityCosts()
	bp := descriptor.TowerBlueprint{
		ID:          "multi_err",
		Name:        "Multi Error",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "X", "atkSpeed": "B", "range": "D"},
		Specialty:   "power",
		Abilities:   []string{"splash", "splash"},
	}

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(errs), errs)
	}
	if !hasValidationError(errs, "tiers") {
		t.Error("expected tiers error")
	}
	if !hasValidationError(errs, "specialty") {
		t.Error("expected specialty error")
	}
	if !hasValidationError(errs, "abilities") {
		t.Error("expected abilities error")
	}
}

func TestValidateBlueprint_ExactlyAtCap(t *testing.T) {
	// 恰好在预算上限时应该通过
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

	errs := descriptor.ValidateBlueprint(bp, rules, costs)

	if hasValidationError(errs, "budget") {
		t.Error("exactly at cap should not trigger budget error")
	}
}

func TestValidateBlueprint_EmptyAbilitiesIsValid(t *testing.T) {
	rules := testBudgetRules()
	bp := descriptor.TowerBlueprint{
		ID:          "no_ab",
		Name:        "No Abilities",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "D", "atkSpeed": "D", "range": "D"},
		Specialty:   "",
		Abilities:   nil,
	}

	errs := descriptor.ValidateBlueprint(bp, rules, nil)

	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}
