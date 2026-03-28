// draw_thunder.go — 闪电视觉渲染。
// 管理闪电弹生命周期和 zigzag 线条绘制。
package render

import (
	"image/color"
	"math"
	"math/rand"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ThunderBolt 闪电视觉效果。
type ThunderBolt struct {
	X1, Y1  float64 // 起点
	X2, Y2  float64 // 终点
	Life    float64 // 剩余时间
	MaxLife float64 // 初始时间
	Primary bool    // 是否主击（粗线）
}

// ThunderBolts 全局闪电效果池。
var ThunderBolts []ThunderBolt

// SpawnThunderBolt 生成一条闪电视觉效果。
func SpawnThunderBolt(x1, y1, x2, y2 float64, primary bool) {
	life := 0.3
	if !primary {
		life = 0.2
	}
	ThunderBolts = append(ThunderBolts, ThunderBolt{
		X1: x1, Y1: y1,
		X2: x2, Y2: y2,
		Life:    life,
		MaxLife: life,
		Primary: primary,
	})
}

// UpdateThunderBolts 每帧更新闪电效果，移除过期的。
func UpdateThunderBolts(dt float64) {
	n := 0
	for i := range ThunderBolts {
		ThunderBolts[i].Life -= dt
		if ThunderBolts[i].Life > 0 {
			ThunderBolts[n] = ThunderBolts[i]
			n++
		}
	}
	ThunderBolts = ThunderBolts[:n]
}

// DrawThunderBolts 绘制所有存活的闪电效果（zigzag 线段 + 发光）。
func DrawThunderBolts(screen *ebiten.Image) {
	for i := range ThunderBolts {
		b := &ThunderBolts[i]
		drawSingleBolt(screen, b)
	}
}

// zigzagSegments 闪电 zigzag 分段数。
const zigzagSegments = 8

// drawSingleBolt 绘制单条闪电（zigzag 折线 + alpha 渐隐）。
func drawSingleBolt(screen *ebiten.Image, b *ThunderBolt) {
	// 透明度随生命衰减
	alpha := b.Life / b.MaxLife
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}

	// 闪电颜色（亮蓝白色）
	baseA := uint8(255 * alpha)
	coreClr := color.RGBA{R: 200, G: 220, B: 255, A: baseA}
	glowClr := color.RGBA{R: 120, G: 160, B: 255, A: uint8(float64(baseA) * 0.4)}

	// 线宽：主击粗，溅射细
	coreW := float32(2.0)
	glowW := float32(6.0)
	if b.Primary {
		coreW = 3.0
		glowW = 10.0
	}

	// 生成 zigzag 路径点
	dx := b.X2 - b.X1
	dy := b.Y2 - b.Y1
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		return
	}

	// 垂直偏移量（距离越长偏移越大）
	jitter := dist * 0.08
	if !b.Primary {
		jitter = dist * 0.05
	}

	// 法向量
	nx := -dy / dist
	ny := dx / dist

	// 构建折线点
	points := make([][2]float32, zigzagSegments+1)
	for i := 0; i <= zigzagSegments; i++ {
		t := float64(i) / float64(zigzagSegments)
		px := b.X1 + dx*t
		py := b.Y1 + dy*t

		// 首尾不偏移
		if i > 0 && i < zigzagSegments {
			offset := (rand.Float64()*2 - 1) * jitter
			px += nx * offset
			py += ny * offset
		}
		points[i] = [2]float32{float32(px), float32(py)}
	}

	// 画发光层（粗线，低透明度）
	for i := 0; i < len(points)-1; i++ {
		draw.Line(screen,
			points[i][0], points[i][1],
			points[i+1][0], points[i+1][1],
			glowW, glowClr, true)
	}

	// 画核心线（细线，高透明度）
	for i := 0; i < len(points)-1; i++ {
		draw.Line(screen,
			points[i][0], points[i][1],
			points[i+1][0], points[i+1][1],
			coreW, coreClr, true)
	}
}
