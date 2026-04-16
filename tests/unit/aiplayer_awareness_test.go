//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

// ── ThreatAnalysis 测试 ──

func TestAnalyzeThreatsEmpty(t *testing.T) {
	ta := aiplayer.AnalyzeThreats(nil)
	if ta.TotalEnemies != 0 {
		t.Errorf("TotalEnemies = %d, want 0", ta.TotalEnemies)
	}
	if ta.ThreatLevel != "low" {
		t.Errorf("ThreatLevel = %q, want low", ta.ThreatLevel)
	}
	if ta.DominantThreat != "none" {
		t.Errorf("DominantThreat = %q, want none", ta.DominantThreat)
	}
}

func TestAnalyzeThreatsBossDetection(t *testing.T) {
	enemies := []aiplayer.AIEnemy{
		{HP: 100, MaxHP: 100, Speed: 1.0, Boss: true, Active: true},
		{HP: 50, MaxHP: 50, Speed: 1.0, Active: true},
		{HP: 50, MaxHP: 50, Speed: 1.0, Active: true},
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	if ta.BossCount != 1 {
		t.Errorf("BossCount = %d, want 1", ta.BossCount)
	}
	if ta.DominantThreat != "boss" {
		t.Errorf("DominantThreat = %q, want boss", ta.DominantThreat)
	}
	if ta.TotalEnemies != 3 {
		t.Errorf("TotalEnemies = %d, want 3", ta.TotalEnemies)
	}
}

func TestAnalyzeThreatsTankClassification(t *testing.T) {
	// 3 个普通 HP=50，2 个高 HP=200（> 1.5x avg of ~100）
	enemies := []aiplayer.AIEnemy{
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 200, Speed: 1.0, Active: true},
		{HP: 200, Speed: 1.0, Active: true},
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	// avgHP = (50*3 + 200*2) / 5 = 110, tank threshold = 165
	// HP=200 > 165 → 2 tanks
	if ta.TankCount != 2 {
		t.Errorf("TankCount = %d, want 2 (avgHP=%.1f, threshold=%.1f)", ta.TankCount, ta.AvgHP, ta.AvgHP*1.5)
	}
	if ta.DominantThreat != "tank" {
		t.Errorf("DominantThreat = %q, want tank", ta.DominantThreat)
	}
}

func TestAnalyzeThreatsFastClassification(t *testing.T) {
	// 4 个慢速=1.0，2 个快速=3.0
	enemies := []aiplayer.AIEnemy{
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 1.0, Active: true},
		{HP: 50, Speed: 3.0, Active: true},
		{HP: 50, Speed: 3.0, Active: true},
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	// avgSpeed = (1*4 + 3*2) / 6 ≈ 1.67, threshold = 2.0
	// Speed=3.0 > 2.0 → 2 fast
	if ta.FastCount != 2 {
		t.Errorf("FastCount = %d, want 2 (avgSpeed=%.2f)", ta.FastCount, ta.AvgSpeed)
	}
	if ta.DominantThreat != "fast" {
		t.Errorf("DominantThreat = %q, want fast", ta.DominantThreat)
	}
}

func TestAnalyzeThreatsSwarmDetection(t *testing.T) {
	// 12 个相似敌人 → swarm
	enemies := make([]aiplayer.AIEnemy, 12)
	for i := range enemies {
		enemies[i] = aiplayer.AIEnemy{HP: 50, Speed: 1.0, Active: true}
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	if ta.TotalEnemies != 12 {
		t.Errorf("TotalEnemies = %d, want 12", ta.TotalEnemies)
	}
	// 所有敌人相同，没有 tank/fast 主导 → swarm
	if ta.DominantThreat != "swarm" {
		t.Errorf("DominantThreat = %q, want swarm", ta.DominantThreat)
	}
}

func TestAnalyzeThreatsCriticalLevel(t *testing.T) {
	enemies := make([]aiplayer.AIEnemy, 20)
	for i := range enemies {
		enemies[i] = aiplayer.AIEnemy{HP: 50, Speed: 1.0, Active: true}
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	if ta.ThreatLevel != "critical" {
		t.Errorf("ThreatLevel = %q, want critical for 20 enemies", ta.ThreatLevel)
	}
}

func TestAnalyzeThreatsInactiveIgnored(t *testing.T) {
	enemies := []aiplayer.AIEnemy{
		{HP: 100, Speed: 1.0, Active: true},
		{HP: 100, Speed: 1.0, Active: false}, // 不活跃，应被忽略
	}
	ta := aiplayer.AnalyzeThreats(enemies)
	if ta.TotalEnemies != 1 {
		t.Errorf("TotalEnemies = %d, want 1 (inactive should be ignored)", ta.TotalEnemies)
	}
}

// ── DefenseGap 测试 ──

func TestDetectDefenseGapsNeedsCC(t *testing.T) {
	towers := []aiplayer.AITower{
		{Row: 5, Col: 10, Damage: 50, AttackSpeed: 1.0, Abilities: []string{"splash"}},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 3, TankCount: 2, BossCount: 0}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	if !gap.NeedsCC {
		t.Error("NeedsCC should be true when no tower has CC and tanks present")
	}
}

func TestDetectDefenseGapsHasCC(t *testing.T) {
	towers := []aiplayer.AITower{
		{Row: 5, Col: 10, Damage: 50, AttackSpeed: 1.0, Abilities: []string{"slow", "splash"}},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 3, TankCount: 2, BossCount: 0}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	if gap.NeedsCC {
		t.Error("NeedsCC should be false when a tower has slow ability")
	}
}

func TestDetectDefenseGapsNeedsAOE(t *testing.T) {
	towers := []aiplayer.AITower{
		{Row: 5, Col: 10, Damage: 50, AttackSpeed: 1.0, Abilities: []string{"stun"}},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 10}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	if !gap.NeedsAOE {
		t.Error("NeedsAOE should be true when >8 enemies and no AOE tower")
	}
}

func TestDetectDefenseGapsNoAOENeeded(t *testing.T) {
	towers := []aiplayer.AITower{
		{Row: 5, Col: 10, Damage: 50, AttackSpeed: 1.0, Abilities: []string{"splash"}},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 10}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	if gap.NeedsAOE {
		t.Error("NeedsAOE should be false when tower has splash")
	}
}

func TestDetectDefenseGapsNeedsDPS(t *testing.T) {
	// 1 塔 DPS=10，面对大量高 HP 敌人
	towers := []aiplayer.AITower{
		{Row: 5, Col: 10, Damage: 10, AttackSpeed: 1.0},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 10, AvgHP: 500}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	// requiredDPS = 500 * 10 * 0.1 = 500, totalDPS = 10
	if !gap.NeedsDPS {
		t.Error("NeedsDPS should be true when DPS is far below requirement")
	}
}

func TestDetectDefenseGapsCoverageZones(t *testing.T) {
	// 只有出口区域有塔，入口和中段空缺
	towers := []aiplayer.AITower{
		{Row: 5, Col: 18, Damage: 50, AttackSpeed: 1.0},
	}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 3}
	gap := aiplayer.DetectDefenseGaps(towers, threats, 30)
	// Col=18 → exit zone (18/20*30=27, zone=27/10=2=exit)
	// entrance(zone 0) 和 middle(zone 1) 无塔
	if len(gap.CoverageGap) < 1 {
		t.Errorf("CoverageGap should have uncovered zones, got %v", gap.CoverageGap)
	}
}

// ── StrategicAdvice 测试 ──

func TestGenerateAdviceCriticalDPS(t *testing.T) {
	gaps := aiplayer.DefenseGap{NeedsDPS: true, WeakestZone: "entrance"}
	threats := aiplayer.ThreatAnalysis{ThreatLevel: "critical", TotalEnemies: 20}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 10}}
	p := aiplayer.Personality{Aggression: 0.5, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	if advice.Priority != "build_dps" {
		t.Errorf("Priority = %q, want build_dps on critical+needsDPS", advice.Priority)
	}
	if advice.Urgency < 0.9 {
		t.Errorf("Urgency = %.2f, want >= 0.9 on critical threat", advice.Urgency)
	}
}

func TestGenerateAdviceBossUpgrade(t *testing.T) {
	gaps := aiplayer.DefenseGap{NeedsAntiTank: true, WeakestZone: "middle"}
	threats := aiplayer.ThreatAnalysis{BossCount: 1, ThreatLevel: "high", TotalEnemies: 5}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 50}}
	p := aiplayer.Personality{Aggression: 0.5, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	if advice.Priority != "upgrade" {
		t.Errorf("Priority = %q, want upgrade on boss+needsAntiTank", advice.Priority)
	}
	if advice.Urgency < 0.8 {
		t.Errorf("Urgency = %.2f, want >= 0.8 on boss threat", advice.Urgency)
	}
}

