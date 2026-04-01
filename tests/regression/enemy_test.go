package regression_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/tests/regression/sim"
)

// BUG: All enemy archetypes were "normal" — SpawnConfig scales were not applied.
// Fix: Pool.Spawn applies cfg.HpScale/SpeedScale/Radius to base values.
func TestRegression_Enemy_ArchetypeScaling(t *testing.T) {
	cfg := &enemy.SpawnConfig{
		HpScale:    2.0,
		SpeedScale: 0.5,
		Radius:     12,
	}
	s := sim.New().
		WithStraightPath(500).
		WithEnemyCfg("tank", 100, 60, cfg).
		Build()

	e := s.SpawnedEnemies[0]
	if e.HP != 200 { // 100 * 2.0
		t.Fatalf("HP should be 200, got %.1f", e.HP)
	}
	if e.Speed != 30 { // 60 * 0.5
		t.Fatalf("Speed should be 30, got %.1f", e.Speed)
	}
	if e.Radius != 12 {
		t.Fatalf("Radius should be 12, got %.1f", e.Radius)
	}
	if e.Archetype != "tank" {
		t.Fatalf("Archetype should be 'tank', got '%s'", e.Archetype)
	}
}

// BUG: Dying enemy was still targeted by towers.
// Fix: FindNearestEnemy skips enemies where IsDying() is true.
func TestRegression_Enemy_DyingNotTargeted(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1, 60).   // will die from 1 hit
		WithEnemy("normal", 100, 60). // backup target
		WithTower("basic", 250, 100, 10, 300).
		Build()

	// Run enough ticks for tower to fire and kill first enemy
	s.RunTicks(120)

	// First enemy should be dead
	s.AssertEnemyDead(t, 0)
}

// BUG: Kill→Count was out of sync.
// Fix: Kill decrements Count immediately; FinishDying only clears Active.
func TestRegression_Enemy_KillCountTiming(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if pool.Count != 1 {
		t.Fatalf("count after spawn should be 1, got %d", pool.Count)
	}

	pool.Kill(e)
	if pool.Count != 0 {
		t.Fatalf("count after Kill should be 0, got %d", pool.Count)
	}
	if !e.Active {
		t.Fatal("enemy should still be Active during dying animation")
	}

	pool.FinishDying(e)
	if e.Active {
		t.Fatal("enemy should be inactive after FinishDying")
	}
}

// BUG: Enemy leaking didn't decrement lives.
// Fix: Sim.Step checks ReachedEnd and decrements Lives.
func TestRegression_Enemy_LeakLosesLife(t *testing.T) {
	s := sim.New().
		WithPath(
			sim.Pt(0, 100),
			sim.Pt(100, 100),
		).
		WithEnemy("runner", 100, 1000). // very fast
		WithLives(20).
		Build()

	s.RunTicks(60)

	s.AssertLives(t, "<", 20)
	s.AssertLeaked(t, ">", 0)
}
