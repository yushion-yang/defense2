// descriptor_blueprint_to_def_test.go — Blueprint → TowerDef 转换测试。
//
// 验证 BlueprintToTowerDef 正确将自定义蓝图转换为可被 Pool.Place() 使用的 TowerDef。
// 所有配置（tierPresets/budgetRules/abilityCosts）通过参数注入，不依赖全局状态。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// floatEq 比较两个浮点数是否近似相等（容差 1e-9）。
func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// testTierPresets 返回测试用的最小 tier 预设表。
func testTierPresets() *config.TierPresets {
	return &config.TierPresets{
		Damage: config.AttrTiers{
			BasePotential: 2,
			Tiers: map[string]config.TierValue{
				"S": {Base: 20, Potential: 5},
				"B": {Base: 10, Potential: 8},
				"D": {Base: 5, Potential: 10},
			},
		},
		AttackSpeed: config.AttrTiers{
			BasePotential: 0.02,
			Tiers: map[string]config.TierValue{
				"S": {Base: 0.8, Potential: 0.05},
				"B": {Base: 0.6, Potential: 0.07},
				"D": {Base: 0.5, Potential: 0.10},
			},
		},
		Range: config.AttrTiers{
			BasePotential: 2,
			Tiers: map[string]config.TierValue{
				"S": {Base: 150, Potential: 7},
				"B": {Base: 130, Potential: 10},
				"D": {Base: 120, Potential: 15},
			},
		},
	}
}

