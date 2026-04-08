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

func TestParseScenarioDataInvalid(t *testing.T) {
	_, err := ParseScenarioData([]byte(`{invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
