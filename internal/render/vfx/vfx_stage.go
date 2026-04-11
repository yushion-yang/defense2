// vfx_stage.go — stage-level visual effects (extracted from stage.go).
// Zero core dependency — all functions accept only primitive values.
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"
	"defense2/internal/render/easing"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── Enemy Ability Trigger VFX ───────────────────────

// DrawBlockFlash draws a blue shield pulse when projectile is blocked.
// timer: remaining time (starts 0.25, decrements to 0).
func DrawBlockFlash(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	alpha := uint8(clampF(timer/0.25*200, 0, 200))
	// Two expanding rings
	draw.CircleOutline(screen, cx, cy, radius+6, 2, color.RGBA{100, 180, 255, alpha})
	draw.CircleOutline(screen, cx, cy, radius+9, 1.5, color.RGBA{100, 180, 255, alpha / 2})
	// Spark lines radiating outward
	sparkA := uint8(clampF(timer/0.25*180, 0, 180))
	for i := 0; i < 3; i++ {
		angle := float64(i)*2.094 + float64(cx+cy)*0.1 // pseudo-random via position
		sparkLen := float32(6 + 4*timer/0.25)
		sx := cx + (radius+6)*float32(math.Cos(angle))
		sy := cy + (radius+6)*float32(math.Sin(angle))
		ex := cx + (radius+6+sparkLen)*float32(math.Cos(angle))
		ey := cy + (radius+6+sparkLen)*float32(math.Sin(angle))
		draw.Line(screen, sx, sy, ex, ey, 1, color.RGBA{180, 220, 255, sparkA}, true)
	}
}

// DrawDodgeFlash draws a white offset afterimage when enemy dodges.
// timer: remaining time (starts 0.3, decrements to 0).
func DrawDodgeFlash(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	alpha := uint8(150 * (timer / 0.3))
	offset := float32(6 * (timer / 0.3))
	draw.FilledCircle(screen, cx-offset, cy, radius*0.8, color.RGBA{R: 255, G: 255, B: 255, A: alpha})
}

// DrawArmorSpark draws a silver-white expanding ring when armor absorbs damage.
// timer: remaining time (starts 0.15, decrements to 0).
func DrawArmorSpark(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	progress := 1 - timer/0.15
	alpha := uint8(clampF((1-progress)*200, 0, 200))
	expandR := radius + 4 + float32(progress)*8
	// Main ring — silver-white instead of gray
	draw.CircleOutline(screen, cx, cy, expandR, 1.5, color.RGBA{220, 225, 235, alpha})
	// Center flash
	draw.FilledCircle(screen, cx, cy, 3, color.RGBA{255, 255, 255, alpha})
	// 3 scattered spark dots
	for i := 0; i < 3; i++ {
		angle := float64(i)*2.094 + float64(cx)*0.1
		dist := expandR * float32(0.5+0.5*progress)
		dx := cx + dist*float32(math.Cos(angle))
		dy := cy + dist*float32(math.Sin(angle))
		draw.FilledCircle(screen, dx, dy, 1.5, color.RGBA{240, 240, 250, alpha})
	}
}

// DrawDamageCapPulse draws dual orange expanding rings when damage cap triggers.
// timer: remaining time (starts 0.3, decrements to 0).
func DrawDamageCapPulse(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	progress := 1 - timer/0.3
	alpha := uint8(clampF((1-progress)*180, 0, 180))
	// Fast inner ring
	innerR := radius + 3 + float32(progress)*12
	draw.CircleOutline(screen, cx, cy, innerR, 1.5, color.RGBA{255, 180, 60, alpha})
	// Slower outer ring (50% speed)
	outerR := radius + 3 + float32(progress)*6
	draw.CircleOutline(screen, cx, cy, outerR, 1, color.RGBA{255, 200, 100, alpha / 2})
}

// DrawPurgeWave draws a dual-ring white ripple when purge triggers.
// timer: remaining time (starts 0.4, decrements to 0).
func DrawPurgeWave(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	progress := 1 - timer/0.4
	alpha := uint8(clampF((1-progress)*200, 0, 200))
	// Leading ring
	mainR := radius + float32(progress)*40
	draw.CircleOutline(screen, cx, cy, mainR, 2, color.RGBA{255, 255, 255, alpha})
	// Trailing ring (5px behind)
	if progress > 0.1 {
		trailProgress := progress - 0.1
		trailR := radius + float32(trailProgress)*40
		trailA := uint8(clampF(float64(alpha)*0.5, 0, 120))
		draw.CircleOutline(screen, cx, cy, trailR, 1, color.RGBA{230, 240, 255, trailA})
	}
}

