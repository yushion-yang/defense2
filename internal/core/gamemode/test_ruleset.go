// test_ruleset.go — 测试模式塔建造规则。
//
// 测试模式用于开发调试，能力按波次自动解锁，每次击杀必掉道具。
package gamemode

// TestRuleset 测试模式的塔建造规则。
// 特征：能力按波次自动解锁、建塔时按当前波次解锁多个位、每次击杀必掉道具。
type TestRuleset struct{ baseTowerRuleset }

func (TestRuleset) AbilityMode() AbilityMode                   { return AbilityModeFreeByWave }
func (TestRuleset) InitialUnlockWaves(currentWavesCleared int) int { return currentWavesCleared }
func (TestRuleset) ShouldAutoRollOnWaveClear() bool            { return true }
func (TestRuleset) ItemDropMode() ItemDropMode                 { return ItemDropEveryKill }
func (TestRuleset) ShowPaidUnlockButton() bool                 { return false }
func (TestRuleset) IncludePresetTowers() bool                  { return true }
