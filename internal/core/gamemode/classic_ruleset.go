// classic_ruleset.go — 经典模式塔建造规则。
//
// 经典模式特征：
//   - 11 种预设塔型，每塔 2 个固定能力，无解锁流程
//   - Strength 上限 200，最多购买 2 次（每次 +50）
//   - 道具按波次掉落（每 10 波 1 个）
//   - 无付费解锁能力按钮
package gamemode

// ClassicRuleset 经典模式的塔建造规则。
type ClassicRuleset struct{ baseTowerRuleset }

func (ClassicRuleset) UsePresetTowers() bool            { return true }
func (ClassicRuleset) AbilityMode() AbilityMode        { return AbilityModePreset }
func (ClassicRuleset) InitialUnlockWaves(_ int) int     { return 0 }
func (ClassicRuleset) ShouldAutoRollOnWaveClear() bool  { return false }
func (ClassicRuleset) MaxStrengthPurchases() int        { return 2 }
func (ClassicRuleset) ItemDropMode() ItemDropMode       { return ItemDropByWave }
func (ClassicRuleset) ShowPaidUnlockButton() bool       { return false }
