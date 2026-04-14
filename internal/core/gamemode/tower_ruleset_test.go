package gamemode

import "testing"

func TestCampaignRuleset(t *testing.T) {
	r := CampaignRuleset{}
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

func TestTestRuleset(t *testing.T) {
	r := TestRuleset{}
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

func TestClassicRuleset(t *testing.T) {
	r := ClassicRuleset{}
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
