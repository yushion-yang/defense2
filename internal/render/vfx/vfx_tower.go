// vfx_tower.go — 炮塔视觉效果（解耦版）。
// 所有函数只接受纯值参数，零 core 包依赖。
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 建造 / 出售 / 射击动画参数 ──────────────────────

// overshootEase 回弹缓动：先超调到 1.1 再回落到 1.0。
func overshootEase(t float64) float64 {
	if t < 0.7 {
		return (t / 0.7) * 1.1
	}
	return 1.1 - 0.1*((t-0.7)/0.3)
}

// BuildAnimParams 计算建造动画的缩放和透明度。
// buildAnim: 剩余时间（初始 0.3，递减到 0）。
// 返回 (scaleMul, alpha)，无动画时返回 (1, 1)。
func BuildAnimParams(buildAnim float64) (float64, float64) {
	if buildAnim <= 0 {
		return 1.0, 1.0
	}
	progress := 1.0 - buildAnim/0.3
	return overshootEase(progress), progress
}

// SellAnimParams 计算出售动画的缩放和透明度。
// sellAnim: 剩余时间（初始 0.25，递减到 0）。
// 返回 (scaleMul, alpha)，无动画时返回 (1, 1)。
func SellAnimParams(sellAnim float64) (float64, float64) {
	if sellAnim <= 0 {
		return 1.0, 1.0
	}
	progress := 1.0 - sellAnim/0.25
	if progress > 1 {
		progress = 1
	}
	var scale float64
	if progress < 0.2 {
		scale = 1.0 + progress*1.0
	} else {
		scale = 1.2 * (1.0 - (progress-0.2)/0.8)
	}
	return scale, 1.0 - progress
}

// FirePulseScale 计算射击脉冲缩放系数。
// fireAnim: 剩余时间（初始 0.15，递减到 0）。
// 返回乘以精灵 scale 的系数（>= 1.0）。
func FirePulseScale(fireAnim float64) float64 {
	if fireAnim <= 0 {
		return 1.0
	}
	return 1.0 + 0.08*(fireAnim/0.15)
}

// ── 建造波纹 ────────────────────────────────────────

// DrawBuildRipple 绘制建造时的多层白色扩散环。
// progress: 0（刚开始）到 1（结束）。
func DrawBuildRipple(screen *ebiten.Image, cx, cy float32, progress float64) {
	// Main ring
	ringR := float32(10 + 20*progress)
	ringA := uint8(float64(150) * (1 - progress))
	draw.CircleOutline(screen, cx, cy, ringR, 2, color.RGBA{255, 255, 255, ringA})

	// Second ring (delayed by 0.15 progress)
	if progress > 0.15 {
		p2 := (progress - 0.15) / 0.85
		r2 := float32(10 + 20*p2)
		a2 := uint8(float64(100) * (1 - p2))
		draw.CircleOutline(screen, cx, cy, r2, 1.5, color.RGBA{200, 220, 255, a2})
	}

	// Third ring (delayed by 0.3)
	if progress > 0.3 {
		p3 := (progress - 0.3) / 0.7
		r3 := float32(10 + 20*p3)
		a3 := uint8(float64(60) * (1 - p3))
		draw.CircleOutline(screen, cx, cy, r3, 1, color.RGBA{180, 200, 255, a3})
	}
}

// ── 旋转弧刃（spin_aoe） ────────────────────────────

