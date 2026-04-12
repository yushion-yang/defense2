// vfx_enemy.go — 敌人视觉效果（解耦版）。
// 所有函数只接受纯值参数，零 core 包依赖。
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 动画参数计算 ────────────────────────────────────

// DyingAnimParams 计算死亡动画参数。
// 返回 (scale, alpha, offsetY)。
func DyingAnimParams(dyingTimer, dyingDuration float64) (scale, alpha, offsetY float64) {
	if dyingDuration <= 0 {
		return 1, 1, 0
	}
	progress := 1.0 - dyingTimer/dyingDuration
	return 1.0 - progress, 1.0 - progress, -progress * 8
}

// SpawnAnimParams 计算出生动画参数。
// 返回 (scale, alpha)。
func SpawnAnimParams(spawnTimer, spawnDuration float64) (scale, alpha float64) {
	if spawnDuration <= 0 || spawnTimer <= 0 {
		return 1, 1
	}
	progress := 1.0 - spawnTimer/spawnDuration
	if progress < 0.7 {
		scale = progress / 0.7 * 1.15
	} else {
		t := (progress - 0.7) / 0.3
		scale = 1.15 - 0.15*t
	}
	alpha = progress * progress
	return scale, alpha
}

// ── Boss 脉冲环 ─────────────────────────────────────

// DrawBossPulse 绘制 Boss 脉冲光环。
func DrawBossPulse(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	// Inner ring — breathing alpha
	innerAlpha := uint8(clampF(float64(theme.EnemyBossInner.A)*(0.5+0.5*math.Sin(animTime*2.5)), 0, 255))
	innerClr := color.RGBA{R: theme.EnemyBossInner.R, G: theme.EnemyBossInner.G, B: theme.EnemyBossInner.B, A: innerAlpha}
	draw.CircleOutline(screen, cx, cy, radius+6, 2, innerClr)

	// Outer ring — anti-phase for breathing feel
	outerAlpha := uint8(clampF(float64(theme.EnemyBossOuter.A)*(0.5+0.5*math.Sin(animTime*2.5+math.Pi)), 0, 255))
	outerClr := color.RGBA{R: theme.EnemyBossOuter.R, G: theme.EnemyBossOuter.G, B: theme.EnemyBossOuter.B, A: outerAlpha}
	draw.CircleOutline(screen, cx, cy, radius+10, 1.5, outerClr)

	// 4 orbiting energy diamonds
	for i := 0; i < 4; i++ {
		angle := animTime*1.5 + float64(i)*math.Pi/2
		dotX := cx + (radius+10)*float32(math.Cos(angle))
		dotY := cy + (radius+10)*float32(math.Sin(angle))
		twinkle := uint8(clampF(120+40*math.Sin(animTime*4+float64(i)*1.5), 80, 170))
		draw.Diamond(screen, dotX, dotY, 2, 1, color.RGBA{R: 255, G: 100, B: 120, A: twinkle})
	}
}

// ── 跑者光环 ────────────────────────────────────────

// DrawRunnerRing 绘制跑者脉冲光环。
func DrawRunnerRing(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	pulseAlpha := uint8(clampF(float64(theme.EnemyRunnerPulse.A)*(0.5+0.5*math.Sin(animTime*3)), 0, 255))
	pulseClr := color.RGBA{R: theme.EnemyRunnerPulse.R, G: theme.EnemyRunnerPulse.G, B: theme.EnemyRunnerPulse.B, A: pulseAlpha}
	draw.DashedCircle(screen, cx, cy, radius+4, 1.5, 5, 3, pulseClr)

	// Speed lines trailing behind (assume moving right as default visual)
	for i := 0; i < 2; i++ {
		lineAlpha := uint8(clampF(float64(60+30*math.Sin(animTime*5+float64(i)*1.5)), 30, 100))
		offset := float32(4 + i*3)
		lineLen := float32(8 + i*2)
		draw.ThickLine(screen, cx-radius-offset-lineLen, cy-2+float32(i*4), cx-radius-offset, cy-2+float32(i*4), 1.2,
			color.RGBA{R: theme.EnemyRunnerPulse.R, G: theme.EnemyRunnerPulse.G, B: theme.EnemyRunnerPulse.B, A: lineAlpha})
	}
}

