// balance_config.go — 游戏平衡参数配置加载。
// 所有影响游戏平衡的数值常量集中在 config/balance.json，由此文件加载。
package config

import (
	"encoding/json"
	"fmt"
)

// SpawnerBalance 出怪相关平衡参数。
type SpawnerBalance struct {
	HpBase            float64 `json:"hpBase"`
	HpPerWave         float64 `json:"hpPerWave"`
	SpeedBase         float64 `json:"speedBase"`
	SpeedPerWave      float64 `json:"speedPerWave"`
	EnemiesPerWave    int     `json:"enemiesPerWave"`
	SpawnInterval     float64 `json:"spawnInterval"`
	WaveInterval      float64 `json:"waveInterval"`
	FirstWaveInterval float64 `json:"firstWaveInterval"`
	BossEveryNWaves   int     `json:"bossEveryNWaves"`
	BossHpMultBase    float64 `json:"bossHpMultBase"`
	BossRadiusScale   float64 `json:"bossRadiusScale"`
	BossEntranceDelay float64 `json:"bossEntranceDelay"`
	BuffChance        float64 `json:"buffChance"`
	BuffMinWaves      []int   `json:"buffMinWaves"`
	BuffMaxBuffs      []int   `json:"buffMaxBuffs"`
	SpawnBaseInterval float64 `json:"spawnBaseInterval"`
	SpawnMinInterval  float64 `json:"spawnMinInterval"`
	SpawnDecayPerWave float64 `json:"spawnDecayPerWave"`
}

// EffectiveSpawnInterval 返回指定波次的出怪间隔（含衰减）。
// SpawnBaseInterval > 0 时启用衰减系统，否则 fallback 到固定 SpawnInterval。
func (b SpawnerBalance) EffectiveSpawnInterval(wave int) float64 {
	if b.SpawnBaseInterval > 0 {
		interval := b.SpawnBaseInterval - float64(wave)*b.SpawnDecayPerWave
		if interval < b.SpawnMinInterval {
			interval = b.SpawnMinInterval
		}
		return interval
	}
	return b.SpawnInterval
}

// EconomyBalance 经济相关平衡参数。
type EconomyBalance struct {
	KillReward      float64 `json:"killReward"`
	SellRefundRatio float64 `json:"sellRefundRatio"`
}

// DotTickIntervals 按 DoT 类型区分的 tick 间隔。
type DotTickIntervals struct {
	Burn   float64 `json:"burn"`
	Bleed  float64 `json:"bleed"`
	Poison float64 `json:"poison"`
}

// CombatBalance 战斗相关平衡参数。
type CombatBalance struct {
	MaxDamageAmplify        float64          `json:"maxDamageAmplify"`
	MinSpeedRatio           float64          `json:"minSpeedRatio"`
	DotTickInterval         float64          `json:"dotTickInterval"`
	BossPercentHpCap        float64          `json:"bossPercentHpCap"`
	DamageDownFloor         float64          `json:"damageDownFloor"`
	CritMultiplier          float64          `json:"critMultiplier"`
	DefaultProjectileSpeed  float64          `json:"defaultProjectileSpeed"`
	DefaultProjectileRadius float64          `json:"defaultProjectileRadius"`
	ScatterBasePellets      int              `json:"scatterBasePellets"`
	ScatterSpreadAngle      float64          `json:"scatterSpreadAngle"`
	RadialBaseShots         int              `json:"radialBaseShots"`
	RadialRangeMult         float64          `json:"radialRangeMult"`
	WideBeamRangeMult       float64          `json:"wideBeamRangeMult"`
	DotTickIntervalsMap     DotTickIntervals `json:"dotTickIntervals"`
}

// TowerBalance 塔相关平衡参数。
type TowerBalance struct {
	StrengthBuyCost   int     `json:"strengthBuyCost"`
	StrengthBuyAmount float64 `json:"strengthBuyAmount"`
	WavesPerUnlock    int     `json:"wavesPerUnlock"`
	ChoicesPerUnlock  int     `json:"choicesPerUnlock"`
	AttackSpeedFloor  float64 `json:"attackSpeedFloor"`
	FireRateFloor     float64 `json:"fireRateFloor"`
}

// ChainBalance 连锁网络相关平衡参数。
type ChainBalance struct {
	Distance         float64 `json:"distance"`
	StrengthPerTower float64 `json:"strengthPerTower"`
}

