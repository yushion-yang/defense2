// recorder.go — JSON 数据收集器。
// 累积每局的统计数据，输出结构化 JSON 报告。
package autoplay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"defense2/internal/core/telemetry"
)

// TowerStat 塔使用统计。
type TowerStat struct {
	Key       string  `json:"key"`
	Count     int     `json:"count"`
	TotalCost int     `json:"total_cost"`
	TotalDPS  float64 `json:"total_dps"`
}

// WaveEntry 单波统计。
type WaveEntry struct {
	Wave       int `json:"wave"`
	Enemies    int `json:"enemies"`
	Leaked     int `json:"leaked"`
	GoldEarned int `json:"gold_earned"`
	GoldSpent  int `json:"gold_spent"`
}

// BossStat Boss 存活时间统计。
type BossStat struct {
	Archetype    string  `json:"archetype"`
	Wave         int     `json:"wave"`
	SpawnTick    int     `json:"spawn_tick"`
	DeathTick    int     `json:"death_tick"`    // 0 = 存活到结算
	AliveSeconds float64 `json:"alive_seconds"` // 存活时间（秒）
	Verdict      string  `json:"verdict"`       // "too_weak"(<5s) / "ok" / "too_strong"(>60s) / "survived"
}

// PaceStat 节奏指标统计。
type PaceStat struct {
	TotalIdleTicks   int     `json:"total_idle_ticks"`   // 无波次活跃的帧数
	TotalCombatTicks int     `json:"total_combat_ticks"` // 波次活跃的帧数
	IdleCombatRatio  float64 `json:"idle_combat_ratio"`  // 空闲/战斗比（>3 太无聊, <0.3 太紧张）
	Verdict          string  `json:"verdict"`            // "too_boring" / "ok" / "too_intense"
}

// ExperienceStat 体验指标（对应手动测试 §八"感觉"测试）。
type ExperienceStat struct {
	EarlyLeakWave      int             `json:"early_leak_wave"`                // 首次泄漏波次（0=从未泄漏）
	EarlyLeakCount     int             `json:"early_leak_count"`               // 前 3 波泄漏总数
	LateZeroLeakWaves  int             `json:"late_zero_leak_waves"`           // 最后 3 波连续 0 泄漏的波数
	SpecialEnemyImpact []SpecialImpact `json:"special_enemy_impact,omitempty"` // 特殊怪影响力
	DifficultyVerdict  string          `json:"difficulty_verdict"`             // too_easy / ok / too_hard / crushing
}

// SpecialImpact 特殊敌人的影响力度量。
type SpecialImpact struct {
	Archetype   string  `json:"archetype"`
	AvgSurvival float64 `json:"avg_survival_ticks"` // 平均存活 tick
	NormalAvg   float64 `json:"normal_avg_ticks"`   // 同期 normal 的平均存活 tick
	ImpactRatio float64 `json:"impact_ratio"`       // 存活比（>1.5 有影响, <1.1 形同虚设）
}

// CoverageData 覆盖率追踪数据。
type CoverageData struct {
	TowersUsed          []string `json:"towers_used"`
	AbilitiesTriggered  []string `json:"abilities_triggered,omitempty"`
	AttackStylesFired   []string `json:"attack_styles_fired,omitempty"`
	EnemyArchetypesSeen []string `json:"enemy_archetypes_seen"`
	InteractionModes    []string `json:"interaction_modes_entered,omitempty"`
}

