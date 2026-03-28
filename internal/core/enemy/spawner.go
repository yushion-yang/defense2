// spawner.go — 波次出怪管理器。
// 控制敌人按波次间隔生成，每波敌人数量和属性随波次递增。
package enemy

import "defense2/internal/core/gamemap"

// Spawner 波次出怪控制器。
type Spawner struct {
	Wave           int             // 当前波次号（从 1 开始）
	MaxWaves       int             // 总波次数
	SpawnTimer     float64         // 单波内两个敌人之间的倒计时（秒）
	SpawnIndex     int             // 当前波已出第几个敌人
	EnemiesPerWave int             // 每波基础敌人数
	SpawnInterval  float64         // 同波内敌人生成间隔（秒）
	WaveInterval   float64         // 两波之间的间隔（秒）
	WaveTimer      float64         // 波间等待倒计时（秒）
	WaveActive     bool            // 当前波是否正在出怪
	AllDone        bool            // 是否所有波次已出完
	Waypoints      []gamemap.Point // 敌人出生路径点（第一个点为出生位置）
}

// NewSpawner 创建出怪管理器。
func NewSpawner(waypoints []gamemap.Point, maxWaves int) *Spawner {
	return &Spawner{
		Wave:           0,
		MaxWaves:       maxWaves,
		EnemiesPerWave: 5,
		SpawnInterval:  0.6,
		WaveInterval:   3.0,
		WaveTimer:      2.0, // 首波前的等待时间
		Waypoints:      waypoints,
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
		spawn := s.Waypoints[0]
		hp := 10.0 + float64(s.Wave)*5    // 血量随波次递增
		speed := 50.0 + float64(s.Wave)*3  // 速度随波次递增
		pool.Spawn(spawn.X, spawn.Y, hp, speed, 8, 1)
		s.SpawnIndex++
		s.SpawnTimer = s.SpawnInterval
	}

	// 当前波出完，切换到波间等待或结束
	if s.SpawnIndex >= s.enemyCount() {
		s.WaveActive = false
		if s.Wave >= s.MaxWaves {
			s.AllDone = true
		} else {
			s.WaveTimer = s.WaveInterval
		}
	}
}

// enemyCount 返回当前波的敌人总数（基础数 + 波次号）。
func (s *Spawner) enemyCount() int {
	return s.EnemiesPerWave + s.Wave
}

// IsClear 判断是否所有波次出完且场上无存活敌人（玩家胜利条件）。
func (s *Spawner) IsClear(pool *Pool) bool {
	return s.AllDone && pool.Count == 0
}
