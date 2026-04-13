// circle.go — draw 包最大的绘图文件，包含所有基础图元函数。
//
// 本文件是游戏渲染的核心 API 层，提供以下图元：
//   - 圆：CircleOutline / FilledCircle / Glow
//   - 线：Line / ThickLine
//   - 矩形：FilledRect
//   - 菱形：Diamond / DiamondRotated
//   - 弧线：Arc
//   - 精灵：Sprite / SpriteRotated / SpriteScaled / SpriteScaledRotated / SpriteScaledRotatedAlpha
//
// 所有函数接受逻辑坐标(1200x540)，内部自动乘以 Scale 转为物理像素。
// 线段类图元(Diamond/Arc/ThickLine)内部检查 lineBatch.active，
// 激活时自动走批量路径——对调用方完全透明，VFX 代码无需任何修改。
package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// circleSegments 圆弧近似的默认分段数（48 段 ≈ 每段 7.5°，视觉上足够平滑）。
const circleSegments = 48

// CircleOutline 绘制圆形轮廓线。
// 坐标和尺寸均为逻辑像素，内部自动 HiDPI 缩放。
func CircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	if r <= 0 {
		return
	}
	vector.StrokeCircle(screen, S32(cx), S32(cy), S32(r), S32(width), clr, AA())
}

// FilledCircle 绘制实心圆。
func FilledCircle(screen *ebiten.Image, cx, cy, r float32, clr color.Color) {
	vector.DrawFilledCircle(screen, S32(cx), S32(cy), S32(r), clr, AA())
}

// Glow 绘制柔和辉光效果：外圈半透明 + 内圈实心，两层叠加模拟光晕。
// 外圈 alpha 自动设为原色 1/4，产生从中心到边缘的渐淡效果。
//
// 当辉光通道激活(BeginGlowPass)时，绘制重定向到离屏缓冲，
// 最终通过加法混合合成，避免半透明叠加变不透明的问题。
func Glow(screen *ebiten.Image, cx, cy, innerR, outerR float32, clr color.RGBA) {
	if outerR <= 0 {
		return
	}

	// 辉光通道激活时重定向到离屏缓冲
	target := screen
	if GlowPassActive() {
		target = GlowTarget()
	}

	scx, scy := S32(cx), S32(cy)
	// 外圈：alpha 降为 1/4，形成柔和边缘
	outerClr := color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 4}
	vector.DrawFilledCircle(target, scx, scy, S32(outerR), outerClr, AA())

	if innerR > 0 {
		vector.DrawFilledCircle(target, scx, scy, S32(innerR), clr, AA())
	}
}

// Diamond 绘制菱形（旋转 45° 的正方形）轮廓，中心在 (cx,cy)，对角半径 r。
// 由 4 条线段组成。当 lineBatch 激活时自动走批量路径。
// 常用于塔 VFX 中的射程/攻击指示器。
func Diamond(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	// 菱形四个顶点：上、右、下、左
	top := [2]float32{scx, scy - sr}
	right := [2]float32{scx + sr, scy}
	bottom := [2]float32{scx, scy + sr}
	left := [2]float32{scx - sr, scy}

	// 批量模式：收集到顶点缓冲，由 FlushLineBatch 统一输出
	if lineBatch.active {
		batchStrokeLine(top[0], top[1], right[0], right[1], sw, clr)
		batchStrokeLine(right[0], right[1], bottom[0], bottom[1], sw, clr)
		batchStrokeLine(bottom[0], bottom[1], left[0], left[1], sw, clr)
		batchStrokeLine(left[0], left[1], top[0], top[1], sw, clr)
		return
	}
	// 非批量模式：逐条直接渲染
	aa := AA()
	vector.StrokeLine(screen, top[0], top[1], right[0], right[1], sw, clr, aa)
	vector.StrokeLine(screen, right[0], right[1], bottom[0], bottom[1], sw, clr, aa)
	vector.StrokeLine(screen, bottom[0], bottom[1], left[0], left[1], sw, clr, aa)
	vector.StrokeLine(screen, left[0], left[1], top[0], top[1], sw, clr, aa)
}

