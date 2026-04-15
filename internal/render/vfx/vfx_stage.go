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

// DrawUpgradeDiamond draws a clean, elegant upgrade-ready indicator above a tower.
// Style: thin outline diamond + subtle light line + minimal particles. No glow blobs.
func DrawUpgradeDiamond(screen *ebiten.Image, cx, cy float32, animTime float64) {
	const (
		baseY     = float32(28) // 菱形偏移（塔上方）
		bobAmp    = float32(3)  // 上下浮动幅度
		bobFreq   = 2.0         // 浮动频率
		diamondR  = float32(7)  // 菱形半径
		pulseFreq = 2.0         // 扩散环周期（秒）
	)

	bob := bobAmp * float32(math.Sin(animTime*bobFreq*2*math.Pi))
	dy := cy - baseY + bob

	// ── 1. 细光线（塔顶到菱形的连接线） ──
	lineBot := cy - 10
	lineA := uint8(40 + 20*math.Sin(animTime*3))
	draw.Line(screen, cx, dy+diamondR-2, cx, lineBot, 1, color.RGBA{R: 255, G: 210, B: 60, A: lineA}, true)

	// ── 2. 浮动菱形（薄描边，缓慢旋转） ──
	rot := animTime * 0.8
	breathe := float32(1.0 + 0.08*math.Sin(animTime*4))
	r := diamondR * breathe

	// 主菱形（金色描边）
	mainA := uint8(180 + 40*math.Sin(animTime*3))
	draw.DiamondRotated(screen, cx, dy, r, 1.8, rot, color.RGBA{R: 255, G: 210, B: 50, A: mainA})
	// 内部小菱形（白色，反旋转）
	innerA := uint8(100 + 30*math.Sin(animTime*5))
	draw.DiamondRotated(screen, cx, dy, r*0.45, 1.0, -rot*0.5, color.RGBA{R: 255, G: 250, B: 200, A: innerA})

	// ── 3. 向上箭头（^ 形，脉冲显隐） ──
	arrowY := dy - r - 4
	arrowPhase := math.Mod(animTime*1.5, 1.0)
	arrowA := uint8(0)
	if arrowPhase < 0.6 {
		arrowA = uint8(140 * (1 - arrowPhase/0.6))
	}
	if arrowA > 5 {
		aClr := color.RGBA{R: 255, G: 230, B: 80, A: arrowA}
		draw.Line(screen, cx, arrowY, cx-4, arrowY+4, 1.2, aClr, true)
		draw.Line(screen, cx, arrowY, cx+4, arrowY+4, 1.2, aClr, true)
	}

	// ── 4. 2 颗上升小粒子（轻微，不抢眼） ──
	for i := 0; i < 2; i++ {
		phase := float64(i) * math.Pi
		t := math.Mod(animTime*0.5+phase*0.3, 1.0)
		sparkY := lineBot - (lineBot-dy)*float32(t)
		sparkX := cx + float32(math.Sin(animTime*2.5+phase)*2.5)
		sparkA := uint8(120 * (1 - t))
		draw.FilledCircle(screen, sparkX, sparkY, 1.0,
			color.RGBA{R: 255, G: 230, B: 100, A: sparkA})
	}

	// ── 5. 扩散环（单层，柔和） ──
	pulseT := math.Mod(animTime, pulseFreq) / pulseFreq
	ringR := float32(4 + 12*pulseT)
	ringA := uint8(80 * (1 - pulseT) * (1 - pulseT))
	if ringA > 3 {
		draw.CircleOutline(screen, cx, dy, ringR, float32(1.0*(1-pulseT)+0.3),
			color.RGBA{R: 255, G: 220, B: 80, A: ringA})
	}
}

// DrawStrengthUpgradeIndicator 绘制经典模式力量升级指示器（蓝绿色上箭头 + 呼吸动画）。
// 用于标识有钱可升级力量的塔，区别于金色菱形（能力选择）。
func DrawStrengthUpgradeIndicator(screen *ebiten.Image, cx, cy float32, animTime float64) {
	const (
		baseY    = float32(24) // 偏移（塔上方）
		bobAmp   = float32(2)  // 上下浮动幅度
		bobFreq  = 1.8         // 浮动频率
		arrowW   = float32(6)  // 箭头半宽
		arrowH   = float32(5)  // 箭头半高
		barW     = float32(3)  // 竖条半宽
		barH     = float32(5)  // 竖条高度
	)

	bob := bobAmp * float32(math.Sin(animTime*bobFreq*2*math.Pi))
	dy := cy - baseY + bob

	// 呼吸透明度
	breatheA := uint8(160 + 60*math.Sin(animTime*3))
	clr := color.RGBA{R: 50, G: 210, B: 180, A: breatheA}

	// ── 上箭头（^ 形） ──
	draw.Line(screen, cx, dy-arrowH, cx-arrowW, dy, 1.5, clr, true)
	draw.Line(screen, cx, dy-arrowH, cx+arrowW, dy, 1.5, clr, true)

	// ── 竖条（箭头下方） ──
	draw.Line(screen, cx, dy, cx, dy+barH, 1.5, clr, true)

	// ── 柔和底部光点 ──
	dotA := uint8(80 + 40*math.Sin(animTime*4))
	draw.FilledCircle(screen, cx, dy+barH+2, 1.5,
		color.RGBA{R: 50, G: 210, B: 180, A: dotA})
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
