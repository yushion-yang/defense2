// spawner.go — 波次出怪管理器（753 行，本项目最复杂的单文件）。
//
// ═══════════════════════════════════════════════════════════════════
// 架构概览
// ═══════════════════════════════════════════════════════════════════
//
// Spawner 是敌人生成的唯一控制器，管理以下子系统：
//
//  1. 【波次状态机】
//     idle(波间等待) → waitingForClear(等待清场) → waveActive(出怪中) → idle...
//     - 等待清场：上波敌人全灭后才开始倒计时（60s 保底超时）
//     - 倒计时到 0 后自动开波（ManualWave 模式需外部调用 StartNextWave）
//     - AllDone=true 表示所有波次出完
//
// 2. 【原型选择】
//   - 波次组合表(wave-compositions.json)：按波次段配置各原型权重
//   - 小队模板(spawner.json squads)：每波开始时随机选一个小队，前 N 个敌人按模板出
//   - EnemyFilter：测试/调试用，限制只出特定类型
//   - 原型排序：所有随机选择前先排序，消除 map 迭代不确定性
//
// 3. 【Boss 注入】
//   - 每 N 波（Boss.EveryNWaves）末尾追加一个 Boss
//   - Boss = 随机原型 + HpMultBase 倍率 + RadiusScale 倍率 + 强制净化能力
//   - Boss 波有入场延迟（EntranceDelay），给玩家准备时间
//
// 4. 【数值缩放】
//   - baseHP = HpBase + wave * HpPerWave + HpQuadratic * wave²（二次增长，后期上翘）
//   - baseSpeed = SpeedBase + wave * SpeedPerWave
//   - 出怪间隔逐波衰减：EffectiveSpawnInterval(wave)
//   - 每波敌人数 = EnemiesPerWave + wave
//   - HPScale/SpeedScale 是难度倍率（外部注入）
//
// 5. 【Wave Buff 注入】
//   - 按波次段从 buff 模板池中随机注入 0~2 个 buff
//   - 1-5 波无 buff → 6-15 波最多 1 个 → 16-25 波最多 2 个 → 26+ 波最多 2 个
//   - Boss 不注入波次 buff
//   - buff 参数从 abilities.json 读取，支持 potential 波次缩放
//
// 6. 【能力 Potential 缩放】
//   - ApplyAbilityPotentials: delta = potential * wave，叠加到 spawn 时的 base 值
//   - 部分能力不缩放（phaseShift/purge/ccImmune 等）
//
// 配置文件依赖：
//   - config/enemies/spawner.json: 数值缩放/时机/Boss/小队/waveBuff 配置
//   - config/enemies/wave-compositions.json: 波次原型权重表
//   - config/enemies/abilities.json: 敌人能力参数（含 potential）
//   - config/enemies/enemies-core.json: 原型定义（由 stage 层加载后注入 Archetypes）
//
// 关联文件：
//   - pool.go: Spawn() 实际创建敌人
//   - spawn_config.go: SpawnConfig 参数定义
//   - ability_type_ids.go: 能力 ID 常量（AbilBerserk 等）
//   - stage.go: 创建 Spawner 并注入 Archetypes/GameMap
package enemy

import (
	"cmp"
	"math/rand"
	"slices"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/gamemap"
	tel "defense2/internal/core/telemetry"
)

// waveEntry 波次组合中一种原型的权重配置。
// 从 wave-compositions.json 加载，用于加权随机选取原型。
type waveEntry struct {
	archetype string // 原型标识（如 "normal"、"runner"、"tank"）
	weight    int    // 权重值（越大越可能被选中）
}

