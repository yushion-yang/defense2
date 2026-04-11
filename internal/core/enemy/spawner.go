// spawner.go — 波次出怪管理器。
// 控制敌人按波次间隔生成，支持单路径和多路径地图。
// 每波敌人数量和属性随波次递增，原型按波次阶段加权随机选取。
package enemy

import (
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/gamemap"
	tel "defense2/internal/core/telemetry"
)

// waveEntry 波次组合中一种原型的权重配置。
type waveEntry struct {
	archetype string
	weight    int
}

// getComposition 根据波次号从配置中获取对应的原型权重表。
func getComposition(wave int) []waveEntry {
	comps := config.GlobalWaveCompositions()
	for _, c := range comps {
		if c.MaxWave == 0 || wave <= c.MaxWave {
			entries := make([]waveEntry, 0, len(c.Enemies))
			for arch, w := range c.Enemies {
				entries = append(entries, waveEntry{archetype: arch, weight: w})
			}
			return entries
		}
	}
	// 回退到最后一个阶段
	if len(comps) > 0 {
		last := comps[len(comps)-1]
		entries := make([]waveEntry, 0, len(last.Enemies))
		for arch, w := range last.Enemies {
			entries = append(entries, waveEntry{archetype: arch, weight: w})
		}
		return entries
	}
	return []waveEntry{{"normal", 100}}
}

// Spawner 波次出怪控制器。
type Spawner struct {
	Wave              int                     // 当前波次号（从 1 开始）
	MaxWaves          int                     // 总波次数
	SpawnTimer        float64                 // 单波内两个敌人之间的倒计时（秒）
	SpawnIndex        int                     // 当前波已出第几个敌人
	EnemiesPerWave    int                     // 每波基础敌人数
	SpawnInterval     float64                 // 同波内敌人生成间隔（秒）
	WaveInterval      float64                 // 两波之间的间隔（秒）
	WaveTimer         float64                 // 波间等待倒计时（秒）
	FirstWaveInterval float64                 // 第一波等待时间（秒），默认 20
	WaveActive        bool                    // 当前波是否正在出怪
	AllDone           bool                    // 是否所有波次已出完
	GameMap           *gamemap.GameMap        // 运行时地图（用于获取路径）
	Archetypes        map[string]*SpawnConfig // 原型名 → 生成配置（由外部注入）
	EnemyFilter       string                  // 敌人过滤器（ground-only/flying-only/elite-only/boss-only/dummy/stress/none/mixed/""）
	HPScale           float64                 // 难度 HP 倍率（默认 1.0）
	SpeedScale        float64                 // 难度速度倍率（默认 1.0）
	ManualWave        bool                    // 手动开波模式：波间到 0 不自动开波，需外部调用 StartNextWave
	FixedCount        int                     // >0 时每波固定该数量（不随波次递增）
	BossEveryWave     bool                    // true 时每波末尾都出 Boss（bossRush 模式用）
	bossQueued        bool                    // 本波是否需要在末尾追加 Boss
	EntranceDelay     float64                 // Boss 波入场延迟（秒），>0 时暂停出怪
}

// NewSpawner 创建出怪管理器。
func NewSpawner(gm *gamemap.GameMap, maxWaves int) *Spawner {
	sc := config.GlobalSpawnerConfig()
	return &Spawner{
		Wave:              0,
		MaxWaves:          maxWaves,
		EnemiesPerWave:    sc.Scaling.EnemiesPerWave,
		SpawnInterval:     sc.Scaling.SpawnInterval,
		WaveInterval:      sc.Timing.WaveInterval,
		FirstWaveInterval: sc.Timing.FirstWaveInterval,
		WaveTimer:         sc.Timing.FirstWaveInterval,
		GameMap:           gm,
	}
}