// DrawSpinBlades 绘制旋转攻击的多层弧线刃。
// activeRatio: SpinActive/0.3，0~1 控制透明度和视觉强度。
// spinAngle: 当前旋转弧度。
func DrawSpinBlades(screen *ebiten.Image, cx, cy, outerR float32, spinAngle, activeRatio float64) {
	a := activeRatio
	if a > 1 {
		a = 1
	}

	// ── Layer 1: 外层柔光轨道（低 alpha，营造旋转扫掠区域感） ──
	trailA := uint8(20 * a)
	draw.CircleOutline(screen, cx, cy, outerR, 1, color.RGBA{R: 163, G: 230, B: 53, A: trailA})

	innerR := float32(towerSpriteSize) * 0.25 // 内核半径（刀刃起始）

	for i := 0; i < 4; i++ {
		ang := spinAngle + float64(i)*math.Pi/2
		cosA, sinA := float32(math.Cos(ang)), float32(math.Sin(ang))

		// ── Blade edge lines: 内核→外刃的连线（刀刃放大虚影） ──
		// 前缘连线（刃的旋转前端）
		leadAng := ang + 0.3
		leadCos, leadSin := float32(math.Cos(leadAng)), float32(math.Sin(leadAng))
		leadIA := uint8(55 * a)
		draw.ThickLine(screen,
			cx+innerR*leadCos, cy+innerR*leadSin,
			cx+outerR*leadCos, cy+outerR*leadSin,
			1.5, color.RGBA{R: 180, G: 240, B: 80, A: leadIA})
		// 后缘连线（刃的旋转尾端，更暗）
		trailAng := ang - 0.3
		trailCos, trailSin := float32(math.Cos(trailAng)), float32(math.Sin(trailAng))
		trailIA := uint8(30 * a)
		draw.ThickLine(screen,
			cx+innerR*trailCos, cy+innerR*trailSin,
			cx+outerR*trailCos, cy+outerR*trailSin,
			1, color.RGBA{R: 140, G: 210, B: 40, A: trailIA})
		// 中轴连线（刃的中心脊线，最亮）
		spineA := uint8(45 * a)
		draw.ThickLine(screen,
			cx+innerR*cosA, cy+innerR*sinA,
			cx+outerR*cosA, cy+outerR*sinA,
			1, color.RGBA{R: 200, G: 255, B: 130, A: spineA})

		// ── Layer 2: 外层宽弧（模糊拖影） ──
		trailOffAng := ang - 0.15
		trailArcA := uint8(35 * a)
		draw.Arc(screen, cx, cy, outerR+1, float32(trailOffAng-0.4), float32(trailOffAng+0.4), 4, color.RGBA{R: 140, G: 210, B: 40, A: trailArcA})

		// ── Layer 3: 主弧刃（明亮锐利） ──
		mainA := uint8(140 * a)
		draw.Arc(screen, cx, cy, outerR, float32(ang-0.32), float32(ang+0.32), 2.5, color.RGBA{R: 163, G: 230, B: 53, A: mainA})

		// ── Layer 4: 内层亮芯弧（白绿色高亮） ──
		coreA := uint8(100 * a)
		draw.Arc(screen, cx, cy, outerR-1, float32(ang-0.2), float32(ang+0.2), 1.5, color.RGBA{R: 210, G: 255, B: 140, A: coreA})

		// ── Layer 5: 刃尖高亮点 ──
		tipX := cx + outerR*leadCos
		tipY := cy + outerR*leadSin
		tipA := uint8(120 * a)
		draw.FilledCircle(screen, tipX, tipY, 2.5, color.RGBA{R: 220, G: 255, B: 160, A: tipA})

		// ── Layer 6: 刃尾衰减点 ──
		tailX := cx + outerR*trailCos
		tailY := cy + outerR*trailSin
		tailA := uint8(40 * a)
		draw.FilledCircle(screen, tailX, tailY, 1.5, color.RGBA{R: 130, G: 200, B: 40, A: tailA})
	}

	// ── Layer 7: 中心旋转光核（与精灵呼应的内部旋涡） ──
	coreR := float32(towerSpriteSize) * 0.22
	corePulse := 0.7 + 0.3*math.Sin(spinAngle*2)
	coreAlpha := uint8(float64(50) * a * corePulse)
	draw.CircleOutline(screen, cx, cy, coreR, 1.5, color.RGBA{R: 180, G: 240, B: 80, A: coreAlpha})
	// 两条交叉内弧（随旋转角旋转，比外刃快 1.5x）
	innerAng := spinAngle * 1.5
	innerA := uint8(60 * a * corePulse)
	for j := 0; j < 2; j++ {
		ja := innerAng + float64(j)*math.Pi
		draw.Arc(screen, cx, cy, coreR+2, float32(ja-0.5), float32(ja+0.5), 1.5, color.RGBA{R: 200, G: 255, B: 120, A: innerA})
	}
}

// ── 力量光环层级 ─────────────────────────────────────

const towerSpriteSize = 64

// DrawStrengthGlow 根据力量溢出值绘制层级光环。
// overflow: 超过基线 100 的力量值。
func DrawStrengthGlow(screen *ebiten.Image, cx, cy float32, overflow float64, animTime float64) {
	if overflow >= 50 {
		// Tier 1: warm yellow ring + subtle glow
		pulse1 := 0.5 + 0.5*math.Sin(animTime*2)
		ringAlpha := uint8(50 + 20*pulse1)
		draw.Glow(screen, cx, cy, float32(towerSpriteSize*0.35), float32(towerSpriteSize*0.5), color.RGBA{255, 230, 150, uint8(20 + 10*pulse1)})
		draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.4), 1.5, color.RGBA{255, 230, 150, ringAlpha})
	}
	if overflow >= 100 {
		// Tier 2: orange second ring
		pulse2 := 0.5 + 0.5*math.Sin(animTime*2.5)
		ringAlpha2 := uint8(45 + 20*pulse2)
		draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.5), 1.5, color.RGBA{255, 200, 80, ringAlpha2})
	}
	if overflow >= 150 {
		// Tier 3: bright gold pulsing + strong glow
		pulse3 := 0.5 + 0.5*math.Sin(animTime*3)
		ringAlpha3 := uint8(50 + 30*pulse3)
		draw.Glow(screen, cx, cy, float32(towerSpriteSize*0.45), float32(towerSpriteSize*0.65), color.RGBA{255, 200, 50, uint8(25 + 15*pulse3)})
		draw.CircleOutline(screen, cx, cy, float32(towerSpriteSize*0.6), 1.5, color.RGBA{255, 200, 50, ringAlpha3})
	}
}

