// flying.go — 飞行敌人路径系统。
// TODO: 飞行将作为能力实现，当前保留函数签名但功能待迁移。
package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// IsFlying 判断敌人是否为飞行单位。
// TODO: 迁移到能力系统后从能力列表判断。
func IsFlying(_ *Enemy) bool {
	return false
}

// ComputeFlyingPath 计算飞行路径（起点到终点的直线，两个路径点）。
func ComputeFlyingPath(startX, startY, endX, endY float64) []gamemap.Point {
	return []gamemap.Point{
		{X: startX, Y: startY},
		{X: endX, Y: endY},
	}
}

// MoveFlyingEnemy 驱动飞行敌人沿直线移动。
func MoveFlyingEnemy(e *Enemy, dt float64) bool {
	if e.ReachedEnd || len(e.Path) < 2 {
		return e.ReachedEnd
	}
	if e.StunTimer > 0 || e.RootTimer > 0 {
		return false
	}
	target := e.Path[len(e.Path)-1]
	dx := target.X - e.X
	dy := target.Y - e.Y
	dist := math.Hypot(dx, dy)
	step := e.Speed * dt
	if dist <= step {
		e.X = target.X
		e.Y = target.Y
		e.ReachedEnd = true
		return true
	}
	e.X += (dx / dist) * step
	e.Y += (dy / dist) * step
	return false
}
