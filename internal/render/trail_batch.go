// trail_batch.go — batched projectile trail renderer.
// Collects all trail dots and connector lines into vertex buffers,
// then renders everything in 1-2 DrawTriangles calls instead of
// 12,000+ individual draw.FilledCircle/ThickLine calls.
//
// Pattern: same as particle.Pool — pre-allocated vertex/index slices,
// quads for circles, stretched quads for lines, single white-pixel source.
package render

import (
	"image/color"
	"math"
	"sync"

	"defense2/internal/core/projectile"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// trailBatch collects trail geometry for batch rendering.
// Not goroutine-safe (Ebitengine is single-threaded).
var trailBatch struct {
	// Circle dots (trail points + glow endpoints)
	circVs []ebiten.Vertex
	circIs []uint16
	// Line segments (connectors between trail points)
	lineVs []ebiten.Vertex
	lineIs []uint16
}

var (
	trailWhiteOnce  sync.Once
	trailWhiteImage *ebiten.Image
)

func trailWhitePixel() *ebiten.Image {
	trailWhiteOnce.Do(func() {
		trailWhiteImage = ebiten.NewImage(3, 3)
		pix := make([]byte, 3*3*4)
		for i := range pix {
			pix[i] = 0xff
		}
		trailWhiteImage.WritePixels(pix)
	})
	return trailWhiteImage
}

func init() {
	// Pre-allocate for worst case: 1024 projectiles × 6 trail points
	const maxDots = 1024 * 7   // 6 trail + 1 glow per projectile
	const maxLines = 1024 * 5  // up to 5 connector lines per projectile
	trailBatch.circVs = make([]ebiten.Vertex, 0, maxDots*4)
	trailBatch.circIs = make([]uint16, 0, maxDots*6)
	trailBatch.lineVs = make([]ebiten.Vertex, 0, maxLines*4)
	trailBatch.lineIs = make([]uint16, 0, maxLines*6)
}

// beginTrailBatch resets the batch buffers for a new frame.
func beginTrailBatch() {
	trailBatch.circVs = trailBatch.circVs[:0]
	trailBatch.circIs = trailBatch.circIs[:0]
	trailBatch.lineVs = trailBatch.lineVs[:0]
	trailBatch.lineIs = trailBatch.lineIs[:0]
}

// addTrailDot adds a filled circle to the batch.
func addTrailDot(cx, cy, radius float32, clr color.RGBA) {
	if clr.A == 0 || radius <= 0 {
		return
	}
	sx := draw.S32(cx)
	sy := draw.S32(cy)
	r := draw.S32(radius)

	cr := float32(clr.R) / 255 * float32(clr.A) / 255
	cg := float32(clr.G) / 255 * float32(clr.A) / 255
	cb := float32(clr.B) / 255 * float32(clr.A) / 255
	ca := float32(clr.A) / 255

	idx := uint16(len(trailBatch.circVs))
	trailBatch.circVs = append(trailBatch.circVs,
		ebiten.Vertex{DstX: sx - r, DstY: sy - r, SrcX: 1, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy - r, SrcX: 2, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx + r, DstY: sy + r, SrcX: 2, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx - r, DstY: sy + r, SrcX: 1, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	trailBatch.circIs = append(trailBatch.circIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// addTrailLine adds a thick line segment to the batch as a rotated quad.
func addTrailLine(x1, y1, x2, y2, width float32, clr color.RGBA) {
	if clr.A == 0 || width <= 0 {
		return
	}
	sx1 := draw.S32(x1)
	sy1 := draw.S32(y1)
	sx2 := draw.S32(x2)
	sy2 := draw.S32(y2)
	w := draw.S32(width) * 0.5

	dx := float64(sx2 - sx1)
	dy := float64(sy2 - sy1)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		return
	}
	// Perpendicular normal
	nx := float32(-dy / length * float64(w))
	ny := float32(dx / length * float64(w))

	cr := float32(clr.R) / 255 * float32(clr.A) / 255
	cg := float32(clr.G) / 255 * float32(clr.A) / 255
	cb := float32(clr.B) / 255 * float32(clr.A) / 255
	ca := float32(clr.A) / 255

	idx := uint16(len(trailBatch.lineVs))
	trailBatch.lineVs = append(trailBatch.lineVs,
		ebiten.Vertex{DstX: sx1 - nx, DstY: sy1 - ny, SrcX: 1, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx1 + nx, DstY: sy1 + ny, SrcX: 2, SrcY: 1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 + nx, DstY: sy2 + ny, SrcX: 2, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: sx2 - nx, DstY: sy2 - ny, SrcX: 1, SrcY: 2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	trailBatch.lineIs = append(trailBatch.lineIs, idx, idx+1, idx+2, idx, idx+2, idx+3)
}

// collectTrail adds one projectile's trail to the batch.
func collectTrail(p *projectile.Projectile) {
	baseClr := vfx.ProjectileTrailColor(p.SourceTowerKey)
	n := projectile.TrailLen

	var prevX, prevY float32
	var prevActive bool
	var prevAlpha uint8
	var prevR float32

	for i := 0; i < n; i++ {
		idx := (p.TrailCursor + i) % n
		pt := p.Trail[idx]
		if !pt.Active {
			prevActive = false
			continue
		}
		frac := float64(i+1) / float64(n)
		alpha := uint8(140 * frac)
		r := float32(theme.ProjDefaultR) * float32(0.3+0.7*frac)
		clr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: alpha}

		curX := float32(pt.X)
		curY := float32(pt.Y)

		// Connector line
		if prevActive {
			lineAlpha := prevAlpha
			if alpha < lineAlpha {
				lineAlpha = alpha
			}
			lineClr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: lineAlpha}
			lineW := prevR * 0.8
			if lineW < 0.5 {
				lineW = 0.5
			}
			addTrailLine(prevX, prevY, curX, curY, lineW, lineClr)
		}

		// Trail dot
		addTrailDot(curX, curY, r, clr)

		// Newest point glow (larger, same color)
		if i == n-1 {
			addTrailDot(curX, curY, r*1.5, color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 3})
		}

		prevX, prevY = curX, curY
		prevAlpha = alpha
		prevR = r
		prevActive = true
	}
}

// flushTrailBatch renders all collected trail geometry.
func flushTrailBatch(screen *ebiten.Image) {
	wp := trailWhitePixel()
	opts := &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
	}

	// Lines first (behind dots)
	if len(trailBatch.lineVs) > 0 {
		screen.DrawTriangles(trailBatch.lineVs, trailBatch.lineIs, wp, opts)
	}
	// Then dots (on top)
	if len(trailBatch.circVs) > 0 {
		screen.DrawTriangles(trailBatch.circVs, trailBatch.circIs, wp, opts)
	}
}
