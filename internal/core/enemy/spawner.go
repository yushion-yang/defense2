// spawner.go — 波次出怪管理器。
// 控制敌人按波次间隔生成，支持单路径和多路径地图。
// 每波敌人数量和属性随波次递增。
package enemy

import "defense2/internal/core/gamemap"

// Spawner 波次出怪控制器。
type Spawner struct {
	Wave           int              // 当前波次号（从 1 开始）
	MaxWaves       int              // 总波次数
	SpawnTimer     float64          // 单波内两个敌人之间的倒计时（秒）
	SpawnIndex     int              // 当前波已出第几个敌人
	EnemiesPerWave int              // 每波基础敌人数
	SpawnInterval  float64          // 同波内敌人生成间隔（秒）
	WaveInterval   float64          // 两波之间的间隔（秒）
	WaveTimer      float64          // 波间等待倒计时（秒）
	WaveActive     bool             // 当前波是否正在出怪
	AllDone        bool             // 是否所有波次已出完
	GameMap        *gamemap.GameMap // 运行时地图（用于获取路径）
}

// NewSpawner 创建出怪管理器。
func NewSpawner(gm *gamemap.GameMap, maxWaves int) *Spawner {
	return &Spawner{
		Wave:           0,
		MaxWaves:       maxWaves,
		EnemiesPerWave: 5,
		SpawnInterval:  0.6,
		WaveInterval:   3.0,
		WaveTimer:      2.0,
		GameMap:        gm,
	}
}

// Update 每帧调用，驱动波次计时和敌人生成。
func (s *Spawner) Update(pool *Pool, dt float64) {
	if s.AllDone {
		return
	}

	// 波间等待
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

	// 波内出怪
	s.SpawnTimer -= dt
	if s.SpawnTimer <= 0 && s.SpawnIndex < s.enemyCount() {
		// 选择路径（多路径地图随机，单路径用默认）
		path := s.GameMap.PickPath()
		if len(path) > 0 {
			spawn := path[0]
			hp := 10.0 + float64(s.Wave)*5
			speed := 50.0 + float64(s.Wave)*3
			e := pool.Spawn(spawn.X, spawn.Y, hp, speed, 8, 1)
			if e != nil {
				// 将路径绑定到敌人（多路径时每个敌人可能走不同路）
				e.Path = path
			}
		}
		s.SpawnIndex++
		s.SpawnTimer = s.SpawnInterval
	}

	// 当前波出完
	if s.SpawnIndex >= s.enemyCount() {
		s.WaveActive = false
		if s.Wave >= s.MaxWaves {
			s.AllDone = true
		} else {
			s.WaveTimer = s.WaveInterval
		}
	}
}

// enemyCount 返回当前波的敌人总数。
func (s *Spawner) enemyCount() int {
	return s.EnemiesPerWave + s.Wave
}

// IsClear 判断是否所有波次出完且场上无存活敌人。
func (s *Spawner) IsClear(pool *Pool) bool {
	return s.AllDone && pool.Count == 0
}