// ── 光环脉冲圈 ──────────────────────────────────────

// DrawAuraPulse 绘制光环能力的脉冲虚线圈。
func DrawAuraPulse(screen *ebiten.Image, cx, cy float32, radius float64, clr color.RGBA, animTime float64) {
	// Main aura ring — much more visible
	pulse := float32(0.7 + 0.3*math.Sin(animTime*2))
	a := uint8(float64(55) * float64(pulse))
	c := color.RGBA{clr.R, clr.G, clr.B, a}
	draw.DashedCircle(screen, cx, cy, float32(radius), 1, 6, 4, c)

	// Inner solid ring at 60% radius
	innerA := uint8(float64(30) * float64(pulse))
	draw.CircleOutline(screen, cx, cy, float32(radius*0.6), 1, color.RGBA{clr.R, clr.G, clr.B, innerA})

	// 2 orbiting dots at edge
	for i := 0; i < 2; i++ {
		angle := animTime*1.5 + float64(i)*math.Pi
		dotX := cx + float32(radius)*float32(math.Cos(angle))
		dotY := cy + float32(radius)*float32(math.Sin(angle))
		draw.FilledCircle(screen, dotX, dotY, 2, color.RGBA{clr.R, clr.G, clr.B, uint8(70 * pulse)})
	}
}

// ── Buff 指示圆点 ───────────────────────────────────

// DrawBuffDots 绘制塔上方的 buff 指示圆点。
func DrawBuffDots(screen *ebiten.Image, cx, cy float32, buffCount int) {
	dotY := cy - float32(towerSpriteSize*0.5) - 6
	dotSpacing := float32(6)
	dotR := float32(2.5)
	n := buffCount
	if n > 5 {
		n = 5
	}
	startX := cx - float32(n-1)*dotSpacing/2
	for i := 0; i < n; i++ {
		dx := startX + float32(i)*dotSpacing
		draw.FilledCircle(screen, dx, dotY, dotR,
			color.RGBA{R: 180, G: 140, B: 255, A: 180})
	}
}

// ── 五角星芒阵（金灵 buff） ─────────────────────────

// DrawPentagram 绘制旋转五角星芒阵 + 外环 + 内光晕。
// remainRatio: buff 剩余比例 0~1，控制渐隐。
func DrawPentagram(screen *ebiten.Image, cx, cy float32, remainRatio float64, animTime float64) {
	p := float32(remainRatio)
	if p > 1 {
		p = 1
	}
	alpha := uint8(float32(100) * p)
	starR := float32(28)
	rotation := float32(animTime * 0.6)
	pulse := float32(1.0 + 0.1*math.Sin(animTime*2.5))

	// 外层金色光环
	draw.CircleOutline(screen, cx, cy, starR*pulse, 1,
		color.RGBA{R: 255, G: 220, B: 80, A: alpha / 2})

	// 五角星
	drawPentagramStar(screen, cx, cy, starR*0.85, rotation, alpha)

	// 内层光晕
	draw.FilledCircle(screen, cx, cy, 8*pulse,
		color.RGBA{R: 255, G: 230, B: 120, A: alpha / 3})
}

func drawPentagramStar(screen *ebiten.Image, cx, cy, r float32, rotation float32, alpha uint8) {
	clr := color.RGBA{R: 255, G: 210, B: 80, A: alpha}

	var pts [5][2]float32
	for i := 0; i < 5; i++ {
		angle := float64(rotation) + float64(i)*2*math.Pi/5 - math.Pi/2
		pts[i] = [2]float32{
			cx + r*float32(math.Cos(angle)),
			cy + r*float32(math.Sin(angle)),
		}
	}

	order := [6]int{0, 2, 4, 1, 3, 0}
	for i := 0; i < 5; i++ {
		a, b := order[i], order[i+1]
		draw.Line(screen, pts[a][0], pts[a][1], pts[b][0], pts[b][1], 1.2, clr, true)
	}

	for _, pt := range pts {
		draw.FilledCircle(screen, pt[0], pt[1], 2,
			color.RGBA{R: 255, G: 240, B: 150, A: alpha})
	}
}