// testConversionBudgetRules 返回转换测试用的预算规则。
func testConversionBudgetRules() *descriptor.BudgetRules {
	return &descriptor.BudgetRules{
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

// testConversionAbilityCosts 返回转换测试用的能力费用表。
func testConversionAbilityCosts() map[string]int {
	return map[string]int{
		"splash":     8,
		"crit":       7,
		"slowPower":  5,
		"stunChance": 10,
	}
}

// TestBlueprintToTowerDef_BasicConversion 测试基础转换：档位→Base/Potential 映射。
func TestBlueprintToTowerDef_BasicConversion(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-basic",
		Name:        "测试塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "S",
			"atkSpeed": "B",
			"range":    "D",
		},
		Specialty: "damage",
		Abilities: []string{"splash"},
		Strength: descriptor.StrengthConfig{
			Cost:         50,
			Amount:       25,
			MaxPurchases: 4,
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	// 验证 damage S 档位 → Base=20, Potential=5 + 专精加成 2 = 7
	if def.CfgBaseDamage != 20 {
		t.Errorf("CfgBaseDamage: got %v, want 20", def.CfgBaseDamage)
	}
	if def.PotentialDamage != 7 { // 5 + basePotential(2) for specialty
		t.Errorf("PotentialDamage: got %v, want 7", def.PotentialDamage)
	}

	// 验证 atkSpeed B 档位 → Base=0.6, Potential=0.07（无专精加成）
	if def.CfgBaseSpeed != 0.6 {
		t.Errorf("CfgBaseSpeed: got %v, want 0.6", def.CfgBaseSpeed)
	}
	if def.PotentialSpeed != 0.07 {
		t.Errorf("PotentialSpeed: got %v, want 0.07", def.PotentialSpeed)
	}

	// 验证 range D 档位 → Base=120, Potential=15（无专精加成）
	if def.CfgBaseRange != 120 {
		t.Errorf("CfgBaseRange: got %v, want 120", def.CfgBaseRange)
	}
	if def.PotentialRange != 15 {
		t.Errorf("PotentialRange: got %v, want 15", def.PotentialRange)
	}
}

// TestBlueprintToTowerDef_FixedTiersFlag 测试 FixedTiers 标记设置。
func TestBlueprintToTowerDef_FixedTiersFlag(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-fixed",
		Name:        "固定档位塔",
		AttackStyle: "scatter",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if !def.FixedTiers {
		t.Error("FixedTiers should be true for blueprint-based tower")
	}
}

// TestBlueprintToTowerDef_PresetAbilities 测试预设能力注入。
func TestBlueprintToTowerDef_PresetAbilities(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-abilities",
		Name:        "多能力塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
		Abilities: []string{"splash", "crit", "slowPower"},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if len(def.PresetAbilities) != 3 {
		t.Errorf("PresetAbilities length: got %d, want 3", len(def.PresetAbilities))
	}
	if def.AbilityAcquireMode != "preset" {
		t.Errorf("AbilityAcquireMode: got %q, want %q", def.AbilityAcquireMode, "preset")
	}

	// 验证能力列表内容
	expected := []string{"splash", "crit", "slowPower"}
	for i, want := range expected {
		if i >= len(def.PresetAbilities) || def.PresetAbilities[i] != want {
			t.Errorf("PresetAbilities[%d]: got %q, want %q", i, def.PresetAbilities[i], want)
		}
	}
}

// TestBlueprintToTowerDef_AttackStyleMapping 测试攻击方式映射。
func TestBlueprintToTowerDef_AttackStyleMapping(t *testing.T) {
	tests := []struct {
		style string
		want  tower.AttackStyle
	}{
		{"projectile", tower.StyleProjectile},
		{"scatter", tower.StyleScatter},
		{"wideBeam", tower.StyleWideBeam},
		{"spin_aoe", tower.StyleSpinAoE},
		{"radial", tower.StyleRadial},
		{"barrage", tower.StyleBarrage},
	}

	for _, tc := range tests {
		bp := descriptor.TowerBlueprint{
			ID:          "test-" + tc.style,
			Name:        tc.style + "塔",
			AttackStyle: tc.style,
			Tiers: map[string]string{
				"damage":   "B",
				"atkSpeed": "B",
				"range":    "B",
			},
		}

		def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

		if def.AttackStyleID != tc.want {
			t.Errorf("AttackStyleID for %s: got %q, want %q", tc.style, def.AttackStyleID, tc.want)
		}
	}
}

// TestBlueprintToTowerDef_SpriteKeyOverride 测试自定义精灵覆盖。
func TestBlueprintToTowerDef_SpriteKeyOverride(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-custom-sprite",
		Name:        "自定义精灵塔",
		AttackStyle: "scatter",
		SpriteKey:   "custom-sprite",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if def.SpriteKeyOverride != "custom-sprite" {
		t.Errorf("SpriteKeyOverride: got %q, want %q", def.SpriteKeyOverride, "custom-sprite")
	}
}

// TestBlueprintToTowerDef_DefaultSprite 测试默认精灵推导。
func TestBlueprintToTowerDef_DefaultSprite(t *testing.T) {
	tests := []struct {
		style      string
		wantSprite string
	}{
		{"projectile", "sentinel"},
		{"scatter", "shotgun"},
		{"wideBeam", "prism"},
		{"spin_aoe", "cyclone"},
		{"radial", "nova"},
		{"barrage", "gatling"},
	}

	for _, tc := range tests {
		bp := descriptor.TowerBlueprint{
			ID:          "test-default-sprite-" + tc.style,
			Name:        tc.style + "塔",
			AttackStyle: tc.style,
			SpriteKey:   "", // 空=使用默认推导
			Tiers: map[string]string{
				"damage":   "B",
				"atkSpeed": "B",
				"range":    "B",
			},
		}

		def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

		if def.SpriteKeyOverride != tc.wantSprite {
			t.Errorf("Default sprite for %s: got %q, want %q", tc.style, def.SpriteKeyOverride, tc.wantSprite)
		}
	}
}

// TestBlueprintToTowerDef_BuildCostFromBudget 测试建造费用计算。
func TestBlueprintToTowerDef_BuildCostFromBudget(t *testing.T) {
	// 预算：attackStyle(scatter=12) + tiers(S+B+D=4+2+0=6) + specialty(2) + abilities(splash=8) = 28
	// 建造费用：BaseBuildCost(40) + usedBudget(28) * CostPerPoint(0.5) = 40 + 14 = 54
	bp := descriptor.TowerBlueprint{
		ID:          "test-cost",
		Name:        "费用测试塔",
		AttackStyle: "scatter",
		Tiers: map[string]string{
			"damage":   "S",
			"atkSpeed": "B",
			"range":    "D",
		},
		Specialty: "damage",
		Abilities: []string{"splash"},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if def.Cost != 54 {
		t.Errorf("Cost: got %d, want 54", def.Cost)
	}
}

// TestBlueprintToTowerDef_StrengthConfig 测试强度配置映射。
func TestBlueprintToTowerDef_StrengthConfig(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-strength",
		Name:        "强度测试塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
		Strength: descriptor.StrengthConfig{
			Cost:         75,
			Amount:       30,
			MaxPurchases: 5,
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if def.StrengthCost != 75 {
		t.Errorf("StrengthCost: got %d, want 75", def.StrengthCost)
	}
	if def.StrengthAmount != 30 {
		t.Errorf("StrengthAmount: got %v, want 30", def.StrengthAmount)
	}
	if def.MaxStrengthBuys != 5 {
		t.Errorf("MaxStrengthBuys: got %d, want 5", def.MaxStrengthBuys)
	}
}

// TestBlueprintToTowerDef_IdentityFields 测试身份字段映射。
func TestBlueprintToTowerDef_IdentityFields(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "my-custom-tower",
		Name:        "我的自定义塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if def.Key != "my-custom-tower" {
		t.Errorf("Key: got %q, want %q", def.Key, "my-custom-tower")
	}
	if def.Label != "我的自定义塔" {
		t.Errorf("Label: got %q, want %q", def.Label, "我的自定义塔")
	}
}

// TestBlueprintToTowerDef_SpecialtyVariants 测试各种专精加成。
func TestBlueprintToTowerDef_SpecialtyVariants(t *testing.T) {
	baseTiers := map[string]string{
		"damage":   "B",
		"atkSpeed": "B",
		"range":    "B",
	}

	tests := []struct {
		specialty       string
		wantDmgPot      float64
		wantSpdPot      float64
		wantRngPot      float64
		wantSpecialtyID int
	}{
		{"damage", 8 + 2, 0.07, 10, 0},   // damage专精：potential(8) + basePotential(2)
		{"atkSpeed", 8, 0.07 + 0.02, 10, 1}, // atkSpeed专精
		{"range", 8, 0.07, 10 + 2, 2},       // range专精
		{"", 8, 0.07, 10, -1},               // 无专精
	}

	for _, tc := range tests {
		bp := descriptor.TowerBlueprint{
			ID:          "test-specialty-" + tc.specialty,
			Name:        "专精测试塔",
			AttackStyle: "projectile",
			Tiers:       baseTiers,
			Specialty:   tc.specialty,
		}

		def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

		if !floatEq(def.PotentialDamage, tc.wantDmgPot) {
			t.Errorf("[%s] PotentialDamage: got %v, want %v", tc.specialty, def.PotentialDamage, tc.wantDmgPot)
		}
		if !floatEq(def.PotentialSpeed, tc.wantSpdPot) {
			t.Errorf("[%s] PotentialSpeed: got %v, want %v", tc.specialty, def.PotentialSpeed, tc.wantSpdPot)
		}
		if !floatEq(def.PotentialRange, tc.wantRngPot) {
			t.Errorf("[%s] PotentialRange: got %v, want %v", tc.specialty, def.PotentialRange, tc.wantRngPot)
		}
		if def.FixedSpecialty != tc.wantSpecialtyID {
			t.Errorf("[%s] FixedSpecialty: got %d, want %d", tc.specialty, def.FixedSpecialty, tc.wantSpecialtyID)
		}
	}
}

// TestBlueprintToTowerDef_ProjectileSpeed 测试弹道速度默认值。
func TestBlueprintToTowerDef_ProjectileSpeed(t *testing.T) {
	bp := descriptor.TowerBlueprint{
		ID:          "test-proj-speed",
		Name:        "弹道测试塔",
		AttackStyle: "projectile",
		Tiers: map[string]string{
			"damage":   "B",
			"atkSpeed": "B",
			"range":    "B",
		},
	}

	def := descriptor.BlueprintToTowerDef(&bp, testTierPresets(), testConversionBudgetRules(), testConversionAbilityCosts())

	if def.ProjectileSpeed != 300 {
		t.Errorf("ProjectileSpeed: got %v, want 300", def.ProjectileSpeed)
	}
}
