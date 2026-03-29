// draw_skill_vfx.go — 技能施展视觉特效（精修版）。
package render

import (
	"image/color"
	"math"
	"math/rand"

	"defense2/internal/core/skill"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// vfxTick 全局帧计数器（用于动画）。
var vfxTick float64

// UpdateVFXTick 每帧调用，驱动 VFX 动画。
func UpdateVFXTick(dt float64) { vfxTick += dt }

// DrawSkillVFX 绘制技能施展特效。
func DrawSkillVFX(screen *ebiten.Image, state *skill.SkillState) {
	vfx := skill.GetSkillVFX(state)
	if vfx == nil || !vfx.Active {
		return
	}

	switch vfx.Type {
	case "lightning":
		drawLightningVFX(screen, vfx)
	case "explosion":
		drawExplosionVFX(screen, vfx)
	case "projectile":
		drawSkillProjectileVFX(screen, vfx)
	case "laser":
		drawLaserVFX(screen, vfx)
	case "blades":
		drawBladesVFX(screen, vfx)
	case "barrage":
		drawBarrageVFX(screen, vfx)
	case "beam_burst":
		drawBeamBurstVFX(screen, vfx)
	case "rain":
		drawRainVFX(screen, vfx)
	case "sweep_beam":
		drawSweepBeamVFX(screen, vfx)
	case "bolts":
		drawBoltsVFX(screen, vfx)
	}
}

// ════════════════════════════════════════════════════════════
// 闪电链 — 锯齿折线 + 分支 + 命中爆闪
// ════════════════════════════════════════════════════════════

func drawLightningVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	for i := 0; i < len(vfx.Points)-1; i++ {
		x1, y1 := float32(vfx.Points[i][0]), float32(vfx.Points[i][1])
		x2, y2 := float32(vfx.Points[i+1][0]), float32(vfx.Points[i+1][1])
		drawJaggedBolt(screen, x1, y1, x2, y2, 6, 1.0)
		// 命中点爆闪
		draw.Glow(screen, x2, y2, 3, 10, color.RGBA{R: 140, G: 210, B: 255, A: 120})
		draw.FilledCircle(screen, x2, y2, 4, color.RGBA{R: 200, G: 240, B: 255, A: 220})
	}
	// 起点光晕
	if len(vfx.Points) > 0 {
		x0, y0 := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
		draw.Glow(screen, x0, y0, 2, 8, color.RGBA{R: 100, G: 180, B: 255, A: 100})
	}
}

// drawJaggedBolt 绘制锯齿状闪电弧线。
func drawJaggedBolt(screen *ebiten.Image, x1, y1, x2, y2 float32, segments int, intensity float64) {
	dx := x2 - x1
	dy := y2 - y1
	dist := float32(math.Hypot(float64(dx), float64(dy)))
	if dist < 2 {
		return
	}
	nx := -dy / dist // 法线
	ny := dx / dist

	pts := make([][2]float32, segments+1)
	pts[0] = [2]float32{x1, y1}
	pts[segments] = [2]float32{x2, y2}

	jitter := dist * 0.12 * float32(intensity)
	for i := 1; i < segments; i++ {
		t := float32(i) / float32(segments)
		mx := x1 + dx*t
		my := y1 + dy*t
		offset := (rand.Float32()*2 - 1) * jitter
		pts[i] = [2]float32{mx + nx*offset, my + ny*offset}
	}

	glowA := uint8(60 * intensity)
	coreA := uint8(220 * intensity)
	brightA := uint8(255 * intensity)
	for i := 0; i < segments; i++ {
		ax, ay := pts[i][0], pts[i][1]
		bx, by := pts[i+1][0], pts[i+1][1]
		draw.Line(screen, ax, ay, bx, by, 5, color.RGBA{R: 60, G: 120, B: 255, A: glowA}, true)
		draw.Line(screen, ax, ay, bx, by, 2, color.RGBA{R: 130, G: 200, B: 255, A: coreA}, true)
		draw.Line(screen, ax, ay, bx, by, 0.8, color.RGBA{R: 220, G: 240, B: 255, A: brightA}, true)
	}
}

