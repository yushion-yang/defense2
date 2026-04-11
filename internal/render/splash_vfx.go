// splash_vfx.go — splash ability impact ring VFX.
// Global object pool, expanding ring + center flash at splash origin.
package render

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// SplashVFX represents a single splash ring effect instance.
type SplashVFX struct {
	X, Y    float64
	Life    float64
	MaxLife float64
	Active  bool
	Radius  float32 // splash radius (final expanded size)
}

const maxSplashVFX = 8

var splashPool [maxSplashVFX]SplashVFX
var splashCursor int

// SpawnSplashRing spawns an expanding orange-yellow ring at the splash origin.
func SpawnSplashRing(x, y, radius float64) {
	v := &splashPool[splashCursor]
	splashCursor = (splashCursor + 1) % maxSplashVFX
	v.X = x
	v.Y = y
	v.Life = 0.3
	v.MaxLife = 0.3
	v.Active = true
	v.Radius = float32(radius)
}

// ClearSplashVFX deactivates all splash VFX instances.
func ClearSplashVFX() {
	for i := range splashPool {
		splashPool[i].Active = false
	}
}

// UpdateSplashVFX ticks all active splash VFX.
func UpdateSplashVFX(dt float64) {
	for i := range splashPool {
		v := &splashPool[i]
		if !v.Active {
			continue
		}
		v.Life -= dt
		if v.Life <= 0 {
			v.Active = false
		}
	}
}

// DrawSplashVFX renders all active splash ring effects.
func DrawSplashVFX(screen *ebiten.Image) {
	for i := range splashPool {
		v := &splashPool[i]
		if !v.Active {
			continue
		}
		alpha := v.Life / v.MaxLife // 1 -> 0
		progress := 1.0 - alpha    // 0 -> 1

		cx := float32(v.X)
		cy := float32(v.Y)

		// Center flash — white, fades quickly
		if alpha > 0.5 {
			flashR := float32(4 * alpha)
			draw.FilledCircle(screen, cx, cy, flashR,
				color.RGBA{R: 255, G: 240, B: 200, A: uint8(200 * alpha)})
		}

		// Expanding ring — orange-yellow, grows to splash radius
		ringR := v.Radius * float32(progress)
		ringW := float32(1.8 * alpha)
		if ringW < 0.4 {
			ringW = 0.4
		}
		ringAlpha := uint8(180 * alpha)
		draw.CircleOutline(screen, cx, cy, ringR, ringW,
			color.RGBA{R: 255, G: 180, B: 60, A: ringAlpha})

		// Secondary ring — slightly behind, dimmer
		if progress > 0.15 {
			innerProgress := progress - 0.15
			innerR := v.Radius * float32(innerProgress)
			innerAlpha := uint8(90 * alpha)
			draw.CircleOutline(screen, cx, cy, innerR, 1,
				color.RGBA{R: 255, G: 210, B: 100, A: innerAlpha})
		}

		// 4 spark lines radiating outward
		sparkAlpha := uint8(140 * alpha)
		if sparkAlpha > 0 {
			sparkLen := float32(float64(v.Radius) * progress * 0.6)
			seed := float64(v.X*7.3 + v.Y*13.7)
			offset := math.Sin(seed) * 0.4
			for j := 0; j < 4; j++ {
				a := float64(j)*math.Pi/2 + offset
				ex := cx + float32(math.Cos(a))*sparkLen
				ey := cy + float32(math.Sin(a))*sparkLen
				draw.Line(screen, cx, cy, ex, ey, 1,
					color.RGBA{R: 255, G: 200, B: 80, A: sparkAlpha}, true)
			}
		}
	}
}
