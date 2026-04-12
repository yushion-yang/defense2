// vfx_mascot.go — Mascot-themed visual effects.
// All functions accept pure value parameters, zero core dependency.
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 击杀闪光 ────────────────────────────────────────

// DrawMascotKillSparkle renders a pink/gold sparkle burst at the given position.
// timer counts up from 0.0; duration is the total effect length (~0.5s).
// Effect: expanding ring that fades + 6 diamond/star shapes radiating outward.
func DrawMascotKillSparkle(screen *ebiten.Image, cx, cy float64, timer, duration float64) {
	if timer < 0 || timer >= duration || duration <= 0 {
		return
	}
	p := timer / duration // 0.0 → 1.0

	fcx, fcy := float32(cx), float32(cy)

	// 1. Expanding pink ring — fades out as it grows.
	ringR := float32(6 + 20*p)
	ringA := uint8(180 * (1 - p))
	if ringA > 2 {
		draw.CircleOutline(screen, fcx, fcy, ringR, float32(1.5*(1-p)+0.5),
			color.RGBA{R: 255, G: 105, B: 180, A: ringA}) // hot pink
	}

	// 2. Six diamond/star shapes expanding outward, alternating pink and gold.
	pink := color.RGBA{R: 255, G: 105, B: 180, A: 0}
	gold := color.RGBA{R: 255, G: 215, B: 0, A: 0}

	for i := 0; i < 6; i++ {
		angle := float64(i) * math.Pi / 3.0
		dist := (8 + 30*p) // expand outward
		dx := fcx + float32(math.Cos(angle)*dist)
		dy := fcy + float32(math.Sin(angle)*dist)

		// Size shrinks as effect progresses.
		size := float32(3.0 * (1 - p*0.6))
		alpha := uint8(200 * (1 - p))

		clr := pink
		if i%2 == 1 {
			clr = gold
		}
		clr.A = alpha
		draw.Diamond(screen, dx, dy, size, 1.0, clr)
	}

	// 3. Central flash — bright white-pink core that fades quickly.
	if p < 0.3 {
		coreP := p / 0.3
		coreA := uint8(255 * (1 - coreP))
		coreR := float32(4 * (1 - coreP*0.5))
		draw.FilledCircle(screen, fcx, fcy, coreR,
			color.RGBA{R: 255, G: 220, B: 240, A: coreA})
	}
}
