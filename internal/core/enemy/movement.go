package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// MoveAlongPath moves an enemy toward the next waypoint.
// Returns true if the enemy reached the end of the path this tick.
func MoveAlongPath(e *Enemy, waypoints []gamemap.Point, dt float64) bool {
	if e.PathIndex >= len(waypoints) {
		e.ReachedEnd = true
		return true
	}
	if e.StunTimer > 0 {
		e.StunTimer -= dt
		return false
	}

	target := waypoints[e.PathIndex]
	dx := target.X - e.X
	dy := target.Y - e.Y
	dist := math.Hypot(dx, dy)
	step := e.Speed * dt

	if dist <= step {
		e.X = target.X
		e.Y = target.Y
		e.PathIndex++
		if e.PathIndex >= len(waypoints) {
			e.ReachedEnd = true
			return true
		}
	} else {
		e.X += (dx / dist) * step
		e.Y += (dy / dist) * step
	}
	return false
}
