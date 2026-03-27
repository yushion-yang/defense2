package render

import (
	"image/color"

	"defense2/internal/core/enemy"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawEnemies renders all active enemies as circles with HP bars.
func DrawEnemies(screen *ebiten.Image, pool *enemy.Pool) {
	pool.Each(func(e *enemy.Enemy) {
		cx := float32(e.X)
		cy := float32(e.Y)
		r := float32(e.Radius)

		// Body
		bodyColor := color.RGBA{R: 200, G: 60, B: 60, A: 255}
		vector.DrawFilledCircle(screen, cx, cy, r, bodyColor, false)

		// HP bar (above enemy)
		if e.HP < e.MaxHP {
			barW := r * 2.5
			barH := float32(3)
			barX := cx - barW/2
			barY := cy - r - 6

			// Background
			vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{R: 60, G: 20, B: 20, A: 200}, false)
			// Fill
			fill := barW * float32(e.HP/e.MaxHP)
			vector.DrawFilledRect(screen, barX, barY, fill, barH, color.RGBA{R: 60, G: 200, B: 60, A: 255}, false)
		}
	})
}
