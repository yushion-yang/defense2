package projectile

import (
	"math"

	"defense2/internal/core/game"
)

// Pool is a ring-buffer projectile pool.
type Pool struct {
	projectiles []Projectile
	cursor      int
	Count       int
}

// NewPool creates a projectile pool.
func NewPool(cap int) *Pool {
	return &Pool{
		projectiles: make([]Projectile, cap),
	}
}

// DefaultPool creates a pool with default capacity.
func DefaultPool() *Pool {
	return NewPool(game.MaxProjectiles)
}

// Fire creates a new projectile aimed at (tx, ty) from (sx, sy).
func (p *Pool) Fire(sx, sy, tx, ty, damage, speed, radius float64) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = 3.0
	proj.Life = proj.MaxLife

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// Update moves all active projectiles and deactivates expired ones.
func (p *Pool) Update(dt float64) {
	for i := range p.projectiles {
		proj := &p.projectiles[i]
		if !proj.Active {
			continue
		}
		proj.X += proj.VX * dt
		proj.Y += proj.VY * dt
		proj.Life -= dt

		if proj.Life <= 0 || proj.X < -50 || proj.X > float64(game.ScreenWidth)+50 ||
			proj.Y < -50 || proj.Y > float64(game.ScreenHeight)+50 {
			proj.Active = false
			p.Count--
		}
	}
}

// Each iterates over active projectiles.
func (p *Pool) Each(fn func(proj *Projectile)) {
	for i := range p.projectiles {
		if p.projectiles[i].Active {
			fn(&p.projectiles[i])
		}
	}
}

// Release deactivates a projectile (on hit).
func (p *Pool) Release(proj *Projectile) {
	if proj.Active {
		proj.Active = false
		p.Count--
	}
}

// ClearAll deactivates everything.
func (p *Pool) ClearAll() {
	for i := range p.projectiles {
		p.projectiles[i].Active = false
	}
	p.Count = 0
	p.cursor = 0
}
