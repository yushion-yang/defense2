// tower_ruleset.go — 模式级塔加载规则。
//
// 与 Mode 分离：Mode 处理游戏流程（胜负/经济/波次），
// TowerRuleset 处理模式级的塔加载策略和道具掉落规则。
//
// 塔自身的行为规则（能力获取方式/强度上限等）已下沉到 TowerDef 配置中，
// 每种塔自带自己的规则，不再由 TowerRuleset 全局控制。
package gamemode

// ItemDropMode 道具掉落策略。
type ItemDropMode int

const (
	// ItemDropProbability 战役模式：概率掉落 + 周期保底。
	ItemDropProbability ItemDropMode = iota
	// ItemDropEveryKill 测试模式：每次击杀必掉。
	ItemDropEveryKill
	// ItemDropByWave 经典模式：特定波次固定掉落。
	ItemDropByWave
	// ItemDropNone 无道具掉落。
	ItemDropNone
)

// TowerRuleset 模式级塔加载规则。
// 塔自身的行为规则（abilityMode/strength）在 TowerDef 配置中定义。
type TowerRuleset interface {
	// UsePresetTowers 是否仅使用预设塔列表（经典模式）。
	UsePresetTowers() bool

	// IncludePresetTowers 是否追加预设塔（测试模式：标准+经典都可用）。
	IncludePresetTowers() bool

	// ItemDropMode 返回道具掉落策略。
	ItemDropMode() ItemDropMode

	// UseClassicWaves 是否使用经典模式确定性出怪配置（classic-waves.json）。
	UseClassicWaves() bool

	// WardenEnabled 是否提供战灵选择。经典模式不提供战灵。
	WardenEnabled() bool

	// AllowCustomBlueprints 是否允许使用玩家自定义蓝图。
	AllowCustomBlueprints() bool

	// CustomBudgetCap 自定义蓝图的预算上限覆盖（-1 = 用默认值）。
	CustomBudgetCap() int

	// GoldShare 是否启用金币共享（合作模式：击杀/产金所有玩家等额获得）。
	GoldShare() bool
}

// baseTowerRuleset 默认实现（= campaign 行为）。
type baseTowerRuleset struct{}

func (baseTowerRuleset) UsePresetTowers() bool       { return false }
func (baseTowerRuleset) IncludePresetTowers() bool   { return false }
func (baseTowerRuleset) ItemDropMode() ItemDropMode  { return ItemDropProbability }
func (baseTowerRuleset) UseClassicWaves() bool       { return false }
func (baseTowerRuleset) WardenEnabled() bool         { return true }
func (baseTowerRuleset) AllowCustomBlueprints() bool { return true }
func (baseTowerRuleset) CustomBudgetCap() int        { return -1 }
func (baseTowerRuleset) GoldShare() bool             { return false }
