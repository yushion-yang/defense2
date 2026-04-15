//go:build unittest

package unit

import (
	"testing"

	"defense2/internal/core/aiplayer"
)

func TestZoneSplit(t *testing.T) {
	z := aiplayer.NewZone(24, 13)

	tests := []struct {
		name     string
		row, col int
		want     int
	}{
		{"player left edge", 5, 0, int(aiplayer.ZoneHuman)},
		{"player right boundary", 5, 11, int(aiplayer.ZoneHuman)},
		{"ai left boundary", 5, 12, int(aiplayer.ZoneAI)},
		{"ai right edge", 5, 23, int(aiplayer.ZoneAI)},
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

func TestZoneBuildCells(t *testing.T) {
	z := aiplayer.NewZone(24, 13)
	grid := make([][]int, 13)
	for r := range grid {
		grid[r] = make([]int, 24)
		for c := range grid[r] {
			if c%3 == 0 {
				grid[r][c] = 2
			}
		}
	}
	aiCells := z.BuildableCells(grid, int(aiplayer.ZoneAI))
	for _, c := range aiCells {
		if c.Col < 12 {
			t.Errorf("AI buildable cell at col %d, should be >= 12", c.Col)
		}
	}
	humanCells := z.BuildableCells(grid, int(aiplayer.ZoneHuman))
	for _, c := range humanCells {
		if c.Col >= 12 {
			t.Errorf("Human buildable cell at col %d, should be < 12", c.Col)
		}
	}
	if len(aiCells) == 0 {
		t.Error("expected non-empty AI buildable cells")
	}
	if len(humanCells) == 0 {
		t.Error("expected non-empty human buildable cells")
	}
}
