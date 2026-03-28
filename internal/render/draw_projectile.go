// draw_projectile.go — projectile rendering.
// Dispatches by SourceTowerKey to render different visual styles per tower type.
// Uses glow effects and diamond shapes for variety.
package render

import (
	"strings"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with per-tower-type visuals.
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	pool.Each(func(p *projectile.Projectile) {
		cx := float32(p.X)
		cy := float32(p.Y)
		key := p.SourceTowerKey

		switch {
		case strings.Contains(key, "sniper"):
			// Sniper: larger solid + glow
			draw.Glow(screen, cx, cy,
				theme.ProjSniperR, theme.ProjSniperGlow, theme.ProjSniper)

		case strings.Contains(key, "rapid"):
			// Rapid fire: small solid circle
			draw.FilledCircle(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjRapid)

		case strings.Contains(key, "freeze"):
			// Freeze: diamond shape
			draw.Diamond(screen, cx, cy,
				theme.ProjDefaultR+1, 1.5, theme.ProjFreeze)

		case strings.Contains(key, "wind"):
			// Wind: solid circle
			draw.FilledCircle(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjWind)

		default:
			// Default: solid + glow
			draw.Glow(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjDefaultGlow, theme.ProjDefault)
		}
	})
}
