// minimap.go — 右下角小地图 HUD。
// 显示路径（灰线）、敌人（红点）、塔（蓝点）、战灵（紫点）。
package hud

import (
	"image/color"

	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
	"defense2/internal/core/tower"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	minimapW      = 120
	minimapH      = 55
	minimapMargin = 8
	minimapAlpha  = 140
)

// DrawMinimap renders a minimap at the bottom-right corner.
func DrawMinimap(screen *ebiten.Image, gm *gamemap.GameMap, enemies *enemy.Pool, towers *tower.Pool, wardenX, wardenY float64) {
	if gm == nil {
		return
	}

	// Position: bottom-right corner
	ox := float32(theme.CanvasW - minimapW - minimapMargin)
	oy := float32(theme.CanvasH - minimapH - minimapMargin)

	// Background
	draw.FilledRect(screen, ox, oy, minimapW, minimapH,
		color.RGBA{R: 10, G: 15, B: 30, A: minimapAlpha}, true)
	draw.FilledRect(screen, ox, oy, minimapW, 1, color.RGBA{R: 60, G: 80, B: 120, A: 100}, true) // top border

	// Scale factors: map coords → minimap coords
	scaleX := float64(minimapW) / float64(theme.CanvasW)
	scaleY := float64(minimapH) / float64(theme.CanvasH)

	// Path lines (gray)
	wps := gm.Waypoints
	for i := 1; i < len(wps); i++ {
		x1 := float32(float64(ox) + wps[i-1].X*scaleX)
		y1 := float32(float64(oy) + wps[i-1].Y*scaleY)
		x2 := float32(float64(ox) + wps[i].X*scaleX)
		y2 := float32(float64(oy) + wps[i].Y*scaleY)
		draw.Line(screen, x1, y1, x2, y2, 1, color.RGBA{R: 80, G: 100, B: 130, A: 180}, false)
	}

	// Towers (blue dots)
	if towers != nil {
		towers.Each(func(t *tower.Tower) {
			tx := float32(float64(ox) + t.X*scaleX)
			ty := float32(float64(oy) + t.Y*scaleY)
			draw.FilledCircle(screen, tx, ty, 1.5, color.RGBA{R: 60, G: 140, B: 255, A: 220})
		})
	}

	// Enemies (red dots)
	if enemies != nil {
		enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() {
				return
			}
			ex := float32(float64(ox) + e.X*scaleX)
			ey := float32(float64(oy) + e.Y*scaleY)
			r := float32(1.0)
			if e.Boss {
				r = 2.0
			}
			draw.FilledCircle(screen, ex, ey, r, color.RGBA{R: 255, G: 60, B: 60, A: 220})
		})
	}

	// Warden (purple dot)
	if wardenX > 0 && wardenY > 0 {
		wx := float32(float64(ox) + wardenX*scaleX)
		wy := float32(float64(oy) + wardenY*scaleY)
		draw.FilledCircle(screen, wx, wy, 2, color.RGBA{R: 180, G: 100, B: 255, A: 220})
	}
}
