package regression_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// ============================================================
// Tower Upgrade System Regressions
// ============================================================

// Verify AddAbility assigns to correct category slot.
func TestRegression_Upgrade_AddAbilitySlot(t *testing.T) {
	if config.GlobalAbilityTable() == nil {
		t.Skip("ability table not loaded")
	}

	tw := &tower.Tower{Active: true, Level: 1}

	ok := tw.AddAbility("slowPower")
	if !ok {
		t.Fatal("AddAbility should succeed for empty slot")
	}
	if tw.AbilitySlots[config.AbilityCatCC] != "slowPower" {
		t.Fatalf("expected CC slot = slowPower, got %s", tw.AbilitySlots[config.AbilityCatCC])
	}
	if tw.Level != 2 {
		t.Fatalf("level should be 2, got %d", tw.Level)
	}

	// 同类别不能再加
	ok = tw.AddAbility("stunChance")
	if ok {
		t.Fatal("AddAbility should fail for occupied category")
	}
}

// Verify PendingSlots based on waves cleared.
func TestRegression_Upgrade_PendingSlots(t *testing.T) {
	tw := &tower.Tower{Active: true, Level: 1}

	if tw.PendingSlots(0) != 0 {
		t.Fatalf("0 waves should have 0 pending, got %d", tw.PendingSlots(0))
	}
	if tw.PendingSlots(2) != 1 {
		t.Fatalf("2 waves should have 1 pending, got %d", tw.PendingSlots(2))
	}
	if tw.PendingSlots(12) != 6 {
		t.Fatalf("12 waves should have 6 pending, got %d", tw.PendingSlots(12))
	}
	if tw.PendingSlots(100) != 6 {
		t.Fatalf("100 waves should cap at 6 pending, got %d", tw.PendingSlots(100))
	}
}

// Verify AllAbilities merges slots and legacy.
func TestRegression_Upgrade_AllAbilities(t *testing.T) {
	tw := &tower.Tower{
		Active:    true,
		Abilities: []string{"crit"},
	}
	tw.AbilitySlots[config.AbilityCatCC] = "stunChance"

	all := tw.AllAbilities()
	if len(all) != 2 {
		t.Fatalf("expected 2 abilities, got %d: %v", len(all), all)
	}
}

// Verify ResolveAttackStyle maps Category 0 ability to style.
func TestRegression_Upgrade_ResolveStyle(t *testing.T) {
	tw := &tower.Tower{Active: true}
	if tw.ResolveAttackStyle() != tower.StyleProjectile {
		t.Fatal("default should be projectile")
	}

	tw.AbilitySlots[0] = "scatter"
	if tw.ResolveAttackStyle() != tower.StyleScatter {
		t.Fatalf("expected scatter, got %s", tw.ResolveAttackStyle())
	}

	tw.AbilitySlots[0] = "spinAoe"
	if tw.ResolveAttackStyle() != tower.StyleSpinAoE {
		t.Fatalf("expected spin_aoe, got %s", tw.ResolveAttackStyle())
	}

	tw.AbilitySlots[0] = "radial"
	if tw.ResolveAttackStyle() != tower.StyleRadial {
		t.Fatalf("expected radial, got %s", tw.ResolveAttackStyle())
	}
}

// Verify CategoryName returns Chinese names.
func TestRegression_Upgrade_CategoryNames(t *testing.T) {
	names := []string{"攻击模式", "控制效果", "命中加伤", "增益光环", "持续伤害", "范围效果"}
	for i, expected := range names {
		got := tower.CategoryName(i)
		if got != expected {
			t.Fatalf("category %d: expected %s, got %s", i, expected, got)
		}
	}
}

// Verify UnlockedSlots calculation.
func TestRegression_Upgrade_UnlockedSlots(t *testing.T) {
	cases := []struct{ waves, expected int }{
		{0, 0}, {1, 0}, {2, 1}, {3, 1}, {4, 2}, {10, 5}, {12, 6}, {20, 6},
	}
	for _, tc := range cases {
		got := tower.UnlockedSlots(tc.waves)
		if got != tc.expected {
			t.Fatalf("UnlockedSlots(%d) = %d, want %d", tc.waves, got, tc.expected)
		}
	}
}
