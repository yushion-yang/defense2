// config_ruleset.go — 配置驱动的 TowerRuleset 实现。
//
// 从 RulesetConfig（JSON 反序列化）直接映射到 TowerRuleset 接口，
// 替代 CampaignRuleset/ClassicRuleset/TestRuleset 三个独立 struct。
//
// ItemDrop 字符串→枚举的映射在 parseItemDropMode 中完成，
// 未知值回退到 ItemDropProbability（战役默认）。
package gamemode

// ConfigRuleset 从 JSON 配置驱动的 TowerRuleset 实现。
type ConfigRuleset struct {
	cfg RulesetConfig
}

// NewConfigRuleset 从 RulesetConfig 创建 TowerRuleset。
func NewConfigRuleset(cfg RulesetConfig) ConfigRuleset {
	return ConfigRuleset{cfg: cfg}
}

func (r ConfigRuleset) UsePresetTowers() bool     { return r.cfg.PresetTowers }
func (r ConfigRuleset) IncludePresetTowers() bool { return r.cfg.IncludePresets }
func (r ConfigRuleset) UseClassicWaves() bool     { return r.cfg.ClassicWaves }
func (r ConfigRuleset) WardenEnabled() bool       { return r.cfg.WardenEnabled }
func (r ConfigRuleset) AllowCustomBlueprints() bool { return r.cfg.Blueprints }
func (r ConfigRuleset) CustomBudgetCap() int        { return r.cfg.BudgetCap }

// ItemDropMode 将 JSON 字符串映射为 ItemDropMode 枚举。
func (r ConfigRuleset) ItemDropMode() ItemDropMode {
	return parseItemDropMode(r.cfg.ItemDrop)
}

// parseItemDropMode 字符串→枚举转换。
// 未知值回退到 ItemDropProbability（战役默认行为）。
func parseItemDropMode(s string) ItemDropMode {
	switch s {
	case "probability":
		return ItemDropProbability
	case "everyKill":
		return ItemDropEveryKill
	case "byWave":
		return ItemDropByWave
	case "none":
		return ItemDropNone
	default:
		return ItemDropProbability
	}
}
