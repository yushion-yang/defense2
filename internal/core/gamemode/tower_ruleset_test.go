package gamemode

import "testing"

// TestConfigRuleset_Campaign 验证零值 ConfigRuleset 等价于旧 CampaignRuleset（全部默认值）。
func TestConfigRuleset_Campaign(t *testing.T) {
	r := ConfigRuleset{}
	if r.UsePresetTowers() {
		t.Error("UsePresetTowers: want false")
	}
	if r.IncludePresetTowers() {
		t.Error("IncludePresetTowers: want false")
	}
	if r.ItemDropMode() != ItemDropProbability {
		t.Errorf("ItemDropMode: got %d, want Probability", r.ItemDropMode())
	}
}

// TestConfigRuleset_Test 验证 test 模式的 ConfigRuleset。
func TestConfigRuleset_Test(t *testing.T) {
	r := NewConfigRuleset(RulesetConfig{
		IncludePresets: true,
		ItemDrop:       "everyKill",
	})
	if r.UsePresetTowers() {
		t.Error("UsePresetTowers: want false")
	}
	if !r.IncludePresetTowers() {
		t.Error("IncludePresetTowers: want true")
	}
	if r.ItemDropMode() != ItemDropEveryKill {
		t.Errorf("ItemDropMode: got %d, want EveryKill", r.ItemDropMode())
	}
}

// TestConfigRuleset_Classic 验证 classic 模式的 ConfigRuleset。
func TestConfigRuleset_Classic(t *testing.T) {
	r := NewConfigRuleset(RulesetConfig{
		PresetTowers: true,
		ClassicWaves: true,
		ItemDrop:     "byWave",
	})
	if !r.UsePresetTowers() {
		t.Error("UsePresetTowers: want true")
	}
	if r.ItemDropMode() != ItemDropByWave {
		t.Errorf("ItemDropMode: got %d, want ByWave", r.ItemDropMode())
	}
}

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
