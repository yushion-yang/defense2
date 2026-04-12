// vfx_mascot.go — Mascot-themed visual effects.
// All functions accept pure value parameters, zero core dependency.
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawMascotKillMark renders a magic-circle mark → explosion at the target.
// timer counts up from 0.0; duration is the total effect length (~0.5s).
//
// Phase 1 (0~30%):  Pink magic circle scales up with rotating cross lines.
// Phase 2 (30~70%): Circle locked, pulsing glow.
// Phase 3 (70~100%): Explosion — ring expands, diamonds fly out, central flash.
func DrawMascotKillMark(screen *ebiten.Image, cx, cy float64, timer, duration float64) {
	if timer < 0 || timer >= duration || duration <= 0 {
		return
	}
	p := timer / duration // 0.0 → 1.0

	fcx, fcy := float32(cx), float32(cy)
	pink := color.RGBA{R: 255, G: 105, B: 180, A: 0}
	gold := color.RGBA{R: 255, G: 215, B: 0, A: 0}

	if p < 0.3 {
		// Phase 1: Magic circle scales up.
		pp := p / 0.3 // 0→1
		radius := float32(4 + 18*pp)
		alpha := uint8(60 + 160*pp)

		// Outer ring.
		pink.A = alpha
		draw.CircleOutline(screen, fcx, fcy, radius, 1.5, pink)

		// Inner ring (smaller, lighter).
		if radius > 6 {
			pink.A = alpha / 2
			draw.CircleOutline(screen, fcx, fcy, radius*0.5, 1.0, pink)
		}

		// Rotating cross lines (4 lines through center).
		rotation := pp * math.Pi * 2 // full rotation during scale-up
		lineAlpha := uint8(120 + 100*pp)
		pink.A = lineAlpha
		for i := 0; i < 4; i++ {
			angle := rotation + float64(i)*math.Pi/4.0
			ex := fcx + float32(math.Cos(angle)*float64(radius))
			ey := fcy + float32(math.Sin(angle)*float64(radius))
			sx := fcx - float32(math.Cos(angle)*float64(radius))
			sy := fcy - float32(math.Sin(angle)*float64(radius))
			draw.Line(screen, sx, sy, ex, ey, 0.8, pink, false)
		}
	} else if p < 0.7 {
		// Phase 2: Circle locked, pulsing.
		pp := (p - 0.3) / 0.4 // 0→1
		radius := float32(22)
		pulse := 0.7 + 0.3*math.Sin(pp*math.Pi*4) // fast pulse
		alpha := uint8(float64(220) * pulse)

		pink.A = alpha
		draw.CircleOutline(screen, fcx, fcy, radius, 1.5, pink)
		pink.A = alpha / 2
		draw.CircleOutline(screen, fcx, fcy, radius*0.5, 1.0, pink)

		// Static cross lines.
		pink.A = uint8(float64(180) * pulse)
		for i := 0; i < 4; i++ {
			angle := float64(i) * math.Pi / 4.0
			ex := fcx + float32(math.Cos(angle)*float64(radius))
			ey := fcy + float32(math.Sin(angle)*float64(radius))
			sx := fcx - float32(math.Cos(angle)*float64(radius))
			sy := fcy - float32(math.Sin(angle)*float64(radius))
			draw.Line(screen, sx, sy, ex, ey, 0.8, pink, false)
		}

		// Small diamonds at cardinal points.
		gold.A = uint8(float64(200) * pulse)
		for i := 0; i < 4; i++ {
			angle := float64(i)*math.Pi/2.0 + math.Pi/4.0
			dx := fcx + float32(math.Cos(angle)*float64(radius))
			dy := fcy + float32(math.Sin(angle)*float64(radius))
			draw.Diamond(screen, dx, dy, 2.5, 1.0, gold)
		}
	} else {
		// Phase 3: Explosion.
		pp := (p - 0.7) / 0.3 // 0→1
		fadeA := uint8(220 * (1 - pp))

		// Expanding ring.
		ringR := float32(22 + 24*pp)
		pink.A = fadeA
		draw.CircleOutline(screen, fcx, fcy, ringR, float32(2.0*(1-pp)+0.5), pink)

		// 6 diamonds radiating outward.
		for i := 0; i < 6; i++ {
			angle := float64(i) * math.Pi / 3.0
			dist := 10 + 30*pp
			dx := fcx + float32(math.Cos(angle)*dist)
			dy := fcy + float32(math.Sin(angle)*dist)
			size := float32(3.5 * (1 - pp*0.6))
			dAlpha := uint8(200 * (1 - pp))

			clr := pink
			if i%2 == 1 {
				clr = gold
			}
			clr.A = dAlpha
			draw.Diamond(screen, dx, dy, size, 1.0, clr)
		}

		// Central flash (bright white-pink, fades fast).
		if pp < 0.5 {
			coreP := pp / 0.5
			coreA := uint8(255 * (1 - coreP))
			coreR := float32(6 * (1 - coreP*0.5))
			draw.FilledCircle(screen, fcx, fcy, coreR,
				color.RGBA{R: 255, G: 220, B: 240, A: coreA})
		}
	}
}
