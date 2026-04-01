package regression_test

import (
	"testing"

	"defense2/internal/core/strength"
)

// ============================================================
// Strength Data Regressions
// ============================================================

// BUG: EnemyMul applied before EnemySub could create unexpected zero values.
// Fix: Order is documented: sum → multiply → subtract → floor at 0.
func TestRegression_Strength_DebuffOrder(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetEnemyMul("aura", 0.5)   // Base(100)*0.5 = 50
	s.SetEnemySub("debuff", 30)  // 50 - 30 = 20
	if s.Effective() != 20 {
		t.Fatalf("expected 20, got %.1f", s.Effective())
	}
}

// BUG: Multiple EnemyMul factors should multiply together.
func TestRegression_Strength_MultipleEnemyMul(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetEnemyMul("aura1", 0.5) // 100 * 0.5 = 50
	s.SetEnemyMul("aura2", 0.5) // 50 * 0.5 = 25
	if s.Effective() != 25 {
		t.Fatalf("expected 25, got %.1f", s.Effective())
	}
}

// BUG: Temp bonuses should add to base before multipliers.
func TestRegression_Strength_TempPlusBase(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetTemp("chain", 30)        // Base(100) + 30 = 130
	s.SetEnemyMul("debuff", 0.5) // 130 * 0.5 = 65
	if s.Effective() != 65 {
		t.Fatalf("expected 65, got %.1f", s.Effective())
	}
}

// BUG: ClearTransient should remove temp bonuses but keep permanent.
func TestRegression_Strength_ClearTransient(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(20)
	s.SetTemp("chain", 50)
	s.SetEnemyMul("debuff", 0.8)
	s.SetEnemySub("aura", 10)

	s.ClearTransient()

	// After clear: Base(100) + Permanent(20) = 120, no temp/mul/sub
	if s.Effective() != 120 {
		t.Fatalf("after ClearTransient, expected 120, got %.1f", s.Effective())
	}
}

// BUG: Ratio() should return Effective/100 for stat scaling.
func TestRegression_Strength_Ratio(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(50) // Effective = 150

	r := s.Ratio()
	if r != 1.5 {
		t.Fatalf("Ratio should be 1.5, got %.2f", r)
	}
}
