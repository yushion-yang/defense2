package core_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"
)

func TestNewGameMapWaypoints(t *testing.T) {
	m, err := config.LoadMap("map_01")
	if err != nil {
		t.Fatalf("load map: %v", err)
	}
	gm := gamemap.NewGameMap(m)

	if len(gm.Waypoints) != len(m.PathOrder) {
		t.Fatalf("waypoint count mismatch: %d vs %d", len(gm.Waypoints), len(m.PathOrder))
	}

	// First waypoint should be center of first path cell
	first := gm.Waypoints[0]
	cs := float64(m.CellSize)
	expectedX := float64(m.PathOrder[0][1])*cs + cs/2
	expectedY := float64(m.PathOrder[0][0])*cs + cs/2
	if first.X != expectedX || first.Y != expectedY {
		t.Fatalf("first waypoint expected (%.0f, %.0f), got (%.0f, %.0f)",
			expectedX, expectedY, first.X, first.Y)
	}
}

func TestCellAt(t *testing.T) {
	m, err := config.LoadMap("map_01")
	if err != nil {
		t.Fatalf("load map: %v", err)
	}
	gm := gamemap.NewGameMap(m)

	// Cell [4][3] is spawn (4) — map_01 is 24x13, spawn at row=4 col=3
	cell := gm.CellAt(210, 270) // col=3 center=210, row=4 center=270
	if cell != config.CellSpawn {
		t.Fatalf("expected CellSpawn (4) at [4][3], got %d", cell)
	}

	// Out of bounds
	cell = gm.CellAt(-10, -10)
	if cell != -1 {
		t.Fatalf("expected -1 for out of bounds, got %d", cell)
	}
}

func TestPixelDimensions(t *testing.T) {
	m, err := config.LoadMap("map_01")
	if err != nil {
		t.Fatalf("load map: %v", err)
	}
	gm := gamemap.NewGameMap(m)

	expectedW := float64(24 * 60) // map_01: 24 cols × 60
	expectedH := float64(13 * 60) // map_01: 13 rows × 60
	if gm.PixelWidth() != expectedW {
		t.Fatalf("expected width %.0f, got %.0f", expectedW, gm.PixelWidth())
	}
	if gm.PixelHeight() != expectedH {
		t.Fatalf("expected height %.0f, got %.0f", expectedH, gm.PixelHeight())
	}
}
