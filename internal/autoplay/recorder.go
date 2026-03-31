// recorder.go — JSON 数据收集器。
// 累积每局的统计数据，输出结构化 JSON 报告。
package autoplay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// CoverageData 覆盖率追踪数据。
type CoverageData struct {
	TowersUsed          []string `json:"towers_used"`
	AbilitiesTriggered   []string `json:"abilities_triggered,omitempty"`
	AttackStylesFired    []string `json:"attack_styles_fired,omitempty"`
	EnemyArchetypesSeen  []string `json:"enemy_archetypes_seen"`
	EventsChosen         []string `json:"events_chosen,omitempty"`
	InteractionModes     []string `json:"interaction_modes_entered,omitempty"`
	SkillsActivated      []string `json:"skills_activated,omitempty"`
}

// SessionRecord 完整的对局报告。
type SessionRecord struct {
	SessionID     string       `json:"session_id"`
	Strategy      string       `json:"strategy"`
	MapID         string       `json:"map_id"`
	Difficulty    string       `json:"difficulty"`
	Warden        string       `json:"warden"`
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
	Screenshots   []string     `json:"screenshots"`
	Coverage      CoverageData `json:"coverage"`
	DPSSnapshots  []float64    `json:"dps_snapshots,omitempty"`
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
	eventsChosen     map[string]bool
	abilitiesSeen    map[string]bool
	attackStylesSeen map[string]bool
	skillsSeen       map[string]bool

	// 累计统计
	totalKills int
	prevGold   int
	prevLives  int
}

// NewRecorder 创建对局数据记录器。
func NewRecorder(sessionID, strategy, mapID, difficulty, warden string) *Recorder {
	return &Recorder{
		sessionID:      sessionID,
		strategy:       strategy,
		mapID:          mapID,
		difficulty:     difficulty,
		warden:         warden,
		dpsSnapshots:     make([]float64, 0, 128),
		towersUsed:       make(map[string]bool),
		archetypesSeen:   make(map[string]bool),
		eventsChosen:     make(map[string]bool),
		abilitiesSeen:    make(map[string]bool),
		attackStylesSeen: make(map[string]bool),
		skillsSeen:       make(map[string]bool),
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
		if t.SkillName != "" {
			r.skillsSeen[t.SkillName] = true
		}
	}

	// 追踪击杀（通过 lives 变化推断泄漏）
	if r.prevLives > 0 && state.Lives < r.prevLives {
		r.waveLeaked += r.prevLives - state.Lives
	}

	r.prevGold = state.Gold
	r.prevLives = state.Lives
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

// OnEventChosen 记录事件选择。
func (r *Recorder) OnEventChosen(eventKind string) {
	r.eventsChosen[eventKind] = true
}

// Finalize 生成最终对局报告。
func (r *Recorder) Finalize(state *GameState, anomalies []Anomaly, screenshots []string) *SessionRecord {
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
		EventsChosen:        mapKeys(r.eventsChosen),
		AbilitiesTriggered:  mapKeys(r.abilitiesSeen),
		AttackStylesFired:   mapKeys(r.attackStylesSeen),
		SkillsActivated:     mapKeys(r.skillsSeen),
	}

	return &SessionRecord{
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
		Screenshots:   screenshots,
		Coverage:      coverage,
		DPSSnapshots:  r.dpsSnapshots,
	}
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

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
