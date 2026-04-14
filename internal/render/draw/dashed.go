// dashed.go — 虚线绘制，与 line_batch 批量化集成。
//
// 提供虚线段和虚线圆两种图元，用于塔射程指示、区域边界等 VFX 效果。
// 当 lineBatch 激活时自动走批量路径，对调用方透明。
//
// 性能防护：DashedCircle 有 64 段硬上限(maxDashes)，防止极端半径导致
// 线段数爆炸。配合 vfxRadius(400) 视觉半径钳制，双重保障。
package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DashedLine 绘制从 (x1,y1) 到 (x2,y2) 的虚线。
// dashLen 为可见段长度，gapLen 为间隔长度，均为逻辑像素。
// 内部自动处理 HiDPI 缩放和 lineBatch 批量化。
func DashedLine(screen *ebiten.Image, x1, y1, x2, y2, width, dashLen, gapLen float32, clr color.Color) {
	// 所有参数一次性缩放到物理像素，后续计算无需再缩放
	x1, y1, x2, y2 = S32(x1), S32(y1), S32(x2), S32(y2)
	width, dashLen, gapLen = S32(width), S32(dashLen), S32(gapLen)
	dx := x2 - x1
	dy := y2 - y1
	totalLen := float32(math.Hypot(float64(dx), float64(dy)))
	if totalLen < 0.001 {
		return
	}

	// 单位方向向量，用于沿线段方向步进
	ux := dx / totalLen
	uy := dy / totalLen

	var dist float32
	drawing := true // 从可见段开始交替：dash → gap → dash → ...

	for dist < totalLen {
		if drawing {
			segEnd := dist + dashLen
			if segEnd > totalLen {
				segEnd = totalLen // 最后一段可能不足完整 dashLen
			}
			sx := x1 + ux*dist
			sy := y1 + uy*dist
			ex := x1 + ux*segEnd
			ey := y1 + uy*segEnd
			if lineBatch.active {
				batchStrokeLine(sx, sy, ex, ey, width, clr)
			} else {
				vector.StrokeLine(screen, sx, sy, ex, ey, width, clr, AA())
			}
			dist = segEnd
		} else {
			dist += gapLen
		}
		drawing = !drawing
	}
}

// DashedCircle 绘制以 (cx,cy) 为圆心、半径 r 的虚线圆。
// dashLen 和 gapLen 是沿圆周的弧长（非角度），单位为逻辑像素。
// 每段 dash 用直线近似弧线（段数足够时视觉上无差异）。
// 自适应：当圆周过大导致段数超过 maxDashes 时，自动等比放大 dashLen/gapLen，
// 保证始终画完整圈且段数不爆炸。
func DashedCircle(screen *ebiten.Image, cx, cy, r, width, dashLen, gapLen float32, clr color.Color) {
	cx, cy, r = S32(cx), S32(cy), S32(r)
	width, dashLen, gapLen = S32(width), S32(dashLen), S32(gapLen)
	if r <= 0 {
		return
	}

	circumference := 2 * math.Pi * float64(r)
	if circumference < 0.001 {
		return
	}

	// 自适应：预估 dash 段数，超过上限时等比放大 dashLen/gapLen
	const maxDashes = 64
	period := float64(dashLen + gapLen)
	if period > 0 {
		estDashes := circumference / period
		if estDashes > maxDashes {
			scale := float32(estDashes / maxDashes)
			dashLen *= scale
			gapLen *= scale
		}
	}

	batch := lineBatch.active
	aa := AA()
	var dist float64
	drawing := true
	draws := 0

	for dist < circumference && draws < maxDashes {
		if drawing {
			segEnd := dist + float64(dashLen)
			if segEnd > circumference {
				segEnd = circumference
			}

			// 弧长 → 角度：angle = arcLength / radius
			startAngle := dist / float64(r)
			endAngle := segEnd / float64(r)

			// 圆上两端点坐标
			sx := float32(float64(cx) + float64(r)*math.Cos(startAngle))
			sy := float32(float64(cy) + float64(r)*math.Sin(startAngle))
			ex := float32(float64(cx) + float64(r)*math.Cos(endAngle))
			ey := float32(float64(cy) + float64(r)*math.Sin(endAngle))

			if batch {
				batchStrokeLine(sx, sy, ex, ey, width, clr)
			} else {
				vector.StrokeLine(screen, sx, sy, ex, ey, width, clr, aa)
			}
			draws++
			dist = segEnd
		} else {
			dist += float64(gapLen)
		}
		drawing = !drawing
	}
}