func TestGenerateAdviceSwarmAOE(t *testing.T) {
	gaps := aiplayer.DefenseGap{NeedsAOE: true, WeakestZone: "entrance"}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 12, ThreatLevel: "high"}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 30}}
	p := aiplayer.Personality{Aggression: 0.5, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	if advice.Priority != "build_aoe" {
		t.Errorf("Priority = %q, want build_aoe on swarm+needsAOE", advice.Priority)
	}
}

func TestGenerateAdviceBalanced(t *testing.T) {
	gaps := aiplayer.DefenseGap{WeakestZone: "middle"}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 2, ThreatLevel: "medium"}
	towers := []aiplayer.AITower{
		{Row: 3, Col: 5, Damage: 50},
		{Row: 5, Col: 10, Damage: 50},
	}
	p := aiplayer.Personality{Aggression: 0.5, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	// 无缺口、塔不多 → balanced 或 low-urgency
	if advice.Urgency > 0.6 {
		t.Errorf("Urgency = %.2f, want <= 0.6 when no gaps", advice.Urgency)
	}
}

func TestGenerateAdvicePersonalityAggressive(t *testing.T) {
	// 激进型 AI 在缺 CC 但不紧急时应偏向 DPS
	gaps := aiplayer.DefenseGap{NeedsCC: true, WeakestZone: "middle"}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 3, TankCount: 1, ThreatLevel: "medium"}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 30}}
	p := aiplayer.Personality{Aggression: 0.9, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	// 高 Aggression → build_cc 被降级为 build_dps
	if advice.Priority != "build_dps" {
		t.Errorf("Priority = %q, want build_dps for aggressive personality", advice.Priority)
	}
}