// ════════════════════════════════════════════════════════════
// 核弹爆炸 — 多层辐射 + 冲击波环 + 碎片
// ════════════════════════════════════════════════════════════

func drawExplosionVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	cx, cy := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
	p := float32(vfx.Timer / 0.3)
	if p > 1 {
		p = 1
	}
	r := float32(vfx.Radius)
	fade := 1.0 - p // 0→1 淡出

	// 外圈冲击波（扩张 + 淡出）
	waveR := r * (0.6 + 0.6*p)
	draw.CircleOutline(screen, cx, cy, waveR, 3, color.RGBA{R: 255, G: 180, B: 60, A: uint8(200 * fade)})
	draw.CircleOutline(screen, cx, cy, waveR*0.85, 1.5, color.RGBA{R: 255, G: 220, B: 100, A: uint8(140 * fade)})

	// 中层火球（收缩）
	fireR := r * 0.5 * (1 - p*0.5)
	draw.Glow(screen, cx, cy, fireR*0.3, fireR, color.RGBA{R: 255, G: 140, B: 30, A: uint8(180 * fade)})
	draw.FilledCircle(screen, cx, cy, fireR*0.4, color.RGBA{R: 255, G: 220, B: 80, A: uint8(220 * fade)})
	draw.FilledCircle(screen, cx, cy, fireR*0.15, color.RGBA{R: 255, G: 255, B: 200, A: uint8(255 * fade)})

	// 辐射碎片线
	rng := rand.New(rand.NewSource(int64(vfx.Timer * 1000)))
	fragCount := 12
	for i := 0; i < fragCount; i++ {
		angle := float64(i) * math.Pi * 2 / float64(fragCount)
		fragLen := float64(r) * (0.4 + rng.Float64()*0.4) * float64(p)
		fx := cx + float32(math.Cos(angle)*fragLen)
		fy := cy + float32(math.Sin(angle)*fragLen)
		draw.Line(screen, cx, cy, fx, fy, 1.5, color.RGBA{R: 255, G: 200, B: 80, A: uint8(120 * fade)}, true)
	}
}

// ════════════════════════════════════════════════════════════
// 核弹飞行体 — 发光弹头 + 拖尾
// ════════════════════════════════════════════════════════════

func drawSkillProjectileVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	px, py := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
	// 拖尾光晕
	draw.Glow(screen, px, py, 3, 14, color.RGBA{R: 255, G: 120, B: 20, A: 60})
	// 弹头
	draw.FilledCircle(screen, px, py, 5, color.RGBA{R: 255, G: 180, B: 40, A: 240})
	draw.FilledCircle(screen, px, py, 2.5, color.RGBA{R: 255, G: 240, B: 180, A: 255})
	// 如果有目标点，画虚线预瞄
	if len(vfx.Points) >= 2 {
		tx, ty := float32(vfx.Points[1][0]), float32(vfx.Points[1][1])
		draw.DashedLine(screen, px, py, tx, ty, 0.8, 4, 6, color.RGBA{R: 255, G: 160, B: 40, A: 40})
	}
}

// ════════════════════════════════════════════════════════════
// 引导激光 — 三层光束 + 边缘粒子 + 脉动宽度
// ════════════════════════════════════════════════════════════

func drawLaserVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 2 {
		return
	}
	x1, y1 := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
	x2, y2 := float32(vfx.Points[1][0]), float32(vfx.Points[1][1])
	w := float32(vfx.Radius)
	if w < 4 {
		w = 4
	}
	// 脉动宽度
	pulse := float32(1.0 + 0.15*math.Sin(vfxTick*12))
	pw := w * pulse

	// 外层光晕
	draw.Line(screen, x1, y1, x2, y2, pw*1.2, color.RGBA{R: 255, G: 60, B: 40, A: 30}, true)
	// 中层
	draw.Line(screen, x1, y1, x2, y2, pw*0.6, color.RGBA{R: 255, G: 100, B: 70, A: 140}, true)
	// 核心
	draw.Line(screen, x1, y1, x2, y2, pw*0.2, color.RGBA{R: 255, G: 200, B: 180, A: 240}, true)
	// 白芯
	draw.Line(screen, x1, y1, x2, y2, pw*0.08, color.RGBA{R: 255, G: 255, B: 240, A: 255}, true)

	// 起点聚能光
	draw.Glow(screen, x1, y1, 3, 12, color.RGBA{R: 255, G: 100, B: 60, A: 80})
}