// SessionRecord 完整的对局报告。
type SessionRecord struct {
	SessionID     string       `json:"session_id"`
	Strategy      string       `json:"strategy"`
	MapID         string       `json:"map_id"`
	Difficulty    string       `json:"difficulty"`
	Warden        string       `json:"warden"`
	Seed          int64        `json:"seed"` // 随机种子（用于复现）
	Result        string       `json:"result"`
	WavesSurvived int          `json:"waves_survived"`
	TotalWaves    int          `json:"total_waves"`
	FinalGold     int          `json:"final_gold"`
	FinalLives    int          `json:"final_lives"`
	TotalKills    int          `json:"total_kills"`
	DurationTicks int          `json:"duration_ticks"`
	TowersBuilt   []TowerStat  `json:"towers_built"`
	WaveLog       []WaveEntry  `json:"wave_log"`
	Anomalies     []Anomaly    `json:"anomalies"`
	Coverage      CoverageData `json:"coverage"`
	DPSSnapshots  []float64    `json:"dps_snapshots,omitempty"`

	// 节奏与平衡指标
	BossStats       []BossStat      `json:"boss_stats,omitempty"`
	PaceStats       *PaceStat       `json:"pace_stats,omitempty"`
	EconomyAlerts   []string        `json:"economy_alerts,omitempty"`
	ExperienceStats *ExperienceStat `json:"experience_stats,omitempty"` // 体验指标

	// 断言结果（能力测试场景）
	Assertions []AssertionResult `json:"assertions,omitempty"`

	// 遥测覆盖
	PipelineSteps      []string `json:"pipeline_steps,omitempty"`
	DamageTypes        []string `json:"damage_types,omitempty"`
	BuffTypesApplied   []string `json:"buff_types_applied,omitempty"`
	BuffStackModes     []string `json:"buff_stack_modes,omitempty"`
	EnemyBuffTemplates []string `json:"enemy_buff_templates,omitempty"`
	BossSpawned        []string `json:"boss_spawned,omitempty"`
	InteractionModes   []string `json:"interaction_modes,omitempty"`
	CCApplied          []string `json:"cc_applied,omitempty"`
}

// Recorder 对局数据记录器。
type Recorder struct {
	sessionID  string
	strategy   string
	mapID      string
	difficulty string
	warden     string

	// DPS 追踪
	dpsWindow    float64
	dpsTimer     float64
	dpsSnapshots []float64
	peakDPS      float64

	// 波次追踪
	waveLog       []WaveEntry
	currentWave   int
	waveEnemies   int
	waveLeaked    int
	waveGoldStart int
	waveGoldSpent int

	// 覆盖率追踪
	towersUsed       map[string]bool
	archetypesSeen   map[string]bool
	abilitiesSeen    map[string]bool
	attackStylesSeen map[string]bool

	// 累计统计
	totalKills int
	prevGold   int
	prevLives  int

	// Boss 追踪
	activeBosses map[int]BossStat // enemyID -> spawn info
	bossStats    []BossStat

	// 节奏追踪
	idleTicks   int
	combatTicks int

	// 经济断档追踪
	econStallTicks int      // 连续"买不起最便宜塔"的帧数
	econAlerts     []string // 断档事件描述
	minTowerCost   int      // 最便宜的塔价格（首帧缓存）

	// 断言结果（由 Controller 注入）
	Assertions []AssertionResult

	// 体验指标追踪
	firstLeakWave  int              // 首次泄漏波次
	earlyLeaks     int              // 前 3 波泄漏数
	perWaveLeaks   map[int]int      // wave → 泄漏数
	enemySurvival  map[string][]int // archetype → 存活 tick 列表
	enemySpawnTick map[int]int      // enemyID → spawn tick
}

// NewRecorder 创建对局数据记录器。
func NewRecorder(sessionID, strategy, mapID, difficulty, warden string) *Recorder {
	return &Recorder{
		sessionID:        sessionID,
		strategy:         strategy,
		mapID:            mapID,
		difficulty:       difficulty,
		warden:           warden,
		dpsSnapshots:     make([]float64, 0, 128),
		towersUsed:       make(map[string]bool),
		archetypesSeen:   make(map[string]bool),
		abilitiesSeen:    make(map[string]bool),
		attackStylesSeen: make(map[string]bool),
		activeBosses:     make(map[int]BossStat),
		perWaveLeaks:     make(map[int]int),
		enemySurvival:    make(map[string][]int),
		enemySpawnTick:   make(map[int]int),
	}
}

