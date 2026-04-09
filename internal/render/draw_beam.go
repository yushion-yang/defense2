// draw_beam.go — 光束渲染。
// 渲染 wideBeam 的视觉效果：多层 glow + 亮芯 + 端点光斑。
package render

import (
	"image/color"

	"defense2/internal/core/combat"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawBeams 渲染所有存活光束。
func DrawBeams(screen *ebiten.Image, beams *combat.BeamPool) {
	if beams == nil {
		return
	}
	beams.Each(func(b *combat.Beam) {
		alpha := b.Alpha()
		if alpha <= 0 {
			return
		}

		w := float32(b.Width)
		a8 := uint8(alpha * 255)

		// 1. 外层 glow（3.2x 宽，40% alpha）
		glowA := uint8(float64(a8) * 0.4)
		glowClr := color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: glowA}
		draw.ThickLine(screen,
			float32(b.X1), float32(b.Y1), float32(b.X2), float32(b.Y2),
			w*3.2, glowClr)

		// 2. 核心束（95% alpha）
		coreA := uint8(float64(a8) * 0.95)
		coreClr := color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: coreA}
		draw.ThickLine(screen,
			float32(b.X1), float32(b.Y1), float32(b.X2), float32(b.Y2),
			w, coreClr)

		// 3. 白色亮芯（34% 宽）
		whiteA := uint8(float64(a8) * 0.8)
		whiteClr := color.RGBA{R: 255, G: 255, B: 255, A: whiteA}
		draw.ThickLine(screen,
			float32(b.X1), float32(b.Y1), float32(b.X2), float32(b.Y2),
			w*0.34, whiteClr)

		// 4. 两端光斑
		flareClr := color.RGBA{R: b.Color[0], G: b.Color[1], B: b.Color[2], A: a8}
		draw.Glow(screen, float32(b.X1), float32(b.Y1), w*0.8, w*2, flareClr)
		draw.Glow(screen, float32(b.X2), float32(b.Y2), w*0.8, w*2, flareClr)
	})
}
