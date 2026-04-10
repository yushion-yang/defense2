// vfx_stage.go — stage-level visual effects (extracted from stage.go).
// Zero core dependency — all functions accept only primitive values.
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── Enemy Ability Trigger VFX ───────────────────────

// DrawBlockFlash draws a blue shield pulse when projectile is blocked.
// timer: remaining time (starts 0.25, decrements to 0).
func DrawBlockFlash(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	r := radius + 6
	alpha := uint8(200 * (timer / 0.25))
	draw.CircleOutline(screen, cx, cy, r, 2, color.RGBA{R: 80, G: 160, B: 255, A: alpha})
	draw.CircleOutline(screen, cx, cy, r+3, 1, color.RGBA{R: 120, G: 200, B: 255, A: alpha / 2})
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

// DrawArmorSpark draws a gray expanding ring when armor absorbs damage.
// timer: remaining time (starts 0.15, decrements to 0).
func DrawArmorSpark(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	alpha := uint8(200 * (timer / 0.15))
	sparkR := radius + float32(4*(1-timer/0.15))
	draw.CircleOutline(screen, cx, cy, sparkR, 1.5, color.RGBA{R: 180, G: 180, B: 180, A: alpha})
}

// DrawDamageCapPulse draws an orange expanding ring when damage cap triggers.
// timer: remaining time (starts 0.3, decrements to 0).
func DrawDamageCapPulse(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	alpha := uint8(180 * (timer / 0.3))
	r := radius + float32(6*(1-timer/0.3))
	draw.CircleOutline(screen, cx, cy, r, 1.5, color.RGBA{R: 255, G: 180, B: 40, A: alpha})
}

// DrawPurgeWave draws a white expanding ring when purge triggers.
// timer: remaining time (starts 0.4, decrements to 0).
func DrawPurgeWave(screen *ebiten.Image, cx, cy, radius float32, timer float64) {
	if timer <= 0 {
		return
	}
	progress := 1 - timer/0.4
	r := radius + float32(40*progress)
	alpha := uint8(200 * (1 - progress))
	draw.CircleOutline(screen, cx, cy, r, 2, color.RGBA{R: 255, G: 255, B: 255, A: alpha})
}

// DrawPhaseAura draws purple pulsating double ring for phase-shifted enemy.
func DrawPhaseAura(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	pulseR := radius + 4 + float32(3*math.Sin(animTime*6))
	draw.CircleOutline(screen, cx, cy, pulseR, 2, color.RGBA{R: 160, G: 80, B: 255, A: 160})
	draw.CircleOutline(screen, cx, cy, pulseR+4, 1, color.RGBA{R: 160, G: 80, B: 255, A: 60})
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

// DrawUpgradeDiamond draws a pulsing gold diamond above a tower with pending upgrade.
func DrawUpgradeDiamond(screen *ebiten.Image, cx, cy float32, animTime float64) {
	pulse := float32(0.6 + 0.4*math.Sin(animTime*5))
	scale := float32(1.0 + 0.15*math.Sin(animTime*5))
	a := uint8(230 * pulse)
	r := float32(7) * scale
	draw.Diamond(screen, cx, cy-24, r+2, 1.0, color.RGBA{R: 250, G: 200, B: 50, A: a / 3})
	draw.Diamond(screen, cx, cy-24, r, 1.8, color.RGBA{R: 250, G: 200, B: 50, A: a})
}

// DrawSelectionRing draws the tower selection ring + range indicator.
func DrawSelectionRing(screen *ebiten.Image, cx, cy, selectionR, selectionW float32, selClr color.RGBA, rangeR, rangeW float32, rangeClr color.RGBA) {
	draw.CircleOutline(screen, cx, cy, selectionR, selectionW, selClr)
	draw.CircleOutline(screen, cx, cy, rangeR, rangeW, rangeClr)
}

// ── Enemy Body Overlays ─────────────────────────────

// DrawSlowOverlay draws blue circle outline on slowed enemy body.
func DrawSlowOverlay(screen *ebiten.Image, cx, cy, spriteR float32) {
	draw.CircleOutline(screen, cx, cy, spriteR+1, 1.5, color.RGBA{80, 160, 255, 80})
}

// DrawBurnOverlay draws orange inner glow on burning enemy body.
func DrawBurnOverlay(screen *ebiten.Image, cx, cy, spriteR float32) {
	draw.FilledCircle(screen, cx, cy, spriteR*0.5, color.RGBA{255, 120, 30, 35})
}
