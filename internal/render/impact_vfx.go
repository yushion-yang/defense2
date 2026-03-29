// impact_vfx.go — 命中冲击特效系统。
// 全局对象池，单目标 impact ring + 中心闪光。
// 由 SpawnChargeImpact 触发（蓄力弹命中时）。
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
	Large   bool // true=蓄力弹大号特效, false=通用小型特效
}

const maxImpactVFX = 16

var impactPool [maxImpactVFX]ImpactVFX
var impactCursor int

// SpawnHitImpact 在指定位置生成通用命中特效（小型扩散环）。
func SpawnHitImpact(x, y float64) {
	v := &impactPool[impactCursor]
	impactCursor = (impactCursor + 1) % maxImpactVFX
	v.X = x
	v.Y = y
	v.Life = 0.15
	v.MaxLife = 0.15
	v.Active = true
	v.Large = false
}

// SpawnChargeImpact 在指定位置生成蓄力弹命中特效（大号扩散环）。
func SpawnChargeImpact(x, y float64) {
	v := &impactPool[impactCursor]
	impactCursor = (impactCursor + 1) % maxImpactVFX
	v.X = x
	v.Y = y
	v.Life = 0.25
	v.MaxLife = 0.25
	v.Active = true
	v.Large = true
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
		progress := 1.0 - alpha    // 0→1 扩展

		cx := float32(v.X)
		cy := float32(v.Y)

		if v.Large {
			// 蓄力弹：白色闪光 + 红色大扩散环
			if alpha > 0.3 {
				flashR := float32(6 * alpha)
				draw.FilledCircle(screen, cx, cy, flashR,
					color.RGBA{R: 254, G: 242, B: 242, A: uint8(255 * alpha)})
			}
			ringR := float32(25 * progress)
			ringW := float32(2.5 * alpha)
			if ringW < 0.5 {
				ringW = 0.5
			}
			draw.CircleOutline(screen, cx, cy, ringR, ringW,
				color.RGBA{R: 239, G: 68, B: 68, A: uint8(220 * alpha)})
		} else {
			// 通用命中：白色小闪光 + 淡黄扩散环
			if alpha > 0.4 {
				flashR := float32(3 * alpha)
				draw.FilledCircle(screen, cx, cy, flashR,
					color.RGBA{R: 255, G: 255, B: 240, A: uint8(200 * alpha)})
			}
			ringR := float32(12 * progress)
			ringW := float32(1.5 * alpha)
			if ringW < 0.3 {
				ringW = 0.3
			}
			draw.CircleOutline(screen, cx, cy, ringR, ringW,
				color.RGBA{R: 253, G: 224, B: 71, A: uint8(160 * alpha)})
		}
	}
}
