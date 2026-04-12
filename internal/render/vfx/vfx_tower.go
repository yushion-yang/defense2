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
		draw.Diamond(screen, tipX, tipY, 2.5, 1, color.RGBA{R: 220, G: 255, B: 160, A: tipA})

		// ── Layer 6: 刃尾衰减点 ──
		tailX := cx + outerR*trailCos
		tailY := cy + outerR*trailSin
		tailA := uint8(40 * a)
		draw.Diamond(screen, tailX, tailY, 1.5, 0.8, color.RGBA{R: 130, G: 200, B: 40, A: tailA})
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

// DrawStrengthGlow 根据力量溢出值绘制层级力量标识。
// overflow: 超过基线 100 的力量值。
// 6 个层级：50/150/300/500/700/900，以旋转菱形、弧线段、虚线环组合表现，
// 不使用实心圆/Glow，紧凑贴合塔体。
func DrawStrengthGlow(screen *ebiten.Image, cx, cy float32, overflow float64, animTime float64) {
	if overflow < 50 {
		return
	}

	// ── T1 (50+): 4 个缓慢旋转的小菱形标记 ──
	{
		rot := animTime * 0.8
		pulse := 0.5 + 0.5*math.Sin(animTime*2)
		r := float32(15)
		a := uint8(45 + 25*pulse)
		clr := color.RGBA{255, 220, 130, a}
		for i := 0; i < 4; i++ {
			angle := rot + float64(i)*math.Pi/2
			dx := r * float32(math.Cos(angle))
			dy := r * float32(math.Sin(angle))
			draw.DiamondRotated(screen, cx+dx, cy+dy, 3, 1, angle, clr)
		}
	}

	// ── T2 (150+): 2 段旋转弧线 ──
	if overflow >= 150 {
		rot := animTime * 1.2
		pulse := 0.5 + 0.5*math.Sin(animTime*2.5)
		r := float32(17)
		a := uint8(50 + 25*pulse)
		clr := color.RGBA{255, 200, 90, a}
		arcLen := float32(math.Pi * 0.45)
		for i := 0; i < 2; i++ {
			start := float32(rot) + float32(i)*math.Pi
			draw.Arc(screen, cx, cy, r, start, start+arcLen, 1.2, clr)
		}
	}

	// ── T3 (300+): 3 段反向旋转弧线（外层） ──
	if overflow >= 300 {
		rot := -animTime * 1.5
		pulse := 0.5 + 0.5*math.Sin(animTime*3)
		r := float32(20)
		a := uint8(50 + 30*pulse)
		clr := color.RGBA{255, 180, 60, a}
		arcLen := float32(math.Pi * 0.35)
		for i := 0; i < 3; i++ {
			start := float32(rot) + float32(i)*math.Pi*2/3
			draw.Arc(screen, cx, cy, r, start, start+arcLen, 1.3, clr)
		}
	}

	// ── T4 (500+): 脉冲虚线内环 ──
	if overflow >= 500 {
		pulse := 0.5 + 0.5*math.Sin(animTime*2)
		r := float32(13 + pulse*1.5)
		a := uint8(40 + 30*pulse)
		clr := color.RGBA{255, 170, 40, a}
		draw.DashedCircle(screen, cx, cy, r, 1, 3, 3, clr)
	}

	// ── T5 (700+): 外层 4 菱形 + 放射短线 ──
	if overflow >= 700 {
		rot := animTime * 0.6
		pulse := 0.5 + 0.5*math.Sin(animTime*3.5)
		a := uint8(55 + 35*pulse)
		clr := color.RGBA{255, 150, 30, a}
		r := float32(23)
		for i := 0; i < 4; i++ {
			angle := rot + float64(i)*math.Pi/2 + math.Pi/4
			dx := r * float32(math.Cos(angle))
			dy := r * float32(math.Sin(angle))
			draw.DiamondRotated(screen, cx+dx, cy+dy, 3.5, 1.2, angle, clr)
			// 放射短线从菱形向外延伸
			outerR := r + 5
			dx2 := outerR * float32(math.Cos(angle))
			dy2 := outerR * float32(math.Sin(angle))
			draw.ThickLine(screen, cx+dx, cy+dy, cx+dx2, cy+dy2, 1, clr)
		}
	}

	// ── T6 (900+): 密集弧段光冠（6 段交替旋转） ──
	if overflow >= 900 {
		rot := animTime * 2.0
		pulse := 0.5 + 0.5*math.Sin(animTime*4)
		r := float32(25)
		a := uint8(60 + 40*pulse)
		clr := color.RGBA{255, 130, 20, a}
		arcLen := float32(math.Pi * 0.2)
		for i := 0; i < 6; i++ {
			start := float32(rot) + float32(i)*math.Pi/3
			draw.Arc(screen, cx, cy, r, start, start+arcLen, 1.5, clr)
		}
	}
}

