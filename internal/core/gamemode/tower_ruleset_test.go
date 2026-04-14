package gamemode

import "testing"

// TestCampaignRuleset 验证战役模式的塔规则与 campaign 行为一致。
func TestCampaignRuleset(t *testing.T) {
	r := CampaignRuleset{}

	if r.AbilityMode() != AbilityModePaidUnlock {
		t.Errorf("AbilityMode: got %d, want PaidUnlock(%d)", r.AbilityMode(), AbilityModePaidUnlock)
	}
	if got := r.InitialUnlockWaves(5); got != 0 {
		t.Errorf("InitialUnlockWaves(5): got %d, want 0", got)
	}
	if r.ShouldAutoRollOnWaveClear() {
		t.Error("ShouldAutoRollOnWaveClear: got true, want false")
	}
	if r.MaxStrengthPurchases() != -1 {
		t.Errorf("MaxStrengthPurchases: got %d, want -1", r.MaxStrengthPurchases())
	}
	if r.ItemDropMode() != ItemDropProbability {
		t.Errorf("ItemDropMode: got %d, want Probability(%d)", r.ItemDropMode(), ItemDropProbability)
	}
	if !r.ShowPaidUnlockButton() {
		t.Error("ShowPaidUnlockButton: got false, want true")
	}
}

// TestTestRuleset 验证测试模式的塔规则。
func TestTestRuleset(t *testing.T) {
	r := TestRuleset{}

	if r.AbilityMode() != AbilityModeFreeByWave {
		t.Errorf("AbilityMode: got %d, want FreeByWave(%d)", r.AbilityMode(), AbilityModeFreeByWave)
	}
	if got := r.InitialUnlockWaves(5); got != 5 {
		t.Errorf("InitialUnlockWaves(5): got %d, want 5", got)
	}
	if !r.ShouldAutoRollOnWaveClear() {
		t.Error("ShouldAutoRollOnWaveClear: got false, want true")
	}
	if r.MaxStrengthPurchases() != -1 {
		t.Errorf("MaxStrengthPurchases: got %d, want -1", r.MaxStrengthPurchases())
	}
	if r.ItemDropMode() != ItemDropEveryKill {
		t.Errorf("ItemDropMode: got %d, want EveryKill(%d)", r.ItemDropMode(), ItemDropEveryKill)
	}
	if r.ShowPaidUnlockButton() {
		t.Error("ShowPaidUnlockButton: got true, want false")
	}
}

// TestAllModesHaveRuleset 验证所有注册模式都返回非 nil 的 Ruleset。
func TestAllModesHaveRuleset(t *testing.T) {
	for _, id := range List() {
		m := Get(id)
		if m == nil {
			t.Errorf("mode %q: Get returned nil", id)
			continue
		}
		if m.Ruleset() == nil {
			t.Errorf("mode %q: Ruleset() returned nil", id)
		}
	}
}
