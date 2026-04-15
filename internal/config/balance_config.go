// balance_config.go — 游戏平衡配置的 Go 映射。
//
// 配置分布在 5 个 JSON 文件中，按子系统归属到对应子目录：
//
//	config/systems/combat.json    — 战斗参数（减速/DoT/暴击/弹道）
//	config/towers/balance.json    — 塔(tower) + 连锁(chain)
//	config/towers/items.json      — 道具定义（6 种）
//	config/enemies/balance.json   — 分裂(split) / 死亡召唤(deathSpawn) / 死亡动画(dying)
//	config/wardens/balance.json   — 战灵默认参数
//	config/systems/gameplay.json  — 游戏性(gameplay) + 道具掉落(itemDrop)
//
// 独立子系统（各自有 GlobalXxx() 访问器）：
//   - 经济(economy) → config/systems/economy.json
//   - 出怪(spawner) → config/systems/spawner.json
//
// 设计决策：
//   - 每个子系统一个独立 struct，方便按模块传递
//   - defaultBalance() 提供完整的硬编码默认值，确保 JSON 缺字段时不会零值崩溃
//   - globalBalance 全局缓存 + GlobalBalance() 访问器模式，与 spawner/enemy 配置一致
//   - LoadBalance() 从多个文件加载合并，消费方通过 GlobalBalance() 统一访问，无感知
package config

import (
	"encoding/json"
	"fmt"
)

// CombatBalance 战斗相关平衡参数。
// 涵盖减速下限、DoT 节奏、暴击、弹道参数。
//
// 已清理的死字段（仅保留在 JSON 中作文档，加 _ 前缀）：
//   - scatterBasePellets/scatterSpreadAngle/radialBaseShots/radialRangeMult/wideBeamRangeMult
//     → 实际从 abilities.json 读取
//   - bossPercentHpCap → 已迁移至 spawner.json percentHpCap
//   - wardenProjectileSpeed/wardenFireballSpeed → 代码硬编码或用 ParamOr
type CombatBalance struct {
	MinSpeedRatio             float64 `json:"minSpeedRatio"`             // 减速下限（0.2 = 最多减速到原速的 20%）
	DotTickInterval           float64 `json:"dotTickInterval"`           // DoT 伤害跳间隔（秒），各 DoT 类型可在 dotTickIntervals 中覆盖
	CritMultiplier            float64 `json:"critMultiplier"`            // 暴击伤害倍率
	DefaultProjectileSpeed    float64 `json:"defaultProjectileSpeed"`    // 默认弹速（像素/秒）
	DefaultProjectileRadius   float64 `json:"defaultProjectileRadius"`   // 默认弹体半径（像素，用于碰撞检测）
	WardenMechProjectileSpeed float64 `json:"wardenMechProjectileSpeed"` // 机甲战灵弹速（更快）
}

// TowerBalance 塔相关平衡参数。
// 塔属性公式：attr = Base + Potential * (Strength / 100)
type TowerBalance struct {
	StrengthBuyCost   int     `json:"strengthBuyCost"`   // 购买力量的金币消耗
	StrengthBuyAmount float64 `json:"strengthBuyAmount"` // 每次购买增加的力量值
	WavesPerUnlock    int     `json:"wavesPerUnlock"`    // 每隔几波解锁一个能力槽（默认 2）
	ChoicesPerUnlock  int     `json:"choicesPerUnlock"`  // 每次解锁提供几个候选能力（默认 3 选 1）
	AttackSpeedFloor  float64 `json:"attackSpeedFloor"`  // 攻击间隔下限（秒），防止叠速后无限攻速
}

// ChainBalance 连锁网络相关平衡参数。
// 连锁：相邻塔形成网络，互相增加力量值。
type ChainBalance struct {
	Distance         float64 `json:"distance"`         // 连锁判定距离（逻辑像素），塔间距 < 此值时建立连锁
	StrengthPerTower float64 `json:"strengthPerTower"` // 每个连锁邻居贡献的力量值
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
// 掉落机制：每 CycleWaves 波为一个周期，周期内每次击杀有 DropChance 概率掉落，
// 达到 MaxPerCycle 上限后本周期不再掉落。
type ItemDropBalance struct {
	CycleWaves  int     `json:"cycleWaves"`  // 掉落周期（波数）
	DropChance  float64 `json:"dropChance"`  // 每次击杀的掉落概率
	MaxPerCycle int     `json:"maxPerCycle"` // 每周期最大掉落数
	GroundSec   float64 `json:"groundSec"`   // 道具落地动画时长（秒）
	FlySec      float64 `json:"flySec"`      // 道具飞向玩家动画时长（秒）
}

// GameplayBalance 通用游戏性参数。
type GameplayBalance struct {
	StarRatingThreshold float64 `json:"starRatingThreshold"`
	MultiKillWindow     float64 `json:"multiKillWindow"`
}

// BalanceConfig 游戏平衡参数总配置。
// 已迁移：economy → economy.json, spawner → spawner.json。
type BalanceConfig struct {
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

// LoadBalance 从多个配置文件加载平衡参数，合并到统一的 BalanceConfig。
func LoadBalance() (*BalanceConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load balance: dataFS not initialized")
	}

	bal := defaultBalance()

	if err := loadCombatBalance(bal); err != nil {
		return nil, err
	}
	if err := loadTowerBalance(bal); err != nil {
		return nil, err
	}
	if err := loadItemsBalance(bal); err != nil {
		return nil, err
	}
	if err := loadEnemyBalance(bal); err != nil {
		return nil, err
	}
	if err := loadWardenBalance(bal); err != nil {
		return nil, err
	}
	if err := loadGameplayBalance(bal); err != nil {
		return nil, err
	}

	globalBalance = bal
	return bal, nil
}

// loadCombatBalance 加载 config/systems/combat.json → bal.Combat。
func loadCombatBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/systems/combat.json")
	if err != nil {
		return fmt.Errorf("load combat balance: %w", err)
	}
	if err := json.Unmarshal(data, &bal.Combat); err != nil {
		return fmt.Errorf("parse combat balance: %w", err)
	}
	return nil
}

