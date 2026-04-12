// draw_beam.go — 光束渲染。
// 渲染 wideBeam 的视觉效果：多层辉光 + 电弧抖动 + 能量流节点 + 边缘火花 + 端点冲击。
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/combat"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// beamAnimClock monotonically increasing timer for beam energy flow animation.
var beamAnimClock float64

// DrawBeams 渲染所有存活光束。animDT 用于推进能量流动动画。
func DrawBeams(screen *ebiten.Image, beams *combat.BeamPool, animDT float64) {
	beamAnimClock += animDT
	if beams == nil {
		return
	}
	beams.Each(func(b *combat.Beam) {
		alpha := b.Alpha()
		if alpha <= 0 {
			return
		}

		// Life progress: 0 = just fired, 1 = about to vanish
		progress := 1.0 - alpha
		// Width pulse: expand on fire then taper
		widthMul := 1.0 + 0.5*math.Exp(-progress*6)
		w := float32(b.Width * widthMul)
		a8 := uint8(alpha * 255)

		x1, y1 := float32(b.X1), float32(b.Y1)
		x2, y2 := float32(b.X2), float32(b.Y2)
		dx, dy := float64(x2-x1), float64(y2-y1)
		beamLen := math.Sqrt(dx*dx + dy*dy)
		perpX, perpY := 0.0, 0.0
		if beamLen > 1 {
			perpX = -dy / beamLen
			perpY = dx / beamLen
		}

		// ── Layer 1: Ultra-wide atmospheric glow ──
		glowA := uint8(float64(a8) * 0.2)
		draw.ThickLine(screen, x1, y1, x2, y2, w*6,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: glowA})

		// ── Layer 2: Outer glow with flicker ──
		flicker := 0.8 + 0.2*math.Sin(beamAnimClock*18)
		outerA := uint8(float64(a8) * 0.4 * flicker)
		draw.ThickLine(screen, x1, y1, x2, y2, w*3.5,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: outerA})

		// ── Layer 3: Electric arc jitter (2 offset lines simulating unstable arc) ──
		if beamLen > 10 {
			for arc := 0; arc < 2; arc++ {
				jitter := math.Sin(beamAnimClock*25+float64(arc)*3.7) * 0.5 * float64(w)
				jx := float32(perpX * jitter)
				jy := float32(perpY * jitter)
				arcA := uint8(float64(a8) * 0.35 * flicker)
				draw.ThickLine(screen, x1+jx, y1+jy, x2+jx, y2+jy, w*0.8,
					color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: arcA})
			}
		}

		// ── Layer 4: Core beam (bright, solid) ──
		coreA := uint8(float64(a8) * 0.95)
		draw.ThickLine(screen, x1, y1, x2, y2, w*1.2,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: coreA})

		// ── Layer 5: White-hot center ──
		whiteA := uint8(float64(a8) * 0.9)
		draw.ThickLine(screen, x1, y1, x2, y2, w*0.4,
			color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: whiteA})

		// ── Layer 6: Energy flow nodes (5 pulses traveling along beam) ──
		if beamLen > 5 {
			for i := 0; i < 5; i++ {
				phase := math.Mod(beamAnimClock*8+float64(i)*0.2, 1.0)
				nx := float32(float64(x1) + dx*phase)
				ny := float32(float64(y1) + dy*phase)
				nodeR := w * 0.5
				nodePulse := 0.6 + 0.4*math.Sin(beamAnimClock*15+float64(i)*2.5)
				nodeA := uint8(float64(a8) * nodePulse)
				draw.FilledCircle(screen, nx, ny, nodeR,
					color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: nodeA})
				// Node glow halo
				draw.FilledCircle(screen, nx, ny, nodeR*2,
					color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: nodeA / 3})
			}
		}

		// ── Layer 7: Edge sparks (bright dots dancing along beam edges) ──
		if beamLen > 20 {
			for i := 0; i < 4; i++ {
				t := math.Mod(beamAnimClock*5+float64(i)*0.25, 1.0)
				side := 1.0
				if i%2 == 0 {
					side = -1.0
				}
				sparkOffset := (0.5 + 0.5*math.Sin(beamAnimClock*20+float64(i)*4)) * float64(w) * 1.5 * side
				sx := float32(float64(x1) + dx*t + perpX*sparkOffset)
				sy := float32(float64(y1) + dy*t + perpY*sparkOffset)
				sparkA := uint8(float64(a8) * (0.4 + 0.3*math.Sin(beamAnimClock*12+float64(i))))
				draw.FilledCircle(screen, sx, sy, 1.5,
					color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: sparkA})
			}
		}

		// ── Source flare (moderate, pulsing) ──
		srcPulse := 0.8 + 0.2*math.Sin(beamAnimClock*10)
		srcFlareA := uint8(float64(a8) * srcPulse)
		draw.Glow(screen, x1, y1, w*0.5, w*2.5,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: srcFlareA})
		draw.FilledCircle(screen, x1, y1, w*0.4,
			color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: uint8(float64(a8) * 0.6 * srcPulse)})

		// ── Target impact (large flare + white core + double expanding rings) ──
		draw.Glow(screen, x2, y2, w*1.2, w*4,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: a8})
		draw.FilledCircle(screen, x2, y2, w*0.6,
			color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: uint8(float64(a8) * 0.8)})
		// Inner expanding ring
		ring1R := w*1.5 + float32(progress)*w*2
		ring1A := uint8(float64(a8) * (1 - progress) * 0.6)
		draw.CircleOutline(screen, x2, y2, ring1R, 2,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: ring1A})
		// Outer expanding ring (delayed)
		if progress > 0.1 {
			ring2Prog := (progress - 0.1) / 0.9
			ring2R := w*2 + float32(ring2Prog)*w*3
			ring2A := uint8(float64(a8) * (1 - ring2Prog) * 0.3)
			draw.CircleOutline(screen, x2, y2, ring2R, 1,
				color.RGBA{R: theme.BeamHighlight.R, G: theme.BeamHighlight.G, B: theme.BeamHighlight.B, A: ring2A})
		}
	})
}