// Tick 每帧调用，驱动波次计时和敌人生成。
func (s *Spawner) Tick(pool *Pool, dt float64) {
	if s.AllDone {
		return
	}
	// "none" 过滤器：不生成任何敌人
	if s.EnemyFilter == FilterNone {
		return
	}

	// 波间等待
	if !s.WaveActive {
		// 倒计时始终递减（UI 显示用），到 0 停住
		if s.WaveTimer > 0 {
			s.WaveTimer -= dt
			if s.WaveTimer < 0 {
				s.WaveTimer = 0
			}
		}
		// ManualWave 模式不自动开波，需外部调用 StartNextWave
		if s.ManualWave {
			return
		}
		if s.WaveTimer <= 0 {
			s.startWave()
		}
		return
	}

	// Boss 入场延迟：倒计时结束前不出怪
	if s.EntranceDelay > 0 {
		s.EntranceDelay -= dt
		if s.EntranceDelay < 0 {
			s.EntranceDelay = 0
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
			sc := config.GlobalSpawnerConfig()
			baseHP := (sc.Scaling.HpBase + float64(s.Wave)*sc.Scaling.HpPerWave) * hpScale
			baseSpeed := (sc.Scaling.SpeedBase + float64(s.Wave)*sc.Scaling.SpeedPerWave) * spdScale

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
					bossCfg.HpScale *= sc.Boss.HpMultBase + float64(s.Wave) // Boss HP 随波次增长
					bossCfg.Radius *= sc.Boss.RadiusScale                   // Boss 体型更大
					cfg = &bossCfg
					tel.T.Record("boss", archetype)
				}
			} else {
				archetype = s.pickArchetype()
				cfg = s.getConfig(archetype)
			}

			e := pool.Spawn(spawn.X, spawn.Y, baseHP, baseSpeed, 1, archetype, cfg)
			if e != nil {
				e.Path = path
				// TODO: Boss 行为将通过能力系统装配
				// 波次 buff 自动注入
				s.applyWaveBuffs(e)
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

// TimeToNextWave 返回距离下一波的剩余倒计时秒数（UI 显示用）。
// 波内或已结束时返回 0。
func (s *Spawner) TimeToNextWave() float64 {
	if s.WaveActive || s.AllDone {
		return 0
	}
	return s.WaveTimer
}

// IsIntermission 返回是否处于波间等待状态（可开波）。
func (s *Spawner) IsIntermission() bool {
	return !s.WaveActive && !s.AllDone
}

// StartNextWave 手动触发下一波（跳过倒计时）。
func (s *Spawner) StartNextWave() {
	if s.WaveActive || s.AllDone {
		return
	}
	s.WaveTimer = 0
	s.startWave()
}

// startWave 内部开始下一波。
func (s *Spawner) startWave() {
	s.Wave++
	s.SpawnIndex = 0
	s.SpawnTimer = 0
	s.WaveActive = true
	sc := config.GlobalSpawnerConfig()
	// 逐波衰减出怪间隔
	s.SpawnInterval = sc.EffectiveSpawnInterval(s.Wave)
	s.bossQueued = s.BossEveryWave || (s.Wave%sc.Boss.EveryNWaves == 0)
	// Boss 波入场延迟：给玩家 3 秒准备时间
	if s.bossQueued {
		s.EntranceDelay = sc.Boss.EntranceDelay
	} else {
		s.EntranceDelay = 0
	}
}

// WavePreviewEntry 下一波预览中的一种敌人。
type WavePreviewEntry struct {
	Archetype string
	Label     string
	Count     int
}

// PreviewWave 返回指定波次的敌人组合预览（无需 Spawner 实例）。
// archetypes 提供原型 Label 映射，可为 nil（此时 Label 用原型名）。
// 返回各原型预计数量、总数量和是否为 Boss 波。
func PreviewWave(wave, maxWaves, enemiesPerWave int, archetypes map[string]*SpawnConfig) ([]WavePreviewEntry, int, bool) {
	if wave < 1 || wave > maxWaves {
		return nil, 0, false
	}

	sc := config.GlobalSpawnerConfig()
	count := enemiesPerWave + wave
	isBoss := wave%sc.Boss.EveryNWaves == 0
	if isBoss {
		count++
	}
	totalCount := count

	comp := getComposition(wave)

	totalWeight := 0
	for _, e := range comp {
		totalWeight += e.weight
	}
	if totalWeight == 0 {
		return nil, totalCount, isBoss
	}

	normalCount := count
	if isBoss {
		normalCount--
	}
	var entries []WavePreviewEntry
	for _, e := range comp {
		n := normalCount * e.weight / totalWeight
		if n <= 0 {
			n = 1
		}
		label := e.archetype
		if archetypes != nil {
			if cfg, ok := archetypes[e.archetype]; ok && cfg != nil && cfg.Label != "" {
				label = cfg.Label
			}
		}
		entries = append(entries, WavePreviewEntry{
			Archetype: e.archetype,
			Label:     label,
			Count:     n,
		})
	}
	return entries, totalCount, isBoss
}

// NextWavePreview 返回下一波的预览信息。
func (s *Spawner) NextWavePreview() (entries []WavePreviewEntry, totalCount int, isBoss bool) {
	nextWave := s.Wave + 1
	if nextWave > s.MaxWaves {
		return nil, 0, false
	}

	// 计算下一波敌人数
	count := s.EnemiesPerWave + nextWave
	if s.FixedCount > 0 {
		count = s.FixedCount
	}
	isBoss = s.BossEveryWave || nextWave%config.GlobalSpawnerConfig().Boss.EveryNWaves == 0
	if isBoss {
		count++ // Boss 额外一个
	}
	totalCount = count

	// 获取原型权重分布
	comp := getComposition(nextWave)

	// 计算各原型大概数量
	totalWeight := 0
	for _, e := range comp {
		totalWeight += e.weight
	}
	if totalWeight == 0 {
		return nil, totalCount, isBoss
	}

	normalCount := count
	if isBoss {
		normalCount-- // Boss 不算在原型分配中
	}
	for _, e := range comp {
		n := normalCount * e.weight / totalWeight
		if n <= 0 {
			n = 1
		}
		label := e.archetype
		if cfg := s.getConfig(e.archetype); cfg != nil && cfg.Label != "" {
			label = cfg.Label
		}
		entries = append(entries, WavePreviewEntry{
			Archetype: e.archetype,
			Label:     label,
			Count:     n,
		})
	}
	return entries, totalCount, isBoss
}

// enemyCount 返回当前波的常规敌人总数（不含 Boss）。
func (s *Spawner) enemyCount() int {
	if s.FixedCount > 0 {
		return s.FixedCount
	}
	return s.EnemiesPerWave + s.Wave
}

// EnemyCountForWave 返回指定波次的敌人数量（契约测试用）。
func (s *Spawner) EnemyCountForWave(wave int) int {
	saved := s.Wave
	s.Wave = wave
	count := s.enemyCount()
	s.Wave = saved
	return count
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
	if s.EnemyFilter == "" || s.EnemyFilter == FilterMixed {
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

	switch s.EnemyFilter {
	case FilterGroundOnly:
		return !isBoss
	case FilterFlyingOnly:
		return false // flying enemies removed
	case FilterBossOnly:
		return isBoss
	case FilterDummy:
		return name == "dummy"
	case FilterStress:
		return !isBoss
	case FilterNone:
		return false // no spawning
	case FilterAllStatic:
		return true // all types (used by spawnAllStatic, not by wave spawning)
	case FilterMixed, "":
		return !isBoss // default: exclude boss (added separately)
	default:
		// 精确原型匹配（如 EnemyFilter="armored" 只出 armored 怪）
		return name == s.EnemyFilter
	}
}

// compositionForWave 返回当前波次对应的原型权重表。
func (s *Spawner) compositionForWave() []waveEntry {
	return getComposition(s.Wave)
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

// ── 波次 buff 自动注入 ──
// 按波次段从 buff 模板池中随机注入 0~2 个 buff。
// 1-5 波：无 buff
// 6-15 波：最多 1 个（基础池）
// 16-25 波：最多 2 个（扩展池）
// 26+ 波：最多 2 个（完整池）

// applyWaveBuffs 根据当前波次为刚生成的敌人随机注入 buff。
// Boss 不注入波次 buff。tier/pool/chance 从 spawner.json waveBuffs 区段读取。
func (s *Spawner) applyWaveBuffs(e *Enemy) {
	if e.Boss {
		return
	}

	wb := config.GlobalSpawnerConfig().WaveBuffs
	var buffPool []string
	maxBuffs := 0
	for _, tier := range wb.Tiers {
		if s.Wave >= tier.MinWave {
			buffPool = tier.Pool
			maxBuffs = tier.MaxBuffs
		}
	}
	if maxBuffs <= 0 || len(buffPool) == 0 {
		return
	}

	// buffChance 概率注入 buff（不是每个敌人都有）
	if rand.Float64() > wb.Chance {
		return
	}

	// 随机选 1~maxBuffs 个不重复 buff
	count := 1 + rand.Intn(maxBuffs)
	if count > len(buffPool) {
		count = len(buffPool)
	}
	perm := rand.Perm(len(buffPool))
	for i := 0; i < count; i++ {
		applyWaveBuff(e, buffPool[perm[i]])
	}
}

// applyWaveBuff 根据 buffID 从 enemies/abilities.json 读取能力参数，注入对应 buff。
func applyWaveBuff(e *Enemy, buffID string) {
	tel.T.Record("enemy_template", buffID)
	table := config.GlobalEnemyAbilityTable()
	if table == nil {
		return
	}
	def := table[buffID]
	if def == nil {
		return
	}
	switch buffID {
	case "berserk":
		// base=threshold(0.5), param=speedScale(1.5)
		e.Buffs.Add(buff.Buff{
			ID: buff.IDBerserk, Category: buff.CatBehavior, Source: "wave_buff",
			Value: def.Param, Value2: def.Base,
			Duration: -1, Remaining: -1,
		})
	case "regen":
		// base=hpRatio(0.02)
		e.Buffs.Add(buff.Buff{
			ID: buff.IDRegen, Category: buff.CatBehavior, Source: "wave_buff",
			Value: e.MaxHP * def.Base, Duration: -1, Remaining: -1,
		})
	case "healAura":
		// base=healPercent(0.05), param=radius(80)
		e.Buffs.Add(buff.Buff{
			ID: buff.IDHealAura, Category: buff.CatBehavior, Source: "wave_buff",
			Value: def.Base, Value2: def.Param,
			Duration: -1, Remaining: -1,
		})
		e.HealInterval = 3 // 固定3s（与 applyEnemyAbilityToSpawnConfig 一致）
		e.HealCooldown = 0
	case "speedAura":
		// base=bonus(0.2), param=radius(80)
		e.Buffs.Add(buff.Buff{
			ID: buff.IDBufferAura, Category: buff.CatBehavior, Source: "wave_buff",
			Value: def.Base, Value2: def.Param,
			Duration: -1, Remaining: -1,
		})
	case "damageReduce":
		// base=ratio(0.3)
		e.Buffs.Add(buff.Buff{
			ID: buff.IDDamageReduce, Category: buff.CatDefense, Source: "wave_buff",
			Value: def.Base, Duration: -1, Remaining: -1,
		})
	case "deathSplit":
		// base=count(2), param=hpRatio(0.3)
		if e.SplitCount <= 0 {
			e.SplitCount = int(def.Base)
		}
		if e.SplitHPRatio <= 0 {
			e.SplitHPRatio = def.Param
		}
		if e.SplitSpeedScale <= 0 {
			e.SplitSpeedScale = 1.4
		}
	}
}

// IsBossWave 返回当前波次是否为 Boss 波。
func (s *Spawner) IsBossWave() bool {
	return s.BossEveryWave || (s.Wave > 0 && s.Wave%config.GlobalSpawnerConfig().Boss.EveryNWaves == 0)
}
