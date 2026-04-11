package contracts

import (
	"encoding/json"
	"testing"

	"defense2/internal/config"
)

func TestScenarioRoundTrip(t *testing.T) {
	original := config.ScenarioData{
		ID:          "roundtrip-test",
		Name:        "Round Trip",
		Description: "test",
		MapID:       "map_test",
		Gold:        9999,
		Lives:       999,
		Waves:       10,
		EnemyFilter: "mixed",
		ManualWave:  true,
		Towers: []config.TowerSnapshot{
			{
				Row: 3, Col: 5, Key: "prism",
				AbilitySlots:    [6]string{"scatter", "", "auraDamage", "", "", ""},
				DamageTier:      "S",
				SpeedTier:       "B",
				RangeTier:       "D",
				BaseDamage:      12.0,
				PotentialDamage: 8.0,
				BaseSpeed:       1.5,
				PotentialSpeed:  0.5,
				BaseRange:       120.0,
				PotentialRange:  30.0,
				Specialty:       2,
			},
			{
				Row: 5, Col: 7, Key: "sentinel",
				AbilitySlots:    [6]string{"enhance", "slow", "", "", "", ""},
				DamageTier:      "B",
				SpeedTier:       "S",
				RangeTier:       "B",
				BaseDamage:      8.0,
				PotentialDamage: 6.0,
				BaseSpeed:       2.0,
				PotentialSpeed:  0.8,
				BaseRange:       100.0,
				PotentialRange:  15.0,
				Specialty:       1,
			},
		},
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Parse back
	parsed, err := config.ParseScenarioData(data)
	if err != nil {
		t.Fatalf("ParseScenarioData failed: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, original.ID)
	}
	if parsed.MapID != original.MapID {
		t.Errorf("MapID = %q, want %q", parsed.MapID, original.MapID)
	}
	if parsed.Gold != original.Gold {
		t.Errorf("Gold = %d, want %d", parsed.Gold, original.Gold)
	}
	if parsed.ManualWave != original.ManualWave {
		t.Errorf("ManualWave = %v, want %v", parsed.ManualWave, original.ManualWave)
	}
	if len(parsed.Towers) != 2 {
		t.Fatalf("Towers len = %d, want 2", len(parsed.Towers))
	}

	// Verify tower 1
	if parsed.Towers[0].Key != "prism" {
		t.Errorf("Tower[0].Key = %q, want %q", parsed.Towers[0].Key, "prism")
	}
	if parsed.Towers[0].AbilitySlots != original.Towers[0].AbilitySlots {
		t.Errorf("Tower[0].AbilitySlots = %v, want %v", parsed.Towers[0].AbilitySlots, original.Towers[0].AbilitySlots)
	}
	if diff := parsed.Towers[0].BaseDamage - 12.0; diff > 0.001 || diff < -0.001 {
		t.Errorf("Tower[0].BaseDamage = %f, want 12.0", parsed.Towers[0].BaseDamage)
	}

	// Verify tower 2
	if parsed.Towers[1].Key != "sentinel" {
		t.Errorf("Tower[1].Key = %q, want %q", parsed.Towers[1].Key, "sentinel")
	}
	if parsed.Towers[1].AbilitySlots != original.Towers[1].AbilitySlots {
		t.Errorf("Tower[1].AbilitySlots = %v, want %v", parsed.Towers[1].AbilitySlots, original.Towers[1].AbilitySlots)
	}
}
