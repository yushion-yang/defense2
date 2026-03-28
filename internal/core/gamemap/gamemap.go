// gamemap.go — 运行时地图状态。
// 基于 MapConfig 生成像素坐标系的路径点，提供格子查询和坐标转换。
package gamemap

import "defense2/internal/config"

// Point 像素坐标点。
type Point struct {
	X, Y float64
}

// GameMap 运行时地图，由 MapConfig 派生。
type GameMap struct {
	Config    *config.MapConfig // 原始地图配置
	Waypoints []Point          // 像素中心路径点序列，敌人沿此移动
	CellSize  int              // 单元格边长（像素）
	OffsetX   float64          // 水平偏移量（用于居中显示）
	OffsetY   float64          // 垂直偏移量（用于居中显示）
}

// NewGameMap 从 MapConfig 创建运行时地图，将网格坐标转换为像素坐标。
func NewGameMap(cfg *config.MapConfig) *GameMap {
	gm := &GameMap{
		Config:   cfg,
		CellSize: cfg.CellSize,
	}
	gm.Waypoints = gm.pathToPixels(cfg.PathOrder)
	return gm
}

// pathToPixels 将 [row, col] 路径序列转换为像素中心坐标序列。
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

// CellAt 返回像素坐标 (x, y) 对应的网格类型值，越界返回 -1。
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

// CellCenter 返回指定网格格子的像素中心坐标。
func (gm *GameMap) CellCenter(row, col int) Point {
	cs := float64(gm.CellSize)
	return Point{
		X: gm.OffsetX + float64(col)*cs + cs/2,
		Y: gm.OffsetY + float64(row)*cs + cs/2,
	}
}

// PixelWidth 返回地图网格的总像素宽度。
func (gm *GameMap) PixelWidth() float64 {
	return float64(gm.Config.Cols * gm.CellSize)
}

// PixelHeight 返回地图网格的总像素高度。
func (gm *GameMap) PixelHeight() float64 {
	return float64(gm.Config.Rows * gm.CellSize)
}
