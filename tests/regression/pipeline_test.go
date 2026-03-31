package regression_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// BUG: Shield absorb was not applied, damage went straight to HP.
// Fix: ProcessDamage step 5 consumes ShieldHP before HP.
func TestRegression_Pipeline_ShieldAbsorb(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("shielded", 100, 60, &enemy.SpawnConfig{
			HpScale: 1, SpeedScale: 1, Radius: 8, ShieldScale: 0.5,
		}).
		Build()

	e := s.SpawnedEnemies[0]
	// Enemy has 100 HP + 50 shield (100 * 0.5)
	result := combat.ProcessDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 30,
	})

	if result.ShieldAbsorbed != 30 {
		t.Fatalf("shield should absorb 30, absorbed %.1f", result.ShieldAbsorbed)
	}
	if e.HP != 100 {
		t.Fatalf("HP should be untouched at 100, got %.1f", e.HP)
	}
}

// BUG: Pure damage did not bypass shield.
// Fix: ProcessDamage step 5 skips shield for pure damage type.
func TestRegression_Pipeline_PureBypassesShield(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("shielded", 100, 60, &enemy.SpawnConfig{
			HpScale: 1, SpeedScale: 1, Radius: 8, ShieldScale: 0.5,
		}).
		Build()

	e := s.SpawnedEnemies[0]
	result := combat.ProcessDamage(combat.DamageInput{
		Target:     e,
		RawDamage:  30,
		DamageType: combat.DmgPure,
	})

	if result.ShieldAbsorbed != 0 {
		t.Fatalf("pure should bypass shield, absorbed %.1f", result.ShieldAbsorbed)
	}
	if e.HP != 70 {
		t.Fatalf("HP should be 70, got %.1f", e.HP)
	}
}

// BUG: DamageCap not disabled when enemy is silenced.
// Fix: ProcessDamage step 4.5 skips damageCap when Silenced=true.
func TestRegression_Pipeline_SilenceDisablesDamageCap(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 60).
		Build()

	e := s.SpawnedEnemies[0]
	e.DamageCap = 50

	// Without silence: damage capped at 50
	r1 := combat.ProcessDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r1.FinalDamage > 50 {
		t.Fatalf("capped damage should be <=50, got %.1f", r1.FinalDamage)
	}

	// With silence: damage cap disabled
	e.HP = 1000
	e.Silenced = true
	r2 := combat.ProcessDamage(combat.DamageInput{
		Target: e, RawDamage: 200,
	})
	if r2.FinalDamage != 200 {
		t.Fatalf("silenced should disable cap, got %.1f", r2.FinalDamage)
	}
}
