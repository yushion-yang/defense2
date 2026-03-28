// targeting.go — 塔索敌逻辑。
// 在塔射程范围内查找最近的存活敌人。
package tower

import (
	"math"

	"defense2/internal/core/enemy"
)

// FindNearestEnemy 返回塔射程内最近的存活敌人，无目标时返回 nil。
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
