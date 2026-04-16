//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

// ── CoopAnalysis 测试 ──

func TestCoopAnalysisFillCC(t *testing.T) {
	// 人类有 DPS 塔（高伤害），无 CC → AI 应该补 CC
	allTowers := []aiplayer.AITower{
		{Owner: 0, Damage: 60, Abilities: []string{"barrage"}},    // 人类 DPS 塔
		{Owner: 0, Damage: 55, Abilities: []string{"multiTarget"}}, // 人类 DPS 塔
		{Owner: 1, Damage: 30, Abilities: []string{}},              // AI 普通塔
	}
	ca := aiplayer.AnalyzeCooperation(allTowers, 1)

	if !ca.HumanHasDPS {
		t.Error("HumanHasDPS should be true (human has high damage towers)")
	}
	if ca.HumanHasCC {
		t.Error("HumanHasCC should be false (no CC abilities)")
	}
	if ca.Recommendation != "fill_cc" {
		t.Errorf("Recommendation = %q, want fill_cc", ca.Recommendation)
	}
	if ca.HumanTowers != 2 {
		t.Errorf("HumanTowers = %d, want 2", ca.HumanTowers)
	}
	if ca.AITowers != 1 {
		t.Errorf("AITowers = %d, want 1", ca.AITowers)
	}
}

func TestCoopAnalysisFillDPS(t *testing.T) {
	// 人类有 CC 塔，无高 DPS → AI 应该补 DPS
	allTowers := []aiplayer.AITower{
		{Owner: 0, Damage: 20, Abilities: []string{"slow"}},  // 人类 CC 塔
		{Owner: 0, Damage: 15, Abilities: []string{"stun"}},  // 人类 CC 塔
		{Owner: 1, Damage: 10, Abilities: []string{}},         // AI 低伤害塔
	}
	ca := aiplayer.AnalyzeCooperation(allTowers, 1)

	if !ca.HumanHasCC {
		t.Error("HumanHasCC should be true (human has slow+stun)")
	}
	if ca.HumanHasDPS {
		t.Error("HumanHasDPS should be false (human damage too low)")
	}
	if ca.Recommendation != "fill_dps" {
		t.Errorf("Recommendation = %q, want fill_dps", ca.Recommendation)
	}
}

func TestCoopAnalysisBalanced(t *testing.T) {
	// 双方都有 CC、DPS 和 AOE → balanced
	allTowers := []aiplayer.AITower{
		{Owner: 0, Damage: 60, Abilities: []string{"slow", "barrage", "splash"}}, // 人类全能塔
		{Owner: 1, Damage: 55, Abilities: []string{"stun", "bounce", "scatter"}}, // AI 全能塔
	}
	ca := aiplayer.AnalyzeCooperation(allTowers, 1)

	if !ca.HumanHasCC {
		t.Error("HumanHasCC should be true")
	}
	if !ca.HumanHasDPS {
		t.Error("HumanHasDPS should be true")
	}
	if ca.TeamNeedsCC {
		t.Error("TeamNeedsCC should be false (both sides have CC)")
	}
	if ca.TeamNeedsDPS {
		t.Error("TeamNeedsDPS should be false (both sides have DPS)")
	}
	if ca.Recommendation != "balanced" {
		t.Errorf("Recommendation = %q, want balanced", ca.Recommendation)
	}
}

func TestCoopAnalysisFillAOE(t *testing.T) {
	// 双方都有 CC 和 DPS，但无 AOE → fill_aoe
	allTowers := []aiplayer.AITower{
		{Owner: 0, Damage: 60, Abilities: []string{"slow", "bounce"}}, // 人类 CC+DPS，无 AOE
		{Owner: 1, Damage: 55, Abilities: []string{"stun", "barrage"}}, // AI CC+DPS，无 AOE
	}
	ca := aiplayer.AnalyzeCooperation(allTowers, 1)

	if !ca.TeamNeedsAOE {
		t.Error("TeamNeedsAOE should be true (neither side has AOE)")
	}
	// CC 和 DPS 都覆盖了，只缺 AOE → 不应触发 fill_cc/fill_dps
	if ca.Recommendation != "fill_aoe" {
		t.Errorf("Recommendation = %q, want fill_aoe", ca.Recommendation)
	}
}

func TestCoopAnalysisEmptyTowers(t *testing.T) {
	// 无塔 → balanced（没有数据可分析）
	ca := aiplayer.AnalyzeCooperation(nil, 1)

	if ca.Recommendation != "balanced" {
		t.Errorf("Recommendation = %q, want balanced for empty towers", ca.Recommendation)
	}
	if ca.HumanTowers != 0 || ca.AITowers != 0 {
		t.Errorf("Tower counts should be 0, got human=%d, ai=%d", ca.HumanTowers, ca.AITowers)
	}
}

