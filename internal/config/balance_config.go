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
	BuffChance        float64 `json:"buffChance"`
	BuffMinWaves      []int   `json:"buffMinWaves"`
	BuffMaxBuffs      []int   `json:"buffMaxBuffs"`
}

// EconomyBalance 经济相关平衡参数。
type EconomyBalance struct {
	KillReward      float64 `json:"killReward"`
	SellRefundRatio float64 `json:"sellRefundRatio"`
}

// CombatBalance 战斗相关平衡参数。
type CombatBalance struct {
	MaxDamageAmplify        float64 `json:"maxDamageAmplify"`
	MinSpeedRatio           float64 `json:"minSpeedRatio"`
	DotTickInterval         float64 `json:"dotTickInterval"`
	CritMultiplier          float64 `json:"critMultiplier"`
	DefaultProjectileSpeed  float64 `json:"defaultProjectileSpeed"`
	DefaultProjectileRadius float64 `json:"defaultProjectileRadius"`
	ScatterBasePellets      int     `json:"scatterBasePellets"`
	ScatterSpreadAngle      float64 `json:"scatterSpreadAngle"`
	RadialBaseShots         int     `json:"radialBaseShots"`
	RadialRangeMult         float64 `json:"radialRangeMult"`
	WideBeamRangeMult       float64 `json:"wideBeamRangeMult"`
}

// TowerBalance 塔相关平衡参数。
type TowerBalance struct {
	StrengthBuyCost   int     `json:"strengthBuyCost"`
	StrengthBuyAmount float64 `json:"strengthBuyAmount"`
	WavesPerUnlock    int     `json:"wavesPerUnlock"`
	ChoicesPerUnlock  int     `json:"choicesPerUnlock"`
	AttackSpeedFloor  float64 `json:"attackSpeedFloor"`
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
	ChildOffset float64 `json:"childOffset"`
}

// DeathSpawnBalance 死亡召唤参数。
type DeathSpawnBalance struct {
	HpRatio     float64 `json:"hpRatio"`
	DefaultArch string  `json:"defaultArch"`
	ChildOffset float64 `json:"childOffset"`
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
			BossEveryNWaves: 5, BossHpMultBase: 8, BossRadiusScale: 1.5,
			BuffChance: 0.3, BuffMinWaves: []int{6, 16, 26}, BuffMaxBuffs: []int{1, 1, 2},
		},
		Economy: EconomyBalance{KillReward: 15, SellRefundRatio: 0.7},
		Combat: CombatBalance{
			MaxDamageAmplify: 0.5, MinSpeedRatio: 0.2, DotTickInterval: 0.5,
			CritMultiplier: 2, DefaultProjectileSpeed: 300, DefaultProjectileRadius: 4,
			ScatterBasePellets: 3, ScatterSpreadAngle: 60,
			RadialBaseShots: 3, RadialRangeMult: 1.2, WideBeamRangeMult: 3,
		},
		Tower: TowerBalance{
			StrengthBuyCost: 10, StrengthBuyAmount: 10,
			WavesPerUnlock: 2, ChoicesPerUnlock: 3, AttackSpeedFloor: 0.1,
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
		Split:      SplitBalance{HpRatio: 0.3, SpeedScale: 1.4, RadiusRatio: 0.7, ChildOffset: 6},
		DeathSpawn: DeathSpawnBalance{HpRatio: 0.2, DefaultArch: "normal", ChildOffset: 8},
		Dying:      DyingBalance{NormalDuration: 0.3, BossDuration: 0.5},
		Warden:     WardenBalance{InitialStrength: 100, DefaultGrowthOnKill: 2, DefaultGrowthOnWaveClear: 5},
	}
}
