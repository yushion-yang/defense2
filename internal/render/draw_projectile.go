// draw_projectile.go — projectile rendering.
// Dispatches by SourceTowerKey to render different visual styles per tower type.
// Renders motion trail (fading history positions) before the projectile body.
package render

import (
	"image/color"
	"strings"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with trail + per-tower-type visuals.
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	pool.Each(func(p *projectile.Projectile) {
		// --- Motion trail (oldest first, fading alpha) ---
		drawProjectileTrail(screen, p)

		// --- Projectile body ---
		cx := float32(p.X)
		cy := float32(p.Y)
		key := p.SourceTowerKey

		// 特殊弹丸类型优先判断
		switch {
		case p.ChargeShot:
			// 蓄力弹：大红色 glow + 白芯
			draw.Glow(screen, cx, cy, 8, 16, color.RGBA{R: 255, G: 80, B: 60, A: 220})
			draw.FilledCircle(screen, cx, cy, 4, color.RGBA{R: 255, G: 255, B: 255, A: 200})

		case p.Pierce:
			// 穿刺弹：紫色拉长椭圆
			draw.Glow(screen, cx, cy, 5, 12, color.RGBA{R: 180, G: 100, B: 255, A: 200})
			draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 220, G: 180, B: 255, A: 230})

		case p.ScatterVisual:
			// 散射视觉弹：小蓝色弹丸
			draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 100, G: 180, B: 255, A: 200})

		case strings.Contains(key, "sniper"):
			draw.Glow(screen, cx, cy,
				theme.ProjSniperR, theme.ProjSniperGlow, theme.ProjSniper)

		case strings.Contains(key, "rapid"):
			draw.FilledCircle(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjRapid)

		case strings.Contains(key, "freeze"):
			draw.Diamond(screen, cx, cy,
				theme.ProjDefaultR+1, 1.5, theme.ProjFreeze)

		case strings.Contains(key, "wind"):
			draw.FilledCircle(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjWind)

		default:
			draw.Glow(screen, cx, cy,
				theme.ProjDefaultR, theme.ProjDefaultGlow, theme.ProjDefault)
		}
	})
}

// drawProjectileTrail renders the fading trail behind a projectile.
func drawProjectileTrail(screen *ebiten.Image, p *projectile.Projectile) {
	baseClr := theme.ProjDefault
	key := p.SourceTowerKey
	switch {
	case strings.Contains(key, "sniper"):
		baseClr = theme.ProjSniper
	case strings.Contains(key, "rapid"):
		baseClr = theme.ProjRapid
	case strings.Contains(key, "freeze"):
		baseClr = theme.ProjFreeze
	case strings.Contains(key, "wind"):
		baseClr = theme.ProjWindTrail
	}

	for i := 0; i < projectile.TrailLen; i++ {
		// Read oldest first: cursor is next-write, so cursor-TrailLen is oldest
		idx := (p.TrailCursor + i) % projectile.TrailLen
		pt := p.Trail[idx]
		if !pt.Active {
			continue
		}

		// Alpha fades from near-transparent (oldest) to semi-opaque (newest)
		frac := float64(i+1) / float64(projectile.TrailLen)
		alpha := uint8(80 * frac)
		// Radius shrinks from body size to tiny
		r := float32(theme.ProjDefaultR) * float32(0.3+0.7*frac)

		clr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: alpha}
		draw.FilledCircle(screen, float32(pt.X), float32(pt.Y), r, clr)
	}
}
