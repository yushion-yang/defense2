package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

func TestSpawnTimerSetOnSpawn(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(100, 100, 100, 50, 0, "test", nil)
	if e == nil {
		t.Fatal("Spawn returned nil")
	}
	if e.SpawnTimer <= 0 {
		t.Errorf("SpawnTimer = %v, want > 0", e.SpawnTimer)
	}
	if e.SpawnDuration <= 0 {
		t.Errorf("SpawnDuration = %v, want > 0", e.SpawnDuration)
	}
}

func TestIsSpawning(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(100, 100, 100, 50, 0, "test", nil)
	if !e.IsSpawning() {
		t.Error("IsSpawning should be true right after spawn")
	}
	e.SpawnTimer = 0
	if e.IsSpawning() {
		t.Error("IsSpawning should be false when timer is 0")
	}
}

func TestSpawnTimerDecreases(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(100, 100, 100, 50, 0, "test", nil)
	initial := e.SpawnTimer
	dt := 0.1
	e.SpawnTimer -= dt
	if e.SpawnTimer >= initial {
		t.Errorf("SpawnTimer should decrease, got %v (was %v)", e.SpawnTimer, initial)
	}
}

func TestBossSpawnDurationLonger(t *testing.T) {
	pool := enemy.NewPool(4)
	normal := pool.Spawn(100, 100, 100, 50, 0, "test", nil)
	bossCfg := enemy.DefaultSpawnConfig()
	bossCfg.Boss = true
	boss := pool.Spawn(200, 100, 100, 50, 0, "boss", bossCfg)
	if boss.SpawnDuration <= normal.SpawnDuration {
		t.Errorf("boss SpawnDuration (%v) should be > normal (%v)", boss.SpawnDuration, normal.SpawnDuration)
	}
}

func TestSpawningEnemyNotTargeted(t *testing.T) {
	p := enemy.NewPool(4)
	e1 := p.Spawn(100, 100, 50, 60, 1, "normal", nil)
	e2 := p.Spawn(110, 100, 50, 60, 1, "normal", nil)

	// e2 finishes spawn anim
	e2.SpawnTimer = 0

	tw := &tower.Tower{X: 100, Y: 100, Range: 200}

	// FindNearestEnemy should skip spawning e1 and find e2
	best := tower.FindNearestEnemy(tw, p)
	if best == nil {
		t.Fatal("should find a non-spawning target")
	}
	if best == e1 {
		t.Fatal("should not target spawning enemy")
	}
	if best != e2 {
		t.Fatal("should target the non-spawning enemy")
	}

	// Make both spawning: no valid targets
	e2.SpawnTimer = 0.2
	best = tower.FindNearestEnemy(tw, p)
	if best != nil {
		t.Fatal("should return nil when all enemies are spawning")
	}
}

func TestSpawnTimerResetOnRespawn(t *testing.T) {
	p := enemy.NewPool(1)
	e := p.Spawn(100, 100, 50, 60, 1, "normal", nil)

	// Complete spawn and then kill+finish to free slot
	e.SpawnTimer = 0
	p.Kill(e)
	p.FinishDying(e)

	// Respawn: should have fresh spawn timer
	e2 := p.Spawn(200, 200, 80, 60, 1, "runner", nil)
	if e2 == nil {
		t.Fatal("respawn should succeed")
	}
	if e2.SpawnTimer <= 0 {
		t.Errorf("respawned enemy SpawnTimer should be > 0, got %f", e2.SpawnTimer)
	}
	if e2.SpawnDuration <= 0 {
		t.Errorf("respawned enemy SpawnDuration should be > 0, got %f", e2.SpawnDuration)
	}
}

func TestNormalSpawnDuration(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(100, 100, 100, 50, 0, "test", nil)
	if e.SpawnTimer != 0.3 {
		t.Errorf("normal SpawnTimer should be 0.3, got %f", e.SpawnTimer)
	}
	if e.SpawnDuration != 0.3 {
		t.Errorf("normal SpawnDuration should be 0.3, got %f", e.SpawnDuration)
	}
}

func TestBossSpawnDuration(t *testing.T) {
	pool := enemy.NewPool(4)
	cfg := enemy.DefaultSpawnConfig()
	cfg.Boss = true
	e := pool.Spawn(100, 100, 100, 50, 0, "boss", cfg)
	if e.SpawnTimer != 0.5 {
		t.Errorf("boss SpawnTimer should be 0.5, got %f", e.SpawnTimer)
	}
	if e.SpawnDuration != 0.5 {
		t.Errorf("boss SpawnDuration should be 0.5, got %f", e.SpawnDuration)
	}
}
