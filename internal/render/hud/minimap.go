// minimap.go — 右上角小地图 HUD。
// 显示路径（灰线）、敌人（红点）、塔（蓝点）、战灵（紫点）。
package hud

import (
	"image/color"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	minimapMargin = 8
	minimapAlpha  = 140
)

// MinimapVM 小地图展示数据（纯值类型，无核心依赖）。
type MinimapVM struct {
	PathPoints []MinimapPoint // 路径点序列
	Towers     []MinimapPoint // 塔位置
	Enemies    []MinimapDot   // 敌人位置
	WardenX    float64        // 战灵 X（0=不显示）
	WardenY    float64        // 战灵 Y
}

// MinimapPoint 小地图上的一个坐标点。
type MinimapPoint struct{ X, Y float64 }

// MinimapDot 小地图上的敌人点（区分 Boss）。
type MinimapDot struct {
	X, Y   float64
	IsBoss bool
}

// DrawMinimap renders a minimap at the top-right corner.
func DrawMinimap(screen *ebiten.Image, vm MinimapVM) {
	// Position: top-right corner (below TopBar)
	ox := float32(theme.CanvasW-theme.MinimapW) - minimapMargin
	oy := float32(theme.TopBarY+theme.TopBarH) + minimapMargin

	// Background
	ui.Panel(screen, ox, oy, theme.MinimapW, theme.MinimapH, ui.PanelStyle{
		BgColor: color.RGBA{R: 10, G: 15, B: 30, A: minimapAlpha}, Radius: 0,
	})
	// 顶部边线用 nolint 豁免（非 HUD 组件，纯视觉装饰）
	draw.FilledRect(screen, ox, oy, theme.MinimapW, 1, color.RGBA{R: 60, G: 80, B: 120, A: 100}, true) //nolint:hud

	scaleX := float64(theme.MinimapW) / float64(theme.CanvasW)
	scaleY := float64(theme.MinimapH) / float64(theme.CanvasH)

	// 以下均为地图图形绘制（路径线段/实体点位），非 HUD 组件
	for i := 1; i < len(vm.PathPoints); i++ { //nolint:hud
		x1 := float32(float64(ox) + vm.PathPoints[i-1].X*scaleX)
		y1 := float32(float64(oy) + vm.PathPoints[i-1].Y*scaleY)
		x2 := float32(float64(ox) + vm.PathPoints[i].X*scaleX)
		y2 := float32(float64(oy) + vm.PathPoints[i].Y*scaleY)
		draw.Line(screen, x1, y1, x2, y2, 1, color.RGBA{R: 80, G: 100, B: 130, A: 180}, false) //nolint:hud
	}

	for _, t := range vm.Towers {
		tx := float32(float64(ox) + t.X*scaleX)
		ty := float32(float64(oy) + t.Y*scaleY)
		draw.FilledCircle(screen, tx, ty, 1.5, color.RGBA{R: 60, G: 140, B: 255, A: 220}) //nolint:hud
	}

	for _, e := range vm.Enemies {
		ex := float32(float64(ox) + e.X*scaleX)
		ey := float32(float64(oy) + e.Y*scaleY)
		r := float32(1.0)
		if e.IsBoss {
			r = 2.0
		}
		draw.FilledCircle(screen, ex, ey, r, color.RGBA{R: 255, G: 60, B: 60, A: 220}) //nolint:hud
	}

	if vm.WardenX > 0 && vm.WardenY > 0 {
		wx := float32(float64(ox) + vm.WardenX*scaleX)
		wy := float32(float64(oy) + vm.WardenY*scaleY)
		draw.FilledCircle(screen, wx, wy, 2, color.RGBA{R: 180, G: 100, B: 255, A: 220}) //nolint:hud
	}
}
