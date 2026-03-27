package enemy

import "defense2/internal/core/gamemap"

// Spawner controls enemy wave spawning.
type Spawner struct {
	Wave         int
	MaxWaves     int
	SpawnTimer   float64
	SpawnIndex   int     // current enemy within wave
	EnemiesPerWave int
	SpawnInterval float64 // seconds between spawns
	WaveInterval float64  // seconds between waves
	WaveTimer    float64
	WaveActive   bool
	AllDone      bool
	Waypoints    []gamemap.Point
}

// NewSpawner creates a spawner for the given map waypoints and wave count.
func NewSpawner(waypoints []gamemap.Point, maxWaves int) *Spawner {
	return &Spawner{
		Wave:           0,
		MaxWaves:       maxWaves,
		EnemiesPerWave: 5,
		SpawnInterval:  0.6,
		WaveInterval:   3.0,
		WaveTimer:      2.0, // initial delay before first wave
		Waypoints:      waypoints,
	}
}

// Update ticks the spawner, spawning enemies into the pool.
func (s *Spawner) Update(pool *Pool, dt float64) {
	if s.AllDone {
		return
	}

	if !s.WaveActive {
		s.WaveTimer -= dt
		if s.WaveTimer <= 0 {
			s.Wave++
			s.SpawnIndex = 0
			s.SpawnTimer = 0
			s.WaveActive = true
		}
		return
	}

	s.SpawnTimer -= dt
	if s.SpawnTimer <= 0 && s.SpawnIndex < s.enemyCount() {
		spawn := s.Waypoints[0]
		hp := 10.0 + float64(s.Wave)*5
		speed := 50.0 + float64(s.Wave)*3
		pool.Spawn(spawn.X, spawn.Y, hp, speed, 8, 1)
		s.SpawnIndex++
		s.SpawnTimer = s.SpawnInterval
	}

	if s.SpawnIndex >= s.enemyCount() {
		s.WaveActive = false
		if s.Wave >= s.MaxWaves {
			s.AllDone = true
		} else {
			s.WaveTimer = s.WaveInterval
		}
	}
}

// enemyCount returns the number of enemies for the current wave.
func (s *Spawner) enemyCount() int {
	return s.EnemiesPerWave + s.Wave
}

// IsClear returns true if all waves are done and no enemies remain.
func (s *Spawner) IsClear(pool *Pool) bool {
	return s.AllDone && pool.Count == 0
}
