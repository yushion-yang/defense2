// draw_beam.go — 光束渲染。
// 渲染 wideBeam 的视觉效果：5 层渲染 + 能量流动节点 + 端点冲击。
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/combat"
	"defense2/internal/render/draw"

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
		widthMul := 1.0 + 0.3*math.Exp(-progress*8)
		w := float32(b.Width * widthMul)
		a8 := uint8(alpha * 255)

		x1, y1 := float32(b.X1), float32(b.Y1)
		x2, y2 := float32(b.X2), float32(b.Y2)

		// ── Layer 1: Ultra-wide soft glow (atmosphere) ──
		glowA := uint8(float64(a8) * 0.25)
		draw.ThickLine(screen, x1, y1, x2, y2, w*5,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: glowA})

		// ── Layer 2: Outer glow (3.2x width) ──
		outerA := uint8(float64(a8) * 0.45)
		draw.ThickLine(screen, x1, y1, x2, y2, w*3.2,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: outerA})

		// ── Layer 3: Core beam ──
		coreA := uint8(float64(a8) * 0.95)
		draw.ThickLine(screen, x1, y1, x2, y2, w,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: coreA})

		// ── Layer 4: White-hot center ──
		whiteA := uint8(float64(a8) * 0.85)
		draw.ThickLine(screen, x1, y1, x2, y2, w*0.38,
			color.RGBA{R: 255, G: 255, B: 255, A: whiteA})

		// ── Layer 5: Energy flow nodes (3 bright pulses traveling along beam) ──
		dx, dy := float64(x2-x1), float64(y2-y1)
		beamLen := math.Sqrt(dx*dx + dy*dy)
		if beamLen > 1 {
			for i := 0; i < 3; i++ {
				// Each node travels from source to target, offset by phase
				phase := math.Mod(beamAnimClock*6+float64(i)*0.33, 1.0)
				nx := float32(float64(x1) + dx*phase)
				ny := float32(float64(y1) + dy*phase)
				nodeR := w * 0.6
				nodeA := uint8(float64(a8) * (0.5 + 0.3*math.Sin(beamAnimClock*12+float64(i)*2)))
				draw.FilledCircle(screen, nx, ny, nodeR,
					color.RGBA{R: 255, G: 255, B: 255, A: nodeA})
			}
		}

		// ── Endpoint flares (source = small, target = large impact) ──
		flareClr := color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: a8}
		// Source: moderate flare
		draw.Glow(screen, x1, y1, w*0.6, w*1.8, flareClr)
		// Target: large impact flare + white core + expanding ring
		draw.Glow(screen, x2, y2, w*1.0, w*3, flareClr)
		impactA := uint8(float64(a8) * 0.7)
		draw.FilledCircle(screen, x2, y2, w*0.5,
			color.RGBA{R: 255, G: 255, B: 255, A: impactA})
		// Expanding impact ring at target
		ringR := w*1.5 + float32(progress)*w*2
		ringA := uint8(float64(a8) * (1 - progress) * 0.5)
		draw.CircleOutline(screen, x2, y2, ringR, 1.5,
			color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: ringA})
	})
}
