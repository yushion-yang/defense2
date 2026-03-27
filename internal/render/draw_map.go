package render

import (
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Cell colors by type.
var cellColors = map[int]color.RGBA{
	config.CellEmpty:     {R: 40, G: 60, B: 40, A: 255},
	config.CellPath:      {R: 80, G: 70, B: 50, A: 255},
	config.CellBuildable:  {R: 50, G: 80, B: 50, A: 255},
	config.CellSpawn:     {R: 180, G: 60, B: 60, A: 255},
	config.CellBase:      {R: 60, G: 60, B: 180, A: 255},
	config.CellHeroBase:  {R: 200, G: 180, B: 60, A: 255},
}

// DrawMap renders the grid and path waypoints.
func DrawMap(screen *ebiten.Image, gm *gamemap.GameMap) {
	cs := float32(gm.CellSize)
	cfg := gm.Config
	ox := float32(gm.OffsetX)
	oy := float32(gm.OffsetY)

	// Draw grid cells
	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			cellType := cfg.Grid[row][col]
			clr, ok := cellColors[cellType]
			if !ok {
				clr = cellColors[config.CellEmpty]
			}
			x := ox + float32(col)*cs
			y := oy + float32(row)*cs
			vector.DrawFilledRect(screen, x, y, cs-1, cs-1, clr, false)
		}
	}

	// Draw path line connecting waypoints
	if len(gm.Waypoints) > 1 {
		pathColor := color.RGBA{R: 160, G: 140, B: 100, A: 200}
		for i := 0; i < len(gm.Waypoints)-1; i++ {
			p1 := gm.Waypoints[i]
			p2 := gm.Waypoints[i+1]
			vector.StrokeLine(screen,
				float32(p1.X), float32(p1.Y),
				float32(p2.X), float32(p2.Y),
				2, pathColor, false)
		}
	}

	// Draw buildable slot markers
	buildColor := color.RGBA{R: 100, G: 160, B: 100, A: 120}
	for row := 0; row < cfg.Rows; row++ {
		for col := 0; col < cfg.Cols; col++ {
			if cfg.Grid[row][col] == config.CellBuildable {
				cx := ox + float32(col)*cs + cs/2
				cy := oy + float32(row)*cs + cs/2
				vector.DrawFilledCircle(screen, cx, cy, cs/4, buildColor, false)
			}
		}
	}
}
