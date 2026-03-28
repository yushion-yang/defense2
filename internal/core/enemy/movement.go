// movement.go — 敌人沿路径移动。
// 敌人按路径点序列逐点前进，眩晕时停止移动。
package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// MoveAlongPath 驱动敌人向下一个路径点移动。
// 到达路径终点时返回 true（表示该敌人抵达基地）。
func MoveAlongPath(e *Enemy, waypoints []gamemap.Point, dt float64) bool {
	if e.PathIndex >= len(waypoints) {
		e.ReachedEnd = true
		return true
	}
	// 眩晕中：只消耗眩晕计时器，不移动
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
		// 足够接近，直接吸附到路径点并切换下一个
		e.X = target.X
		e.Y = target.Y
		e.PathIndex++
		if e.PathIndex >= len(waypoints) {
			e.ReachedEnd = true
			return true
		}
	} else {
		// 沿方向向量前进一步
		e.X += (dx / dist) * step
		e.Y += (dy / dist) * step
	}
	return false
}
