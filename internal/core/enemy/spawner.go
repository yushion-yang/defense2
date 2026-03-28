// spawner.go — 波次出怪管理器。
// 控制敌人按波次间隔生成，支持单路径和多路径地图。
// 每波敌人数量和属性随波次递增，原型按波次阶段加权随机选取。
package enemy

import (
	"math/rand"
	"strings"

	"defense2/internal/core/gamemap"
)

// waveEntry 波次组合中一种原型的权重配置。
type waveEntry struct {
	archetype string
	weight    int
}

// waveCompositions 按波次阶段定义原型权重分布。
// 索引越大对应波次越高。Spawner 根据当前 Wave 选择合适的阶段。
var waveCompositions = []struct {
	maxWave int // 该阶段适用的最大波次号（含），0 表示无上限
	entries []waveEntry
}{
	{maxWave: 3, entries: []waveEntry{
		{"normal", 100},
	}},
	{maxWave: 6, entries: []waveEntry{
		{"normal", 70}, {"runner", 20}, {"swarm", 10},
	}},
	{maxWave: 9, entries: []waveEntry{
		{"normal", 50}, {"runner", 20}, {"tank", 15}, {"armored", 10}, {"shielded", 5},
	}},
	{maxWave: 14, entries: []waveEntry{
		{"normal", 40}, {"runner", 15}, {"tank", 15}, {"armored", 10},
		{"flying", 10}, {"healer", 5}, {"stealth", 5},
	}},
	{maxWave: 0, entries: []waveEntry{ // wave 15+
		{"normal", 30}, {"runner", 10}, {"tank", 15}, {"armored", 10},
		{"flying", 10}, {"healer", 5}, {"stealth", 5},
		{"splitter", 5}, {"buffer", 5}, {"teleporter", 5},
	}},
}

// Spawner 波次出怪控制器。
type Spawner struct {
	Wave           int                       // 当前波次号（从 1 开始）
	MaxWaves       int                       // 总波次数
	SpawnTimer     float64                   // 单波内两个敌人之间的倒计时（秒）
	SpawnIndex     int                       // 当前波已出第几个敌人
	EnemiesPerWave int                       // 每波基础敌人数
	SpawnInterval  float64                   // 同波内敌人生成间隔（秒）
	WaveInterval   float64                   // 两波之间的间隔（秒）
	WaveTimer      float64                   // 波间等待倒计时（秒）
	WaveActive     bool                      // 当前波是否正在出怪
	AllDone        bool                      // 是否所有波次已出完
	GameMap        *gamemap.GameMap          // 运行时地图（用于获取路径）
	Archetypes     map[string]*SpawnConfig   // 原型名 → 生成配置（由外部注入）
	EnemyFilter    string                    // 敌人过滤器（ground-only/flying-only/elite-only/boss-only/dummy/stress/none/mixed/""）
	HPScale        float64                   // 难度 HP 倍率（默认 1.0）
	SpeedScale     float64                   // 难度速度倍率（默认 1.0）
	ManualWave     bool                      // 手动开波模式：波间不自动倒计时
	FixedCount     int                       // >0 时每波固定该数量（不随波次递增）
	bossQueued     bool                      // 本波是否需要在末尾追加 Boss
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
	// "none" 过滤器：不生成任何敌人
	if s.EnemyFilter == "none" {
		return
	}

	// 波间等待（ManualWave 模式下不自动倒计时，需外部调用 StartNextWave）
	if !s.WaveActive {
		if s.ManualWave {
			return
		}
		s.WaveTimer -= dt
		if s.WaveTimer <= 0 {
			s.startWave()
		}
		return
	}

	// 波内出怪
	s.SpawnTimer -= dt
	total := s.enemyCount()
	if s.bossQueued {
		total++ // Boss 额外一个名额
	}
	if s.SpawnTimer <= 0 && s.SpawnIndex < total {
		path := s.GameMap.PickPath()
		if len(path) > 0 {
			spawn := path[0]
			hpScale := s.HPScale
			if hpScale <= 0 {
				hpScale = 1.0
			}
			spdScale := s.SpeedScale
			if spdScale <= 0 {
				spdScale = 1.0
			}
			baseHP := (10.0 + float64(s.Wave)*5) * hpScale
			baseSpeed := (50.0 + float64(s.Wave)*3) * spdScale

			var archetype string
			var cfg *SpawnConfig

			if s.bossQueued && s.SpawnIndex == total-1 {
				// 波末 Boss：使用 tank 原型 + Boss 标记 + 额外倍率
				archetype = "tank"
				cfg = s.getConfig(archetype)
				if cfg != nil {
					// 复制一份避免修改原始配置
					bossCfg := *cfg
					bossCfg.Boss = true
					bossCfg.HpScale *= 3   // Boss 额外 3 倍 HP
					bossCfg.Radius *= 1.5  // Boss 体型更大
					cfg = &bossCfg
				}
			} else {
				archetype = s.pickArchetype()
				cfg = s.getConfig(archetype)
			}

			e := pool.Spawn(spawn.X, spawn.Y, baseHP, baseSpeed, 1, archetype, cfg)
			if e != nil {
				e.Path = path
			}
		}
		s.SpawnIndex++
		s.SpawnTimer = s.SpawnInterval
	}

	// 当前波出完
	if s.SpawnIndex >= total {
		s.WaveActive = false
		s.bossQueued = false
		if s.Wave >= s.MaxWaves {
			s.AllDone = true
		} else {
			s.WaveTimer = s.WaveInterval
		}
	}
}

