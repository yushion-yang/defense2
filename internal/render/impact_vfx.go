// impact_vfx.go — 命中冲击特效系统。
// 全局对象池，单目标 impact ring + 中心闪光。
package render

import (
	"image/color"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImpactVFX 单个命中特效实例。
type ImpactVFX struct {
	X, Y    float64
	Life    float64
	MaxLife float64
	Active  bool
	Color   color.RGBA // 扩散环颜色
	Radius  float32    // 最大扩散半径
}

const maxImpactVFX = 16

var impactPool [maxImpactVFX]ImpactVFX
var impactCursor int

// SpawnHitImpact 在指定位置生成通用命中特效（小型扩散环）。
func SpawnHitImpact(x, y float64) {
	spawnImpact(x, y, color.RGBA{R: 255, G: 220, B: 100, A: 200}, 12, 0.15)
}

// SpawnTypedImpact 根据攻击方式生成对应元素颜色的命中特效。
func SpawnTypedImpact(x, y float64, attackStyle string) {
	switch attackStyle {
	case "scatter": // ice — blue
		spawnImpact(x, y, color.RGBA{R: 100, G: 180, B: 255, A: 200}, 8, 0.18)
	case "spin_aoe", "projectile": // fire/physical — orange
		spawnImpact(x, y, color.RGBA{R: 255, G: 140, B: 40, A: 200}, 8, 0.15)
	case "wideBeam": // energy — purple
		spawnImpact(x, y, color.RGBA{R: 200, G: 100, B: 255, A: 200}, 6, 0.12)
	default: // warm yellow default
		spawnImpact(x, y, color.RGBA{R: 255, G: 220, B: 100, A: 200}, 7, 0.15)
	}
}

// spawnImpact 生成一个自定义颜色和半径的小型命中特效。
func spawnImpact(x, y float64, clr color.RGBA, radius float32, life float64) {
	v := &impactPool[impactCursor]
	impactCursor = (impactCursor + 1) % maxImpactVFX
	v.X = x
	v.Y = y
	v.Life = life
	v.MaxLife = life
	v.Active = true
	v.Color = clr
	v.Radius = radius
}

// ClearImpactVFX 清除所有存活命中特效。
func ClearImpactVFX() {
	for i := range impactPool {
		impactPool[i].Active = false
	}
}

// UpdateImpactVFX 每帧更新所有命中特效。
func UpdateImpactVFX(dt float64) {
	for i := range impactPool {
		v := &impactPool[i]
		if !v.Active {
			continue
		}
		v.Life -= dt
		if v.Life <= 0 {
			v.Active = false
		}
	}
}

// DrawImpactVFX 绘制所有存活命中特效。
func DrawImpactVFX(screen *ebiten.Image) {
	for i := range impactPool {
		v := &impactPool[i]
		if !v.Active {
			continue
		}
		alpha := v.Life / v.MaxLife // 1→0 渐隐
		progress := 1.0 - alpha     // 0→1 扩展

		cx := float32(v.X)
		cy := float32(v.Y)

		// 通用命中：白色小闪光 + 扩散环
		if alpha > 0.4 {
			flashR := float32(3 * alpha)
			draw.FilledCircle(screen, cx, cy, flashR,
				color.RGBA{R: 255, G: 255, B: 240, A: uint8(200 * alpha)})
		}
		ringR := v.Radius * float32(progress)
		ringW := float32(1.5 * alpha)
		if ringW < 0.3 {
			ringW = 0.3
		}
		draw.CircleOutline(screen, cx, cy, ringR, ringW,
			color.RGBA{R: v.Color.R, G: v.Color.G, B: v.Color.B, A: uint8(float64(v.Color.A) * alpha)})
	}
}