// OnTick 每帧调用，更新统计。
func (r *Recorder) OnTick(state *GameState, gameDT float64) {
	// 追踪可见敌人原型
	for _, e := range state.Enemies {
		if e.Active {
			r.archetypesSeen[e.Archetype] = true
		}
	}

	// 追踪已建塔类型、能力、攻击方式、技能
	for _, t := range state.Towers {
		r.towersUsed[t.Key] = true
		for _, ab := range t.Abilities {
			r.abilitiesSeen[ab] = true
		}
		if t.AttackStyle != "" {
			r.attackStylesSeen[t.AttackStyle] = true
		}
	}

	// 追踪击杀（通过 lives 变化推断泄漏）
	if r.prevLives > 0 && state.Lives < r.prevLives {
		leaked := r.prevLives - state.Lives
		r.waveLeaked += leaked
		// 体验指标：泄漏追踪
		r.perWaveLeaks[state.Wave] += leaked
		if r.firstLeakWave == 0 {
			r.firstLeakWave = state.Wave
		}
		if state.Wave <= 3 {
			r.earlyLeaks += leaked
		}
	}

	// 体验指标：敌人存活追踪
	for _, e := range state.Enemies {
		if !e.Active {
			continue
		}
		if _, tracked := r.enemySpawnTick[e.ID]; !tracked {
			r.enemySpawnTick[e.ID] = state.Tick
		}
	}
	// 检测消失的敌人 → 记录存活时长
	activeIDs := make(map[int]bool)
	for _, e := range state.Enemies {
		if e.Active {
			activeIDs[e.ID] = true
		}
	}
	for id, spawnTick := range r.enemySpawnTick {
		if !activeIDs[id] {
			survived := state.Tick - spawnTick
			// 找回原型（从最近帧的 enemies 列表中）
			arch := "normal"
			for _, e := range state.Enemies {
				if e.ID == id {
					arch = e.Archetype
					break
				}
			}
			r.enemySurvival[arch] = append(r.enemySurvival[arch], survived)
			delete(r.enemySpawnTick, id)
		}
	}

	// ── Boss 存活追踪 ──
	r.trackBosses(state)

	// ── 节奏追踪 ──
	if state.WaveActive {
		r.combatTicks++
	} else if state.Wave > 0 { // 游戏已开始但当前无波
		r.idleTicks++
	}

	// ── 经济断档追踪 ──
	r.trackEconomyStall(state)

	r.prevGold = state.Gold
	r.prevLives = state.Lives
}

// trackBosses 追踪 Boss 出生和死亡。
func (r *Recorder) trackBosses(state *GameState) {
	activeIDs := make(map[int]bool)
	for _, e := range state.Enemies {
		if !e.Active {
			continue
		}
		if e.Boss {
			activeIDs[e.ID] = true
			if _, tracked := r.activeBosses[e.ID]; !tracked {
				r.activeBosses[e.ID] = BossStat{
					Archetype: e.Archetype,
					Wave:      state.Wave,
					SpawnTick: state.Tick,
				}
			}
		}
	}
	// 检测已死亡/消失的 Boss
	for id, bs := range r.activeBosses {
		if !activeIDs[id] {
			bs.DeathTick = state.Tick
			bs.AliveSeconds = float64(bs.DeathTick-bs.SpawnTick) / 60.0
			switch {
			case bs.AliveSeconds < 5:
				bs.Verdict = "too_weak"
			case bs.AliveSeconds > 60:
				bs.Verdict = "too_strong"
			default:
				bs.Verdict = "ok"
			}
			r.bossStats = append(r.bossStats, bs)
			delete(r.activeBosses, id)
		}
	}
}

// trackEconomyStall 检测经济断档（连续 15s 买不起最便宜的塔）。
func (r *Recorder) trackEconomyStall(state *GameState) {
	// 缓存最便宜塔价格
	if r.minTowerCost == 0 && len(state.TowerDefs) > 0 {
		r.minTowerCost = state.TowerDefs[0].Cost
		for _, d := range state.TowerDefs[1:] {
			if d.Cost < r.minTowerCost {
				r.minTowerCost = d.Cost
			}
		}
	}
	if r.minTowerCost == 0 {
		return
	}

	// 还有空位且买不起最便宜的塔
	if len(state.BuildCells) > 0 && state.Gold < r.minTowerCost {
		r.econStallTicks++
		if r.econStallTicks == 900 { // 15s @60fps
			r.econAlerts = append(r.econAlerts, fmt.Sprintf(
				"wave=%d tick=%d: 连续 15s 金币(%d)不够最便宜的塔(%d)",
				state.Wave, state.Tick, state.Gold, r.minTowerCost))
		}
	} else {
		r.econStallTicks = 0
	}
}

