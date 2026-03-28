// collision.go — 碰撞检测工具库。
// 提供圆形、点、线段等几何碰撞检测的纯数学函数，无外部依赖。
// 用于替换 tick_combat.go 中的内联距离计算。
package physics

import (
	"math"
	"sort"
)

// Circle 圆形碰撞体。
type Circle struct {
	X      float64 // 圆心X
	Y      float64 // 圆心Y
	Radius float64 // 半径
}

// Point 二维点。
type Point struct {
	X float64
	Y float64
}

// RangeResult 范围查询结果。
type RangeResult struct {
	Index    int     // 在候选列表中的索引
	Distance float64 // 到查询中心的距离
}

// LineHitResult 线段-圆碰撞结果。
type LineHitResult struct {
	Hit      bool    // 是否命中
	HitX     float64 // 命中点X
	HitY     float64 // 命中点Y
	Distance float64 // 命中点到线段起点的距离
}

// LineTargetResult 线段沿途命中结果。
type LineTargetResult struct {
	Index    int     // 在候选列表中的索引
	HitX     float64 // 命中点X
	HitY     float64 // 命中点Y
	Distance float64 // 命中点到线段起点的距离
}

// PointDistance 计算两点间的欧几里得距离。
func PointDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// PointDistanceSq 计算两点间距离的平方（避免开方，用于比较）。
func PointDistanceSq(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return dx*dx + dy*dy
}

// CirclesOverlap 判断两个圆是否重叠。
func CirclesOverlap(a, b Circle) bool {
	dx := b.X - a.X
	dy := b.Y - a.Y
	sumR := a.Radius + b.Radius
	return dx*dx+dy*dy <= sumR*sumR
}

// PointInCircle 判断点是否在圆内。
func PointInCircle(px, py float64, c Circle) bool {
	dx := px - c.X
	dy := py - c.Y
	return dx*dx+dy*dy <= c.Radius*c.Radius
}

// FindCirclesInRange 查找所有在指定范围内的圆，按距离升序排列。
// center: 查询中心; rangeVal: 查询半径; candidates: 候选圆列表。
func FindCirclesInRange(cx, cy, rangeVal float64, candidates []Circle) []RangeResult {
	var results []RangeResult
	for i, c := range candidates {
		dist := PointDistance(cx, cy, c.X, c.Y)
		if dist <= rangeVal+c.Radius {
			results = append(results, RangeResult{Index: i, Distance: dist})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})
	return results
}

// LineSegmentIntersectsCircle 检测线段是否与圆相交。
// 使用二次方程判别式法（quadratic discriminant method）。
func LineSegmentIntersectsCircle(x1, y1, x2, y2 float64, c Circle) LineHitResult {
	dx := x2 - x1
	dy := y2 - y1
	fx := x1 - c.X
	fy := y1 - c.Y

	a := dx*dx + dy*dy
	if a < 1e-12 {
		// 零长度线段，退化为点-圆检测
		if PointInCircle(x1, y1, c) {
			return LineHitResult{Hit: true, HitX: x1, HitY: y1, Distance: 0}
		}
		return LineHitResult{}
	}

	b := 2 * (fx*dx + fy*dy)
	cVal := fx*fx + fy*fy - c.Radius*c.Radius

	discriminant := b*b - 4*a*cVal
	if discriminant < 0 {
		return LineHitResult{} // 无交点
	}

	sqrtD := math.Sqrt(discriminant)

	// 检查两个交点（t1 <= t2）
	t1 := (-b - sqrtD) / (2 * a)
	t2 := (-b + sqrtD) / (2 * a)

	// 取第一个在 [0,1] 范围内的 t 值
	t := -1.0
	if t1 >= 0 && t1 <= 1 {
		t = t1
	} else if t2 >= 0 && t2 <= 1 {
		t = t2
	} else if t1 < 0 && t2 > 1 {
		// 线段完全在圆内
		t = 0
	}

	if t < 0 {
		return LineHitResult{}
	}

	hitX := x1 + t*dx
	hitY := y1 + t*dy
	dist := math.Sqrt(t * t * a) // t * sqrt(a) = t * 线段长度
	return LineHitResult{Hit: true, HitX: hitX, HitY: hitY, Distance: dist}
}

// FindCirclesAlongLine 查找线段沿途命中的所有圆，按命中距离升序排列。
// excludeIDs: 跳过的索引集合（已命中的目标）。
func FindCirclesAlongLine(x1, y1, x2, y2 float64, candidates []Circle, excludeIDs map[int]bool) []LineTargetResult {
	var results []LineTargetResult
	for i, c := range candidates {
		if excludeIDs != nil && excludeIDs[i] {
			continue
		}
		hit := LineSegmentIntersectsCircle(x1, y1, x2, y2, c)
		if hit.Hit {
			results = append(results, LineTargetResult{
				Index:    i,
				HitX:     hit.HitX,
				HitY:     hit.HitY,
				Distance: hit.Distance,
			})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})
	return results
}
