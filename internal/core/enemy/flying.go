// flying.go — 飞行敌人路径系统。
// 飞行单位无视地面路径，直线飞向终点。
package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// IsFlying 判断敌人是否为飞行单位。
func IsFlying(e *Enemy) bool {
	return e.MovementType == "flying"
}

// ComputeFlyingPath 计算飞行路径（起点到终点的直线，两个路径点）。
func ComputeFlyingPath(startX, startY, endX, endY float64) []gamemap.Point {
	return []gamemap.Point{
		{X: startX, Y: startY},
		{X: endX, Y: endY},
	}
}

// MoveFlyingEnemy 驱动飞行敌人沿直线移动。
// 返回 true 表示已到达终点。
func MoveFlyingEnemy(e *Enemy, dt float64) bool {
	// 已到达终点
	if e.ReachedEnd {
		return true
	}

	// 路径点不足
	if len(e.Path) < 2 {
		return false
	}

	// 眩晕中不移动
	if e.StunTimer > 0 {
		return false
	}

	// 定身中不移动
	if e.RootTimer > 0 {
		return false
	}

	// 飞行终点为路径最后一个点
	target := e.Path[len(e.Path)-1]
	dx := target.X - e.X
	dy := target.Y - e.Y
	dist := math.Hypot(dx, dy)
	step := e.Speed * dt

	if dist <= step {
		// 到达终点
		e.X = target.X
		e.Y = target.Y
		e.ReachedEnd = true
		return true
	}

	// 沿方向向量前进
	e.X += (dx / dist) * step
	e.Y += (dy / dist) * step
	return false
}