// RecordDamage 记录一次伤害（用于 DPS 计算）。
func (r *Recorder) RecordDamage(amount float64) {
	r.dpsWindow += amount
}

// TickDPS 驱动 DPS 采样（每秒一次）。
func (r *Recorder) TickDPS(dt float64) {
	r.dpsTimer += dt
	if r.dpsTimer >= 1.0 {
		dps := r.dpsWindow / r.dpsTimer
		r.dpsSnapshots = append(r.dpsSnapshots, dps)
		if dps > r.peakDPS {
			r.peakDPS = dps
		}
		r.dpsWindow = 0
		r.dpsTimer = 0
	}
}

// OnWaveStart 波次开始时调用。
func (r *Recorder) OnWaveStart(wave int, state *GameState) {
	r.currentWave = wave
	r.waveEnemies = len(state.Enemies)
	r.waveLeaked = 0
	r.waveGoldStart = state.Gold
	r.waveGoldSpent = 0
}

// OnWaveEnd 波次结束时调用。
func (r *Recorder) OnWaveEnd(wave int, state *GameState) {
	goldEarned := state.Gold - r.waveGoldStart + r.waveGoldSpent
	if goldEarned < 0 {
		goldEarned = 0
	}
	r.waveLog = append(r.waveLog, WaveEntry{
		Wave:       wave,
		Enemies:    r.waveEnemies,
		Leaked:     r.waveLeaked,
		GoldEarned: goldEarned,
		GoldSpent:  r.waveGoldSpent,
	})
}

// OnTowerBuilt 记录建塔花费（供波次统计）。
func (r *Recorder) OnTowerBuilt(cost int) {
	r.waveGoldSpent += cost
}

// OnKill 记录击杀。
func (r *Recorder) OnKill() {
	r.totalKills++
}

