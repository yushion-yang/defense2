package regression_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// ============================================================
// CC (Crowd Control) Regressions
// ============================================================

// BUG: SlowFactor=0 caused enemy speed to drop to 0.
// Fix: combat.ApplySlow clamps factor to MinSpeedRatio (0.2).
func TestRegression_CC_SlowMinSpeedClamp(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 100, 100).
		Build()

	combat.ApplySlow(s.SpawnedEnemies[0], 0.0, 5.0, "test")
	s.RunTicks(1)

	s.AssertEnemySpeed(t, 0, ">=", 100*enemy.MinSpeedRatio)
	s.AssertEnemySpeed(t, 0, ">", 0)
}

// BUG: Stun did not prevent movement.
// Fix: MoveAlongPath checks StunTimer > 0 and skips movement.
func TestRegression_CC_StunPreventsMovement(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 100, 100).
		Build()

	startX := s.SpawnedEnemies[0].X
	combat.ApplyStun(s.SpawnedEnemies[0], 2.0, "test")
	s.RunTicks(60) // 1 second

	if s.SpawnedEnemies[0].X != startX {
		t.Fatalf("stunned enemy moved: startX=%.1f nowX=%.1f", startX, s.SpawnedEnemies[0].X)
	}
}

// BUG: Tenacity was not reducing CC duration.
// Fix: ApplyStun/ApplySlow multiply duration by (1 - Tenacity).
func TestRegression_CC_TenacityReducesDuration(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("boss", 1000, 100).
		Build()

	e := s.SpawnedEnemies[0]
	e.Tenacity = 0.5

	combat.ApplyStun(e, 2.0, "test")
	if e.StunTimer != 1.0 {
		t.Fatalf("stun with 50%% tenacity should be 1.0s, got %.2f", e.StunTimer)
	}

	combat.ApplySlow(e, 0.5, 4.0, "test")
	if e.SlowTimer != 2.0 {
		t.Fatalf("slow with 50%% tenacity should be 2.0s, got %.2f", e.SlowTimer)
	}
}

// BUG: Full tenacity (1.0) should make enemy CC immune.
func TestRegression_CC_FullTenacityImmune(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("boss", 1000, 100).
		Build()

	e := s.SpawnedEnemies[0]
	e.Tenacity = 1.0

	ok := combat.ApplyStun(e, 2.0, "test")
	if ok {
		t.Fatal("stun should fail with tenacity=1.0")
	}
	if e.StunTimer != 0 {
		t.Fatalf("stun timer should be 0, got %.2f", e.StunTimer)
	}
}

// ============================================================
// Projectile Regressions
// ============================================================

// BUG: Projectile was fire-and-forget (fixed VX/VY), missed when enemies turned.
// Fix: Projectile.Update() recalculates VX/VY toward target each frame.
func TestRegression_Projectile_TrackingTarget(t *testing.T) {
	s := sim.New().
		WithPath(
			sim.Pt(0, 0),
			sim.Pt(200, 0),
			sim.Pt(200, 200),
		).
		WithEnemy("normal", 50, 100).
		WithTower("archer", 100, 0, 50, 300).
		Build()

	dead := s.RunUntil(func(s *sim.Sim) bool {
		e := s.SpawnedEnemies[0]
		return e.IsDying() || !e.Active
	}, 300)

	if !dead {
		t.Fatalf("projectile should have tracked and killed enemy, HP=%.1f",
			s.SpawnedEnemies[0].HP)
	}
}

// ============================================================
// DoT Regressions
// ============================================================

// BUG: Burn was treated as bleed (shared timer).
// Fix: Burn has independent BurnTimer/BurnDPS fields.
func TestRegression_DoT_BurnIndependentOfBleed(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 0). // stationary
		ApplyBurn(0, 100, 3.0).
		ApplyBleed(0, 50, 3.0).
		Build()

	e := s.SpawnedEnemies[0]
	if e.BurnTimer <= 0 || e.BleedTimer <= 0 {
		t.Fatal("both burn and bleed should be active")
	}
	if e.BurnDPS != 100 || e.BleedDPS != 50 {
		t.Fatalf("DPS values wrong: burn=%.0f bleed=%.0f", e.BurnDPS, e.BleedDPS)
	}

	// Run 1 DoT tick cycle (0.5s = 30 ticks)
	s.RunTicks(30)

	// Combined DoT per tick: (100+50) * 0.5 = 75
	expectedHP := 1000.0 - 75.0
	// Allow +-10 tolerance for timing
	if e.HP > expectedHP+10 || e.HP < expectedHP-10 {
		t.Fatalf("HP after 1 DoT tick should be ~%.0f, got %.1f", expectedHP, e.HP)
	}
}
