package render

import (
	"image/color"
	"math"

	"defense2/internal/core/tower"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawTowers renders all active towers.
func DrawTowers(screen *ebiten.Image, pool *tower.Pool) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)

		// Base square
		size := float32(20)
		vector.DrawFilledRect(screen, cx-size/2, cy-size/2, size, size,
			color.RGBA{R: 80, G: 140, B: 220, A: 255}, false)

		// Range circle (subtle)
		drawCircleOutline(screen, cx, cy, float32(t.Range), 1,
			color.RGBA{R: 80, G: 140, B: 220, A: 40})
	})
}

// DrawTowerRangePreview draws a range circle for tower placement preview.
func DrawTowerRangePreview(screen *ebiten.Image, cx, cy float32, r float64, valid bool) {
	clr := color.RGBA{R: 100, G: 200, B: 100, A: 80}
	if !valid {
		clr = color.RGBA{R: 200, G: 80, B: 80, A: 80}
	}
	drawCircleOutline(screen, cx, cy, float32(r), 1.5, clr)

	size := float32(20)
	fillClr := color.RGBA{R: 100, G: 200, B: 100, A: 120}
	if !valid {
		fillClr = color.RGBA{R: 200, G: 80, B: 80, A: 120}
	}
	vector.DrawFilledRect(screen, cx-size/2, cy-size/2, size, size, fillClr, false)
}

func drawCircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.RGBA) {
	const segments = 48
	for i := 0; i < segments; i++ {
		a1 := float64(i) * 2 * math.Pi / segments
		a2 := float64(i+1) * 2 * math.Pi / segments
		x1 := cx + r*float32(math.Cos(a1))
		y1 := cy + r*float32(math.Sin(a1))
		x2 := cx + r*float32(math.Cos(a2))
		y2 := cy + r*float32(math.Sin(a2))
		vector.StrokeLine(screen, x1, y1, x2, y2, width, clr, false)
	}
}
