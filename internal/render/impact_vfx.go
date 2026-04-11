// impact_vfx.go — 命中冲击特效系统。
// 全局对象池，单目标 impact ring + 中心闪光。
package render

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImpactVFX 单个命中特效实例。
type ImpactVFX struct {
	X, Y    float64    // 快照坐标（fallback）
	Life    float64
	MaxLife float64
	Active  bool
	Color   color.RGBA // 扩散环颜色
	Radius  float32    // 最大扩散半径
	TrackX  *float64   // 跟踪目标 X（非 nil 时每帧读取最新位置）
	TrackY  *float64   // 跟踪目标 Y
}

const maxImpactVFX = 16

var impactPool [maxImpactVFX]ImpactVFX
var impactCursor int

// SpawnTypedImpact 根据攻击方式生成对应元素颜色的命中特效。
// trackX/trackY 为目标坐标指针，非 nil 时特效跟随目标移动。
func SpawnTypedImpact(trackX, trackY *float64, attackStyle string) {
	x, y := *trackX, *trackY
	switch attackStyle {
	case "scatter": // ice — blue
		spawnImpact(x, y, trackX, trackY, color.RGBA{R: 100, G: 180, B: 255, A: 200}, 8, 0.30)
	case "spin_aoe", "projectile": // fire/physical — orange
		spawnImpact(x, y, trackX, trackY, color.RGBA{R: 255, G: 140, B: 40, A: 200}, 8, 0.25)
	case "wideBeam": // energy — purple
		spawnImpact(x, y, trackX, trackY, color.RGBA{R: 200, G: 100, B: 255, A: 200}, 6, 0.20)
	default: // warm yellow default
		spawnImpact(x, y, trackX, trackY, color.RGBA{R: 255, G: 220, B: 100, A: 200}, 7, 0.25)
	}
}

// spawnImpact 生成一个自定义颜色和半径的小型命中特效。
func spawnImpact(x, y float64, trackX, trackY *float64, clr color.RGBA, radius float32, life float64) {
	v := &impactPool[impactCursor]
	impactCursor = (impactCursor + 1) % maxImpactVFX
	v.X = x
	v.Y = y
	v.TrackX = trackX
	v.TrackY = trackY
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
		impactPool[i].TrackX = nil
		impactPool[i].TrackY = nil
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
			v.TrackX = nil
			v.TrackY = nil
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

		// 跟踪目标最新位置，fallback 到快照坐标
		posX, posY := v.X, v.Y
		if v.TrackX != nil && v.TrackY != nil {
			posX, posY = *v.TrackX, *v.TrackY
		}
		cx := float32(posX)
		cy := float32(posY)

		// White center flash — larger, lingers longer
		if alpha > 0.5 {
			flashR := float32(5 * alpha)
			draw.FilledCircle(screen, cx, cy, flashR,
				color.RGBA{R: 255, G: 255, B: 240, A: uint8(200 * alpha)})
		}

		// Expanding ring — wider radius, thicker stroke
		ringR := v.Radius * 1.8 * float32(progress)
		ringW := float32(1.5 * alpha)
		if ringW < 0.3 {
			ringW = 0.3
		}
		ringAlpha := uint8(float64(v.Color.A) * alpha)
		draw.CircleOutline(screen, cx, cy, ringR, ringW,
			color.RGBA{R: v.Color.R, G: v.Color.G, B: v.Color.B, A: ringAlpha})

		// 4 spark lines radiating outward
		sparkAlpha := uint8(float64(ringAlpha) * 0.7)
		if sparkAlpha > 0 {
			sparkLen := float32(float64(v.Radius) * progress * 1.5)
			// Use position as deterministic seed for slight angle offset
			seed := float64(v.X*7.3 + v.Y*13.7)
			offset := math.Sin(seed) * 0.3
			for j := 0; j < 4; j++ {
				a := float64(j)*math.Pi/2 + offset
				ex := cx + float32(math.Cos(a))*sparkLen
				ey := cy + float32(math.Sin(a))*sparkLen
				draw.Line(screen, cx, cy, ex, ey, 1,
					color.RGBA{R: v.Color.R, G: v.Color.G, B: v.Color.B, A: sparkAlpha}, true)
			}
		}
	}
}
