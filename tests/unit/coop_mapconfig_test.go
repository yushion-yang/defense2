//go:build unittest

package unit

import (
	"encoding/json"
	"testing"

	"defense2/internal/config"
)

func TestMapConfigCoopParsing(t *testing.T) {
	raw := `{
		"id": "map_co01", "cols": 28, "rows": 13, "cellSize": 50,
		"coop": {
			"playerCount": 2,
			"sections": [
				{"id": "A", "colStart": 0, "colEnd": 14, "theme": "forest", "owner": 0},
				{"id": "B", "colStart": 14, "colEnd": 28, "theme": "ice", "owner": 1}
			]
		},
		"grid": [], "pathOrder": []
	}`
	var mc config.MapConfig
	if err := json.Unmarshal([]byte(raw), &mc); err != nil {
		t.Fatal(err)
	}
	if mc.Coop == nil {
		t.Fatal("Coop should not be nil")
	}
	if mc.Coop.PlayerCount != 2 {
		t.Errorf("PlayerCount = %d, want 2", mc.Coop.PlayerCount)
	}
	if len(mc.Coop.Sections) != 2 {
		t.Fatalf("Sections len = %d, want 2", len(mc.Coop.Sections))
	}
	s := mc.Coop.Sections[1]
	if s.Theme != "ice" {
		t.Errorf("Section B theme = %q, want ice", s.Theme)
	}
	if s.Owner != 1 {
		t.Errorf("Section B owner = %d, want 1", s.Owner)
	}
	if s.ColStart != 14 || s.ColEnd != 28 {
		t.Errorf("Section B cols = [%d,%d), want [14,28)", s.ColStart, s.ColEnd)
	}
}

func TestMapConfigNonCoopNil(t *testing.T) {
	raw := `{"id": "map_01", "cols": 24, "rows": 13, "cellSize": 60, "grid": [], "pathOrder": []}`
	var mc config.MapConfig
	if err := json.Unmarshal([]byte(raw), &mc); err != nil {
		t.Fatal(err)
	}
	if mc.Coop != nil {
		t.Error("Coop should be nil for non-coop map")
	}
}
