package config

import (
	"testing"
)

func TestParseScenarioData(t *testing.T) {
	raw := []byte(`{
		"id": "test-save",
		"name": "Test Save",
		"description": "unit test",
		"mapID": "map_test",
		"gold": 9999,
		"lives": 999,
		"waves": 10,
		"enemyFilter": "mixed",
		"manualWave": true,
		"towers": [{
			"row": 3, "col": 5,
			"key": "prism",
			"abilitySlots": ["scatter","","auraDamage","","",""],
			"damageTier": "A", "speedTier": "B", "rangeTier": "S",
			"baseDamage": 12.0, "potentialDamage": 8.0,
			"baseSpeed": 1.5, "potentialSpeed": 0.5,
			"baseRange": 120.0, "potentialRange": 30.0
		}]
	}`)
	sd, err := ParseScenarioData(raw)
	if err != nil {
		t.Fatalf("ParseScenarioData failed: %v", err)
	}
	if sd.ID != "test-save" {
		t.Errorf("ID = %q, want %q", sd.ID, "test-save")
	}
	if sd.MapID != "map_test" {
		t.Errorf("MapID = %q, want %q", sd.MapID, "map_test")
	}
	if sd.Gold != 9999 {
		t.Errorf("Gold = %d, want %d", sd.Gold, 9999)
	}
	if !sd.ManualWave {
		t.Error("ManualWave = false, want true")
	}
	if len(sd.Towers) != 1 {
		t.Fatalf("Towers len = %d, want 1", len(sd.Towers))
	}
	tw := sd.Towers[0]
	if tw.Row != 3 || tw.Col != 5 {
		t.Errorf("Tower pos = (%d,%d), want (3,5)", tw.Row, tw.Col)
	}
	if tw.Key != "prism" {
		t.Errorf("Tower key = %q, want %q", tw.Key, "prism")
	}
	wantSlots := [6]string{"scatter", "", "auraDamage", "", "", ""}
	if tw.AbilitySlots != wantSlots {
		t.Errorf("AbilitySlots = %v, want %v", tw.AbilitySlots, wantSlots)
	}
	if tw.DamageTier != "A" {
		t.Errorf("DamageTier = %q, want %q", tw.DamageTier, "A")
	}
	if diff := tw.BaseDamage - 12.0; diff > 0.01 || diff < -0.01 {
		t.Errorf("BaseDamage = %f, want 12.0", tw.BaseDamage)
	}
}

func TestParseScenarioDataEmpty(t *testing.T) {
	raw := []byte(`{"id":"empty","name":"Empty","mapID":"map_test","gold":100,"lives":10,"waves":5,"towers":[]}`)
	sd, err := ParseScenarioData(raw)
	if err != nil {
		t.Fatalf("ParseScenarioData failed: %v", err)
	}
	if len(sd.Towers) != 0 {
		t.Errorf("Towers len = %d, want 0", len(sd.Towers))
	}
}

func TestParseScenarioDataWithEnemies(t *testing.T) {
	raw := []byte(`{
		"id": "with-enemies",
		"name": "With Enemies",
		"mapID": "map_test",
		"gold": 9999, "lives": 999, "waves": 10,
		"towers": [],
		"enemies": [
			{"archetype": "orc_warrior", "x": 100.5, "y": 200.0, "pathIndex": 3, "hp": 500, "maxHP": 1000},
			{"archetype": "runner", "x": 300.0, "y": 150.0, "pathIndex": 1}
		]
	}`)
	sd, err := ParseScenarioData(raw)
	if err != nil {
		t.Fatalf("ParseScenarioData failed: %v", err)
	}
	if len(sd.Enemies) != 2 {
		t.Fatalf("Enemies len = %d, want 2", len(sd.Enemies))
	}
	e0 := sd.Enemies[0]
	if e0.Archetype != "orc_warrior" {
		t.Errorf("Enemy[0].Archetype = %q, want %q", e0.Archetype, "orc_warrior")
	}
	if e0.PathIndex != 3 {
		t.Errorf("Enemy[0].PathIndex = %d, want 3", e0.PathIndex)
	}
	if diff := e0.HP - 500; diff > 0.01 || diff < -0.01 {
		t.Errorf("Enemy[0].HP = %f, want 500", e0.HP)
	}
	if diff := e0.MaxHP - 1000; diff > 0.01 || diff < -0.01 {
		t.Errorf("Enemy[0].MaxHP = %f, want 1000", e0.MaxHP)
	}
	// Second enemy has no HP fields → should be zero
	e1 := sd.Enemies[1]
	if e1.HP != 0 {
		t.Errorf("Enemy[1].HP = %f, want 0 (omitted)", e1.HP)
	}
}

func TestParseScenarioDataNoEnemies(t *testing.T) {
	// Old format without enemies field should work (backward compatible)
	raw := []byte(`{"id":"old","name":"Old","mapID":"map_test","gold":100,"lives":10,"waves":5,"towers":[]}`)
	sd, err := ParseScenarioData(raw)
	if err != nil {
		t.Fatalf("ParseScenarioData failed: %v", err)
	}
	if len(sd.Enemies) != 0 {
		t.Errorf("Enemies len = %d, want 0", len(sd.Enemies))
	}
}

func TestParseScenarioDataInvalid(t *testing.T) {
	_, err := ParseScenarioData([]byte(`{invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