// Finalize 生成最终对局报告。
func (r *Recorder) Finalize(state *GameState, anomalies []Anomaly) *SessionRecord {
	result := "timeout"
	if state.Victory {
		result = "victory"
	} else if state.GameOver {
		result = "defeat"
	}

	// 塔统计
	towerCounts := make(map[string]*TowerStat)
	for _, t := range state.Towers {
		ts, ok := towerCounts[t.Key]
		if !ok {
			ts = &TowerStat{Key: t.Key}
			towerCounts[t.Key] = ts
		}
		ts.Count++
		ts.TotalCost += t.Cost
		ts.TotalDPS += t.Damage // 近似
	}
	var towerStats []TowerStat
	for _, ts := range towerCounts {
		towerStats = append(towerStats, *ts)
	}

	// 覆盖率
	coverage := CoverageData{
		TowersUsed:          mapKeys(r.towersUsed),
		EnemyArchetypesSeen: mapKeys(r.archetypesSeen),
		AbilitiesTriggered:  mapKeys(r.abilitiesSeen),
		AttackStylesFired:   mapKeys(r.attackStylesSeen),
	}

	rec := &SessionRecord{
		SessionID:     r.sessionID,
		Strategy:      r.strategy,
		MapID:         r.mapID,
		Difficulty:    r.difficulty,
		Warden:        r.warden,
		Result:        result,
		WavesSurvived: state.Wave,
		TotalWaves:    state.MaxWaves,
		FinalGold:     state.Gold,
		FinalLives:    state.Lives,
		TotalKills:    r.totalKills,
		DurationTicks: state.Tick,
		TowersBuilt:   towerStats,
		WaveLog:       r.waveLog,
		Anomalies:     anomalies,
		Coverage:      coverage,
		DPSSnapshots:  r.dpsSnapshots,
	}

	// Boss 存活统计（含未死亡的 Boss 标记为 survived）
	for _, bs := range r.activeBosses {
		bs.AliveSeconds = float64(state.Tick-bs.SpawnTick) / 60.0
		bs.Verdict = "survived"
		r.bossStats = append(r.bossStats, bs)
	}
	if len(r.bossStats) > 0 {
		rec.BossStats = r.bossStats
	}

	// 节奏指标
	if r.combatTicks > 0 {
		ratio := float64(r.idleTicks) / float64(r.combatTicks)
		verdict := "ok"
		if ratio > 3.0 {
			verdict = "too_boring"
		} else if ratio < 0.3 {
			verdict = "too_intense"
		}
		rec.PaceStats = &PaceStat{
			TotalIdleTicks:   r.idleTicks,
			TotalCombatTicks: r.combatTicks,
			IdleCombatRatio:  ratio,
			Verdict:          verdict,
		}
	}

	// 经济断档事件
	if len(r.econAlerts) > 0 {
		rec.EconomyAlerts = r.econAlerts
	}

	// 体验指标
	exp := &ExperienceStat{
		EarlyLeakWave:  r.firstLeakWave,
		EarlyLeakCount: r.earlyLeaks,
	}
	// 最后 3 波 0 泄漏检测
	if state.Wave >= 3 {
		zeroCount := 0
		for w := state.Wave; w > state.Wave-3 && w > 0; w-- {
			if r.perWaveLeaks[w] == 0 {
				zeroCount++
			}
		}
		exp.LateZeroLeakWaves = zeroCount
	}
	// 特殊怪影响力
	normalAvg := avgTicks(r.enemySurvival["normal"])
	specials := []string{"runner", "tank", "armored", "stealth", "splitter", "teleporter", "healer", "buffer", "flying", "swarm"}
	for _, arch := range specials {
		ticks := r.enemySurvival[arch]
		if len(ticks) == 0 {
			continue
		}
		avg := avgTicks(ticks)
		ratio := 0.0
		if normalAvg > 0 {
			ratio = avg / normalAvg
		}
		exp.SpecialEnemyImpact = append(exp.SpecialEnemyImpact, SpecialImpact{
			Archetype:   arch,
			AvgSurvival: avg,
			NormalAvg:   normalAvg,
			ImpactRatio: ratio,
		})
	}
	// 难度判定
	switch {
	case state.Lives == state.MaxWaves && r.firstLeakWave == 0:
		exp.DifficultyVerdict = "too_easy" // 全程无泄漏
	case r.earlyLeaks > 3:
		exp.DifficultyVerdict = "crushing" // 前 3 波就漏 3+
	case r.firstLeakWave > 0 && r.firstLeakWave <= 3:
		exp.DifficultyVerdict = "too_hard" // 前 3 波就开始漏
	default:
		exp.DifficultyVerdict = "ok"
	}
	rec.ExperienceStats = exp

	// 遥测数据
	tel := state.Telemetry
	rec.PipelineSteps = telemetry.Keys(tel.PipelineSteps)
	rec.DamageTypes = telemetry.Keys(tel.DamageTypes)
	rec.BuffTypesApplied = telemetry.Keys(tel.BuffTypesApplied)
	rec.BuffStackModes = telemetry.Keys(tel.BuffStackModes)
	rec.EnemyBuffTemplates = telemetry.Keys(tel.EnemyBuffTemplates)
	rec.BossSpawned = telemetry.Keys(tel.BossSpawned)
	rec.InteractionModes = telemetry.Keys(tel.InteractionModes)
	rec.CCApplied = telemetry.Keys(tel.CCApplied)

	// 断言结果
	if len(r.Assertions) > 0 {
		rec.Assertions = r.Assertions
	}

	return rec
}

// WriteJSON 将对局报告写入 JSON 文件。
func WriteJSON(record *SessionRecord, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "report.json")
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func avgTicks(ticks []int) float64 {
	if len(ticks) == 0 {
		return 0
	}
	sum := 0
	for _, t := range ticks {
		sum += t
	}
	return float64(sum) / float64(len(ticks))
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