// DrawPhaseAura draws purple pulsating ring + dashed outer ring for phase-shifted enemy.
func DrawPhaseAura(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	innerPulse := 0.5 + 0.5*math.Sin(animTime*4)
	innerAlpha := uint8(50 + 30*innerPulse)
	draw.CircleOutline(screen, cx, cy, radius+3, 1.5, color.RGBA{180, 100, 255, innerAlpha})

	// Outer dashed ring with jitter (instability feel)
	jitterX := float32(math.Sin(animTime*17) * 1)
	jitterY := float32(math.Cos(animTime*23) * 1)
	outerAlpha := uint8(35 + 20*math.Sin(animTime*4+math.Pi))
	draw.DashedCircle(screen, cx+jitterX, cy+jitterY, radius+7, 1, 3, 3, color.RGBA{160, 80, 240, outerAlpha})
}

// DrawDashTrails draws orange speed lines trailing behind a dashing enemy.
// dirX, dirY: normalized movement direction.
func DrawDashTrails(screen *ebiten.Image, cx, cy float32, dirX, dirY float64) {
	for i := 0; i < 3; i++ {
		alpha := uint8(160 - i*50)
		length := float32(12 + i*4)
		perpX := -dirY * float64(i*3-3)
		perpY := dirX * float64(i*3-3)
		tailX := cx + float32(perpX) - float32(dirX)*length
		tailY := cy + float32(perpY) - float32(dirY)*length
		headX := cx + float32(perpX)
		headY := cy + float32(perpY)
		draw.ThickLine(screen, headX, headY, tailX, tailY, 1.5,
			color.RGBA{R: 255, G: 200, B: 80, A: alpha})
	}
}

// ── Enemy Aura VFX ──────────────────────────────────

// DrawHealerAura draws green healer aura: range circle + 3 rotating dots.
func DrawHealerAura(screen *ebiten.Image, cx, cy, healRadius float32, animTime float64) {
	alpha := uint8(25 + 10*math.Sin(animTime*2))
	draw.CircleOutline(screen, cx, cy, healRadius, 1, color.RGBA{R: 60, G: 220, B: 100, A: alpha})
	for i := 0; i < 3; i++ {
		angle := animTime*1.5 + float64(i)*2.094
		px := cx + float32(math.Cos(angle))*healRadius*0.7
		py := cy + float32(math.Sin(angle))*healRadius*0.7
		draw.FilledCircle(screen, px, py, 2, color.RGBA{R: 80, G: 255, B: 120, A: 100})
	}
}

// DrawHealPulse draws the expanding heal pulse ring on heal trigger.
// progress: 0 (just triggered) to 1 (fully expanded).
func DrawHealPulse(screen *ebiten.Image, cx, cy, baseRadius, maxRadius float32, progress float64) {
	if progress < 0 || progress > 1 {
		return
	}
	pulseR := baseRadius + float32(progress)*maxRadius
	pulseAlpha := uint8(200 * (1 - progress))
	draw.CircleOutline(screen, cx, cy, pulseR, 2, color.RGBA{R: 60, G: 255, B: 100, A: pulseAlpha})
}

// DrawSpeedAura draws orange pulsating circle for speed-buffing enemies.
func DrawSpeedAura(screen *ebiten.Image, cx, cy float32, auraRange float64, animTime float64) {
	alpha := uint8(35 + 15*math.Sin(animTime*1.5))
	draw.CircleOutline(screen, cx, cy, float32(auraRange), 1, color.RGBA{R: 255, G: 180, B: 60, A: alpha})
}

// ── Strength Drain ──────────────────────────────────

// DrawStrengthDrainLink draws purple drain line + 3 flowing particles from tower to enemy.
func DrawStrengthDrainLink(screen *ebiten.Image, tx, ty, ex, ey float32, animTime float64) {
	draw.ThickLine(screen, ex, ey, tx, ty, 1.5, color.RGBA{R: 140, G: 40, B: 180, A: 60})

	const particleCount = 3
	speed := 1.2
	for i := 0; i < particleCount; i++ {
		phase := math.Mod(animTime*speed+float64(i)/particleCount, 1.0)
		px := float32(float64(tx) + float64(ex-tx)*phase)
		py := float32(float64(ty) + float64(ey-ty)*phase)
		size := float32(2 + phase*2)
		alpha := uint8(100 + phase*155)
		draw.FilledCircle(screen, px, py, size, color.RGBA{R: 200, G: 80, B: 255, A: alpha})
	}
}

// ── Tower UI VFX ────────────────────────────────────

