// emitter.go — preset particle emitter functions.
package particle

import (
	"image/color"
	"math"
	"math/rand"
)

// randRange returns a random float64 in [lo, hi].
func randRange(lo, hi float64) float64 {
	return lo + rand.Float64()*(hi-lo)
}

// EmitDeathBurst spawns 8-16 particles as a death explosion at (x, y).
func EmitDeathBurst(pool *Pool, x, y float64) {
	count := 8 + rand.Intn(9) // 8-16
	for i := 0; i < count; i++ {
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: 40, SpeedVar: 30,
			Angle: rand.Float64() * 2 * math.Pi, AngleVar: 0.3,
			Life: 0.55, LifeVar: 0.15,
			Size: 3, SizeEnd: 1,
			Color:    color.RGBA{R: 255, G: 100, B: 50, A: 255},
			EndAlpha: 0,
			Gravity:  120,
		})
	}
}

// EmitMuzzleFlash spawns 4 particles along the firing direction.
func EmitMuzzleFlash(pool *Pool, x, y, angle float64) {
	for i := 0; i < 4; i++ {
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: 70, SpeedVar: 20,
			Angle: angle, AngleVar: 0.4,
			Life: 0.2, LifeVar: 0.05,
			Size: 2.5, SizeEnd: 0.5,
			Color:    color.RGBA{R: 255, G: 220, B: 120, A: 255},
			EndAlpha: 0,
		})
	}
}

// EmitFireParticles spawns upward-drifting fire particles.
func EmitFireParticles(pool *Pool, x, y float64, count int) {
	for i := 0; i < count; i++ {
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			SpreadX: 6, SpreadY: 4,
			Speed: 20, SpeedVar: 12,
			Angle: -math.Pi / 2, AngleVar: 0.5,
			Life: 0.65, LifeVar: 0.15,
			Size: 3, SizeEnd: 1,
			Color:    color.RGBA{R: 255, G: 140, B: 30, A: 200},
			EndAlpha: 0,
			Gravity:  -30,
		})
	}
}

// EmitIceParticles spawns downward-floating ice particles.
func EmitIceParticles(pool *Pool, x, y float64, count int) {
	for i := 0; i < count; i++ {
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			SpreadX: 10, SpreadY: 6,
			Speed: 10, SpeedVar: 6,
			Angle: math.Pi/2 + randRange(-0.4, 0.4),
			Life: 1.0, LifeVar: 0.2,
			Size: 2, SizeEnd: 0.5,
			Color:    color.RGBA{R: 180, G: 220, B: 255, A: 180},
			EndAlpha: 0,
			Gravity:  15,
		})
	}
}

// EmitGoldCollect spawns 5 gold sparkle particles flying upward-left toward HUD.
func EmitGoldCollect(pool *Pool, fromX, fromY float64) {
	for i := 0; i < 5; i++ {
		pool.Spawn(ParticleConfig{
			X: fromX, Y: fromY,
			Speed: 35, SpeedVar: 15,
			Angle: -math.Pi*0.7 + randRange(-0.2, 0.2),
			Life: 0.6, LifeVar: 0.1,
			Size: 2.5, SizeEnd: 1,
			Color:    color.RGBA{R: 255, G: 215, B: 0, A: 255},
			EndAlpha: 0,
			Gravity:  -20,
		})
	}
}

// EmitAmbient spawns ambient floating particles across the screen area.
// Call once per second to maintain ~40 visible particles.
func EmitAmbient(pool *Pool, screenW, screenH float64) {
	// Floating dust motes: 5 per call, very slow, long life.
	for i := 0; i < 5; i++ {
		pool.Spawn(ParticleConfig{
			X:        rand.Float64() * screenW,
			Y:        rand.Float64() * screenH,
			Speed:    3 + rand.Float64()*8,
			Angle:    rand.Float64() * 2 * math.Pi,
			AngleVar: 0,
			Life:     8 + rand.Float64()*4,
			Size:     1.5, SizeEnd: 0.5,
			Color:    color.RGBA{R: 255, G: 255, B: 240, A: 50},
			EndAlpha: 0,
		})
	}
	// Rising light particles: 3 per call, slow upward.
	for i := 0; i < 3; i++ {
		pool.Spawn(ParticleConfig{
			X:        rand.Float64() * screenW,
			Y:        screenH + 10,
			Speed:    6 + rand.Float64()*6,
			Angle:    -math.Pi / 2, AngleVar: 0.3,
			Life:     10 + rand.Float64()*5,
			Size:     2, SizeEnd: 0.8,
			Color:    color.RGBA{R: 200, G: 220, B: 255, A: 35},
			EndAlpha: 0,
			Gravity:  -2,
		})
	}
}

// EmitElectricSparks spawns fast electric spark particles in random directions.
func EmitElectricSparks(pool *Pool, x, y float64, count int) {
	for i := 0; i < count; i++ {
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: 65, SpeedVar: 40,
			Angle: rand.Float64() * 2 * math.Pi, AngleVar: 0.2,
			Life: 0.17, LifeVar: 0.08,
			Size: 1.5, SizeEnd: 0.5,
			Color:    color.RGBA{R: 150, G: 200, B: 255, A: 255},
			EndAlpha: 0,
		})
	}
}