// DiamondRotated 绘制旋转菱形轮廓。angle 为弧度，围绕中心旋转。
// 用于 spin_aoe 等旋转攻击方式的 VFX 动画。
func DiamondRotated(screen *ebiten.Image, cx, cy, r, width float32, angle float64, clr color.Color) {
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	cos, sin := float32(math.Cos(angle)), float32(math.Sin(angle))

	// 先计算未旋转菱形顶点的偏移量，再应用 2D 旋转矩阵
	offsets := [4][2]float32{
		{0, -sr}, // 上
		{sr, 0},  // 右
		{0, sr},  // 下
		{-sr, 0}, // 左
	}
	var pts [4][2]float32
	for i, o := range offsets {
		// 旋转变换: x' = x*cos - y*sin, y' = x*sin + y*cos
		pts[i][0] = scx + o[0]*cos - o[1]*sin
		pts[i][1] = scy + o[0]*sin + o[1]*cos
	}

	if lineBatch.active {
		batchStrokeLine(pts[0][0], pts[0][1], pts[1][0], pts[1][1], sw, clr)
		batchStrokeLine(pts[1][0], pts[1][1], pts[2][0], pts[2][1], sw, clr)
		batchStrokeLine(pts[2][0], pts[2][1], pts[3][0], pts[3][1], sw, clr)
		batchStrokeLine(pts[3][0], pts[3][1], pts[0][0], pts[0][1], sw, clr)
		return
	}
	aa := AA()
	vector.StrokeLine(screen, pts[0][0], pts[0][1], pts[1][0], pts[1][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[1][0], pts[1][1], pts[2][0], pts[2][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[2][0], pts[2][1], pts[3][0], pts[3][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[3][0], pts[3][1], pts[0][0], pts[0][1], sw, clr, aa)
}

// ThickLine 绘制带圆端帽的粗线段。
// 非批量模式下使用 StrokePath + LineCapRound 获得圆润端点；
// 批量模式下退化为矩形线段（无圆端帽，但性能更高且视觉差异极小）。
func ThickLine(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color) {
	if lineBatch.active {
		batchStrokeLine(S32(x1), S32(y1), S32(x2), S32(y2), S32(width), clr)
		return
	}
	var path vector.Path
	path.MoveTo(S32(x1), S32(y1))
	path.LineTo(S32(x2), S32(y2))

	vector.StrokePath(screen, &path, &vector.StrokeOptions{
		Width:   S32(width),
		LineCap: vector.LineCapRound,
	}, &vector.DrawPathOptions{
		AntiAlias:  AA(),
		ColorScale: colorScale(clr),
	})
}

// Line 绘制普通线段（自动 HiDPI 缩放）。
// aa 参数由调用方控制，通常传 AA() 的返回值。
func Line(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color, aa bool) {
	vector.StrokeLine(screen, S32(x1), S32(y1), S32(x2), S32(y2), S32(width), clr, aa)
}

// FilledRect 绘制实心矩形（自动 HiDPI 缩放）。
func FilledRect(screen *ebiten.Image, x, y, w, h float32, clr color.Color, aa bool) {
	vector.DrawFilledRect(screen, S32(x), S32(y), S32(w), S32(h), clr, aa)
}

// Sprite 以逻辑坐标 (cx,cy) 为中心绘制精灵图像。
// displaySize 是逻辑空间中期望的显示宽度，函数自动计算缩放比并处理 HiDPI。
//
// GeoM 变换顺序：平移原点到图像中心 → 缩放 → 平移到目标位置。
// 这保证图像始终以 (cx,cy) 为中心对齐。
func Sprite(screen, img *ebiten.Image, cx, cy, displaySize float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	_ = h // h 未使用但保持可读性（缩放基于宽度，宽高比由图片自然保持）
	s := displaySize / w * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)         // 原点移到图像中心
	op.GeoM.Scale(s, s)                   // 统一缩放（保持宽高比）
	op.GeoM.Translate(cx*Scale, cy*Scale) // 移到目标物理像素位置
	screen.DrawImage(img, &op)
}

// SpriteRotated 绘制带旋转的精灵。rotation 为弧度，offsetY 为逻辑垂直偏移。
// offsetY 用于敌人行走抖动等动画效果——在最终平移时叠加。
func SpriteRotated(screen, img *ebiten.Image, cx, cy, displaySize, rotation, offsetY float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := displaySize / w * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2) // 原点居中（旋转轴 = 图像中心）
	op.GeoM.Rotate(rotation)      // 先旋转再缩放，保持旋转中心
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, (cy+offsetY)*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaled 以预计算的逻辑缩放因子绘制精灵。
// 与 Sprite 不同，这里 logicalScale 是直接乘以图像原始尺寸的比例，
// 适合需要动态缩放的场景（如火焰脉冲动画）。
func SpriteScaled(screen, img *ebiten.Image, cx, cy, logicalScale float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaledRotated 绘制带缩放和旋转的精灵。
// rotation 为弧度（0=原始方向，正值=顺时针）。
func SpriteScaledRotated(screen, img *ebiten.Image, cx, cy, logicalScale, rotation float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaledRotatedAlpha 绘制带缩放、旋转和透明度的精灵（最完整的变体）。
// alpha 范围 0.0(完全透明) ~ 1.0(不透明)，用于淡入淡出和隐身敌人(alpha=15%)等。
func SpriteScaledRotatedAlpha(screen, img *ebiten.Image, cx, cy, logicalScale, rotation, alpha float64) {
	if img == nil || alpha <= 0 {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	if alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(img, &op)
}

// Arc 绘制弧线（部分圆轮廓），从 startAngle 到 endAngle（弧度）。
// 用折线段近似弧线，段数与弧度跨度成正比（基于 circleSegments=48 的全圆分辨率）。
// 最少 4 段，保证小弧度也有足够平滑度。
// 当 lineBatch 激活时自动走批量路径。
func Arc(screen *ebiten.Image, cx, cy, r, startAngle, endAngle, width float32, clr color.Color) {
	if r <= 0 {
		return
	}
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	span := endAngle - startAngle
	// 段数与弧度跨度成正比：完整圆=48段，半圆=24段，以此类推
	segments := int(float64(circleSegments) * float64(span) / (2 * math.Pi))
	if segments < 4 {
		segments = 4
	}
	step := float64(span) / float64(segments)
	batch := lineBatch.active
	aa := AA()
	var prevX, prevY float32
	for i := 0; i <= segments; i++ {
		angle := float64(startAngle) + float64(i)*step
		x := scx + sr*float32(math.Cos(angle))
		y := scy + sr*float32(math.Sin(angle))
		if i > 0 {
			if batch {
				batchStrokeLine(prevX, prevY, x, y, sw, clr)
			} else {
				vector.StrokeLine(screen, prevX, prevY, x, y, sw, clr, aa)
			}
		}
		prevX = x
		prevY = y
	}
}
