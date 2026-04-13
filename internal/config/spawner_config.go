// spawner_config.go — 出怪系统完整配置加载（config/systems/spawner.json）。
//
// 此文件是出怪配置的 **唯一权威源**，涵盖 6 个维度：
//
//	缩放(scaling) / 时机(timing) / Boss / 波次组合(compositions) /
//	波次buff(waveBuffs) / 战术小队(squads)
//
// 历史：原本分散在 balance.json 的 spawner/buffs 区段和独立的 wave-compositions.json，
// 为了消除配置散落导致的一致性问题，统一合并到 spawner.json。
//
// 关键机制：
//   - SpawnInterval 衰减系统：每波出怪间隔递减（SpawnBaseInterval - wave*DecayPerWave），
//     让后期波次节奏加快，SpawnMinInterval 为下限兜底
//   - WaveBuffs 分档：按波次解锁 buff 池（6→16→26 波逐步增加可用 buff 种类和数量）
//   - Compositions 定义各波次的敌人原型权重分布，spawner 按权重随机选取原型
package config

import (
	"encoding/json"
	"fmt"
)

// SpawnerScaling 敌人数值缩放参数。
// 控制每波敌人的基础属性如何随波次增长。
// HP 公式：enemyHP = (HpBase + HpPerWave * wave) * archetype.HpScale
// 速度公式：enemySpeed = SpeedBase + SpeedPerWave * wave（当前 SpeedPerWave=0，速度不随波次增长）
type SpawnerScaling struct {
	HpBase            float64 `json:"hpBase"`            // 血量基准值
	HpPerWave         float64 `json:"hpPerWave"`         // 每波血量增量
	SpeedBase         float64 `json:"speedBase"`         // 速度基准值（像素/秒）
	SpeedPerWave      float64 `json:"speedPerWave"`      // 每波速度增量（当前为 0）
	EnemiesPerWave    int     `json:"enemiesPerWave"`    // 每波基础出怪数
	SpawnInterval     float64 `json:"spawnInterval"`     // 固定出怪间隔（秒，衰减系统关闭时使用）
	SpawnBaseInterval float64 `json:"spawnBaseInterval"` // 衰减系统起始间隔（>0 时启用衰减）
	SpawnMinInterval  float64 `json:"spawnMinInterval"`  // 衰减下限（出怪间隔不会低于此值）
	SpawnDecayPerWave float64 `json:"spawnDecayPerWave"` // 每波衰减量（秒）
}

// SpawnerTiming 波次时机参数。
type SpawnerTiming struct {
	WaveInterval      float64 `json:"waveInterval"`      // 两波之间的间歇时间（秒）
	FirstWaveInterval float64 `json:"firstWaveInterval"` // 开局到第一波的等待时间（秒，比 WaveInterval 长，给玩家建塔时间）
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
// 出怪器在每波开始时，按概率从当前最高可用档位的 Pool 中随机选取 buff 赋予敌人。
type WaveBuffTier struct {
	MinWave  int      `json:"minWave"`  // 此档位的最低波次要求
	MaxBuffs int      `json:"maxBuffs"` // 此档位单波最多赋予几种 buff
	Pool     []string `json:"pool"`     // 可选 buff 类型池（如 berserk/regen/healAura 等）
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

// EffectiveSpawnInterval 返回指定波次的实际出怪间隔（秒）。
// 衰减公式：interval = SpawnBaseInterval - wave * SpawnDecayPerWave，下限 SpawnMinInterval。
// 例：基础 0.92s，每波减 0.03s → 第 10 波 = 0.62s，第 25 波 = 0.18s（触底）。
// SpawnBaseInterval <= 0 时回退到固定 SpawnInterval（无衰减模式）。
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
// 加载成功后会同步更新 globalWaveCompositions，保持旧 API 的向后兼容。
// 空 compositions 数组被视为配置错误（没有波次组合意味着无法生成敌人）。
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
			HpBase: 80, HpPerWave: 60, SpeedBase: 50, SpeedPerWave: 0,
			EnemiesPerWave: 5, SpawnInterval: 0.6,
			SpawnBaseInterval: 0.92, SpawnMinInterval: 0.18, SpawnDecayPerWave: 0.03,
		},
		Timing: SpawnerTiming{
			WaveInterval: 10, FirstWaveInterval: 20,
		},
		Boss: SpawnerBoss{
			EveryNWaves: 4, HpMultBase: 3, RadiusScale: 1.5,
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
