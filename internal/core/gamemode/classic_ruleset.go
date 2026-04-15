// classic_ruleset.go — 经典模式塔加载规则。
package gamemode

// ClassicRuleset 经典模式：仅预设塔，确定性出怪，按波次掉落道具。
type ClassicRuleset struct{ baseTowerRuleset }

func (ClassicRuleset) UsePresetTowers() bool     { return true }
func (ClassicRuleset) UseClassicWaves() bool     { return true }
func (ClassicRuleset) ItemDropMode() ItemDropMode { return ItemDropByWave }
