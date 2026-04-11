// balance_config.go — 游戏平衡参数配置加载。
// 所有影响游戏平衡的数值常量集中在 config/balance.json，由此文件加载。
package config

import (
	"encoding/json"
	"fmt"
)

// EconomyBalance 经济相关平衡参数。
type EconomyBalance struct {
	KillReward      float64 `json:"killReward"`
	SellRefundRatio float64 `json:"sellRefundRatio"`
}

// CombatBalance 战斗相关平衡参数。
type CombatBalance struct {
	MinSpeedRatio             float64 `json:"minSpeedRatio"`
	DotTickInterval           float64 `json:"dotTickInterval"`
	BossPercentHpCap          float64 `json:"bossPercentHpCap"`
	CritMultiplier            float64 `json:"critMultiplier"`
	DefaultProjectileSpeed    float64 `json:"defaultProjectileSpeed"`
	DefaultProjectileRadius   float64 `json:"defaultProjectileRadius"`
	ScatterBasePellets        int     `json:"scatterBasePellets"`
	ScatterSpreadAngle        float64 `json:"scatterSpreadAngle"`
	RadialBaseShots           int     `json:"radialBaseShots"`
	RadialRangeMult           float64 `json:"radialRangeMult"`
	WideBeamRangeMult         float64 `json:"wideBeamRangeMult"`
	WardenProjectileSpeed     float64 `json:"wardenProjectileSpeed"`     // 战灵默认弹速
	WardenMechProjectileSpeed float64 `json:"wardenMechProjectileSpeed"` // 机甲战灵弹速
	WardenFireballSpeed       float64 `json:"wardenFireballSpeed"`       // 火灵火球速度
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
	Kind        string  `json:"kind"`
	Label       string  `json:"label"`
	Boost       float64 `json:"boost"`
	StartCount  int     `json:"startCount"`
	Description string  `json:"description"` // 描述模板，{v} 占位符替换为 Boost 值
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

// ItemDropBalance 道具掉落参数。
type ItemDropBalance struct {
	CycleWaves  int     `json:"cycleWaves"`
	DropChance  float64 `json:"dropChance"`
	MaxPerCycle int     `json:"maxPerCycle"`
	GroundSec   float64 `json:"groundSec"`
	FlySec      float64 `json:"flySec"`
}

// GameplayBalance 通用游戏性参数。
type GameplayBalance struct {
	StarRatingThreshold float64 `json:"starRatingThreshold"`
	MultiKillWindow     float64 `json:"multiKillWindow"`
}

// BalanceConfig 游戏平衡参数总配置。
// 出怪相关参数（spawner/buffs）已迁移至 config/systems/spawner.json，
// 通过 spawner_config.go 的 GlobalSpawnerConfig() 访问。
type BalanceConfig struct {
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
	ItemDrop   ItemDropBalance   `json:"itemDrop"`
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
		Economy: EconomyBalance{KillReward: 15, SellRefundRatio: 0.7},
		Combat: CombatBalance{
			MinSpeedRatio: 0.2, DotTickInterval: 0.5, BossPercentHpCap: 0.05,
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
		Split:      SplitBalance{HpRatio: 0.3, SpeedScale: 1.4, RadiusRatio: 0.7, ChildOffset: 6, RewardScale: 0.2},
		DeathSpawn: DeathSpawnBalance{HpRatio: 0.2, DefaultArch: "normal", ChildOffset: 8, RewardScale: 0.3},
		Dying:      DyingBalance{NormalDuration: 0.3, BossDuration: 0.5},
		Warden:     WardenBalance{InitialStrength: 100, DefaultGrowthOnKill: 2, DefaultGrowthOnWaveClear: 5},
		Gameplay:   GameplayBalance{StarRatingThreshold: 0.8, MultiKillWindow: 1.5},
		ItemDrop:   ItemDropBalance{CycleWaves: 3, DropChance: 0.08, MaxPerCycle: 3, GroundSec: 1.0, FlySec: 0.6},
	}
}
