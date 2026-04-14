// tower_ruleset.go — 塔建造规则策略接口。
//
// 与 Mode 分离：Mode 处理游戏流程（胜负/经济/波次），
// TowerRuleset 处理塔建造策略（能力解锁/属性生成/强度上限/道具掉落）。
// 每个 Mode 通过 Ruleset() 方法暴露自己的塔规则。
//
// 扩展方式：新模式只需实现 TowerRuleset 接口（可嵌入 baseTowerRuleset 获取默认值），
// 在对应 Mode 的 Ruleset() 中返回即可，Stage 层零修改。
package gamemode

// ── 枚举 ─────────────────────────────────────────────

// AbilityMode 能力系统运行模式。
type AbilityMode int

const (
	// AbilityModePaidUnlock 战役模式：建塔时解锁第 1 个攻击位，后续花钱解锁，3 选 1。
	AbilityModePaidUnlock AbilityMode = iota
	// AbilityModeFreeByWave 测试模式：按波次自动解锁，所有类别可选。
	AbilityModeFreeByWave
	// AbilityModePreset 经典模式：预设固定能力，无解锁流程。
	AbilityModePreset
)

// ItemDropMode 道具掉落策略。
type ItemDropMode int

const (
	// ItemDropProbability 战役模式：概率掉落 + 周期保底。
	ItemDropProbability ItemDropMode = iota
	// ItemDropEveryKill 测试模式：每次击杀必掉。
	ItemDropEveryKill
	// ItemDropByWave 经典模式：每 N 波固定掉落。
	ItemDropByWave
	// ItemDropNone 无道具掉落。
	ItemDropNone
)

// ── 接口 ─────────────────────────────────────────────

// TowerRuleset 塔建造规则策略。
// Stage 层通过此接口获取当前模式的塔建造行为，避免散落的 if/else 分支。
type TowerRuleset interface {
	// AbilityMode 返回能力系统的运行模式。
	AbilityMode() AbilityMode

	// InitialUnlockWaves 建塔时传给 RollAndCachePendingChoices 的 wavesCleared 参数。
	// 战役返回 0（仅解锁攻击位），测试返回实际 wavesCleared。
	InitialUnlockWaves(currentWavesCleared int) int

	// ShouldAutoRollOnWaveClear 波次清除时是否自动为所有塔 roll 新的待选能力。
	ShouldAutoRollOnWaveClear() bool

	// MaxStrengthPurchases 单塔最大强度购买次数。-1 表示无限制。
	MaxStrengthPurchases() int

	// ItemDropMode 返回道具掉落策略。
	ItemDropMode() ItemDropMode

	// ShowPaidUnlockButton 信息面板是否显示"花钱解锁能力槽位"按钮。
	ShowPaidUnlockButton() bool

	// UsePresetTowers 是否仅使用预设塔列表（经典模式从 classic-presets.json 加载）。
	// false 时使用标准 towers.json + 随机 tier。
	UsePresetTowers() bool

	// IncludePresetTowers 是否在标准塔列表后追加预设塔（测试模式用）。
	// 仅当 UsePresetTowers()=false 时生效，true 时同时包含标准塔和经典预设塔。
	IncludePresetTowers() bool
}

// ── 默认实现 ─────────────────────────────────────────

// baseTowerRuleset 提供 TowerRuleset 的默认实现（= campaign 行为）。
// 具体 Ruleset 通过嵌入此类型继承默认值，只覆写差异方法。
type baseTowerRuleset struct{}

func (baseTowerRuleset) UsePresetTowers() bool                      { return false }
func (baseTowerRuleset) IncludePresetTowers() bool                  { return false }
func (baseTowerRuleset) AbilityMode() AbilityMode                   { return AbilityModePaidUnlock }
func (baseTowerRuleset) InitialUnlockWaves(_ int) int               { return 0 }
func (baseTowerRuleset) ShouldAutoRollOnWaveClear() bool            { return false }
func (baseTowerRuleset) MaxStrengthPurchases() int                  { return -1 }
func (baseTowerRuleset) ItemDropMode() ItemDropMode                 { return ItemDropProbability }
func (baseTowerRuleset) ShowPaidUnlockButton() bool                 { return true }