// ════════════════════════════════════════════════════════════
// 风刃旋舞 — 旋转叶片 + 螺旋气流线
// ════════════════════════════════════════════════════════════

func drawBladesVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	cx, cy := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
	rot := vfxTick * 8 // 旋转角度

	clrOuter := color.RGBA{R: 100, G: 255, B: 180, A: 80}
	clrBlade := color.RGBA{R: 160, G: 255, B: 200, A: 200}
	clrCore := color.RGBA{R: 220, G: 255, B: 240, A: 255}

	// 外圈气流
	draw.CircleOutline(screen, cx, cy, 24, 1, clrOuter)
	draw.CircleOutline(screen, cx, cy, 18, 0.8, color.RGBA{R: 120, G: 255, B: 190, A: 50})

	// 6 片旋转叶片
	for i := 0; i < 6; i++ {
		a := rot + float64(i)*math.Pi/3
		// 内端
		ix := cx + 6*float32(math.Cos(a))
		iy := cy + 6*float32(math.Sin(a))
		// 外端
		ox := cx + 22*float32(math.Cos(a))
		oy := cy + 22*float32(math.Sin(a))
		draw.Line(screen, ix, iy, ox, oy, 2.5, clrBlade, true)
		draw.Line(screen, ix, iy, ox, oy, 1, clrCore, true)
	}

	// 中心旋涡
	draw.FilledCircle(screen, cx, cy, 4, color.RGBA{R: 180, G: 255, B: 220, A: 180})
	draw.Glow(screen, cx, cy, 2, 8, color.RGBA{R: 100, G: 255, B: 180, A: 60})
}

// ════════════════════════════════════════════════════════════
// 导弹齐射 — 发射闪光 + 烟尘环
// ════════════════════════════════════════════════════════════

func drawBarrageVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	ox, oy := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])

	// 发射烟尘环
	pulse := float32(0.8 + 0.4*math.Sin(vfxTick*16))
	draw.Glow(screen, ox, oy, 6, 20*pulse, color.RGBA{R: 255, G: 160, B: 40, A: 60})
	draw.CircleOutline(screen, ox, oy, 16*pulse, 1.5, color.RGBA{R: 255, G: 180, B: 60, A: 120})

	// 闪光中心
	draw.FilledCircle(screen, ox, oy, 5, color.RGBA{R: 255, G: 200, B: 80, A: 220})
	draw.FilledCircle(screen, ox, oy, 2.5, color.RGBA{R: 255, G: 255, B: 200, A: 255})
}

// ════════════════════════════════════════════════════════════
// 审判光束（旧版爆发，保留兼容）
// ════════════════════════════════════════════════════════════

func drawBeamBurstVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	ox, oy := float32(vfx.Points[0][0]), float32(vfx.Points[0][1])
	p := float32(vfx.Timer / 0.5)
	if p > 1 {
		p = 1
	}
	// 从中心向每个命中点画光束
	for i := 1; i < len(vfx.Points); i++ {
		tx, ty := float32(vfx.Points[i][0]), float32(vfx.Points[i][1])
		draw.Line(screen, ox, oy, tx, ty, 4*p, color.RGBA{R: 255, G: 240, B: 160, A: uint8(200 * p)}, true)
		draw.Line(screen, ox, oy, tx, ty, 1.5*p, color.RGBA{R: 255, G: 255, B: 220, A: uint8(255 * p)}, true)
		draw.Glow(screen, tx, ty, 2, float32(8*p), color.RGBA{R: 255, G: 255, B: 200, A: uint8(140 * p)})
	}
	draw.Glow(screen, ox, oy, 4, float32(16*p), color.RGBA{R: 255, G: 255, B: 180, A: uint8(80 * p)})
}

// ════════════════════════════════════════════════════════════
// 审判之雨 — 光柱 + 落地水花 + 星芒
// ════════════════════════════════════════════════════════════

func drawRainVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	for _, pt := range vfx.Points {
		tx, ty := float32(pt[0]), float32(pt[1])
		// 天降光柱（长尾 + 渐隐）
		topY := ty - 60
		draw.Line(screen, tx, topY, tx, ty, 3, color.RGBA{R: 180, G: 140, B: 255, A: 40}, true)
		draw.Line(screen, tx, topY+20, tx, ty, 1.5, color.RGBA{R: 200, G: 160, B: 255, A: 140}, true)
		draw.Line(screen, tx, topY+40, tx, ty, 0.8, color.RGBA{R: 230, G: 200, B: 255, A: 220}, true)
		// 雨滴头部亮点
		draw.FilledCircle(screen, tx, ty, 3.5, color.RGBA{R: 220, G: 190, B: 255, A: 230})
		draw.FilledCircle(screen, tx, ty, 1.5, color.RGBA{R: 255, G: 240, B: 255, A: 255})
		// 落地光圈
		draw.Glow(screen, tx, ty, 1, 8, color.RGBA{R: 180, G: 140, B: 255, A: 50})
	}
}

// ════════════════════════════════════════════════════════════
// 审判光束 — 全图纵向扫荡 + 边缘辉光 + 横向散射线
// ════════════════════════════════════════════════════════════

func drawSweepBeamVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	w := float32(vfx.Radius)
	if w < 4 {
		w = 4
	}
	p := float32(vfx.Timer / 1.0)
	if p > 1 {
		p = 1
	}
	a := p // 淡出因子

	const screenH float32 = 540
	pulse := float32(1.0 + 0.1*math.Sin(vfxTick*14))
	pw := w * pulse

	for _, pt := range vfx.Points {
		bx := float32(pt[0])
		// 最外层泛光
		draw.Line(screen, bx, 0, bx, screenH, pw*1.5, color.RGBA{R: 255, G: 220, B: 60, A: uint8(25 * a)}, true)
		// 外层辉光
		draw.Line(screen, bx, 0, bx, screenH, pw, color.RGBA{R: 255, G: 200, B: 40, A: uint8(60 * a)}, true)
		// 核心
		draw.Line(screen, bx, 0, bx, screenH, pw*0.35, color.RGBA{R: 255, G: 240, B: 120, A: uint8(200 * a)}, true)
		// 白芯
		draw.Line(screen, bx, 0, bx, screenH, pw*0.1, color.RGBA{R: 255, G: 255, B: 220, A: uint8(255 * a)}, true)

		// 横向散射线（每隔 40px 一条短横线，增加能量感）
		for y := float32(20); y < screenH; y += 40 {
			halfW := pw * 0.6
			draw.Line(screen, bx-halfW, y, bx+halfW, y, 0.5, color.RGBA{R: 255, G: 255, B: 200, A: uint8(40 * a)}, true)
		}
	}
}

// ════════════════════════════════════════════════════════════
// 闪电风暴 — 闪电球 + 锯齿尾迹 + 电弧光晕
// ════════════════════════════════════════════════════════════

func drawBoltsVFX(screen *ebiten.Image, vfx *skill.SkillVFX) {
	if len(vfx.Points) < 1 {
		return
	}
	for _, pt := range vfx.Points {
		bx, by := float32(pt[0]), float32(pt[1])
		// 外层电弧光晕
		draw.Glow(screen, bx, by, 4, 16, color.RGBA{R: 60, G: 140, B: 255, A: 50})
		// 闪电球核心
		draw.FilledCircle(screen, bx, by, 6, color.RGBA{R: 120, G: 200, B: 255, A: 240})
		draw.FilledCircle(screen, bx, by, 3, color.RGBA{R: 200, G: 240, B: 255, A: 255})
		// 随机电弧分支（4 条短线）
		for i := 0; i < 4; i++ {
			a := rand.Float64() * math.Pi * 2
			l := 6 + rand.Float32()*8
			ex := bx + l*float32(math.Cos(a))
			ey := by + l*float32(math.Sin(a))
			draw.Line(screen, bx, by, ex, ey, 1, color.RGBA{R: 140, G: 210, B: 255, A: 160}, true)
		}
	}
}