// ── 眩晕星星 ────────────────────────────────────────

// DrawStunStars 绘制眩晕旋转星星（3 颗黄色钻石绕头顶椭圆轨道，带闪烁）。
func DrawStunStars(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	orbitR := radius + 6
	for i := 0; i < 3; i++ {
		angle := animTime*5 + float64(i)*2.094
		sx := cx + orbitR*float32(math.Cos(angle))
		sy := cy - radius - 4 + orbitR*0.4*float32(math.Sin(angle))
		// Per-star twinkle
		twinkle := uint8(clampF(150+50*math.Sin(animTime*8+float64(i)*2), 100, 255))
		draw.Diamond(screen, sx, sy, 2.5, 1.2, color.RGBA{255, 255, 100, twinkle})
	}
}

// ── 命中闪红 ────────────────────────────────────────

// DrawHitFlash 绘制命中红色闪烁叠加层。
// hitFlash: 剩余时间（初始约 0.1）。
func DrawHitFlash(screen *ebiten.Image, cx, cy, spriteR float32, hitFlash float64) {
	if hitFlash <= 0 {
		return
	}
	// Outer red overlay — stronger than before
	flashAlpha := uint8(clampF(hitFlash*500, 0, 160))
	draw.FilledCircle(screen, cx, cy, spriteR, color.RGBA{R: 255, G: 80, B: 60, A: flashAlpha})
	// Inner white core — bright, fades faster
	coreAlpha := uint8(clampF(hitFlash*800, 0, 200))
	draw.FilledCircle(screen, cx, cy, spriteR*0.3, color.RGBA{R: 255, G: 255, B: 255, A: coreAlpha})
}

// ── 状态效果圆点 ────────────────────────────────────

// StatusDot 单个状态效果圆点。
type StatusDot struct {
	Color color.RGBA
}

// DrawStatusDots 绘制状态效果指示圆点（带轮廓和脉冲动画）。
func DrawStatusDots(screen *ebiten.Image, cx, dotY float32, dots []StatusDot, animTime float64) {
	dotSpacing := float32(7)
	n := len(dots)
	startX := cx - float32(n-1)*dotSpacing/2
	for i, d := range dots {
		dx := startX + float32(i)*dotSpacing
		// Dark outline for contrast
		draw.CircleOutline(screen, dx, dotY, 4, 0.8, color.RGBA{0, 0, 0, 100})
		// Pulsing fill — different speeds per color
		pulse := 20 * math.Sin(animTime*float64(3+i*2))
		r, g, b := d.Color.R, d.Color.G, d.Color.B
		a := int(d.Color.A) + int(pulse)
		if a > 255 {
			a = 255
		}
		if a < 0 {
			a = 0
		}
		draw.FilledCircle(screen, dx, dotY, 3.5, color.RGBA{r, g, b, uint8(a)})
	}
}

// ── Buffer 光环 ─────────────────────────────────────

// DrawBufferAura 绘制 buffer 行为敌人的金色光环圈。
func DrawBufferAura(screen *ebiten.Image, cx, cy float32, buffRadius float64, animTime float64) {
	// Outer ring — more visible
	auraAlpha := uint8(clampF(55+25*math.Sin(animTime*3), 35, 90))
	draw.CircleOutline(screen, cx, cy, float32(buffRadius), 1.5, color.RGBA{R: 245, G: 158, B: 11, A: auraAlpha})

	// Inner dashed ring at 80%
	innerA := uint8(clampF(30+15*math.Sin(animTime*3+1), 20, 50))
	draw.DashedCircle(screen, cx, cy, float32(buffRadius*0.8), 1, 4, 3, color.RGBA{R: 245, G: 180, B: 50, A: innerA})

	// 2 orbiting gold dots
	for i := 0; i < 2; i++ {
		angle := animTime*2 + float64(i)*math.Pi
		dotX := cx + float32(buffRadius)*float32(math.Cos(angle))
		dotY := cy + float32(buffRadius)*float32(math.Sin(angle))
		draw.FilledCircle(screen, dotX, dotY, 1.5, color.RGBA{R: 255, G: 200, B: 50, A: uint8(60 + 30*math.Sin(animTime*4+float64(i)))})
	}
}