// getComposition 根据波次号从配置中获取对应的原型权重表。
// wave-compositions.json 按 MaxWave 分段，找到第一个 MaxWave>=wave 的段（MaxWave=0 表示无上限）。
// 返回值排序后可稳定用于加权随机（消除 map 迭代不确定性）。
func getComposition(wave int) []waveEntry {
	comps := config.GlobalWaveCompositions()
	var enemies map[string]int
	for _, c := range comps {
		if c.MaxWave == 0 || wave <= c.MaxWave {
			enemies = c.Enemies
			break
		}
	}
	if enemies == nil && len(comps) > 0 {
		enemies = comps[len(comps)-1].Enemies
	}
	if len(enemies) == 0 {
		return []waveEntry{{"normal", 100}}
	}
	entries := make([]waveEntry, 0, len(enemies))
	for arch, w := range enemies {
		entries = append(entries, waveEntry{archetype: arch, weight: w})
	}
	// 按原型名排序，确保 map 迭代的不确定顺序不影响后续计算
	slices.SortFunc(entries, func(a, b waveEntry) int {
		return cmp.Compare(a.archetype, b.archetype)
	})
	return entries
}

// Spawner 波次出怪控制器。
//
// 由 stage.go 在关卡开始时创建，通过 Tick() 每帧驱动。
// 外部需注入 Archetypes（原型配置表）和 GameMap（提供路径信息）。
type Spawner struct {
	// ── 波次进度 ──
	Wave     int  // 当前波次号（从 1 开始，startWave 时递增）
	MaxWaves int  // 总波次数（由关卡配置决定）
	AllDone  bool // 是否所有波次已出完（Wave >= MaxWaves 且最后一波出完）

	// ── 波内出怪控制 ──
	SpawnTimer     float64 // 单波内两个敌人之间的倒计时（秒，到 0 生成下一个）
	SpawnIndex     int     // 当前波已出第几个敌人（0-based）
	EnemiesPerWave int     // 每波基础敌人数（实际数 = EnemiesPerWave + Wave）
	SpawnInterval  float64 // 同波内敌人生成间隔（秒，逐波衰减）
	WaveActive     bool    // 当前波是否正在出怪

	// ── 波间等待 ──
	WaveInterval      float64 // 两波之间的间隔（秒，从 spawner.json 读取）
	WaveTimer         float64 // 波间等待倒计时（秒，UI 显示用，到 0 自动开波）
	FirstWaveInterval float64 // 第一波等待时间（秒），默认 20
	WaitingForClear   bool    // 出怪完毕，等待场上敌人全灭后才开始倒计时
	clearWaitElapsed  float64 // 等待清场已过秒数（保底 60s 超时防卡死）

	// ── 地图与原型 ──
	GameMap    *gamemap.GameMap        // 运行时地图（PickPath() 获取路径）
	Archetypes map[string]*SpawnConfig // 原型名 → 生成配置（由 stage 层从 enemies-core.json 加载后注入）

	// ── 难度与模式 ──
	EnemyFilter   string  // 敌人过滤器（ground-only/boss-only/dummy/stress/none/mixed/""）
	HPScale       float64 // 难度 HP 倍率（默认 1.0，由 gamemode 设置）
	SpeedScale    float64 // 难度速度倍率（默认 1.0）
	ManualWave    bool    // 手动开波模式：倒计时到 0 不自动开波，需外部调用 StartNextWave
	FixedCount    int     // >0 时每波固定该数量（不随波次递增，测试模式用）
	BossEveryWave bool    // true 时每波末尾都出 Boss（bossRush 模式用）

	// ── Boss 控制 ──
	bossQueued    bool    // 本波是否需要在末尾追加 Boss（startWave 时计算）
	EntranceDelay float64 // Boss 波入场延迟（秒），>0 时暂停出怪给玩家准备

	// ── 小队系统 ──
	squadMembers []string // 本波小队成员原型列表（startWave 时从模板池随机选取）
	// 出怪逻辑：SpawnIndex < len(squadMembers) 时按列表顺序出，之后转加权随机
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

// Tick 每帧调用，驱动波次状态机。
//
// 状态机流程：
//  1. AllDone / FilterNone → 直接返回
//  2. 波间等待 (!WaveActive):
//     a. WaitingForClear → 等敌人全灭或超时 60s → 进入倒计时
//     b. 倒计时递减 → 到 0 → startWave()（ManualWave 模式除外）
//  3. Boss 入场延迟：EntranceDelay > 0 → 倒计时后再出怪
//  4. 波内出怪：SpawnTimer 递减 → 到 0 → 生成一个敌人
//     a. Boss 波末尾追加 Boss（随机原型 + 倍率 + 净化）
//     b. 前 N 个按小队模板出，之后加权随机
//     c. 生成后应用 AbilityPotentials + WaveBuffs
//  5. 当前波出完 → WaveActive=false, 进入 WaitingForClear
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
		// 等待清场阶段：出怪完毕但场上还有敌人
		if s.WaitingForClear {
			s.clearWaitElapsed += dt
			// 场上敌人全部清理 OR 保底 60 秒后 → 进入倒计时
			if pool.Count == 0 || s.clearWaitElapsed >= 60 {
				s.WaitingForClear = false
				s.WaveTimer = s.WaveInterval
			}
			return
		}
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

	// ── 波内出怪循环 ──
	// SpawnTimer 控制生成节奏，每到 0 生成一个敌人
	s.SpawnTimer -= dt
	total := s.enemyCount()
	if s.bossQueued {
		total++ // Boss 额外占一个名额，排在最后
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
			w := float64(s.Wave)
			baseHP := (sc.Scaling.HpBase + w*sc.Scaling.HpPerWave + sc.Scaling.HpQuadratic*w*w) * hpScale
			baseSpeed := (sc.Scaling.SpeedBase + float64(s.Wave)*sc.Scaling.SpeedPerWave) * spdScale

			// ── 原型选择：三层优先级 ──
			// 1) Boss：本波最后一个，随机原型 + Boss 增强
			// 2) 小队：SpawnIndex < len(squadMembers)，按模板顺序
			// 3) 随机：加权随机从波次组合表选取
			var archetype string
			var cfg *SpawnConfig

			if s.bossQueued && s.SpawnIndex == total-1 {
				// 波末 Boss：随机原型 + Boss 标记 + 额外倍率 + 必带净化
				archetype = s.pickRandomArchetype()
				cfg = s.getConfig(archetype)
				if cfg != nil {
					bossCfg := *cfg
					bossCfg.Boss = true
					bossCfg.HpScale *= sc.Boss.HpMultBase
					bossCfg.Radius *= sc.Boss.RadiusScale
					// Boss 必带净化能力（参数从 abilities.json purge 条目读取）
					if bossCfg.PurgeInterval <= 0 {
						if aDef := config.GlobalEnemyAbilityDef(AbilPurge); aDef != nil {
							bossCfg.PurgeInterval = aDef.Base
							bossCfg.PurgeImmuneDur = aDef.Param
						} else {
							bossCfg.PurgeInterval = 4.0
							bossCfg.PurgeImmuneDur = 2.0
						}
					}
					if !containsStr(bossCfg.AbilityIDs, AbilPurge) {
						bossCfg.AbilityIDs = append(append([]string{}, bossCfg.AbilityIDs...), AbilPurge)
					}
					cfg = &bossCfg
					tel.T.Record("boss", archetype)
				}
			} else if s.SpawnIndex < len(s.squadMembers) {
				// 小队成员优先出场
				archetype = s.squadMembers[s.SpawnIndex]
				cfg = s.getConfig(archetype)
			} else {
				archetype = s.pickArchetype()
				cfg = s.getConfig(archetype)
			}

			e := pool.Spawn(spawn.X, spawn.Y, baseHP, baseSpeed, 1, archetype, cfg)
			if e != nil {
				e.Path = path
				// 能力 potential 按波次叠加
				ApplyAbilityPotentials(e, cfg, s.Wave)
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
			// 进入等待清场状态，场上敌人全灭后才开始倒计时
			s.WaitingForClear = true
			s.clearWaitElapsed = 0
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

// StartNextWave 手动触发下一波（跳过等待清场和倒计时）。
func (s *Spawner) StartNextWave() {
	if s.WaveActive || s.AllDone {
		return
	}
	s.WaitingForClear = false
	s.WaveTimer = 0
	s.startWave()
}

// startWave 内部开始下一波。
// 递增 Wave → 重置出怪索引 → 计算本波出怪间隔 → 决定是否 Boss 波 → 选小队模板。
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
	// 选小队模板
	s.squadMembers = s.pickSquad()
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
	// 按原型名排序，保证每帧渲染顺序一致（map 迭代无序）
	slices.SortFunc(entries, func(a, b WavePreviewEntry) int {
		return cmp.Compare(a.Archetype, b.Archetype)
	})
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

// pickSquad 根据当前波次，从可用小队模板中随机选取一个，返回成员列表。
// 仅选择 minWave <= 当前波次 且 成员原型在 Archetypes 中都存在的模板。
// 若无可用模板或小队系统未启用，返回 nil（全部走随机）。
func (s *Spawner) pickSquad() []string {
	sc := config.GlobalSpawnerConfig()
	if !sc.Squads.Enabled || len(sc.Squads.Templates) == 0 {
		return nil
	}

	var candidates []config.SquadTemplate
	for _, t := range sc.Squads.Templates {
		if t.MinWave > s.Wave {
			continue
		}
		// 检查所有成员原型是否可用
		valid := true
		for _, m := range t.Members {
			if s.getConfig(m) == nil {
				valid = false
				break
			}
		}
		if valid {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	chosen := candidates[rand.Intn(len(candidates))]
	tel.T.Record("squad", chosen.ID)
	// 复制一份成员列表避免修改原始配置
	members := make([]string, len(chosen.Members))
	copy(members, chosen.Members)
	return members
}

// pickRandomArchetype 从所有可用原型中随机选取一个（用于 Boss）。
// 排除 dummy 原型。若无可用原型则回退到 "normal"。
func (s *Spawner) pickRandomArchetype() string {
	if s.Archetypes == nil {
		return "normal"
	}
	var candidates []string
	for name := range s.Archetypes {
		if name == "dummy" {
			continue
		}
		candidates = append(candidates, name)
	}
	if len(candidates) == 0 {
		return "normal"
	}
	slices.Sort(candidates) // 保证稳定顺序
	return candidates[rand.Intn(len(candidates))]
}

// containsStr 检查切片中是否包含指定字符串。
func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
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
		applyWaveBuff(e, buffPool[perm[i]], s.Wave)
	}
}

// applyWaveBuff 根据 buffID 从 enemies/abilities.json 读取能力参数，注入对应 buff。
// wave 用于应用 potential 缩放：effectiveBase = base + potential * wave。
//
// 各 buff 类型的参数映射：
//   - berserk: Value=speedScale(Param), Value2=threshold(effectiveBase)
//   - regen: Value=MaxHP*effectiveBase（绝对回血量/秒）
//   - healAura: Value=effectiveBase(治疗力), Value2=Param(范围)
//   - speedAura: Value=effectiveBase(速度加成), Value2=Param(范围)
//   - damageReduce: Value=effectiveBase(减伤比例)
//   - deathSplit: 直接设置 SplitCount/SplitHPRatio/SplitSpeedScale
func applyWaveBuff(e *Enemy, buffID string, wave int) {
	tel.T.Record("enemy_template", buffID)
	table := config.GlobalEnemyAbilityTable()
	if table == nil {
		return
	}
	def := table[buffID]
	if def == nil {
		return
	}
	// 计算波次缩放后的 effectiveBase
	effectiveBase := def.Base
	if wave > 0 && def.Potential != 0 {
		effectiveBase += def.Potential * float64(wave)
	}
	switch buffID {
	case AbilBerserk:
		e.Buffs.Add(buff.Buff{
			ID: buff.IDBerserk, Category: buff.CatBehavior, Source: "wave_buff",
			Value: def.Param, Value2: effectiveBase,
			Duration: -1, Remaining: -1,
		})
	case AbilRegen:
		e.Buffs.Add(buff.Buff{
			ID: buff.IDRegen, Category: buff.CatBehavior, Source: "wave_buff",
			Value: e.MaxHP * effectiveBase, Duration: -1, Remaining: -1,
		})
	case AbilHealAura:
		e.Buffs.Add(buff.Buff{
			ID: buff.IDHealAura, Category: buff.CatBehavior, Source: "wave_buff",
			Value: effectiveBase, Value2: def.Param,
			Duration: -1, Remaining: -1,
		})
		e.HealInterval = def.Param2 // param2=治疗间隔(3s)
		if e.HealInterval <= 0 {
			e.HealInterval = 3
		}
		e.HealCooldown = 0
	case AbilSpeedAura:
		e.Buffs.Add(buff.Buff{
			ID: buff.IDBufferAura, Category: buff.CatBehavior, Source: "wave_buff",
			Value: effectiveBase, Value2: def.Param,
			Duration: -1, Remaining: -1,
		})
	case AbilDamageReduce:
		e.Buffs.Add(buff.Buff{
			ID: buff.IDDamageReduce, Category: buff.CatDefense, Source: "wave_buff",
			Value: effectiveBase, Duration: -1, Remaining: -1,
		})
	case AbilDeathSplit:
		if e.SplitCount <= 0 {
			e.SplitCount = int(effectiveBase)
		}
		if e.SplitHPRatio <= 0 {
			e.SplitHPRatio = def.Param
		}
		if e.SplitSpeedScale <= 0 {
			e.SplitSpeedScale = def.Param2 // param2=子体速度倍率
			if e.SplitSpeedScale <= 0 {
				e.SplitSpeedScale = config.GlobalBalance().Split.SpeedScale
			}
		}
	}
}

// ApplyAbilityPotentials 根据波次为原型能力叠加 potential 增量。
// delta = potential * wave，叠加到 spawn 时已设置的 base 值之上。
//
// 这是敌人越来越强的核心机制之一：随波次推进，能力数值线性增长。
// 例如 armorPlating potential=0.5，第 20 波时 ArmorFlat 额外增加 10。
//
// 部分能力有上限钳制（evasion/projectileBlock/strDrain ≤ 1.0, damageReduce ≤ 1.0）。
// phaseShift/purge/ccImmune/slowImmune/deathSplit/deathSpawn 不参与缩放。
func ApplyAbilityPotentials(e *Enemy, cfg *SpawnConfig, wave int) {
	if cfg == nil || wave <= 0 || len(cfg.AbilityPotentials) == 0 {
		return
	}
	for _, ap := range cfg.AbilityPotentials {
		delta := ap.Potential * float64(wave)
		switch ap.Type {
		case "armorPlating":
			e.ArmorFlat += delta
		case "damageCap":
			e.DamageCap += delta
		case "damageCapPercent":
			e.DamageCapPercent += delta
		case "evasion":
			e.EvasionChance += delta
			if e.EvasionChance > 1 {
				e.EvasionChance = 1
			}
		case "projectileBlock":
			e.ProjectileBlockChance += delta
			if e.ProjectileBlockChance > 1 {
				e.ProjectileBlockChance = 1
			}
		case "strengthDrain":
			e.StrDrainRatio += delta
			if e.StrDrainRatio > 1 {
				e.StrDrainRatio = 1
			}
		case "regen":
			// regen base=hpRatio，叠加到 regen buff 的 Value（= MaxHP * ratio）
			if b := e.Buffs.GetPtr(buff.IDRegen); b != nil {
				b.Value += e.MaxHP * delta
			}
		case "healAura":
			if b := e.Buffs.GetPtr(buff.IDHealAura); b != nil {
				b.Value += delta
			}
		case "speedAura":
			if b := e.Buffs.GetPtr(buff.IDBufferAura); b != nil {
				b.Value += delta
			}
		case "damageReduce":
			if b := e.Buffs.GetPtr(buff.IDDamageReduce); b != nil {
				b.Value += delta
				if b.Value > 1 {
					b.Value = 1
				}
			}
		case "dashOnHit":
			e.DashSpeedBoost += delta
		case "berserk":
			if b := e.Buffs.GetPtr(buff.IDBerserk); b != nil {
				b.Value2 += delta // Value2=threshold
			}
		}
		// phaseShift/purge/ccImmune/slowImmune/deathSplit/deathSpawn: 不缩放
	}
}

// IsBossWave 返回当前波次是否为 Boss 波。
func (s *Spawner) IsBossWave() bool {
	return s.BossEveryWave || (s.Wave > 0 && s.Wave%config.GlobalSpawnerConfig().Boss.EveryNWaves == 0)
}
