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
		Towers:   nil,
		Wave:     1,
		MaxWaves: 12,
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
		Gold:       5,
		TowerDefs:  []aiplayer.AITowerDef{{Key: "basic", Cost: 50}},
		BuildCells: []aiplayer.AICell{{Row: 3, Col: 15}},
		Towers:     []aiplayer.AITower{{Row: 5, Col: 18, Damage: 30, Strength: 100}},
		Wave:       3,
		MaxWaves:   12,
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
		Wave:     10,
		MaxWaves: 12,
	}
	d := eng.Evaluate(snap)
	if d.Type != aiplayer.DecisionUpgrade {
		t.Errorf("decision = %d, want DecisionUpgrade in late game with no build cells", d.Type)
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
		MapCenterX: 600,
		MapCenterY: 270,
		Wave:       1,
		MaxWaves:   12,
	}
	d := eng.Evaluate(snap)
	if d.Row != 4 || d.Col != 18 {
		t.Errorf("picked (%d,%d), want (4,18) closest to center", d.Row, d.Col)
	}
}
