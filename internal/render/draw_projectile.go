// draw_projectile.go — projectile rendering.
// Dispatches by SourceTowerKey to render different visual styles per tower type.
// Renders motion trail (fading history positions) before the projectile body.
package render

import (
	"math"

	"defense2/internal/core/projectile"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with trail + per-tower-type visuals.
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	pool.Each(func(p *projectile.Projectile) {
		// Viewport culling: skip projectiles outside camera view
		if !IsInView(p.X, p.Y) {
			return
		}

		// Trail first (behind body)
		baseClr := vfx.ProjectileTrailColor(p.SourceTowerKey)
		var trailArr [projectile.TrailLen]vfx.TrailPt
		for i := 0; i < projectile.TrailLen; i++ {
			trailArr[i] = vfx.TrailPt{X: p.Trail[i].X, Y: p.Trail[i].Y, Active: p.Trail[i].Active}
		}
		vfx.DrawProjectileTrail(screen, trailArr[:], p.TrailCursor, baseClr)

		// Body
		angle := math.Atan2(p.VY, p.VX)
		vfx.DrawProjectileBody(screen, float32(p.X), float32(p.Y), angle,
			p.SourceTowerKey, p.Penetrate, p.ScatterVisual)
	})
}