// StartNextWave 手动触发下一波（ManualWave 模式下由外部调用）。
func (s *Spawner) StartNextWave() {
	if s.WaveActive || s.AllDone {
		return
	}
	s.startWave()
}

// startWave 内部开始下一波。
func (s *Spawner) startWave() {
	s.Wave++
	s.SpawnIndex = 0
	s.SpawnTimer = 0
	s.WaveActive = true
	s.bossQueued = (s.Wave%5 == 0)
}

// enemyCount 返回当前波的常规敌人总数（不含 Boss）。
func (s *Spawner) enemyCount() int {
	if s.FixedCount > 0 {
		return s.FixedCount
	}
	return s.EnemiesPerWave + s.Wave
}

// IsClear 判断是否所有波次出完且场上无存活敌人。
func (s *Spawner) IsClear(pool *Pool) bool {
	return s.AllDone && pool.Count == 0
}

// pickArchetype 根据当前波次，从对应阶段的权重表中随机选取原型。
// 若设置了 EnemyFilter，先过滤候选池再加权随机。
func (s *Spawner) pickArchetype() string {
	entries := s.compositionForWave()

	// 应用 EnemyFilter：保留符合条件的权重条目
	filtered := s.filterEntries(entries)
	if len(filtered) == 0 {
		return "normal"
	}

	totalWeight := 0
	for _, e := range filtered {
		totalWeight += e.weight
	}
	if totalWeight <= 0 {
		return filtered[0].archetype
	}

	r := rand.Intn(totalWeight)
	for _, e := range filtered {
		r -= e.weight
		if r < 0 {
			return e.archetype
		}
	}
	return filtered[0].archetype
}

// filterEntries 根据 EnemyFilter 过滤权重条目。
func (s *Spawner) filterEntries(entries []waveEntry) []waveEntry {
	if s.EnemyFilter == "" || s.EnemyFilter == "mixed" {
		// mixed/默认：排除 boss 原型（boss 由波末追加逻辑处理）
		return entries
	}

	var result []waveEntry
	for _, e := range entries {
		cfg := s.getConfig(e.archetype)
		if s.matchesFilter(e.archetype, cfg) {
			result = append(result, e)
		}
	}
	return result
}

// filteredArchetypes 根据 EnemyFilter 从 Archetypes 映射中筛选原型名列表。
func (s *Spawner) filteredArchetypes() []string {
	if s.Archetypes == nil {
		return nil
	}
	var result []string
	for name, cfg := range s.Archetypes {
		if s.matchesFilter(name, cfg) {
			result = append(result, name)
		}
	}
	return result
}

// matchesFilter 判断一个原型是否符合当前 EnemyFilter。
func (s *Spawner) matchesFilter(name string, cfg *SpawnConfig) bool {
	isBoss := cfg != nil && cfg.Boss
	isFlying := strings.HasPrefix(name, "fly-") || strings.HasPrefix(name, "flying")
	isElite := strings.HasPrefix(name, "el-") || strings.Contains(name, "elite")

	switch s.EnemyFilter {
	case "ground-only":
		return !isBoss && !isFlying
	case "flying-only":
		return isFlying
	case "elite-only":
		return isElite
	case "boss-only":
		return isBoss
	case "dummy":
		return name == "dummy"
	case "stress":
		return !isBoss && !isFlying
	case "none":
		return false // no spawning
	case "all-static":
		return true // all types (used by spawnAllStatic, not by wave spawning)
	case "mixed", "":
		return !isBoss // default: exclude boss (added separately)
	default:
		return true
	}
}

// compositionForWave 返回当前波次对应的原型权重表。
func (s *Spawner) compositionForWave() []waveEntry {
	for _, c := range waveCompositions {
		if c.maxWave == 0 || s.Wave <= c.maxWave {
			return c.entries
		}
	}
	// 回退到最后一个阶段
	return waveCompositions[len(waveCompositions)-1].entries
}

// getConfig 查找原型对应的 SpawnConfig。
// 未找到时返回 nil（Pool.Spawn 会使用默认值）。
func (s *Spawner) getConfig(archetype string) *SpawnConfig {
	if s.Archetypes == nil {
		return nil
	}
	cfg, ok := s.Archetypes[archetype]
	if !ok {
		return nil
	}
	return cfg
}
