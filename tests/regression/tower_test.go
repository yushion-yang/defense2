package regression_test

import (
	"testing"

	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/tests/regression/sim"
)

// ============================================================
// Strength System Regressions
// ============================================================

// BUG: Tower strength dropping to 0 produced incorrect stats.
// Fix: RecalcStats uses ratio = Effective()/100; at 0 strength only BaseDamage remains.
func TestRegression_Tower_StrengthZeroFloor(t *testing.T) {
	tw := &tower.Tower{
		BaseDamage:      10,
		PotentialDamage: 20,
		BaseSpeed:       0.3,
		PotentialSpeed:  0.7,
		BaseRange:       100,
		PotentialRange:  50,
		Strength:        strength.NewStrengthData(),
		Active:          true,
	}

	// Reduce strength to 0 via AddPermanent
	tw.Strength.AddPermanent(-100)
	if tw.Strength.Effective() != 0 {
		t.Fatalf("expected Effective()=0, got %.1f", tw.Strength.Effective())
	}

	tw.RecalcStats()

	// At strength=0, ratio=0, only base stats remain
	if tw.Damage != 10 {
		t.Fatalf("damage at strength 0 should be BaseDamage=10, got %.1f", tw.Damage)
	}
	if tw.AttackSpeed != 0.3 {
		t.Fatalf("attackSpeed at strength 0 should be BaseSpeed=0.3, got %.3f", tw.AttackSpeed)
	}
	if tw.Range != 100 {
		t.Fatalf("range at strength 0 should be BaseRange=100, got %.1f", tw.Range)
	}
}

// BUG: Strength clamp at -Base caused confusing debug behavior with AddPermanent.
// Fix: AddPermanent clamps Permanent >= -Base (so Effective is floored at 0).
func TestRegression_Tower_StrengthPermanentClamp(t *testing.T) {
	s := strength.NewStrengthData()

	// Base=100; subtract 200 should clamp Permanent to -100
	s.AddPermanent(-200)
	if s.Effective() != 0 {
		t.Fatalf("after -200, Effective should be 0, got %.1f", s.Effective())
	}

	// Now add back 50
	s.AddPermanent(50)
	if s.Effective() != 50 {
		t.Fatalf("after +50 from clamped, Effective should be 50, got %.1f", s.Effective())
	}

	// Another 50 recovers to baseline
	s.AddPermanent(50)
	if s.Effective() != 100 {
		t.Fatalf("after +50+50, Effective should be 100, got %.1f", s.Effective())
	}
}

// BUG: Strength ratio below 0 with enemy debuffs.
// Fix: Effective() floors at 0 before dividing.
func TestRegression_Tower_StrengthNegativeDebuff(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetEnemySub("debuff1", 150) // subtract 150 from base 100
	if s.Effective() != 0 {
		t.Fatalf("Effective should floor at 0, got %.1f", s.Effective())
	}
}

// BUG: Full strength (100) should produce base + potential stats.
func TestRegression_Tower_StrengthFullStats(t *testing.T) {
	tw := &tower.Tower{
		BaseDamage:      10,
		PotentialDamage: 20,
		BaseSpeed:       0.3,
		PotentialSpeed:  0.7,
		BaseRange:       100,
		PotentialRange:  50,
		Strength:        strength.NewStrengthData(),
		Active:          true,
	}

	tw.RecalcStats()

	// At strength=100, ratio=1.0, base + potential
	if tw.Damage != 30 {
		t.Fatalf("damage at full strength should be 30, got %.1f", tw.Damage)
	}
	if tw.AttackSpeed != 1.0 {
		t.Fatalf("attackSpeed should be 1.0, got %.3f", tw.AttackSpeed)
	}
	if tw.Range != 150 {
		t.Fatalf("range should be 150, got %.1f", tw.Range)
	}
}

// ============================================================
// Direct Damage Tower Regressions (laser/spin_aoe/aura_dot)
// ============================================================

// BUG: Laser tower direct-damage didn't kill enemies.
// Fix: Sim handles direct-damage styles (laser/wideBeam/spinAoE/auraDot) without projectiles.
func TestRegression_Tower_DirectDamageKill(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyAt(250, 100, 50, 0). // stationary enemy right next to tower
		WithTowerFull("laser", tower.StyleLaser, 250, 100, 100, 200, 2.0, 0).
		Build()

	s.RunTicks(60) // 1 second, tower should fire twice (atkSpd=2)

	s.AssertEnemyDead(t, 0)
}

// BUG: Spin AoE tower hits all enemies in range.
// Verify: Multiple enemies within range are all damaged.
func TestRegression_Tower_SpinAoEHitsAll(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyAt(250, 100, 200, 0). // stationary in range
		WithEnemyAt(260, 100, 200, 0). // also in range
		WithEnemyAt(800, 100, 200, 0). // out of range
		WithTowerFull("spin", tower.StyleSpinAoE, 250, 100, 50, 100, 1.0, 0).
		Build()

	s.RunTicks(120) // 2 seconds

	// Both in-range enemies should be damaged
	s.AssertEnemyHP(t, 0, "<", 200)
	s.AssertEnemyHP(t, 1, "<", 200)
	// Out-of-range enemy untouched
	s.AssertEnemyHP(t, 2, "==", 200)
}
