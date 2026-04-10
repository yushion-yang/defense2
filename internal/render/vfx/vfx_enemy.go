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
	flashAlpha := uint8(clampF(hitFlash*300, 0, 90))
	draw.FilledCircle(screen, cx, cy, spriteR, color.RGBA{R: 255, G: 80, B: 60, A: flashAlpha})
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

// ── 格挡闪光 ────────────────────────────────────────

// DrawBlockFlash 绘制格挡蓝色盾形脉冲（双层圆环）。
// blockFlash: 剩余时间（初始 0.25）。
func DrawBlockFlash(screen *ebiten.Image, cx, cy, radius float32, blockFlash float64) {
	r := radius + 6
	alpha := uint8(200 * (blockFlash / 0.25))
	draw.CircleOutline(screen, cx, cy, r, 2, color.RGBA{R: 80, G: 160, B: 255, A: alpha})
	draw.CircleOutline(screen, cx, cy, r+3, 1, color.RGBA{R: 120, G: 200, B: 255, A: alpha / 2})
}

// ── 闪避残影 ────────────────────────────────────────

// DrawDodgeFlash 绘制闪避白色偏移残影。
// dodgeFlash: 剩余时间（初始 0.3）。
func DrawDodgeFlash(screen *ebiten.Image, cx, cy, radius float32, dodgeFlash float64) {
	alpha := uint8(150 * (dodgeFlash / 0.3))
	offset := float32(6 * (dodgeFlash / 0.3))
	draw.FilledCircle(screen, cx-offset, cy, radius*0.8, color.RGBA{R: 255, G: 255, B: 255, A: alpha})
}

// ── 装甲火花 ────────────────────────────────────────

// DrawArmorSpark 绘制装甲灰色小火花。
// armorSpark: 剩余时间（初始 0.15）。
func DrawArmorSpark(screen *ebiten.Image, cx, cy, radius float32, armorSpark float64) {
	alpha := uint8(200 * (armorSpark / 0.15))
	sparkR := radius + float32(4*(1-armorSpark/0.15))
	draw.CircleOutline(screen, cx, cy, sparkR, 1.5, color.RGBA{R: 180, G: 180, B: 180, A: alpha})
}

// ── 坚韧脉冲 ────────────────────────────────────────

// DrawDamageCapPulse 绘制坚韧触发橙色扩散圈。
// damageCapHit: 剩余时间（初始 0.3）。
func DrawDamageCapPulse(screen *ebiten.Image, cx, cy, radius float32, damageCapHit float64) {
	alpha := uint8(180 * (damageCapHit / 0.3))
	r := radius + float32(6*(1-damageCapHit/0.3))
	draw.CircleOutline(screen, cx, cy, r, 1.5, color.RGBA{R: 255, G: 180, B: 40, A: alpha})
}

// ── 净化脉冲 ────────────────────────────────────────

// DrawPurgeWave 绘制净化白色扩散圈。
// purgeFlash: 剩余时间（初始 0.4）。
func DrawPurgeWave(screen *ebiten.Image, cx, cy, radius float32, purgeFlash float64) {
	progress := 1 - purgeFlash/0.4
	r := radius + float32(40*progress)
	alpha := uint8(200 * (1 - progress))
	draw.CircleOutline(screen, cx, cy, r, 2, color.RGBA{R: 255, G: 255, B: 255, A: alpha})
}

// ── 相位光环 ────────────────────────────────────────

// DrawPhaseAura 绘制相位偏移紫色脉冲光环。
func DrawPhaseAura(screen *ebiten.Image, cx, cy, radius float32, animTime float64) {
	pulseR := radius + 4 + float32(3*math.Sin(animTime*6))
	draw.CircleOutline(screen, cx, cy, pulseR, 2, color.RGBA{R: 160, G: 80, B: 255, A: 160})
	draw.CircleOutline(screen, cx, cy, pulseR+4, 1, color.RGBA{R: 160, G: 80, B: 255, A: 60})
}

// ── 冲刺拖尾 ────────────────────────────────────────

// DrawDashTrails 绘制受击冲刺的速度拖尾线。
// dirX, dirY: 归一化行进方向。
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

// ── 治疗光环 ────────────────────────────────────────

// DrawHealerAura 绘制治疗光环范围圈 + 旋转光点。
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

// DrawHealPulse 绘制治疗触发扩散脉冲。
// radius: 敌人半径, healRadius: 治疗范围, progress: 0~1 动画进度。
func DrawHealPulse(screen *ebiten.Image, cx, cy, radius, healRadius float32, progress float64) {
	pulseR := radius + float32(progress)*healRadius
	pulseAlpha := uint8(200 * (1 - progress))
	draw.CircleOutline(screen, cx, cy, pulseR, 2, color.RGBA{R: 60, G: 255, B: 100, A: pulseAlpha})
}

// ── 加速光环 ────────────────────────────────────────

// DrawSpeedAura 绘制加速光环橙色范围圈。
func DrawSpeedAura(screen *ebiten.Image, cx, cy float32, auraRange float64, animTime float64) {
	alpha := uint8(35 + 15*math.Sin(animTime*1.5))
	draw.CircleOutline(screen, cx, cy, float32(auraRange), 1, color.RGBA{R: 255, G: 180, B: 60, A: alpha})
}

// ── 削强连接 ────────────────────────────────────────

// DrawStrengthDrainLink 绘制削强连接线 + 流动粒子。
// tx,ty: 塔位置; ex,ey: 敌人位置。
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

// ── 定根地面 ────────────────────────────────────────

// DrawRootGround 绘制定根地面效果（敌人脚下泥色圆）。
func DrawRootGround(screen *ebiten.Image, cx, cy, radius float32) {
	draw.FilledCircle(screen, cx, cy+radius, radius*0.8, color.RGBA{100, 70, 40, 60})
}

// ── 飞行阴影 ────────────────────────────────────────

// DrawFlyingShadow 绘制飞行敌人地面阴影。
func DrawFlyingShadow(screen *ebiten.Image, cx, cy, radius float32, clr color.RGBA) {
	draw.FilledCircle(screen, cx+2, cy+8, radius*1.3, clr)
}

// ── 减速覆盖 ────────────────────────────────────────

// DrawSlowOverlay 绘制减速蓝色轮廓。
func DrawSlowOverlay(screen *ebiten.Image, cx, cy, spriteR float32) {
	draw.CircleOutline(screen, cx, cy, spriteR+1, 1.5, color.RGBA{80, 160, 255, 80})
}

// ── 灼烧覆盖 ────────────────────────────────────────

// DrawBurnOverlay 绘制灼烧橙色光圈。
func DrawBurnOverlay(screen *ebiten.Image, cx, cy, spriteR float32) {
	draw.FilledCircle(screen, cx, cy, spriteR*0.5, color.RGBA{255, 120, 30, 35})
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
