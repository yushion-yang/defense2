package enemy

import "defense2/internal/core/game"

// Pool is a fixed-size enemy pool.
type Pool struct {
	enemies []Enemy
	Count   int
}

// NewPool creates an enemy pool with the given capacity.
func NewPool(cap int) *Pool {
	return &Pool{
		enemies: make([]Enemy, cap),
	}
}

// DefaultPool creates a pool with default capacity.
func DefaultPool() *Pool {
	return NewPool(game.MaxEnemies)
}

// Spawn activates an enemy slot with the given parameters.
// pathIndex is typically 1 (enemy spawns at waypoint[0], moves toward [1]).
// Returns a pointer to the spawned enemy, or nil if pool is full.
func (p *Pool) Spawn(x, y, hp, speed, radius float64, pathIndex int) *Enemy {
	for i := range p.enemies {
		if !p.enemies[i].Active {
			e := &p.enemies[i]
			e.X = x
			e.Y = y
			e.HP = hp
			e.MaxHP = hp
			e.Speed = speed
			e.BaseSpeed = speed
			e.Radius = radius
			e.PathIndex = pathIndex
			e.ReachedEnd = false
			e.Active = true
			e.StunTimer = 0
			e.SlowTimer = 0
			e.SlowFactor = 1
			e.BleedTimer = 0
			e.BleedDPS = 0
			p.Count++
			return e
		}
	}
	return nil
}

// Kill deactivates an enemy.
func (p *Pool) Kill(e *Enemy) {
	if e.Active {
		e.Active = false
		p.Count--
	}
}

// Each iterates over all active enemies.
func (p *Pool) Each(fn func(e *Enemy)) {
	for i := range p.enemies {
		if p.enemies[i].Active {
			fn(&p.enemies[i])
		}
	}
}

// ClearAll deactivates all enemies.
func (p *Pool) ClearAll() {
	for i := range p.enemies {
		p.enemies[i].Active = false
	}
	p.Count = 0
}
