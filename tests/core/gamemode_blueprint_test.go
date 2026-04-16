package core_test

import (
	"testing"

	"defense2/internal/core/gamemode"
)

// ── AllowCustomBlueprints ──
// 通过 GetOrDefault 获取已注册的 UniversalMode，测试其 Ruleset 行为。

func TestCasualRuleset_AllowCustomBlueprints(t *testing.T) {
	m := gamemode.GetOrDefault("casual")
	r := m.Ruleset()
	if !r.AllowCustomBlueprints() {
		t.Error("casual should allow custom blueprints")
	}
	if r.CustomBudgetCap() != -1 {
		t.Errorf("casual budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}

func TestTestRuleset_AllowCustomBlueprints(t *testing.T) {
	m := gamemode.GetOrDefault("test")
	r := m.Ruleset()
	if !r.AllowCustomBlueprints() {
		t.Error("test should allow custom blueprints")
	}
	if r.CustomBudgetCap() != -1 {
		t.Errorf("test budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}

func TestClassicRuleset_DisallowCustomBlueprints(t *testing.T) {
	m := gamemode.GetOrDefault("classic")
	r := m.Ruleset()
	if r.AllowCustomBlueprints() {
		t.Error("classic should NOT allow custom blueprints")
	}
	if r.CustomBudgetCap() != -1 {
		t.Errorf("classic budget cap should be -1, got %d", r.CustomBudgetCap())
	}
}
