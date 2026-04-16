//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestDecisionBuildWhenRich(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:      200,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
		BuildCells: []aiplayer.AICell{
			{Row: 3, Col: 15, X: 900, Y: 180},
			{Row: 5, Col: 18, X: 1080, Y: 300},
		},
		Towers:      nil,
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionBuild {
		t.Errorf("decision = %d, want DecisionBuild when gold=200 and no towers", d.Type)
	}
	if d.TowerKey != "basic" {
		t.Errorf("towerKey = %q, want basic", d.TowerKey)
	}
}

func TestDecisionIdleWhenBroke(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:        5,
		TowerDefs:   []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells:  []aiplayer.AICell{{Row: 3, Col: 15}},
		Towers:      []aiplayer.AITower{{Row: 5, Col: 18, Damage: 30, Strength: 100}},
		Wave:        3,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionIdle {
		t.Errorf("decision = %d, want DecisionIdle when gold=5", d.Type)
	}
}

func TestDecisionUpgradeLateGame(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:       100,
		TowerDefs:  []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells: nil, // 没有可建造格子
		Towers: []aiplayer.AITower{
			{Row: 5, Col: 18, Damage: 50, Strength: 100},
		},
		Wave:        10,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionUpgrade {
		t.Errorf("decision = %d, want DecisionUpgrade in late game with no build cells", d.Type)
	}
}

// TestDecisionChooseAbilityPriority 验证有待选能力时优先于造塔/升级。
func TestDecisionChooseAbilityPriority(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:      200,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
		BuildCells: []aiplayer.AICell{
			{Row: 3, Col: 15, X: 900, Y: 180},
		},
		Towers: []aiplayer.AITower{
			{
				Row: 5, Col: 18, Damage: 30, Strength: 100, Owner: 1,
				PendingSlots: []aiplayer.AIPendingSlot{
					{
						SlotIndex: 0,
						Choices: []aiplayer.AIAbilityChoice{
							{Name: "scatter", Label: "Scatter", Category: "attack"},
							{Name: "wideBeam", Label: "Prism", Category: "attack"},
							{Name: "bounce", Label: "Ricochet", Category: "attack"},
						},
					},
				},
			},
		},
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionChooseAbility {
		t.Errorf("decision = %d, want DecisionChooseAbility when tower has pending slots", d.Type)
	}
	if d.Row != 5 || d.Col != 18 {
		t.Errorf("target (%d,%d), want (5,18)", d.Row, d.Col)
	}
	// 选中的能力应该是三个候选之一
	validChoices := map[string]bool{"scatter": true, "wideBeam": true, "bounce": true}
	if !validChoices[d.AbilityName] {
		t.Errorf("abilityName = %q, want one of scatter/wideBeam/bounce", d.AbilityName)
	}
	if d.SlotIndex != 0 {
		t.Errorf("slotIndex = %d, want 0", d.SlotIndex)
	}
}

// TestDecisionChooseAbilityNotWhenEmpty 验证无待选能力时不选能力。
func TestDecisionChooseAbilityNotWhenEmpty(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:      200,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
		BuildCells: []aiplayer.AICell{
			{Row: 3, Col: 15, X: 900, Y: 180},
		},
		Towers: []aiplayer.AITower{
			{Row: 5, Col: 18, Damage: 30, Strength: 100, Owner: 1},
		},
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Type == aiplayer.DecisionChooseAbility {
		t.Error("should not choose ability when no pending slots")
	}
}

// TestDecisionChooseAbilityCCScoring 验证 CC 能力在低 Aggression 时评分更高。
func TestDecisionChooseAbilityCCScoring(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	// 低 Aggression → 偏好 CC
	eng.SetPersonality(aiplayer.Personality{Aggression: 0.1, Economy: 0.5, Risk: 0.5, Reaction: 0.5, Compliance: 0.5})

	snap := aiplayer.AISnapshot{
		Gold: 200,
		Towers: []aiplayer.AITower{
			{
				Row: 5, Col: 18, Damage: 30, Strength: 100, Owner: 1,
				PendingSlots: []aiplayer.AIPendingSlot{
					{
						SlotIndex: 1,
						Choices: []aiplayer.AIAbilityChoice{
							{Name: "stun", Label: "Stun", Category: "cc"},
							{Name: "crit", Label: "Crit", Category: "damage"},
						},
					},
				},
			},
		},
		WardenReady: true,
		WaveActive:  true,
	}

	// 多次评估以统计偏好（由于有随机噪声，用统计验证趋势）
	ccCount, dmgCount := 0, 0
	for i := 0; i < 200; i++ {
		d := eng.Evaluate(snap)
		if d.Type == aiplayer.DecisionChooseAbility {
			switch d.AbilityName {
			case "stun":
				ccCount++
			case "crit":
				dmgCount++
			}
		}
	}
	// 低 Aggression AI 应该偏好 CC（至少 55% 以上选 CC）
	if ccCount < dmgCount {
		t.Errorf("low aggression AI picked cc=%d, damage=%d; expected cc > damage", ccCount, dmgCount)
	}
}

func TestDecisionPicksBestCell(t *testing.T) {
	eng := aiplayer.NewDecisionEngine()
	snap := aiplayer.AISnapshot{
		Gold:      200,
		TowerDefs: []aiplayer.AITowerDef{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
		BuildCells: []aiplayer.AICell{
			{Row: 0, Col: 15, X: 900, Y: 0},   // 远离中心
			{Row: 4, Col: 18, X: 600, Y: 270},  // 接近中心
			{Row: 12, Col: 20, X: 900, Y: 540}, // 远离中心
		},
		MapCenterX:  600,
		MapCenterY:  270,
		Wave:        1,
		MaxWaves:    12,
		WardenReady: true,
		WaveActive:  true,
	}
	d := eng.Evaluate(snap)
	if d.Row != 4 || d.Col != 18 {
		t.Errorf("picked (%d,%d), want (4,18) closest to center", d.Row, d.Col)
	}
}
