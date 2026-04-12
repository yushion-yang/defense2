// draw_projectile.go — projectile rendering.
// Trail dots and connector lines are batch-rendered via trail_batch.go
// (1-2 DrawTriangles calls for ALL projectiles instead of 12,000+ individual calls).
// Body rendering: Full detail per-projectile at VFXFull, batched dots at VFXReduced+.
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with batched trails + body.
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

	// Phase 3: bodies
	if CurrentVFXLevel >= VFXReduced {
		// 高负载：所有弹体用批量化彩色圆点替代（1 次 DrawTriangles）
		beginBodyBatch()
		pool.Each(func(p *projectile.Projectile) {
			if !IsInView(p.X, p.Y) {
				return
			}
			clr := vfx.ProjectileTrailColor(p.SourceTowerKey)
			// 提亮到不透明
			clr.A = 220
			addBodyDot(float32(p.X), float32(p.Y), 4, clr)
		})
		flushBodyBatch(screen)
	} else {
		// 正常：每弹独立渲染完整视觉
		pool.Each(func(p *projectile.Projectile) {
			if !IsInView(p.X, p.Y) {
				return
			}
			angle := math.Atan2(p.VY, p.VX)
			vfx.DrawProjectileBody(screen, float32(p.X), float32(p.Y), angle,
				p.SourceTowerKey, p.Penetrate, p.ScatterVisual)
		})
	}
}

// ── Body batch (reuses circle texture from trail_batch.go) ──

var bodyBatch struct {
	vs []ebiten.Vertex
	is []uint16
}

func init() {
	bodyBatch.vs = make([]ebiten.Vertex, 0, 1024*4)
	bodyBatch.is = make([]uint16, 0, 1024*6)
}

func beginBodyBatch() {
	bodyBatch.vs = bodyBatch.vs[:0]
	bodyBatch.is = bodyBatch.is[:0]
}

func addBodyDot(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)
	cr := float32(clr.R) / 255 * float32(clr.A) / 255
	cg := float32(clr.G) / 255 * float32(clr.A) / 255
	cb := float32(clr.B) / 255 * float32(clr.A) / 255
	ca := float32(clr.A) / 255
	idx := uint16(len(bodyBatch.vs))
	bodyBatch.vs = append(bodyBatch.vs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBatch.is = append(bodyBatch.is, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

func flushBodyBatch(screen *ebiten.Image) {
	if len(bodyBatch.vs) == 0 {
		return
	}
	screen.DrawTriangles(bodyBatch.vs, bodyBatch.is, trailCircleTex(),
		&ebiten.DrawTrianglesOptions{})
}
