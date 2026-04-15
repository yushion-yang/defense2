package core_test

import (
	"testing"

	"defense2/internal/core/gamemode"
)

// ── AllowCustomBlueprints ──

func TestCampaignRuleset_AllowCustomBlueprints(t *testing.T) {
	r := gamemode.CampaignRuleset{}
	if !r.AllowCustomBlueprints() {
		t.Error("campaign should allow custom blueprints")
	}
	if r.CustomBudgetCap() != -1 {
		t.Errorf("campaign budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}

func TestTestRuleset_AllowCustomBlueprints(t *testing.T) {
	r := gamemode.TestRuleset{}
	if !r.AllowCustomBlueprints() {
		t.Error("test should allow custom blueprints")
	}
	if r.CustomBudgetCap() != -1 {
		t.Errorf("test budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}

func TestClassicRuleset_DisallowCustomBlueprints(t *testing.T) {
	r := gamemode.ClassicRuleset{}
	if r.AllowCustomBlueprints() {
		t.Error("classic should NOT allow custom blueprints")
	}
	// CustomBudgetCap 继承 base 默认值 -1
	if r.CustomBudgetCap() != -1 {
		t.Errorf("classic budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}