func TestCoopAnalysisAIAlreadyHasCC(t *testing.T) {
	// 人类无 CC，但 AI 已有 CC + AOE → 全队不缺 CC → balanced
	allTowers := []aiplayer.AITower{
		{Owner: 0, Damage: 60, Abilities: []string{"barrage", "splash"}}, // 人类 DPS + AOE
		{Owner: 1, Damage: 30, Abilities: []string{"slow", "scatter"}},   // AI 有 CC + AOE
	}
	ca := aiplayer.AnalyzeCooperation(allTowers, 1)

	if ca.HumanHasCC {
		t.Error("HumanHasCC should be false")
	}
	if ca.TeamNeedsCC {
		t.Error("TeamNeedsCC should be false (AI already covers CC)")
	}
	// 人类缺 CC 但全队不缺 → balanced（不需要再补）
	if ca.Recommendation != "balanced" {
		t.Errorf("Recommendation = %q, want balanced (AI already has CC)", ca.Recommendation)
	}
}

// ── MergeCoopWithAdvice 测试 ──

func TestMergeCoopWithAdviceCoopWins(t *testing.T) {
	// 个人建议 build_dps，协作建议 fill_cc → 协作覆盖
	coop := aiplayer.CoopAnalysis{
		Recommendation: "fill_cc",
		Reason:         "队友全输出，我来补控制",
	}
	advice := aiplayer.StrategicAdvice{
		Priority: "build_dps",
		Urgency:  0.5,
		Reason:   "需要更多伤害",
	}

	merged := aiplayer.MergeCoopWithAdvice(coop, advice)
	if merged.Priority != "build_cc" {
		t.Errorf("Priority = %q, want build_cc (coop override)", merged.Priority)
	}
	if merged.Urgency < 0.6 {
		t.Errorf("Urgency = %.2f, want >= 0.6 (coop minimum)", merged.Urgency)
	}
}

func TestMergeCoopWithAdviceAgreement(t *testing.T) {
	// 双方都说 build_cc → urgency 加成
	coop := aiplayer.CoopAnalysis{
		Recommendation: "fill_cc",
		Reason:         "队友全输出，我来补控制",
	}
	advice := aiplayer.StrategicAdvice{
		Priority: "build_cc",
		Urgency:  0.7,
		Reason:   "缺控制",
	}

	merged := aiplayer.MergeCoopWithAdvice(coop, advice)
	if merged.Priority != "build_cc" {
		t.Errorf("Priority = %q, want build_cc", merged.Priority)
	}
	// 0.7 * 1.2 = 0.84
	if merged.Urgency < 0.8 {
		t.Errorf("Urgency = %.2f, want >= 0.8 (agreement boost)", merged.Urgency)
	}
}

func TestMergeCoopWithAdviceBalancedNoChange(t *testing.T) {
	// 协作 balanced → 不修改个人建议
	coop := aiplayer.CoopAnalysis{
		Recommendation: "balanced",
	}
	advice := aiplayer.StrategicAdvice{
		Priority: "build_dps",
		Urgency:  0.6,
		Reason:   "需要更多伤害",
	}

	merged := aiplayer.MergeCoopWithAdvice(coop, advice)
	if merged.Priority != "build_dps" {
		t.Errorf("Priority = %q, want build_dps (unchanged)", merged.Priority)
	}
	if merged.Urgency != 0.6 {
		t.Errorf("Urgency = %.2f, want 0.6 (unchanged)", merged.Urgency)
	}
}

// ── CoopDescription 测试 ──

func TestCoopDescription(t *testing.T) {
	ca := aiplayer.CoopAnalysis{
		HumanTowers: 2,
		HumanHasCC:  true,
		HumanHasDPS: false,
		TeamNeedsDPS: true,
	}
	desc := aiplayer.CoopDescription(ca)
	if !containsStr(desc, "队友有控制") {
		t.Errorf("desc should contain '队友有控制', got: %s", desc)
	}
	if !containsStr(desc, "全队缺输出") {
		t.Errorf("desc should contain '全队缺输出', got: %s", desc)
	}
}

func TestCoopDescriptionNoHumanTowers(t *testing.T) {
	ca := aiplayer.CoopAnalysis{HumanTowers: 0}
	desc := aiplayer.CoopDescription(ca)
	if !containsStr(desc, "队友还没建塔") {
		t.Errorf("desc should contain '队友还没建塔', got: %s", desc)
	}
}
