// draw_projectile.go — projectile rendering.
// Both trails AND bodies are batch-rendered via DrawTriangles for maximum throughput.
// Trail: circle texture quads for dots + line quads for connectors (trail_batch.go).
// Body: circle texture quads for velocity tails, body dots, and glow halos.
// Only wind-type projectiles (with arcs) fall back to individual vector calls.
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawProjectiles renders all alive projectiles with fully batched rendering.
// Single pool.Each pass collects trail + body geometry, then flushes in 3-5 DrawTriangles.
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	beginTrailBatch()
	beginBodyBatch()

	pool.Each(func(p *projectile.Projectile) {
		if !IsInView(p.X, p.Y) {
			return
		}
		// Trail (dots + connectors)
		collectTrail(p)
		// Body (velocity tail + type-specific shape)
		collectBody(p)
	})

	// Flush trails: lines behind, dots on top
	flushTrailBatch(screen)
	// Flush body: tails first, then body circles, then glow halos to glow target
	flushBodyBatch(screen)
}

// ── Body batch buffers ──────────────────────────────────────────────

var bodyBuf struct {
	// Velocity tails (line quads, drawn to screen)
	tailVs []ebiten.Vertex
	tailIs []uint16
	// Body circles (drawn to screen)
	bodyVs []ebiten.Vertex
	bodyIs []uint16
	// Glow outer circles (drawn to glow target for additive blending)
	glowVs []ebiten.Vertex
	glowIs []uint16
	// Individual renders deferred (wind type with arcs)
	deferred []deferredBody
}

type deferredBody struct {
	cx, cy float32
	angle  float64
	style  string
}

func init() {
	const cap = 1024
	bodyBuf.tailVs = make([]ebiten.Vertex, 0, cap*4)
	bodyBuf.tailIs = make([]uint16, 0, cap*6)
	bodyBuf.bodyVs = make([]ebiten.Vertex, 0, cap*4)
	bodyBuf.bodyIs = make([]uint16, 0, cap*6)
	bodyBuf.glowVs = make([]ebiten.Vertex, 0, cap*4)
	bodyBuf.glowIs = make([]uint16, 0, cap*6)
	bodyBuf.deferred = make([]deferredBody, 0, 32)
}

func beginBodyBatch() {
	bodyBuf.tailVs = bodyBuf.tailVs[:0]
	bodyBuf.tailIs = bodyBuf.tailIs[:0]
	bodyBuf.bodyVs = bodyBuf.bodyVs[:0]
	bodyBuf.bodyIs = bodyBuf.bodyIs[:0]
	bodyBuf.glowVs = bodyBuf.glowVs[:0]
	bodyBuf.glowIs = bodyBuf.glowIs[:0]
	bodyBuf.deferred = bodyBuf.deferred[:0]
}

// collectBody adds one projectile's body to batch buffers.
func collectBody(p *projectile.Projectile) {
	cx := float32(p.X)
	cy := float32(p.Y)
	angle := math.Atan2(p.VY, p.VX)

	// Velocity tail (all projectiles)
	const tailLen = 8.0
	tailX := float64(cx) - math.Cos(angle)*tailLen
	tailY := float64(cy) - math.Sin(angle)*tailLen
	addBodyLine(cx, cy, float32(tailX), float32(tailY), 1.5,
		color.RGBA{R: 255, G: 255, B: 255, A: 120})

	switch {
	case p.Penetrate:
		// Outer glow (to glow target)
		addGlowCircle(cx, cy, 12, color.RGBA{R: 180, G: 100, B: 255, A: 50})
		// Inner bright core
		addBodyCircle(cx, cy, 5, color.RGBA{R: 180, G: 100, B: 255, A: 200})
		addBodyCircle(cx, cy, 3, color.RGBA{R: 220, G: 180, B: 255, A: 230})

	case p.ScatterVisual:
		addBodyCircle(cx, cy, 3, color.RGBA{R: 100, G: 180, B: 255, A: 200})
		scTailX := float64(cx) - math.Cos(angle)*5
		scTailY := float64(cy) - math.Sin(angle)*5
		addBodyLine(cx, cy, float32(scTailX), float32(scTailY), 1,
			color.RGBA{R: 100, G: 180, B: 255, A: 140})

	case isSniper(p.SourceTowerKey):
		addGlowCircle(cx, cy, theme.ProjSniperGlow, theme.ProjSniper)
		addBodyCircle(cx, cy, theme.ProjSniperR, theme.ProjSniper)

	case isRapid(p.SourceTowerKey):
		addBodyCircle(cx, cy, theme.ProjDefaultR, theme.ProjRapid)
		trailCx := cx - float32(math.Cos(angle)*3)
		trailCy := cy - float32(math.Sin(angle)*3)
		addBodyCircle(trailCx, trailCy, theme.ProjDefaultR*0.6,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 160})
		addGlowCircle(cx, cy, theme.ProjDefaultR+4,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 60})

	case isFreeze(p.SourceTowerKey):
		// Diamond as rotated square quad
		addBodyDiamond(cx, cy, theme.ProjDefaultR+1, float32(angle), theme.ProjFreeze)

	case isWind(p.SourceTowerKey):
		// Wind has arcs that are hard to batch — defer to individual render
		addBodyCircle(cx, cy, theme.ProjDefaultR, theme.ProjWind)
		bodyBuf.deferred = append(bodyBuf.deferred, deferredBody{cx, cy, angle, p.SourceTowerKey})

	default:
		addGlowCircle(cx, cy, 10, theme.ProjDefault)
		addBodyCircle(cx, cy, 3, theme.ProjDefault)
	}
}

