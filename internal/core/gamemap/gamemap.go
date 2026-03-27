package gamemap

import "defense2/internal/config"

// Point represents a pixel coordinate.
type Point struct {
	X, Y float64
}

// GameMap holds the runtime map state derived from a MapConfig.
type GameMap struct {
	Config    *config.MapConfig
	Waypoints []Point // pixel-center waypoints for enemy movement
	CellSize  int
	OffsetX   float64 // horizontal offset to center the map
	OffsetY   float64 // vertical offset to center the map
}

// NewGameMap creates a GameMap from a MapConfig, converting grid cells to pixel coords.
func NewGameMap(cfg *config.MapConfig) *GameMap {
	gm := &GameMap{
		Config:   cfg,
		CellSize: cfg.CellSize,
	}
	gm.Waypoints = gm.pathToPixels(cfg.PathOrder)
	return gm
}

// pathToPixels converts a list of [row, col] pairs into pixel-center Points.
func (gm *GameMap) pathToPixels(path [][2]int) []Point {
	cs := float64(gm.CellSize)
	pts := make([]Point, len(path))
	for i, cell := range path {
		row, col := cell[0], cell[1]
		pts[i] = Point{
			X: gm.OffsetX + float64(col)*cs + cs/2,
			Y: gm.OffsetY + float64(row)*cs + cs/2,
		}
	}
	return pts
}

// CellAt returns the grid value at pixel position (x, y), or -1 if out of bounds.
func (gm *GameMap) CellAt(x, y float64) int {
	cs := float64(gm.CellSize)
	fx := (x - gm.OffsetX) / cs
	fy := (y - gm.OffsetY) / cs
	if fx < 0 || fy < 0 {
		return -1
	}
	col := int(fx)
	row := int(fy)
	if row >= gm.Config.Rows || col >= gm.Config.Cols {
		return -1
	}
	return gm.Config.Grid[row][col]
}

// CellCenter returns the pixel center of a grid cell.
func (gm *GameMap) CellCenter(row, col int) Point {
	cs := float64(gm.CellSize)
	return Point{
		X: gm.OffsetX + float64(col)*cs + cs/2,
		Y: gm.OffsetY + float64(row)*cs + cs/2,
	}
}

// PixelWidth returns the total pixel width of the map grid.
func (gm *GameMap) PixelWidth() float64 {
	return float64(gm.Config.Cols * gm.CellSize)
}

// PixelHeight returns the total pixel height of the map grid.
func (gm *GameMap) PixelHeight() float64 {
	return float64(gm.Config.Rows * gm.CellSize)
}