// DrawUpgradeDiamond draws a prominent upgrade-ready indicator above a tower.
// Industry standard: large bobbing icon + light pillar + rising particles + pulse ring + arrow hint.
func DrawUpgradeDiamond(screen *ebiten.Image, cx, cy float32, animTime float64) {
	const (
		baseY      = float32(32) // 菱形基准偏移（塔上方）
		bobAmp     = float32(4)  // 上下浮动幅度
		bobFreq    = 2.0         // 浮动频率
		diamondR   = float32(8)  // 菱形基础半径（加大）
		pillarW    = float32(8)  // 光柱宽度（加宽）
		pulseFreq  = 1.5         // 扩散环频率
		sparkCount = 6           // 上升粒子数（增多）
	)

	bob := bobAmp * float32(math.Sin(animTime*bobFreq*2*math.Pi))
	dy := cy - baseY + bob
	pillarTop := dy - 6
	pillarBot := cy - 8

	// ── 1. 底部圆形高光（塔脚下的金色圆盘，提示"这里有事"） ──
	groundPulse := float32(0.4 + 0.3*math.Sin(animTime*3))
	groundA := uint8(40 * groundPulse)
	draw.FilledCircle(screen, cx, cy-2, 14, color.RGBA{R: 255, G: 210, B: 60, A: groundA})

	// ── 2. 光柱（金色渐变，从塔顶延伸到菱形位置） ──
	pillarAlpha := float32(0.3 + 0.2*math.Sin(animTime*3))
	pa := uint8(255 * pillarAlpha)
	draw.FilledRect(screen, cx-pillarW/2, pillarTop, pillarW, pillarBot-pillarTop,
		color.RGBA{R: 255, G: 210, B: 60, A: pa}, true)
	// 中心亮线
	draw.FilledRect(screen, cx-1.5, pillarTop, 3, pillarBot-pillarTop,
		color.RGBA{R: 255, G: 240, B: 150, A: uint8(float32(pa) * 0.8)}, true)

	// ── 3. 浮动菱形（大、旋转、多层） ──
	breathe := float32(1.0 + 0.15*math.Sin(animTime*4))
	r := diamondR * breathe
	rot := animTime * 1.0

	// 最外层柔光（大范围低透明度辉光）
	draw.Glow(screen, cx, dy, r*0.5, r*2.5, color.RGBA{R: 255, G: 200, B: 40, A: uint8(30 * groundPulse)})
	// 外层辉光菱形
	glowA := uint8(60 + 40*math.Sin(animTime*3))
	draw.DiamondRotated(screen, cx, dy, r+4, 1.5, rot, color.RGBA{R: 255, G: 220, B: 80, A: glowA})
	// 主菱形（粗描边）
	mainA := uint8(220 + 35*math.Sin(animTime*5))
	draw.DiamondRotated(screen, cx, dy, r, 2.5, rot, color.RGBA{R: 255, G: 210, B: 50, A: mainA})
	// 内菱形（反向旋转，增加动态层次）
	innerA := uint8(160 + 40*math.Sin(animTime*4))
	draw.DiamondRotated(screen, cx, dy, r*0.55, 1.5, -rot*0.6, color.RGBA{R: 255, G: 240, B: 120, A: innerA})
	// 白色核心高光点
	coreA := uint8(180 + 60*math.Sin(animTime*6))
	draw.FilledCircle(screen, cx, dy, 2.5*breathe, color.RGBA{R: 255, G: 255, B: 230, A: coreA})

	// ── 4. 上升粒子（金色小圆点沿光柱两侧螺旋上升） ──
	for i := 0; i < sparkCount; i++ {
		phase := float64(i) * (2 * math.Pi / sparkCount)
		t := math.Mod(animTime*0.6+phase*0.25, 1.0) // 0→1 slower
		sparkY := pillarBot - (pillarBot-pillarTop)*float32(t)
		// 螺旋水平摆动
		sparkX := cx + float32(math.Sin(animTime*3+phase)*4)
		sparkA := uint8(200 * (1 - t) * (0.4 + 0.6*t))
		sparkR := float32(1.2 + 0.8*math.Sin(animTime*6+phase))
		draw.FilledCircle(screen, sparkX, sparkY, sparkR,
			color.RGBA{R: 255, G: 230, B: 100, A: sparkA})
	}

	// ── 5. 向上箭头提示（三条短线构成 ^） ──
	arrowY := dy - r - 5
	arrowPulse := float32(0.5 + 0.5*math.Sin(animTime*4))
	arrowA := uint8(150 * arrowPulse)
	arrowClr := color.RGBA{R: 255, G: 240, B: 100, A: arrowA}
	arrowSize := float32(5)
	draw.Line(screen, cx, arrowY, cx-arrowSize, arrowY+arrowSize, 1.5, arrowClr, true)
	draw.Line(screen, cx, arrowY, cx+arrowSize, arrowY+arrowSize, 1.5, arrowClr, true)

	// ── 6. 双层周期扩散环 ──
	pulseT := math.Mod(animTime, pulseFreq) / pulseFreq
	ringR := float32(6 + 18*pulseT)
	ringA := uint8(160 * (1 - pulseT) * (1 - pulseT))
	if ringA > 3 {
		draw.CircleOutline(screen, cx, dy, ringR, float32(1.8*(1-pulseT)+0.3),
			color.RGBA{R: 255, G: 220, B: 80, A: ringA})
	}
	// 第二层扩散环（延迟半周期）
	pulseT2 := math.Mod(animTime+pulseFreq*0.5, pulseFreq) / pulseFreq
	ring2R := float32(6 + 18*pulseT2)
	ring2A := uint8(100 * (1 - pulseT2) * (1 - pulseT2))
	if ring2A > 3 {
		draw.CircleOutline(screen, cx, dy, ring2R, float32(1.2*(1-pulseT2)+0.3),
			color.RGBA{R: 255, G: 240, B: 120, A: ring2A})
	}
}