func flushBodyBatch(screen *ebiten.Image) {
	wp := trailWhitePixel()
	ct := trailCircleTex()
	noAA := &ebiten.DrawTrianglesOptions{}

	// 1. Velocity tails (line quads on screen)
	if len(bodyBuf.tailVs) > 0 {
		screen.DrawTriangles(bodyBuf.tailVs, bodyBuf.tailIs, wp, noAA)
	}

	// 2. Body circles (on screen)
	if len(bodyBuf.bodyVs) > 0 {
		screen.DrawTriangles(bodyBuf.bodyVs, bodyBuf.bodyIs, ct, noAA)
	}

	// 3. Glow circles (on glow target for additive compositing)
	if len(bodyBuf.glowVs) > 0 {
		if gt := draw.GlowTarget(); gt != nil {
			gt.DrawTriangles(bodyBuf.glowVs, bodyBuf.glowIs, ct, noAA)
		} else {
			// No glow pass active: draw directly with lighter blend
			screen.DrawTriangles(bodyBuf.glowVs, bodyBuf.glowIs, ct,
				&ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter})
		}
	}

	// 4. Deferred individual renders (wind arcs — rare, ~1-2 towers worth)
	for _, d := range bodyBuf.deferred {
		vfx.DrawWindArcs(screen, d.cx, d.cy, d.angle)
	}
}

// ── Batch helpers ───────────────────────────────────────────────────

func addBodyCircle(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)
	cr, cg, cb, ca := premulColor(clr)
	idx := uint16(len(bodyBuf.bodyVs))
	bodyBuf.bodyVs = append(bodyBuf.bodyVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.bodyIs = append(bodyBuf.bodyIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

func addGlowCircle(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)
	// Glow outer circle: reduce alpha for soft halo
	a := clr.A / 4
	if a == 0 {
		a = 1
	}
	cr, cg, cb, ca := premulColor(color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: a})
	idx := uint16(len(bodyBuf.glowVs))
	bodyBuf.glowVs = append(bodyBuf.glowVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: circU0, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: circU1, SrcY: circV0, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: circU1, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: circU0, SrcY: circV1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.glowIs = append(bodyBuf.glowIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

func addBodyLine(x1, y1, x2, y2, width float32, clr color.RGBA) {
	if clr.A == 0 {
		return
	}
	sx1, sy1 := draw.S32(x1), draw.S32(y1)
	sx2, sy2 := draw.S32(x2), draw.S32(y2)
	w := draw.S32(width) * 0.5
	dx, dy := float64(sx2-sx1), float64(sy2-sy1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return
	}
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))
	cr, cg, cb, ca := premulColor(clr)
	idx := uint16(len(bodyBuf.tailVs))
	bodyBuf.tailVs = append(bodyBuf.tailVs,
		ebiten.Vertex{DstX: sx1 - nx, DstY: sy1 - ny, SrcX: 1, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx1 + nx, DstY: sy1 + ny, SrcX: 2, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 + nx, DstY: sy2 + ny, SrcX: 2, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 - nx, DstY: sy2 - ny, SrcX: 1, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.tailIs = append(bodyBuf.tailIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

func addBodyDiamond(cx, cy, size, angle float32, clr color.RGBA) {
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(size)
	cr, cg, cb, ca := premulColor(clr)
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	// Diamond: 4 vertices at 45° rotated by angle
	idx := uint16(len(bodyBuf.bodyVs))
	bodyBuf.bodyVs = append(bodyBuf.bodyVs,
		ebiten.Vertex{DstX: sx + r*cos, DstY: sy + r*sin, SrcX: circU1/2 + circU1/4, SrcY: circV0 + 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r*sin, DstY: sy + r*cos, SrcX: circU1 - 1, SrcY: circV1/2 + circV1/4, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r*cos, DstY: sy - r*sin, SrcX: circU1/2 + circU1/4, SrcY: circV1 - 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r*sin, DstY: sy - r*cos, SrcX: circU0 + 1, SrcY: circV1/2 + circV1/4, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	bodyBuf.bodyIs = append(bodyBuf.bodyIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

func premulColor(clr color.RGBA) (float32, float32, float32, float32) {
	a := float32(clr.A) / 255
	return float32(clr.R) / 255 * a, float32(clr.G) / 255 * a, float32(clr.B) / 255 * a, a
}

// ── Style matchers (avoid strings.Contains per projectile per frame) ──

func isSniper(key string) bool  { return len(key) >= 6 && key[:6] == "sniper" || containsStr(key, "sniper") }
func isRapid(key string) bool   { return len(key) >= 5 && key[:5] == "rapid" || containsStr(key, "rapid") }
func isFreeze(key string) bool  { return len(key) >= 6 && key[:6] == "freeze" || containsStr(key, "freeze") }
func isWind(key string) bool    { return len(key) >= 4 && key[:4] == "wind" || containsStr(key, "wind") }

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
