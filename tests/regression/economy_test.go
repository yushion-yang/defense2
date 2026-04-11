package regression_test

import (
	"testing"

	"defense2/tests/regression/sim"
)

// ============================================================
// Economy / Gold Regressions
// ============================================================

// Verify that killing an enemy marks it dead (gold awarding is handled
// by the event bus in the real game, not by the sim).
func TestRegression_Economy_KillMarksEnemyDead(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyAt(250, 100, 10, 0). // 10 HP, easy kill
		WithTower("basic", 250, 100, 50, 300).
		WithGold(100).
		Build()

	s.RunTicks(120)

	s.AssertEnemyDead(t, 0)
}

// BUG: Leak should not award gold.
// Fix: Only Kill awards gold, not KillImmediate (used for leaks).
func TestRegression_Economy_LeakNoGold(t *testing.T) {
	s := sim.New().
		WithPath(sim.Pt(0, 100), sim.Pt(50, 100)). // very short path
		WithEnemy("runner", 9999, 500).               // fast, unkillable
		WithGold(100).
		WithLives(20).
		Build()

	s.RunTicks(60)

	s.AssertLives(t, "<", 20)    // leaked
	s.AssertGold(t, "==", 100)   // no gold from leak
}