// ItemBalance 单个道具定义。
type ItemBalance struct {
	Kind  string  `json:"kind"`
	Label string  `json:"label"`
	Boost float64 `json:"boost"`
}

// SplitBalance 分裂子体参数。
type SplitBalance struct {
	HpRatio     float64 `json:"hpRatio"`
	SpeedScale  float64 `json:"speedScale"`
	RadiusRatio float64 `json:"radiusRatio"`
	RewardScale float64 `json:"rewardScale"` // 分裂子体奖励倍率（0.2=20%奖励）
	ChildOffset float64 `json:"childOffset"`
}

// DeathSpawnBalance 死亡召唤参数。
type DeathSpawnBalance struct {
	HpRatio     float64 `json:"hpRatio"`
	DefaultArch string  `json:"defaultArch"`
	ChildOffset float64 `json:"childOffset"`
	RewardScale float64 `json:"rewardScale"` // 召唤小怪奖励倍率（0.3=30%奖励）
}

// DyingBalance 死亡动画参数。
type DyingBalance struct {
	NormalDuration float64 `json:"normalDuration"`
	BossDuration   float64 `json:"bossDuration"`
}

// WardenBalance 战灵默认参数。
type WardenBalance struct {
	InitialStrength          float64 `json:"initialStrength"`
	DefaultGrowthOnKill      float64 `json:"defaultGrowthOnKill"`
	DefaultGrowthOnWaveClear float64 `json:"defaultGrowthOnWaveClear"`
}

// GameplayBalance 通用游戏性参数。
type GameplayBalance struct {
	StarRatingThreshold float64 `json:"starRatingThreshold"`
	MultiKillWindow     float64 `json:"multiKillWindow"`
	MultiKillAnnounce1  int     `json:"multiKillAnnounce1"`
	MultiKillAnnounce2  int     `json:"multiKillAnnounce2"`
}

// BerserkBuff 狂暴 buff 配置。
type BerserkBuff struct {
	Threshold  float64 `json:"threshold"`  // 激活血量比例
	SpeedScale float64 `json:"speedScale"` // 激活后速度倍率
}

// RegenBuff 回复 buff 配置。
type RegenBuff struct {
	HpRatio float64 `json:"hpRatio"` // 每秒回复占 MaxHP 比例
}

// HealAuraBuff 治疗光环配置。
type HealAuraBuff struct {
	Power    float64 `json:"power"`    // 每次治疗占 MaxHP 比例
	Radius   float64 `json:"radius"`   // 治疗范围
	Interval float64 `json:"interval"` // 治疗间隔（秒）
}

// SpeedAuraBuff 加速光环配置。
type SpeedAuraBuff struct {
	Radius  float64 `json:"radius"`  // 光环范围
	SpeedUp float64 `json:"speedUp"` // 加速比例
}

// DamageReduceBuff 减伤配置。
type DamageReduceBuff struct {
	Ratio float64 `json:"ratio"` // 减伤比例
}

// DeathSplitBuff 死亡分裂配置。
type DeathSplitBuff struct {
	Count      int     `json:"count"`      // 分裂数量
	HpRatio    float64 `json:"hpRatio"`    // 子体 HP 比例
	SpeedScale float64 `json:"speedScale"` // 子体速度倍率
}

// BuffsBalance 波次 buff 配置。
type BuffsBalance struct {
	Berserk      BerserkBuff      `json:"berserk"`
	Regen        RegenBuff        `json:"regen"`
	HealAura     HealAuraBuff     `json:"healAura"`
	SpeedAura    SpeedAuraBuff    `json:"speedAura"`
	DamageReduce DamageReduceBuff `json:"damageReduce"`
	DeathSplit   DeathSplitBuff   `json:"deathSplit"`
}

// BalanceConfig 游戏平衡参数总配置。
type BalanceConfig struct {
	Spawner    SpawnerBalance    `json:"spawner"`
	Economy    EconomyBalance    `json:"economy"`
	Combat     CombatBalance     `json:"combat"`
	Tower      TowerBalance      `json:"tower"`
	Chain      ChainBalance      `json:"chain"`
	Items      []ItemBalance     `json:"items"`
	Split      SplitBalance      `json:"split"`
	DeathSpawn DeathSpawnBalance `json:"deathSpawn"`
	Dying      DyingBalance      `json:"dying"`
	Warden     WardenBalance     `json:"warden"`
	Gameplay   GameplayBalance   `json:"gameplay"`
	Buffs      BuffsBalance      `json:"buffs"`
}

