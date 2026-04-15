//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestCoopZone2P(t *testing.T) {
	sections := []aiplayer.ZoneSection{
		{ID: "A", ColStart: 0, ColEnd: 14, Owner: 0},
		{ID: "B", ColStart: 14, ColEnd: 28, Owner: 1},
	}
	z := aiplayer.NewCoopZone(28, 13, sections)

	tests := []struct {
		name     string
		row, col int
		want     int
	}{
		{"human left", 5, 3, 0},
		{"human boundary", 5, 13, 0},
		{"ai left", 5, 14, 1},
		{"ai right", 5, 27, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := z.OwnerOf(tt.row, tt.col)
			if got != tt.want {
				t.Errorf("OwnerOf(%d,%d) = %d, want %d", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

func TestCoopZone4P(t *testing.T) {
	sections := []aiplayer.ZoneSection{
		{ID: "A", ColStart: 0, ColEnd: 13, RowStart: 0, RowEnd: 13, Owner: 0},
		{ID: "B", ColStart: 13, ColEnd: 26, RowStart: 0, RowEnd: 13, Owner: 1},
		{ID: "C", ColStart: 0, ColEnd: 13, RowStart: 13, RowEnd: 26, Owner: 2},
		{ID: "D", ColStart: 13, ColEnd: 26, RowStart: 13, RowEnd: 26, Owner: 3},
	}
	z := aiplayer.NewCoopZone(26, 26, sections)

	if z.OwnerOf(5, 5) != 0 {
		t.Error("top-left should be owner 0")
	}
	if z.OwnerOf(5, 20) != 1 {
		t.Error("top-right should be owner 1")
	}
	if z.OwnerOf(20, 5) != 2 {
		t.Error("bottom-left should be owner 2")
	}
	if z.OwnerOf(20, 20) != 3 {
		t.Error("bottom-right should be owner 3")
	}
	if z.PlayerCount() != 4 {
		t.Errorf("PlayerCount = %d, want 4", z.PlayerCount())
	}
}

func TestCoopZoneBuildCells(t *testing.T) {
	sections := []aiplayer.ZoneSection{
		{ID: "A", ColStart: 0, ColEnd: 14, Owner: 0},
		{ID: "B", ColStart: 14, ColEnd: 28, Owner: 1},
	}
	z := aiplayer.NewCoopZone(28, 13, sections)
	grid := make([][]int, 13)
	for r := range grid {
		grid[r] = make([]int, 28)
		for c := 0; c < 28; c += 3 {
			grid[r][c] = 2
		}
	}
	cells := z.BuildCellsForOwner(grid, 1)
	if len(cells) == 0 {
		t.Fatal("expected non-empty cells for owner 1")
	}
	for _, c := range cells {
		if c.Col < 14 {
			t.Errorf("owner 1 cell at col %d, should be >= 14", c.Col)
		}
	}
}

func TestCoopZoneBoundaries(t *testing.T) {
	sections := []aiplayer.ZoneSection{
		{ID: "A", ColStart: 0, ColEnd: 14, Owner: 0},
		{ID: "B", ColStart: 14, ColEnd: 28, Owner: 1},
	}
	z := aiplayer.NewCoopZone(28, 13, sections)
	xs := z.SectionBoundaries(50)
	if len(xs) != 1 {
		t.Fatalf("boundaries = %v, want 1 boundary at x=700", xs)
	}
	if xs[0] != 700 {
		t.Errorf("boundary = %f, want 700", xs[0])
	}
}
