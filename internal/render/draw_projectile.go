// draw_projectile.go — projectile rendering.
// Trail dots and connector lines are batch-rendered via trail_batch.go
// (1-2 DrawTriangles calls for ALL projectiles instead of 12,000+ individual calls).
// Body rendering remains per-projectile (visually distinct per tower type).
package render

import (
	"math"

	"defense2/internal/core/projectile"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with batched trails + per-type body.
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	// Phase 1: collect all trail geometry into batch buffers
	beginTrailBatch()
	pool.Each(func(p *projectile.Projectile) {
		if !IsInView(p.X, p.Y) {
			return
		}
		collectTrail(p)
	})
	// Phase 2: flush all trails in 1-2 DrawTriangles calls
	flushTrailBatch(screen)

	// Phase 3: draw bodies individually (each type looks different)
	pool.Each(func(p *projectile.Projectile) {
		if !IsInView(p.X, p.Y) {
			return
		}
		angle := math.Atan2(p.VY, p.VX)
		vfx.DrawProjectileBody(screen, float32(p.X), float32(p.Y), angle,
			p.SourceTowerKey, p.Penetrate, p.ScatterVisual)
	})
}