// DrawSelectionRing draws the tower selection ring + range indicator.
func DrawSelectionRing(screen *ebiten.Image, cx, cy, selectionR, selectionW float32, selClr color.RGBA, rangeR, rangeW float32, rangeClr color.RGBA) {
	draw.CircleOutline(screen, cx, cy, selectionR, selectionW, selClr)
	draw.CircleOutline(screen, cx, cy, rangeR, rangeW, rangeClr)
}

// ── Root / Flying Ground Effects ─────────────────────

// DrawRootGround draws brown ground circle under rooted enemy.
func DrawRootGround(screen *ebiten.Image, cx, cy, radius float32) {
	draw.FilledCircle(screen, cx, cy+radius, radius*0.8, color.RGBA{100, 70, 40, 60})
}

// ── Enemy Body Overlays ─────────────────────────────

// DrawSlowOverlay draws blue ice overlay on slowed enemy body.
func DrawSlowOverlay(screen *ebiten.Image, cx, cy, spriteR float32, animTime float64) {
	// Faint ice floor
	draw.FilledCircle(screen, cx, cy, spriteR*0.6, color.RGBA{100, 180, 255, 20})
	// Pulsing blue ring
	alpha := uint8(60 + 30*math.Sin(animTime*4))
	draw.CircleOutline(screen, cx, cy, spriteR+1, 1.5, color.RGBA{80, 160, 255, alpha})
}

// DrawBurnOverlay draws flickering fire glow on burning enemy body.
func DrawBurnOverlay(screen *ebiten.Image, cx, cy, spriteR float32, animTime float64) {
	// Flickering dual-layer fire glow
	flicker := math.Sin(animTime*8) * 15
	outerA := uint8(easing.Clamp01(float64(40+flicker)/255) * 255)
	innerA := uint8(easing.Clamp01(float64(55+flicker)/255) * 255)
	draw.FilledCircle(screen, cx, cy, spriteR*0.55, color.RGBA{255, 120, 30, outerA})
	draw.FilledCircle(screen, cx, cy, spriteR*0.3, color.RGBA{255, 220, 100, innerA})
}

// DrawItemDropGlow draws a pulsing colored glow at item drop position.
// timer: remaining ground time, maxTime: total ground duration.
func DrawItemDropGlow(screen *ebiten.Image, cx, cy float32, timer, maxTime float64, clr color.RGBA) {
	if timer <= 0 || maxTime <= 0 {
		return
	}
	t := timer / maxTime // 1→0
	pulse := 0.6 + 0.4*math.Sin(timer*8)
	alpha := uint8(float64(clr.A) * pulse * t)
	// Inner glow
	draw.Glow(screen, cx, cy, 4, 14, color.RGBA{clr.R, clr.G, clr.B, alpha})
	// Core dot
	draw.FilledCircle(screen, cx, cy, 3, color.RGBA{255, 255, 255, alpha})
	// Outer breathing ring
	ringA := uint8(float64(alpha) * 0.5)
	ringR := float32(10 + 4*math.Sin(timer*4))
	draw.CircleOutline(screen, cx, cy, ringR, 1.5, color.RGBA{clr.R, clr.G, clr.B, ringA})
}

// DrawPoisonOverlay draws pulsing poison glow on poisoned enemy body.
func DrawPoisonOverlay(screen *ebiten.Image, cx, cy, spriteR float32, animTime float64) {
	// Pulsing dual-layer poison glow
	pulse := math.Sin(animTime*3) * 10
	outerA := uint8(easing.Clamp01(float64(30+pulse)/255) * 255)
	innerA := uint8(easing.Clamp01(float64(45+pulse)/255) * 255)
	draw.FilledCircle(screen, cx, cy, spriteR*0.55, color.RGBA{40, 160, 30, outerA})
	draw.FilledCircle(screen, cx, cy, spriteR*0.3, color.RGBA{100, 220, 60, innerA})
}
