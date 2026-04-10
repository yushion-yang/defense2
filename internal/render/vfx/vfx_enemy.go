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
	innerAlpha := uint8(clampF(float64(theme.EnemyBossInner.A)*
		(0.5+0.5*math.Sin(animTime*2.5)), 0, 255))
	innerClr := color.RGBA{R: theme.EnemyBossInner.R, G: theme.EnemyBossInner.G,
		B: theme.EnemyBossInner.B, A: innerAlpha}
	draw.CircleOutline(screen, cx, cy, radius+6, 2, innerClr)

	outerAlpha := uint8(clampF(float64(theme.EnemyBossOuter.A)*
		(0.5+0.5*math.Sin(animTime*2.5+1.5)), 0, 255))
	outerClr := color.RGBA{R: theme.EnemyBossOuter.R, G: theme.EnemyBossOuter.G,
		B: theme.EnemyBossOuter.B, A: outerAlpha}
	draw.CircleOutline(screen, cx, cy, radius+10, 1.5, outerClr)
}

// ── 跑者光环 ────────────────────────────────────────

// DrawRunnerRing 绘制跑者脉冲光环。
func DrawRunnerRing(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	pulseAlpha := uint8(clampF(float64(theme.EnemyRunnerPulse.A)*
		(0.5+0.5*math.Sin(animTime*3)), 0, 255))
	pulseClr := color.RGBA{R: theme.EnemyRunnerPulse.R, G: theme.EnemyRunnerPulse.G,
		B: theme.EnemyRunnerPulse.B, A: pulseAlpha}
	draw.CircleOutline(screen, cx, cy, radius+4, 1.5, pulseClr)
}

// ── 眩晕星星 ────────────────────────────────────────

// DrawStunStars 绘制眩晕旋转星星（3 颗黄色圆点绕头顶椭圆轨道）。
func DrawStunStars(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	starR := float32(2)
	orbitR := radius + 4
	for i := 0; i < 3; i++ {
		angle := animTime*5 + float64(i)*2.094
		sx := cx + orbitR*float32(math.Cos(angle))
		sy := cy - radius - 4 + orbitR*0.4*float32(math.Sin(angle))
		draw.FilledCircle(screen, sx, sy, starR, color.RGBA{255, 255, 100, 200})
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

// DrawStatusDots 绘制状态效果指示圆点。
func DrawStatusDots(screen *ebiten.Image, cx, dotY float32, dots []StatusDot) {
	dotX := cx - 8.0
	dotR := float32(2.5)
	for _, d := range dots {
		draw.FilledCircle(screen, dotX, dotY, dotR, d.Color)
		dotX += 6
	}
}

// ── Buffer 光环 ─────────────────────────────────────

// DrawBufferAura 绘制 buffer 行为敌人的金色光环圈。
func DrawBufferAura(screen *ebiten.Image, cx, cy float32, buffRadius float64, animTime float64) {
	auraAlpha := uint8(clampF(40+20*math.Sin(animTime*3), 20, 70))
	draw.CircleOutline(screen, cx, cy, float32(buffRadius), 1.5,
		color.RGBA{R: 245, G: 158, B: 11, A: auraAlpha})
}

// ── 免疫脚环 ────────────────────────────────────────

// DrawImmunityRing 绘制天生能力免疫脚环。
func DrawImmunityRing(screen *ebiten.Image, cx, cy, footR float32, clr color.RGBA) {
	draw.CircleOutline(screen, cx, cy+footR*0.3, footR, 1, clr)
}

// ── 净化微光 ────────────────────────────────────────

// DrawPurgeGlow 绘制净化免疫期白色微光环。
func DrawPurgeGlow(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	glowAlpha := uint8(60 + 30*math.Sin(animTime*6))
	draw.CircleOutline(screen, cx, cy, radius+3, 1.5, color.RGBA{R: 255, G: 255, B: 255, A: glowAlpha})
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