// ── 光环呼吸圈（统一极简方案）────────────────────────
//
// 设计原则：所有光环/区域共享"单圈呼吸 + 圈上小标记"模式。
// 颜色区分类型，2~3 个小标记符号区分具体能力，整体极度克制。
// 多塔同屏时画面不会变花。

// auraBreathAlpha 计算光环呼吸透明度（若隐若现，峰值很低）。
func auraBreathAlpha(animTime, freq, minA, maxA float64) uint8 {
	t := 0.5 + 0.5*math.Sin(animTime*freq)
	return uint8(minA + (maxA-minA)*t)
}

// drawAuraMarkers 在圈上等距绘制 n 个小标记，由 markerFn 绘制每个标记。
func drawAuraMarkers(screen *ebiten.Image, cx, cy, r float32, n int, animTime float64, rotSpeed float64, markerFn func(*ebiten.Image, float32, float32, uint8)) {
	baseRot := animTime * rotSpeed
	for i := 0; i < n; i++ {
		angle := baseRot + float64(i)*2*math.Pi/float64(n)
		mx := cx + r*float32(math.Cos(angle))
		my := cy + r*float32(math.Sin(angle))
		ma := auraBreathAlpha(animTime, 1.8, 40, 90)
		markerFn(screen, mx, my, ma)
	}
}

// DrawAuraPulse 通用光环脉冲圈（preview 用）。
func DrawAuraPulse(screen *ebiten.Image, cx, cy float32, radius float64, clr color.RGBA, animTime float64) {
	a := auraBreathAlpha(animTime, 1.5, 15, 40)
	draw.DashedCircle(screen, cx, cy, float32(radius), 0.8, 6, 4, color.RGBA{clr.R, clr.G, clr.B, a})
}

// ── Buff 光环（4 种，半径来自配置 param=150） ────────────

// DrawDamageAura 增伤光环 — 橙色单圈 + 3 菱形标记。
func DrawDamageAura(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.5, 12, 35)
	clr := color.RGBA{R: 255, G: 160, B: 60, A: a}
	draw.DashedCircle(screen, cx, cy, r32, 0.8, 6, 4, clr)
	// 3 菱形标记
	drawAuraMarkers(screen, cx, cy, r32, 3, animTime, 0.3, func(s *ebiten.Image, mx, my float32, ma uint8) {
		draw.Diamond(s, mx, my, 3, 0.8, color.RGBA{R: 255, G: 180, B: 80, A: ma})
	})
}

// DrawSpeedAuraRing 攻速光环 — 绿色单圈 + 3 短箭头。
func DrawSpeedAuraRing(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.8, 12, 35)
	clr := color.RGBA{R: 100, G: 220, B: 100, A: a}
	draw.DashedCircle(screen, cx, cy, r32, 0.8, 4, 3, clr)
	// 3 径向短箭头（从圈内往圈外的短线，表示"加速"）
	baseRot := animTime * 0.4
	for i := 0; i < 3; i++ {
		angle := baseRot + float64(i)*2*math.Pi/3
		ma := auraBreathAlpha(animTime, 1.8, 35, 80)
		x1 := cx + r32*0.88*float32(math.Cos(angle))
		y1 := cy + r32*0.88*float32(math.Sin(angle))
		x2 := cx + r32*1.0*float32(math.Cos(angle))
		y2 := cy + r32*1.0*float32(math.Sin(angle))
		draw.Line(screen, x1, y1, x2, y2, 1.0, color.RGBA{R: 100, G: 220, B: 100, A: ma}, true)
	}
}

// DrawRangeAura 射程光环 — 蓝色单圈 + 4 十字标记。
func DrawRangeAura(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.3, 12, 35)
	clr := color.RGBA{R: 100, G: 160, B: 255, A: a}
	draw.DashedCircle(screen, cx, cy, r32, 0.8, 6, 4, clr)
	// 4 十字标记（NSEW）
	drawAuraMarkers(screen, cx, cy, r32, 4, animTime, 0.0, func(s *ebiten.Image, mx, my float32, ma uint8) {
		c := color.RGBA{R: 100, G: 160, B: 255, A: ma}
		draw.Line(s, mx-3, my, mx+3, my, 0.8, c, false)
		draw.Line(s, mx, my-3, mx, my+3, 0.8, c, false)
	})
}

