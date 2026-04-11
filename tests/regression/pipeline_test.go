package regression_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/tests/regression/sim"
)

// BUG: DamageCap not disabled when enemy is silenced.
// Fix: ApplyDamage step 4.5 skips damageCap when Silenced=true.
func TestRegression_Pipeline_SilenceDisablesDamageCap(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 60).
		Build()

	e := s.SpawnedEnemies[0]
	e.DamageCap = 50

	// Without silence: damage capped at 50
	r1 := combat.ApplyDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r1.FinalDamage > 50 {
		t.Fatalf("capped damage should be <=50, got %.1f", r1.FinalDamage)
	}

	// With silence: damage cap disabled
	e.HP = 1000
	e.Silenced = true
	r2 := combat.ApplyDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r2.FinalDamage != 200 {
		t.Fatalf("silenced should disable cap, got %.1f", r2.FinalDamage)
	}
}
