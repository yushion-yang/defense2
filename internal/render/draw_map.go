// draw_map.go — 地图渲染。
// 绘制网格单元格（按类型着色）、路径连线、可建造位置标记。
package render

import (
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/gamemap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 各类型网格单元格的显示颜色。
var cellColors = map[int]color.RGBA{
	config.CellEmpty:     {R: 40, G: 60, B: 40, A: 255},   // 空地：深绿
	config.CellPath:      {R: 80, G: 70, B: 50, A: 255},   // 路径：土黄
	config.CellBuildable:  {R: 50, G: 80, B: 50, A: 255},   // 可建造：浅绿
	config.CellSpawn:     {R: 180, G: 60, B: 60, A: 255},  // 出怪点：红色
	config.CellBase:      {R: 60, G: 60, B: 180, A: 255},  // 基地：蓝色
	config.CellHeroBase:  {R: 200, G: 180, B: 60, A: 255}, // 英雄点：金色
}

// DrawMap 渲染地图网格和路径连线。
func DrawMap(screen *ebiten.Image, gm *gamemap.GameMap) {
	cs := float32(gm.CellSize)
	cfg := gm.Config
	ox := float32(gm.OffsetX)
	oy := float32(gm.OffsetY)

	// 绘制网格单元格
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

	// 绘制路径连线（相邻路径点之间的线段）
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

	// 绘制可建造位置标记（小圆点）
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
