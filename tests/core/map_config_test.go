package core_test

import (
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
}

func TestLoadMap01(t *testing.T) {
	m, err := config.LoadMap("map_01")
	if err != nil {
		t.Fatalf("failed to load map_01: %v", err)
	}
	if m.ID != "map_01" {
		t.Fatalf("expected id map_01, got %s", m.ID)
	}
	if m.Cols <= 0 || m.Rows <= 0 {
		t.Fatalf("地图尺寸应 > 0, 实际 %dx%d", m.Cols, m.Rows)
	}
	if m.CellSize <= 0 {
		t.Fatalf("cellSize 应 > 0, 实际 %d", m.CellSize)
	}
	if len(m.Grid) != m.Rows {
		t.Fatalf("grid rows mismatch: %d vs %d", len(m.Grid), m.Rows)
	}
	if len(m.Grid[0]) != m.Cols {
		t.Fatalf("grid cols mismatch: %d vs %d", len(m.Grid[0]), m.Cols)
	}
	if len(m.PathOrder) == 0 {
		t.Fatal("pathOrder should not be empty")
	}
	// First cell should be spawn (4)
	spawn := m.PathOrder[0]
	if m.Grid[spawn[0]][spawn[1]] != config.CellSpawn {
		t.Fatalf("first path cell should be spawn (4), got %d", m.Grid[spawn[0]][spawn[1]])
	}
	// Last cell should be base (5)
	last := m.PathOrder[len(m.PathOrder)-1]
	if m.Grid[last[0]][last[1]] != config.CellBase {
		t.Fatalf("last path cell should be base (5), got %d", m.Grid[last[0]][last[1]])
	}
}

func TestLoadMapNotFound(t *testing.T) {
	_, err := config.LoadMap("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent map")
	}
}