// ── 免疫脚环 ────────────────────────────────────────

// DrawImmunityRing 绘制天生能力免疫脚环。
func DrawImmunityRing(screen *ebiten.Image, cx, cy, footR float32, clr color.RGBA, animTime float64) {
	// Pulsing radius
	pulse := float32(1.0 + 0.1*math.Sin(animTime*2))
	r := footR * pulse
	// Dashed circle with rotation effect (use DashedCircle)
	a := uint8(clampF(float64(clr.A)*(0.6+0.4*math.Sin(animTime*3)), float64(clr.A)*0.3, float64(clr.A)))
	c := color.RGBA{clr.R, clr.G, clr.B, a}
	draw.DashedCircle(screen, cx, cy+footR*0.3, r, 1, 4, 3, c)
}

// ── 净化微光 ────────────────────────────────────────

// DrawPurgeGlow 绘制净化免疫期白色微光环。
func DrawPurgeGlow(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	// Inner tight ring
	innerAlpha := uint8(60 + 30*math.Sin(animTime*6))
	draw.CircleOutline(screen, cx, cy, radius+2, 1.5, color.RGBA{R: 255, G: 255, B: 255, A: innerAlpha})

	// Outer loose ring (anti-phase)
	outerAlpha := uint8(40 + 20*math.Sin(animTime*6+math.Pi))
	draw.CircleOutline(screen, cx, cy, radius+6, 1, color.RGBA{R: 230, G: 240, B: 255, A: outerAlpha})

	// 4 small rotating cross marks for "cleansed" feel
	crossAlpha := uint8(35 + 15*math.Sin(animTime*4))
	for i := 0; i < 4; i++ {
		a := animTime*1.5 + float64(i)*math.Pi/2
		dx := cx + (radius+4)*float32(math.Cos(a))
		dy := cy + (radius+4)*float32(math.Sin(a))
		draw.Diamond(screen, dx, dy, 2, 0.8, color.RGBA{R: 220, G: 235, B: 255, A: crossAlpha})
	}
}

// ── 减伤护盾 ────────────────────────────────────────

// DrawDamageReduceShield 绘制减伤能力的灰色半透明护盾弧。
// 缓慢旋转的弧线 + 呼吸脉冲 alpha + 弧线边缘装甲菱形标记。
func DrawDamageReduceShield(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	// Shield arc rotates slowly around the enemy
	baseAngle := float32(animTime * 0.8)

	// Breathing alpha for the shield
	breathAlpha := clampF(50+25*math.Sin(animTime*2.5), 25, 80)

	arcR := radius + 5

	// Main shield arc — semi-circle facing forward, rotating
	arcAlpha := uint8(breathAlpha)
	arcClr := color.RGBA{R: 190, G: 195, B: 205, A: arcAlpha}
	startAng := baseAngle - math.Pi/2
	endAng := baseAngle + math.Pi/2
	draw.Arc(screen, cx, cy, arcR, startAng, endAng, 2, arcClr)

	// Inner thinner arc for depth
	innerArcAlpha := uint8(clampF(breathAlpha*0.6, 15, 50))
	draw.Arc(screen, cx, cy, radius+2, startAng+0.2, endAng-0.2, 1,
		color.RGBA{R: 200, G: 205, B: 215, A: innerArcAlpha})

	// 3 armor plate diamond markers along the arc edge
	for i := 0; i < 3; i++ {
		angle := float64(baseAngle) + (float64(i)-1)*0.7
		dx := cx + arcR*float32(math.Cos(angle))
		dy := cy + arcR*float32(math.Sin(angle))
		// Per-marker twinkle
		twinkle := uint8(clampF(60+30*math.Sin(animTime*3+float64(i)*2.1), 30, 100))
		draw.Diamond(screen, dx, dy, 2.5, 1, color.RGBA{R: 210, G: 215, B: 225, A: twinkle})
	}
}

