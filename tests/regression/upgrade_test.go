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

	// 建塔时(0波)立即解锁1个槽位（攻击模式）
	if tw.PendingSlots(0) != 1 {
		t.Fatalf("0 waves should have 1 pending, got %d", tw.PendingSlots(0))
	}
	if tw.PendingSlots(2) != 2 {
		t.Fatalf("2 waves should have 2 pending, got %d", tw.PendingSlots(2))
	}
	if tw.PendingSlots(10) != 6 {
		t.Fatalf("10 waves should have 6 pending, got %d", tw.PendingSlots(10))
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

// Verify CategoryName returns i18n keys (without i18n.Init, T() returns the key itself).
func TestRegression_Upgrade_CategoryNames(t *testing.T) {
	names := []string{"tower.category.attack", "tower.category.cc", "tower.category.damage", "tower.category.buff", "tower.category.dot", "tower.category.zone"}
	for i, expected := range names {
		got := tower.CategoryName(i)
		if got != expected {
			t.Fatalf("category %d: expected %s, got %s", i, expected, got)
		}
	}
}

// Verify UnlockedSlots calculation.
func TestRegression_Upgrade_UnlockedSlots(t *testing.T) {
	// 公式: 1 + wavesCleared/WavesPerUnlock, cap at 6
	cases := []struct{ waves, expected int }{
		{0, 1}, {1, 1}, {2, 2}, {3, 2}, {4, 3}, {10, 6}, {12, 6}, {20, 6},
	}
	for _, tc := range cases {
		got := tower.UnlockedSlots(tc.waves)
		if got != tc.expected {
			t.Fatalf("UnlockedSlots(%d) = %d, want %d", tc.waves, got, tc.expected)
		}
	}
}
