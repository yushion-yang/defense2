// test_ruleset.go — 测试模式塔加载规则。
package gamemode

// TestRuleset 测试模式：标准+经典塔都可用，每次击杀必掉道具。
type TestRuleset struct{ baseTowerRuleset }

func (TestRuleset) IncludePresetTowers() bool { return true }
func (TestRuleset) ItemDropMode() ItemDropMode { return ItemDropEveryKill }
