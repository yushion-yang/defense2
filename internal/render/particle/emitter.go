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

// EmitBleedDrip spawns a single red downward-dripping particle at (x, y).
// Call per-frame with a probability gate (~15% at 60fps ≈ 9 particles/sec).
func EmitBleedDrip(pool *Pool, x, y, radius float64) {
	pool.Spawn(ParticleConfig{
		X: x, Y: y + radius,
		SpreadX: radius * 0.5,
		Speed: 8, SpeedVar: 4,
		Angle: math.Pi / 2, AngleVar: 0.3, // downward
		Life: 0.4, LifeVar: 0.1,
		Size: 1.5, SizeEnd: 0.5,
		Color:    color.RGBA{R: 200, G: 30, B: 30, A: 180},
		EndAlpha: 0,
		Gravity:  80,
	})
}

// EmitSpawnBurst emits a brief burst of white/cyan particles at spawn point.
func EmitSpawnBurst(pool *Pool, x, y float64) {
	if pool == nil {
		return
	}
	count := 6 + rand.Intn(3) // 6-8 particles
	for i := 0; i < count; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 20 + rand.Float64()*30
		g := uint8(200 + rand.Intn(56))
		b := uint8(220 + rand.Intn(36))
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: speed, SpeedVar: 5,
			Angle: angle, AngleVar: 0.2,
			Life: 0.4 + rand.Float64()*0.2, LifeVar: 0.05,
			Size: 2, SizeEnd: 0.5,
			Color:    color.RGBA{R: 230, G: g, B: b, A: 200},
			EndAlpha: 0,
			Gravity:  -30, // mostly upward drift
		})
	}
}

// EmitDeathBurstLarge emits a larger death burst for multi-kills and overkills.
// 16-24 particles with wider spread and warmer colors.
func EmitDeathBurstLarge(pool *Pool, x, y float64) {
	if pool == nil {
		return
	}
	count := 16 + rand.Intn(9) // 16-24
	for i := 0; i < count; i++ {
		r := uint8(200 + rand.Intn(56))
		g := uint8(100 + rand.Intn(80))
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: 50, SpeedVar: 40,
			Angle: rand.Float64() * 2 * math.Pi, AngleVar: 0.3,
			Life: 0.55, LifeVar: 0.2,
			Size: 4, SizeEnd: 1.5,
			Color:    color.RGBA{R: r, G: g, B: 40, A: 220},
			EndAlpha: 0,
			Gravity:  100,
		})
	}
}

// EmitBossDeathBurst emits a massive multi-color burst for boss kills.
// 30-40 particles with vibrant colors and large radius.
func EmitBossDeathBurst(pool *Pool, x, y float64) {
	if pool == nil {
		return
	}
	colors := []color.RGBA{
		{R: 255, G: 80, B: 40, A: 240},  // orange-red
		{R: 255, G: 200, B: 40, A: 240},  // gold
		{R: 255, G: 120, B: 60, A: 240},  // amber
		{R: 200, G: 60, B: 255, A: 240},  // purple
		{R: 80, G: 200, B: 255, A: 240},  // cyan
	}
	count := 30 + rand.Intn(11) // 30-40
	for i := 0; i < count; i++ {
		c := colors[rand.Intn(len(colors))]
		pool.Spawn(ParticleConfig{
			X: x, Y: y,
			Speed: 60, SpeedVar: 60,
			Angle: rand.Float64() * 2 * math.Pi, AngleVar: 0.3,
			Life: 0.7, LifeVar: 0.3,
			Size: 5, SizeEnd: 2,
			Color:    c,
			EndAlpha: 0,
			Gravity:  80,
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
