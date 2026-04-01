package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

func TestEnemyIsDying(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("spawn should succeed")
	}

	// Fresh enemy is not dying
	if e.IsDying() {
		t.Fatal("new enemy should not be dying")
	}

	// Kill starts the dying animation
	p.Kill(e)
	if !e.IsDying() {
		t.Fatal("killed enemy should be dying")
	}
	if e.DyingTimer <= 0 {
		t.Fatalf("DyingTimer should be >0, got %f", e.DyingTimer)
	}
	if !e.Active {
		t.Fatal("dying enemy should still be Active")
	}
}

func TestEnemyFinishDying(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)

	p.Kill(e)
	if !e.IsDying() {
		t.Fatal("enemy should be dying after Kill")
	}

	// Simulate ticking DyingTimer to 0
	e.DyingTimer = 0
	if e.IsDying() {
		t.Fatal("enemy with DyingTimer=0 should not be dying")
	}

	// FinishDying deactivates the enemy
	p.FinishDying(e)
	if e.Active {
		t.Fatal("enemy should be inactive after FinishDying")
	}
	if e.DyingTimer != 0 {
		t.Fatalf("DyingTimer should be 0 after FinishDying, got %f", e.DyingTimer)
	}
}

func TestDyingEnemyNotTargeted(t *testing.T) {
	p := enemy.NewPool(4)
	e1 := p.Spawn(100, 100, 50, 60, 1, "normal", nil)
	e2 := p.Spawn(110, 100, 50, 60, 1, "normal", nil)

	tw := &tower.Tower{X: 100, Y: 100, Range: 200}

	// Both alive: FindNearestEnemy should find one
	best := tower.FindNearestEnemy(tw, p)
	if best == nil {
		t.Fatal("should find a target")
	}

	// Kill e1 (closer to tower): it enters dying state
	p.Kill(e1)
	if !e1.IsDying() {
		t.Fatal("e1 should be dying")
	}

	// FindNearestEnemy should skip the dying enemy and find e2
	best = tower.FindNearestEnemy(tw, p)
	if best == nil {
		t.Fatal("should find non-dying target")
	}
	if best == e1 {
		t.Fatal("should not target dying enemy")
	}
	if best != e2 {
		t.Fatal("should target the alive, non-dying enemy")
	}

	// Kill e2 too: no valid targets
	p.Kill(e2)
	best = tower.FindNearestEnemy(tw, p)
	if best != nil {
		t.Fatal("should return nil when all enemies are dying")
	}
}

func TestBossDyingDuration(t *testing.T) {
	p := enemy.NewPool(4)
	cfg := enemy.DefaultSpawnConfig()
	cfg.Boss = true
	e := p.Spawn(100, 100, 500, 30, 1, "boss", cfg)
	if e == nil {
		t.Fatal("spawn should succeed")
	}
	if !e.Boss {
		t.Fatal("enemy should be a boss")
	}

	p.Kill(e)
	if !e.IsDying() {
		t.Fatal("boss should be dying")
	}
	if e.DyingTimer != 0.5 {
		t.Fatalf("boss DyingTimer should be 0.5, got %f", e.DyingTimer)
	}
	if e.DyingDuration != 0.5 {
		t.Fatalf("boss DyingDuration should be 0.5, got %f", e.DyingDuration)
	}
}

func TestPoolCountDecrementsOnKill(t *testing.T) {
	p := enemy.NewPool(4)
	p.Spawn(100, 100, 50, 60, 1, "normal", nil)
	e2 := p.Spawn(200, 200, 50, 60, 1, "normal", nil)
	if p.Count != 2 {
		t.Fatalf("count should be 2, got %d", p.Count)
	}

	// Kill decrements Count immediately (even though enemy is still Active/dying)
	p.Kill(e2)
	if p.Count != 1 {
		t.Fatalf("count should be 1 after Kill, got %d", p.Count)
	}
	if !e2.Active {
		t.Fatal("dying enemy should still be Active")
	}

	// FinishDying does NOT decrement Count again
	p.FinishDying(e2)
	if p.Count != 1 {
		t.Fatalf("count should still be 1 after FinishDying, got %d", p.Count)
	}
	if e2.Active {
		t.Fatal("enemy should be inactive after FinishDying")
	}
}

func TestKillImmediateBypassesDying(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)
	if p.Count != 1 {
		t.Fatalf("count should be 1, got %d", p.Count)
	}

	// KillImmediate deactivates without dying animation
	p.KillImmediate(e)
	if e.Active {
		t.Fatal("enemy should be immediately inactive")
	}
	if e.IsDying() {
		t.Fatal("immediately killed enemy should not be dying")
	}
	if p.Count != 0 {
		t.Fatalf("count should be 0, got %d", p.Count)
	}
}

func TestKillImmediateOnDyingEnemy(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)

	// Kill starts dying (Count decrements)
	p.Kill(e)
	if p.Count != 0 {
		t.Fatalf("count should be 0 after Kill, got %d", p.Count)
	}

	// KillImmediate on a dying enemy should not double-decrement Count
	p.KillImmediate(e)
	if p.Count != 0 {
		t.Fatalf("count should still be 0, got %d", p.Count)
	}
	if e.Active {
		t.Fatal("enemy should be inactive")
	}
}

func TestDoubleKillNoEffect(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)

	p.Kill(e)
	if p.Count != 0 {
		t.Fatalf("count should be 0, got %d", p.Count)
	}

	// Second Kill should have no effect (already dying)
	p.Kill(e)
	if p.Count != 0 {
		t.Fatalf("count should still be 0 after double kill, got %d", p.Count)
	}
}

func TestSpawnResetsDyingFields(t *testing.T) {
	p := enemy.NewPool(4)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)

	// Kill and finish dying to free the slot
	p.Kill(e)
	p.FinishDying(e)

	// Respawn in same slot
	e2 := p.Spawn(200, 200, 80, 60, 1, "runner", nil)
	if e2 == nil {
		t.Fatal("respawn should succeed")
	}
	if e2.DyingTimer != 0 {
		t.Fatalf("respawned enemy DyingTimer should be 0, got %f", e2.DyingTimer)
	}
	if e2.DyingDuration != 0 {
		t.Fatalf("respawned enemy DyingDuration should be 0, got %f", e2.DyingDuration)
	}
	if e2.IsDying() {
		t.Fatal("respawned enemy should not be dying")
	}
}
