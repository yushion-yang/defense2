// gamemap.go — 运行时地图状态。
// 基于 MapConfig 生成像素坐标系的路径点，支持单路径和多路径地图。
// 提供格子查询和坐标转换。
package gamemap

import (
	"math/rand"

	"defense2/internal/config"
)

// Point 像素坐标点。
type Point struct {
	X, Y float64
}

// PathEntry 多路径入口（运行时表示）。
type PathEntry struct {
	ID        string  // 入口标识（如 "top"、"bottom"）
	Waypoints []Point // 该入口的路径点序列
	Weight    float64 // 出怪权重
}

// GameMap 运行时地图，由 MapConfig 派生。
type GameMap struct {
	Config    *config.MapConfig // 原始地图配置
	Waypoints []Point          // 默认路径点序列（单路径或 pathOrder）
	Paths     []PathEntry      // 多路径入口列表（多路径地图使用）
	MultiPath bool             // 是否为多路径地图
	CellSize  int              // 单元格边长（像素）
	OffsetX   float64          // 水平偏移量（用于居中显示）
	OffsetY   float64          // 垂直偏移量（用于居中显示）
	Theme     string           // 地图环境主题（desert/forest/tech/...）
}

// NewGameMap 从 MapConfig 创建运行时地图。
func NewGameMap(cfg *config.MapConfig) *GameMap {
	gm := &GameMap{
		Config:   cfg,
		CellSize: cfg.CellSize,
		Theme:    cfg.Theme,
	}

	// 默认路径
	gm.Waypoints = gm.pathToPixels(cfg.PathOrder)

	// 多路径
	if len(cfg.PathOrders) > 0 && len(cfg.Entries) > 0 {
		gm.MultiPath = true
		gm.Paths = make([]PathEntry, 0, len(cfg.Entries))
		for _, entry := range cfg.Entries {
			path, ok := cfg.PathOrders[entry.ID]
			if !ok {
				continue
			}
			gm.Paths = append(gm.Paths, PathEntry{
				ID:        entry.ID,
				Waypoints: gm.pathToPixels(path),
				Weight:    entry.Weight,
			})
		}
	}

	return gm
}

// PickPath 为一个敌人随机选择一条路径（按权重）。
// 单路径地图返回默认 Waypoints。
func (gm *GameMap) PickPath() []Point {
	if !gm.MultiPath || len(gm.Paths) == 0 {
		return gm.Waypoints
	}

	totalWeight := 0.0
	for _, p := range gm.Paths {
		totalWeight += p.Weight
	}
	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for _, p := range gm.Paths {
		cumulative += p.Weight
		if r <= cumulative {
			return p.Waypoints
		}
	}
	return gm.Paths[len(gm.Paths)-1].Waypoints
}

// GetPathByID 按 ID 查找多路径入口，返回该路径的路径点序列。
// 未找到返回 nil（调用方应 fallback 到默认 Waypoints）。
func (gm *GameMap) GetPathByID(id string) []Point {
	for i := range gm.Paths {
		if gm.Paths[i].ID == id {
			return gm.Paths[i].Waypoints
		}
	}
	return nil
}

// SpawnPoint 返回默认出生点（第一个路径点的像素坐标）。
func (gm *GameMap) SpawnPoint() Point {
	if len(gm.Waypoints) > 0 {
		return gm.Waypoints[0]
	}
	return Point{}
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

