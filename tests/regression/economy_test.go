package regression_test

import (
	"testing"

	"defense2/tests/regression/sim"
)

// ============================================================
// Economy / Gold Regressions
// ============================================================

// BUG: Kill should award gold (enemy.Reward).
// Fix: Sim.Step adds enemy Reward to Gold on kill.
func TestRegression_Economy_KillAwardsGold(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemyAt(250, 100, 10, 0). // 10 HP, easy kill
		WithTower("basic", 250, 100, 50, 300).
		WithGold(100).
		Build()

	// Set reward manually (builder doesn't set it)
	s.SpawnedEnemies[0].Reward = 10

	s.RunTicks(120)

	s.AssertEnemyDead(t, 0)
	s.AssertGold(t, ">=", 110) // 100 + 10 reward
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

	s.SpawnedEnemies[0].Reward = 10

	s.RunTicks(60)

	s.AssertLives(t, "<", 20) // leaked
	s.AssertGold(t, "==", 100) // no gold from leak
}