func TestGenerateAdvicePersonalityDefensive(t *testing.T) {
	// 防守型 AI 在缺 DPS 但不危急时应偏向 CC
	gaps := aiplayer.DefenseGap{NeedsDPS: true, WeakestZone: "entrance"}
	threats := aiplayer.ThreatAnalysis{TotalEnemies: 4, ThreatLevel: "medium"}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 30}}
	p := aiplayer.Personality{Aggression: 0.1, Economy: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	// 低 Aggression → build_dps 被升级为 build_cc
	if advice.Priority != "build_cc" {
		t.Errorf("Priority = %q, want build_cc for defensive personality", advice.Priority)
	}
}

func TestGenerateAdvicePreferredSlot(t *testing.T) {
	gaps := aiplayer.DefenseGap{NeedsDPS: true, WeakestZone: "exit"}
	threats := aiplayer.ThreatAnalysis{ThreatLevel: "critical", TotalEnemies: 20}
	towers := []aiplayer.AITower{{Row: 5, Col: 10, Damage: 10}}
	p := aiplayer.Personality{Aggression: 0.5}

	advice := aiplayer.GenerateAdvice(gaps, threats, towers, p)
	if advice.PreferredSlot != "exit" {
		t.Errorf("PreferredSlot = %q, want exit (from WeakestZone)", advice.PreferredSlot)
	}
}