// DrawCritAura 暴击光环 — 金色单圈 + 3 菱形旋转（比 Damage 稍快辨别）。
func DrawCritAura(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 2.0, 12, 35)
	clr := color.RGBA{R: 255, G: 220, B: 60, A: a}
	draw.DashedCircle(screen, cx, cy, r32, 0.8, 3, 5, clr)
	// 3 旋转小菱形（旋转自身角度，与 DamageAura 固定菱形区分）
	baseRot := animTime * 0.5
	for i := 0; i < 3; i++ {
		angle := baseRot + float64(i)*2*math.Pi/3
		mx := cx + r32*float32(math.Cos(angle))
		my := cy + r32*float32(math.Sin(angle))
		ma := auraBreathAlpha(animTime, 2.0, 40, 90)
		draw.DiamondRotated(screen, mx, my, 3, 0.8, animTime*2, color.RGBA{R: 255, G: 240, B: 100, A: ma})
	}
}

// DrawSoloAura 独行加成 — 紫色极淡虚线圈（无标记，最内敛）。
func DrawSoloAura(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	a := auraBreathAlpha(animTime, 1.2, 8, 25)
	draw.DashedCircle(screen, cx, cy, float32(radius), 0.8, 8, 6, color.RGBA{R: 200, G: 120, B: 255, A: a})
}

// ── Zone 效果（4 种，半径=塔射程）─────────────────────

// DrawPoisonZone 毒区 — 绿色实线圈 + 2 小圆点。
func DrawPoisonZone(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.6, 15, 35)
	draw.CircleOutline(screen, cx, cy, r32, 0.8, color.RGBA{R: 120, G: 200, B: 60, A: a})
	drawAuraMarkers(screen, cx, cy, r32, 2, animTime, 0.2, func(s *ebiten.Image, mx, my float32, ma uint8) {
		draw.Diamond(s, mx, my, 2, 0.8, color.RGBA{R: 120, G: 200, B: 60, A: ma})
	})
}

// DrawSilenceZone 沉默区 — 紫色实线圈 + 2 短横（禁止符号）。
func DrawSilenceZone(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.4, 15, 35)
	draw.CircleOutline(screen, cx, cy, r32, 0.8, color.RGBA{R: 180, G: 80, B: 220, A: a})
	drawAuraMarkers(screen, cx, cy, r32, 2, animTime, 0.15, func(s *ebiten.Image, mx, my float32, ma uint8) {
		draw.Line(s, mx-3, my, mx+3, my, 1.0, color.RGBA{R: 180, G: 80, B: 220, A: ma}, false)
	})
}

// DrawCurseZone 诅咒区 — 暗红实线圈 + 2 X 标记。
func DrawCurseZone(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.0, 15, 35)
	draw.CircleOutline(screen, cx, cy, r32, 0.8, color.RGBA{R: 160, G: 50, B: 50, A: a})
	drawAuraMarkers(screen, cx, cy, r32, 2, animTime, 0.1, func(s *ebiten.Image, mx, my float32, ma uint8) {
		c := color.RGBA{R: 160, G: 50, B: 50, A: ma}
		draw.Line(s, mx-2.5, my-2.5, mx+2.5, my+2.5, 0.8, c, false)
		draw.Line(s, mx+2.5, my-2.5, mx-2.5, my+2.5, 0.8, c, false)
	})
}

// DrawWeakenZone 脆弱区 — 橙色实线圈 + 3 向下斜线。
func DrawWeakenZone(screen *ebiten.Image, cx, cy float32, radius float64, animTime float64) {
	r32 := float32(radius)
	a := auraBreathAlpha(animTime, 1.5, 15, 35)
	draw.CircleOutline(screen, cx, cy, r32, 0.8, color.RGBA{R: 220, G: 140, B: 60, A: a})
	drawAuraMarkers(screen, cx, cy, r32, 3, animTime, 0.2, func(s *ebiten.Image, mx, my float32, ma uint8) {
		draw.Line(s, mx-2, my-2, mx+2, my+2, 0.8, color.RGBA{R: 220, G: 140, B: 60, A: ma}, false)
	})
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
		draw.Diamond(screen, dx, dotY, dotR, 0.8,
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
	draw.DiamondRotated(screen, cx, cy, 6*pulse, 1.2, float64(rotation),
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
		draw.Diamond(screen, pt[0], pt[1], 2, 0.8,
			color.RGBA{R: 255, G: 240, B: 150, A: alpha})
	}
}