// globalBalance 全局缓存。
var globalBalance *BalanceConfig

// GlobalBalance 返回全局平衡配置。LoadBalance 成功后可用。
func GlobalBalance() *BalanceConfig {
	if globalBalance == nil {
		return defaultBalance()
	}
	return globalBalance
}

// LoadBalance 从 config/balance.json 加载平衡配置。
func LoadBalance() (*BalanceConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load balance: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/balance.json")
	if err != nil {
		return nil, fmt.Errorf("load balance: %w", err)
	}

	bal := defaultBalance()
	if err := json.Unmarshal(data, bal); err != nil {
		return nil, fmt.Errorf("parse balance: %w", err)
	}
	globalBalance = bal
	return bal, nil
}

// defaultBalance 返回所有参数的默认值（确保向后兼容）。
func defaultBalance() *BalanceConfig {
	return &BalanceConfig{
		Spawner: SpawnerBalance{
			HpBase: 52, HpPerWave: 21, SpeedBase: 58, SpeedPerWave: 5,
			EnemiesPerWave: 5, SpawnInterval: 0.6, WaveInterval: 10, FirstWaveInterval: 20,
			BossEveryNWaves: 5, BossHpMultBase: 8, BossRadiusScale: 1.5, BossEntranceDelay: 3.0,
			BuffChance: 0.3, BuffMinWaves: []int{6, 16, 26}, BuffMaxBuffs: []int{1, 1, 2},
		},
		Economy: EconomyBalance{KillReward: 15, SellRefundRatio: 0.7},
		Combat: CombatBalance{
			MaxDamageAmplify: 0.5, MinSpeedRatio: 0.2, DotTickInterval: 0.5, BossPercentHpCap: 0.05, DamageDownFloor: 0.2,
			CritMultiplier: 2, DefaultProjectileSpeed: 300, DefaultProjectileRadius: 4,
			ScatterBasePellets: 3, ScatterSpreadAngle: 60,
			RadialBaseShots: 3, RadialRangeMult: 1.2, WideBeamRangeMult: 3,
			DotTickIntervalsMap: DotTickIntervals{Burn: 0.5, Bleed: 0.5, Poison: 1.0},
		},
		Tower: TowerBalance{
			StrengthBuyCost: 10, StrengthBuyAmount: 10,
			WavesPerUnlock: 2, ChoicesPerUnlock: 3, AttackSpeedFloor: 0.1,
			FireRateFloor: 0.18,
		},
		Items: []ItemBalance{
			{Kind: "baseDamage", Label: "攻击磨石", Boost: 2},
			{Kind: "potentialDamage", Label: "攻击秘卷", Boost: 3},
			{Kind: "baseSpeed", Label: "速射齿轮", Boost: 0.15},
			{Kind: "potentialSpeed", Label: "速射秘卷", Boost: 0.2},
			{Kind: "baseRange", Label: "瞄准镜片", Boost: 12},
			{Kind: "potentialRange", Label: "瞄准秘卷", Boost: 18},
		},
		Chain:      ChainBalance{Distance: 150, StrengthPerTower: 10},
		Split:      SplitBalance{HpRatio: 0.3, SpeedScale: 1.4, RadiusRatio: 0.7, ChildOffset: 6, RewardScale: 0.2},
		DeathSpawn: DeathSpawnBalance{HpRatio: 0.2, DefaultArch: "normal", ChildOffset: 8, RewardScale: 0.3},
		Dying:      DyingBalance{NormalDuration: 0.3, BossDuration: 0.5},
		Warden:     WardenBalance{InitialStrength: 100, DefaultGrowthOnKill: 2, DefaultGrowthOnWaveClear: 5},
		Gameplay:   GameplayBalance{StarRatingThreshold: 0.8, MultiKillWindow: 1.5, MultiKillAnnounce1: 5, MultiKillAnnounce2: 10},
		Buffs: BuffsBalance{
			Berserk:      BerserkBuff{Threshold: 0.5, SpeedScale: 1.5},
			Regen:        RegenBuff{HpRatio: 0.02},
			HealAura:     HealAuraBuff{Power: 0.05, Radius: 80, Interval: 3},
			SpeedAura:    SpeedAuraBuff{Radius: 80, SpeedUp: 0.2},
			DamageReduce: DamageReduceBuff{Ratio: 0.3},
			DeathSplit:   DeathSplitBuff{Count: 2, HpRatio: 0.3, SpeedScale: 1.4},
		},
	}
}
