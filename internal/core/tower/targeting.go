package tower

import (
	"math"

	"defense2/internal/core/enemy"
)

// FindNearestEnemy returns the closest active enemy within tower's range, or nil.
func FindNearestEnemy(t *Tower, pool *enemy.Pool) *enemy.Enemy {
	var best *enemy.Enemy
	bestDist := math.MaxFloat64

	pool.Each(func(e *enemy.Enemy) {
		dx := e.X - t.X
		dy := e.Y - t.Y
		dist := math.Hypot(dx, dy)
		if dist <= t.Range && dist < bestDist {
			bestDist = dist
			best = e
		}
	})
	return best
}
