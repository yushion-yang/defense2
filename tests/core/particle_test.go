package core_test

import (
	"image/color"
	"math"
	"testing"

	"defense2/internal/render/particle"
)

func TestParticleSpawnAndUpdate(t *testing.T) {
	pool := particle.NewPool()

	pool.Spawn(particle.ParticleConfig{
		X: 100, Y: 200,
		Speed: 50, Angle: 0,
		Life: 1.0,
		Size: 4, SizeEnd: 1,
		Color: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})

	if pool.ActiveCount() != 1 {
		t.Errorf("expected 1 active, got %d", pool.ActiveCount())
	}

	// Update past lifetime.
	pool.Update(1.5)
	if pool.ActiveCount() != 0 {
		t.Errorf("expected 0 active after expiry, got %d", pool.ActiveCount())
	}
}

func TestParticleGravity(t *testing.T) {
	pool := particle.NewPool()
	pool.Spawn(particle.ParticleConfig{
		X: 0, Y: 0,
		Speed: 0, Angle: 0,
		Life: 10, Size: 1, SizeEnd: 1,
		Color:   color.RGBA{A: 255},
		Gravity: 100,
	})

	pool.Update(1.0)
	if pool.ActiveCount() != 1 {
		t.Error("particle should still be alive after 1s")
	}
}

func TestPoolRingBuffer(t *testing.T) {
	pool := particle.NewPool()
	// Spawn more than MaxParticles.
	for i := 0; i < particle.MaxParticles+10; i++ {
		pool.Spawn(particle.ParticleConfig{
			X: 0, Y: 0, Speed: 0, Life: 10,
			Size: 1, SizeEnd: 1,
			Color: color.RGBA{A: 255},
		})
	}
	// Should wrap around without crash.
	count := pool.ActiveCount()
	if count > particle.MaxParticles {
		t.Errorf("active count %d exceeds max %d", count, particle.MaxParticles)
	}
}

func TestEmitDeathBurst(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitDeathBurst(pool, 100, 200)
	count := pool.ActiveCount()
	if count < 8 || count > 16 {
		t.Errorf("death burst should spawn 8-16 particles, got %d", count)
	}
}

func TestEmitMuzzleFlash(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitMuzzleFlash(pool, 100, 200, math.Pi/4)
	if pool.ActiveCount() != 4 {
		t.Errorf("muzzle flash should spawn 4, got %d", pool.ActiveCount())
	}
}

func TestEmitGoldCollect(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitGoldCollect(pool, 300, 150)
	if pool.ActiveCount() != 5 {
		t.Errorf("gold collect should spawn 5, got %d", pool.ActiveCount())
	}
}

func TestEmitElectricSparks(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitElectricSparks(pool, 50, 50, 6)
	if pool.ActiveCount() != 6 {
		t.Errorf("electric sparks should spawn 6, got %d", pool.ActiveCount())
	}
}

func TestEmitFireParticles(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitFireParticles(pool, 100, 100, 3)
	if pool.ActiveCount() != 3 {
		t.Errorf("fire particles should spawn 3, got %d", pool.ActiveCount())
	}
}

func TestEmitIceParticles(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitIceParticles(pool, 100, 100, 4)
	if pool.ActiveCount() != 4 {
		t.Errorf("ice particles should spawn 4, got %d", pool.ActiveCount())
	}
}
