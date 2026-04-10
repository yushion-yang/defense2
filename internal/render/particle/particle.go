// particle.go — GPU-friendly particle pool with batch rendering.
// Fixed-size ring buffer of particles rendered in a single DrawTriangles call.
package particle

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// MaxParticles is the fixed pool capacity. Oldest particles are overwritten.
const MaxParticles = 2048

// Particle is a single visual particle.
type Particle struct {
	X, Y       float64
	VX, VY     float64
	Life       float64 // remaining life (seconds)
	MaxLife    float64
	Size       float64 // current size (logical pixels)
	SizeEnd    float64 // size at end of life (lerped)
	R, G, B, A float64 // colour components 0-1
	EndA       float64 // alpha at end of life (0-1)
	Gravity    float64 // Y acceleration per second
	Active     bool
}

// ParticleConfig describes how to spawn a single particle.
type ParticleConfig struct {
	X, Y             float64
	SpreadX, SpreadY float64    // random position spread
	Speed, SpeedVar  float64    // speed + random variance
	Angle, AngleVar  float64    // direction + random variance (radians)
	Life, LifeVar    float64    // lifetime + random variance
	Size, SizeEnd    float64    // start/end size (lerp over life)
	Color            color.RGBA // start colour
	EndAlpha         float64    // alpha at death (0-1)
	Gravity          float64    // Y acceleration
}

// Pool manages a fixed-size ring buffer of particles.
type Pool struct {
	particles [MaxParticles]Particle
	cursor    int
	active    int // tracked count of alive particles
	MaxActive int // quality cap; 0 = unlimited (use MaxParticles)
	Additive  bool // use additive (BlendLighter) blending instead of default BlendSourceOver
	vertices  []ebiten.Vertex
	indices   []uint16
}

// NewPool creates a new particle pool with pre-allocated vertex buffers.
func NewPool() *Pool {
	return &Pool{
		vertices: make([]ebiten.Vertex, 0, MaxParticles*4),
		indices:  make([]uint16, 0, MaxParticles*6),
	}
}

// Spawn adds a particle to the pool using the given config.
// Uses the ring-buffer cursor so oldest particles are overwritten when full.
// If MaxActive > 0 and the tracked active count has reached that cap,
// the spawn is silently skipped to respect quality settings.
func (p *Pool) Spawn(cfg ParticleConfig) {
	if p.MaxActive > 0 && p.active >= p.MaxActive {
		return
	}
	pt := &p.particles[p.cursor]
	// Track: if we are overwriting a still-active particle, active count stays the same.
	if !pt.Active {
		p.active++
	}
	p.cursor = (p.cursor + 1) % MaxParticles

	// Position with random spread.
	pt.X = cfg.X + randRange(-cfg.SpreadX, cfg.SpreadX)
	pt.Y = cfg.Y + randRange(-cfg.SpreadY, cfg.SpreadY)

	// Velocity from angle + speed.
	angle := cfg.Angle + randRange(-cfg.AngleVar, cfg.AngleVar)
	speed := cfg.Speed + randRange(-cfg.SpeedVar, cfg.SpeedVar)
	if speed < 0 {
		speed = 0
	}
	pt.VX = math.Cos(angle) * speed
	pt.VY = math.Sin(angle) * speed

	// Lifetime.
	pt.Life = cfg.Life + randRange(-cfg.LifeVar, cfg.LifeVar)
	if pt.Life < 0.01 {
		pt.Life = 0.01
	}
	pt.MaxLife = pt.Life

	// Size.
	pt.Size = cfg.Size
	pt.SizeEnd = cfg.SizeEnd

	// Colour (RGBA 0-1).
	pt.R = float64(cfg.Color.R) / 255.0
	pt.G = float64(cfg.Color.G) / 255.0
	pt.B = float64(cfg.Color.B) / 255.0
	pt.A = float64(cfg.Color.A) / 255.0
	pt.EndA = cfg.EndAlpha

	pt.Gravity = cfg.Gravity
	pt.Active = true
}

// Update ticks all active particles by dt seconds.
func (p *Pool) Update(dt float64) {
	for i := range p.particles {
		pt := &p.particles[i]
		if !pt.Active {
			continue
		}
		pt.Life -= dt
		if pt.Life <= 0 {
			pt.Active = false
			p.active--
			continue
		}
		pt.VY += pt.Gravity * dt
		pt.X += pt.VX * dt
		pt.Y += pt.VY * dt
	}
}

// whitePixel is lazily initialised to avoid GPU panics during init().
var whitePixel *ebiten.Image

func getWhitePixel() *ebiten.Image {
	if whitePixel == nil {
		whitePixel = ebiten.NewImage(3, 3)
		whitePixel.Fill(color.White)
	}
	return whitePixel
}

// Draw renders all active particles with a single DrawTriangles call.
// Positions and sizes are scaled via draw.S() for HiDPI.
func (p *Pool) Draw(screen *ebiten.Image) {
	p.vertices = p.vertices[:0]
	p.indices = p.indices[:0]

	idx := uint16(0)
	for i := range p.particles {
		pt := &p.particles[i]
		if !pt.Active {
			continue
		}

		// Progress 0→1 over lifetime.
		t := 1.0 - pt.Life/pt.MaxLife

		// Lerp size.
		size := pt.Size + (pt.SizeEnd-pt.Size)*t
		// Lerp alpha.
		alpha := pt.A + (pt.EndA-pt.A)*t

		// Scale to physical pixels.
		sx := draw.S(pt.X)
		sy := draw.S(pt.Y)
		half := draw.S(size * 0.5)

		r := float32(pt.R * alpha)
		g := float32(pt.G * alpha)
		b := float32(pt.B * alpha)
		a := float32(alpha)

		// Quad: 4 vertices, 6 indices.
		v0 := ebiten.Vertex{DstX: float32(sx - half), DstY: float32(sy - half), SrcX: 1, SrcY: 1, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
		v1 := ebiten.Vertex{DstX: float32(sx + half), DstY: float32(sy - half), SrcX: 2, SrcY: 1, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
		v2 := ebiten.Vertex{DstX: float32(sx + half), DstY: float32(sy + half), SrcX: 2, SrcY: 2, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
		v3 := ebiten.Vertex{DstX: float32(sx - half), DstY: float32(sy + half), SrcX: 1, SrcY: 2, ColorR: r, ColorG: g, ColorB: b, ColorA: a}

		p.vertices = append(p.vertices, v0, v1, v2, v3)
		p.indices = append(p.indices, idx, idx+1, idx+2, idx, idx+2, idx+3)
		idx += 4
	}

	if len(p.vertices) == 0 {
		return
	}

	blend := ebiten.BlendSourceOver
	if p.Additive {
		blend = ebiten.BlendLighter
	}
	screen.DrawTriangles(p.vertices, p.indices, getWhitePixel(), &ebiten.DrawTrianglesOptions{
		Blend: blend,
	})
}

// Clear deactivates all particles immediately.
func (p *Pool) Clear() {
	for i := range p.particles {
		p.particles[i].Active = false
	}
	p.active = 0
}

// ActiveCount returns the number of currently active (alive) particles.
// Uses the incrementally tracked counter — O(1).
func (p *Pool) ActiveCount() int {
	return p.active
}