// towerChainFile 辅助结构：config/towers/balance.json 的顶层布局。
type towerChainFile struct {
	Tower TowerBalance `json:"tower"`
	Chain ChainBalance `json:"chain"`
}

// loadTowerBalance 加载 config/towers/balance.json → bal.Tower + bal.Chain。
func loadTowerBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/towers/balance.json")
	if err != nil {
		return fmt.Errorf("load tower balance: %w", err)
	}
	// 先用默认值填充，再让 JSON 覆盖
	tmp := towerChainFile{Tower: bal.Tower, Chain: bal.Chain}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("parse tower balance: %w", err)
	}
	bal.Tower = tmp.Tower
	bal.Chain = tmp.Chain
	return nil
}

// loadItemsBalance 加载 config/towers/items.json → bal.Items。
func loadItemsBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/towers/items.json")
	if err != nil {
		return fmt.Errorf("load items: %w", err)
	}
	if err := json.Unmarshal(data, &bal.Items); err != nil {
		return fmt.Errorf("parse items: %w", err)
	}
	return nil
}

// enemyBalanceFile 辅助结构：config/enemies/balance.json 的顶层布局。
type enemyBalanceFile struct {
	Split      SplitBalance      `json:"split"`
	DeathSpawn DeathSpawnBalance `json:"deathSpawn"`
	Dying      DyingBalance      `json:"dying"`
}

// loadEnemyBalance 加载 config/enemies/balance.json → bal.Split + DeathSpawn + Dying。
func loadEnemyBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/enemies/balance.json")
	if err != nil {
		return fmt.Errorf("load enemy balance: %w", err)
	}
	tmp := enemyBalanceFile{Split: bal.Split, DeathSpawn: bal.DeathSpawn, Dying: bal.Dying}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("parse enemy balance: %w", err)
	}
	bal.Split = tmp.Split
	bal.DeathSpawn = tmp.DeathSpawn
	bal.Dying = tmp.Dying
	return nil
}

// loadWardenBalance 加载 config/wardens/balance.json → bal.Warden。
func loadWardenBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/wardens/balance.json")
	if err != nil {
		return fmt.Errorf("load warden balance: %w", err)
	}
	if err := json.Unmarshal(data, &bal.Warden); err != nil {
		return fmt.Errorf("parse warden balance: %w", err)
	}
	return nil
}

// gameplayFile 辅助结构：config/systems/gameplay.json 的顶层布局。
type gameplayFile struct {
	Gameplay GameplayBalance `json:"gameplay"`
	ItemDrop ItemDropBalance `json:"itemDrop"`
}

// loadGameplayBalance 加载 config/systems/gameplay.json → bal.Gameplay + bal.ItemDrop。
func loadGameplayBalance(bal *BalanceConfig) error {
	data, err := dataFS.ReadFile("config/systems/gameplay.json")
	if err != nil {
		return fmt.Errorf("load gameplay balance: %w", err)
	}
	tmp := gameplayFile{Gameplay: bal.Gameplay, ItemDrop: bal.ItemDrop}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("parse gameplay balance: %w", err)
	}
	bal.Gameplay = tmp.Gameplay
	bal.ItemDrop = tmp.ItemDrop
	return nil
}

// defaultBalance 返回所有参数的硬编码默认值。
// 设计目的：当 balance.json 缺少某个字段时，不会使用 Go 零值（如 0.0），
// 而是回退到合理的游戏默认值。这在配置迭代中避免了"新增字段忘更新 JSON"的问题。
func defaultBalance() *BalanceConfig {
	return &BalanceConfig{
		Combat: CombatBalance{
			MinSpeedRatio: 0.2, DotTickInterval: 0.5,
			CritMultiplier: 2, DefaultProjectileSpeed: 300, DefaultProjectileRadius: 4,
			WardenMechProjectileSpeed: 400,
		},
		Tower: TowerBalance{
			StrengthBuyCost: 10, StrengthBuyAmount: 10,
			WavesPerUnlock: 2, ChoicesPerUnlock: 3, AttackSpeedFloor: 0.1,
		},
		Items: []ItemBalance{
			{Kind: "baseDamage", Label: "攻击磨石", Boost: 5, Description: "基础伤害+{v}"},
			{Kind: "baseSpeed", Label: "速射齿轮", Boost: 0.1, Description: "基础攻速+{v}"},
			{Kind: "baseRange", Label: "瞄准镜片", Boost: 5, Description: "基础射程+{v}"},
			{Kind: "potentialDamage", Label: "攻击秘卷", Boost: 2, Description: "潜力伤害+{v}"},
			{Kind: "potentialSpeed", Label: "速射秘卷", Boost: 0.02, Description: "潜力攻速+{v}"},
			{Kind: "potentialRange", Label: "瞄准秘卷", Boost: 2, Description: "潜力射程+{v}"},
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
