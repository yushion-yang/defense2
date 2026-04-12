// spawner_config.go — 出怪系统完整配置加载。
// 从 config/systems/spawner.json 读取所有出怪相关参数（数值缩放、时机、Boss、波次组合、波次buff）。
// 此文件是出怪配置的唯一权威源，替代 balance.json 中的 spawner/buffs 区段和 wave-compositions.json。
package config

import (
	"encoding/json"
	"fmt"
)

// SpawnerScaling 敌人数值缩放参数。
type SpawnerScaling struct {
	HpBase            float64 `json:"hpBase"`
	HpPerWave         float64 `json:"hpPerWave"`
	SpeedBase         float64 `json:"speedBase"`
	SpeedPerWave      float64 `json:"speedPerWave"`
	EnemiesPerWave    int     `json:"enemiesPerWave"`
	SpawnInterval     float64 `json:"spawnInterval"`
	SpawnBaseInterval float64 `json:"spawnBaseInterval"`
	SpawnMinInterval  float64 `json:"spawnMinInterval"`
	SpawnDecayPerWave float64 `json:"spawnDecayPerWave"`
}

// SpawnerTiming 波次时机参数。
type SpawnerTiming struct {
	WaveInterval      float64 `json:"waveInterval"`
	FirstWaveInterval float64 `json:"firstWaveInterval"`
}

// SpawnerBoss Boss 完整配置参数（出怪+战斗+动画）。
type SpawnerBoss struct {
	EveryNWaves      int     `json:"everyNWaves"`
	HpMultBase       float64 `json:"hpMultBase"`
	RadiusScale      float64 `json:"radiusScale"`
	EntranceDelay    float64 `json:"entranceDelay"`
	RewardMultiplier float64 `json:"rewardMultiplier"`
	PercentHpCap     float64 `json:"percentHpCap"`  // %HP 伤害上限（原 combat.bossPercentHpCap）
	DyingDuration    float64 `json:"dyingDuration"` // 死亡动画时长（原 dying.bossDuration）
}

// WaveBuffTier 波次 buff 档位。
type WaveBuffTier struct {
	MinWave  int      `json:"minWave"`
	MaxBuffs int      `json:"maxBuffs"`
	Pool     []string `json:"pool"`
}

// WaveBuffConfig 波次 buff 完整配置。
// 各 buff 具体参数从 enemies/abilities.json 读取。
type WaveBuffConfig struct {
	Chance float64        `json:"chance"`
	Tiers  []WaveBuffTier `json:"tiers"`
}

// SquadTemplate 战术小队模板——一组有配合的敌人组合。
type SquadTemplate struct {
	ID      string   `json:"id"`      // 模板唯一标识
	MinWave int      `json:"minWave"` // 最低波次要求
	Members []string `json:"members"` // 成员原型列表（按出场顺序）
}

// SquadsConfig 小队系统配置。
type SquadsConfig struct {
	Enabled   bool            `json:"enabled"`   // 是否启用小队系统
	Templates []SquadTemplate `json:"templates"` // 所有可用小队模板
}

// SpawnerConfig 出怪系统完整配置。
type SpawnerConfig struct {
	Scaling      SpawnerScaling    `json:"scaling"`
	Timing       SpawnerTiming     `json:"timing"`
	Boss         SpawnerBoss       `json:"boss"`
	Compositions []WaveComposition `json:"compositions"`
	WaveBuffs    WaveBuffConfig    `json:"waveBuffs"`
	Squads       SquadsConfig      `json:"squads"`
}

// EffectiveSpawnInterval 返回指定波次的出怪间隔（含衰减）。
// SpawnBaseInterval > 0 时启用衰减系统，否则 fallback 到固定 SpawnInterval。
func (sc *SpawnerConfig) EffectiveSpawnInterval(wave int) float64 {
	s := sc.Scaling
	if s.SpawnBaseInterval > 0 {
		interval := s.SpawnBaseInterval - float64(wave)*s.SpawnDecayPerWave
		if interval < s.SpawnMinInterval {
			interval = s.SpawnMinInterval
		}
		return interval
	}
	return s.SpawnInterval
}

// globalSpawnerConfig 全局缓存。
var globalSpawnerConfig *SpawnerConfig

// GlobalSpawnerConfig 返回全局出怪配置。LoadSpawnerConfig 成功后可用。
func GlobalSpawnerConfig() *SpawnerConfig {
	if globalSpawnerConfig == nil {
		return defaultSpawnerConfig()
	}
	return globalSpawnerConfig
}

// LoadSpawnerConfig 从 config/systems/spawner.json 加载出怪配置。
// 必须在 SetDataFS() 之后调用。
func LoadSpawnerConfig() error {
	if dataFS == nil {
		return fmt.Errorf("load spawner config: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/systems/spawner.json")
	if err != nil {
		return fmt.Errorf("load spawner config: %w", err)
	}

	cfg := defaultSpawnerConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse spawner config: %w", err)
	}
	if len(cfg.Compositions) == 0 {
		return fmt.Errorf("spawner config: empty compositions array")
	}
	globalSpawnerConfig = cfg

	// 同步更新 globalWaveCompositions，保持向后兼容
	globalWaveCompositions = cfg.Compositions

	return nil
}

// defaultSpawnerConfig 返回所有参数的默认值（确保向后兼容）。
func defaultSpawnerConfig() *SpawnerConfig {
	return &SpawnerConfig{
		Scaling: SpawnerScaling{
			HpBase: 52, HpPerWave: 21, SpeedBase: 58, SpeedPerWave: 5,
			EnemiesPerWave: 5, SpawnInterval: 0.6,
			SpawnBaseInterval: 0.92, SpawnMinInterval: 0.18, SpawnDecayPerWave: 0.03,
		},
		Timing: SpawnerTiming{
			WaveInterval: 10, FirstWaveInterval: 20,
		},
		Boss: SpawnerBoss{
			EveryNWaves: 5, HpMultBase: 8, RadiusScale: 1.5,
			EntranceDelay: 3.0, RewardMultiplier: 5,
		},
		Compositions: []WaveComposition{
			{MaxWave: 3, Enemies: map[string]int{"normal": 100}},
		},
		WaveBuffs: WaveBuffConfig{
			Chance: 0.3,
			Tiers: []WaveBuffTier{
				{MinWave: 6, MaxBuffs: 1, Pool: []string{"berserk", "regen", "healAura", "speedAura"}},
				{MinWave: 16, MaxBuffs: 1, Pool: []string{"berserk", "regen", "healAura", "speedAura", "damageReduce"}},
				{MinWave: 26, MaxBuffs: 2, Pool: []string{"berserk", "regen", "healAura", "speedAura", "damageReduce", "deathSplit"}},
			},
		},
	}
}