// ── 狂暴光焰 ────────────────────────────────────────

// DrawBerserkFlare 绘制狂暴状态的红色脉冲光环 + 速度拖线。
func DrawBerserkFlare(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	// Contracting/expanding thin red ring
	ringPulse := float32(1.0 + 0.15*math.Sin(animTime*8))
	ringR := (radius + 4) * ringPulse
	ringAlpha := uint8(clampF(70+40*math.Sin(animTime*5), 30, 120))
	draw.CircleOutline(screen, cx, cy, ringR, 1.2,
		color.RGBA{R: 255, G: 50, B: 30, A: ringAlpha})

	// 4 rotating speed lines (short radial lines trailing outward)
	for i := 0; i < 4; i++ {
		angle := animTime*4 + float64(i)*math.Pi/2
		innerR := radius + 3
		outerR := radius + 10 + float32(3*math.Sin(animTime*6+float64(i)*1.5))
		x1 := cx + float32(innerR)*float32(math.Cos(angle))
		y1 := cy + float32(innerR)*float32(math.Sin(angle))
		x2 := cx + outerR*float32(math.Cos(angle))
		y2 := cy + outerR*float32(math.Sin(angle))
		lineAlpha := uint8(clampF(80+40*math.Sin(animTime*7+float64(i)*1.8), 40, 130))
		draw.ThickLine(screen, x1, y1, x2, y2, 1.5,
			color.RGBA{R: 255, G: 100, B: 40, A: lineAlpha})
	}

	// Faint outer orange halo
	haloAlpha := uint8(clampF(15+10*math.Sin(animTime*4+1), 5, 30))
	draw.CircleOutline(screen, cx, cy, radius+10, 0.8,
		color.RGBA{R: 255, G: 140, B: 50, A: haloAlpha})
}

// ── 再生光环 ────────────────────────────────────────

// DrawRegenAura 绘制再生能力的绿色上浮光点 + 基底辉光。
func DrawRegenAura(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	// 5 floating green diamonds rising around the enemy
	for i := 0; i < 5; i++ {
		phase := float64(i) * 1.257 // 2*pi/5
		orbAngle := animTime*1.2 + phase
		orbR := float64(radius) * 0.8
		px := float64(cx) + orbR*math.Cos(orbAngle)
		// Rising motion: particles float up and loop back
		risePhase := math.Mod(animTime*0.8+phase*0.5, 1.0) // 0→1 cycle
		riseY := float64(cy) - float64(radius)*0.5 - risePhase*float64(radius)*1.5
		px += 2 * math.Sin(animTime*2+phase)
		// Fade as particle rises
		particleAlpha := uint8(clampF(180*(1-risePhase), 0, 180))
		// Size shrinks as it rises
		dotR := float32(clampF(2.5*(1-risePhase*0.5), 0.5, 2.5))
		draw.Diamond(screen, float32(px), float32(riseY), dotR, 0.8,
			color.RGBA{R: 80, G: 220, B: 100, A: particleAlpha})
	}

	// Faint green ring at base
	ringAlpha := uint8(clampF(25+12*math.Sin(animTime*3), 12, 40))
	draw.CircleOutline(screen, cx, cy, radius+2, 0.8,
		color.RGBA{R: 60, G: 200, B: 90, A: ringAlpha})
}

// ── 辅助 ────────────────────────────────────────────

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
