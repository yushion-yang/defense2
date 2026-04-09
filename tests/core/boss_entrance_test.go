package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
)

// testGameMap returns a minimal GameMap with a simple 2-waypoint path.
func testGameMap() *gamemap.GameMap {
	return &gamemap.GameMap{
		Waypoints: []gamemap.Point{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
		},
		CellSize: 32,
	}
}

func TestBossWaveHasEntranceDelay(t *testing.T) {
	gm := testGameMap()
	s := enemy.NewSpawner(gm, 10)

	// Advance waves 1-4 (non-boss), then start wave 5 (boss)
	for i := 0; i < 4; i++ {
		s.StartNextWave()
		// Complete the wave by updating until WaveActive goes false
		pool := enemy.NewPool(200)
		for s.WaveActive {
			s.Update(pool, 0.6)
		}
	}

	// Wave 5 should be boss (BossEveryNWaves defaults to 5)
	s.StartNextWave()
	if s.EntranceDelay <= 0 {
		t.Errorf("Boss wave EntranceDelay = %v, want > 0", s.EntranceDelay)
	}
	if s.EntranceDelay != 3.0 {
		t.Errorf("Boss wave EntranceDelay = %v, want 3.0", s.EntranceDelay)
	}
}

func TestNonBossWaveNoDelay(t *testing.T) {
	gm := testGameMap()
	s := enemy.NewSpawner(gm, 10)

	s.StartNextWave() // wave 1 - not boss
	if s.EntranceDelay != 0 {
		t.Errorf("Non-boss wave EntranceDelay = %v, want 0", s.EntranceDelay)
	}
}

func TestEntranceDelayPreventsSpawning(t *testing.T) {
	gm := testGameMap()
	s := enemy.NewSpawner(gm, 10)
	pool := enemy.NewPool(200)

	// Advance to wave 5 (boss)
	for i := 0; i < 4; i++ {
		s.StartNextWave()
		for s.WaveActive {
			s.Update(pool, 0.6)
		}
	}

	initialCount := pool.Count
	s.StartNextWave() // wave 5 = boss
	if !s.WaveActive {
		t.Fatal("Wave should be active after StartNextWave")
	}

	// Update with small dt — should NOT spawn during entrance delay
	s.Update(pool, 0.5)
	if pool.Count > initialCount {
		t.Error("Enemies should not spawn during EntranceDelay")
	}
	if s.EntranceDelay <= 0 {
		t.Error("EntranceDelay should still be active after 0.5s")
	}

	// Update enough to exhaust the 3s delay (0.5 + 2.5 = 3.0)
	s.Update(pool, 2.5)
	if s.EntranceDelay != 0 {
		t.Errorf("EntranceDelay should be 0 after 3s total, got %v", s.EntranceDelay)
	}

	// Now spawning should proceed
	s.Update(pool, 0.1)
	if pool.Count <= initialCount {
		t.Error("Enemies should spawn after EntranceDelay expires")
	}
}

func TestEntranceDelayDecrementsCorrectly(t *testing.T) {
	gm := testGameMap()
	s := enemy.NewSpawner(gm, 10)
	pool := enemy.NewPool(200)

	// Advance to wave 5 (boss)
	for i := 0; i < 4; i++ {
		s.StartNextWave()
		for s.WaveActive {
			s.Update(pool, 0.6)
		}
	}

	s.StartNextWave()
	initial := s.EntranceDelay

	s.Update(pool, 1.0)
	expected := initial - 1.0
	if s.EntranceDelay != expected {
		t.Errorf("EntranceDelay after 1s = %v, want %v", s.EntranceDelay, expected)
	}
}
