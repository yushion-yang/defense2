// movement.go — 敌人沿路径移动。
// 纯函数，无渲染/DOM 依赖。支持眩晕和定身冻结。
// 优先使用敌人自身绑定的路径（多路径地图），回退到传入的默认路径。
package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// MoveAlongPath 驱动敌人向下一个路径点移动。
// 到达路径终点时返回 true（表示该敌人抵达基地）。
func MoveAlongPath(e *Enemy, fallbackWaypoints []gamemap.Point, dt float64) bool {
	// 静止敌人（造怪放置/全怪展示）：不移动、不到达终点，但仍递减控制计时器
	if e.BaseSpeed == 0 && e.Speed == 0 {
		if e.StunTimer > 0 {
			e.StunTimer -= dt
		}
		if e.RootTimer > 0 {
			e.RootTimer -= dt
		}
		return false
	}

	// 确定行进路径（优先使用敌人自带路径）
	waypoints := e.Path
	if len(waypoints) == 0 {
		waypoints = fallbackWaypoints
	}

	if e.PathIndex >= len(waypoints) {
		e.ReachedEnd = true
		return true
	}

	// 眩晕中：只消耗计时器，不移动
	if e.StunTimer > 0 {
		e.StunTimer -= dt
		return false
	}

	// 定身中：不移动（计时在 TickStatusEffects 中处理）
	if e.RootTimer > 0 {
		return false
	}

	// 应用旗手光环加速（SpeedBuff 由 TickBehaviors 每帧写入）
	speed := e.Speed
	if e.SpeedBuff > 0 {
		speed *= (1 + e.SpeedBuff)
	}

	target := waypoints[e.PathIndex]
	dx := target.X - e.X
	dy := target.Y - e.Y
	dist := math.Hypot(dx, dy)
	step := speed * dt

	if dist <= step {
		// 足够接近，吸附到路径点并切换下一个
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
