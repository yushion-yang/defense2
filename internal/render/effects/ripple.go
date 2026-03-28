// ripple.go — 波纹特效系统。
// 提供命中波纹和点击波纹两种预设，对象池管理生命周期。
package effects

import (
	"image/color"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// Ripple 波纹特效。
type Ripple struct {
	X, Y      float64    // 中心坐标
	Radius    float64    // 当前半径
	MaxRadius float64    // 最大半径
	Life      float64    // 剩余时间
	MaxLife   float64    // 初始时间
	Color     color.RGBA // 波纹颜色
}

// RipplePool 波纹对象池。
type RipplePool struct {
	ripples []Ripple
	cap     int // 最大容量
}

// NewRipplePool 创建指定容量的波纹对象池。
func NewRipplePool(capacity int) *RipplePool {
	if capacity <= 0 {
		capacity = 64
	}
	return &RipplePool{
		ripples: make([]Ripple, 0, capacity),
		cap:     capacity,
	}
}

// SpawnHit 生成命中波纹（小半径、红色）。
func (p *RipplePool) SpawnHit(x, y float64) {
	p.spawn(Ripple{
		X: x, Y: y,
		Radius:    0,
		MaxRadius: 18,
		Life:      0.25,
		MaxLife:   0.25,
		Color:     color.RGBA{R: 255, G: 80, B: 60, A: 200},
	})
}

// SpawnTap 生成点击波纹（大半径、白色）。
func (p *RipplePool) SpawnTap(x, y float64) {
	p.spawn(Ripple{
		X: x, Y: y,
		Radius:    0,
		MaxRadius: 36,
		Life:      0.4,
		MaxLife:   0.4,
		Color:     color.RGBA{R: 255, G: 255, B: 255, A: 160},
	})
}

// spawn 添加波纹到池中（超出容量时丢弃最旧的）。
func (p *RipplePool) spawn(r Ripple) {
	if len(p.ripples) >= p.cap {
		// 移除最旧的（索引 0）
		copy(p.ripples, p.ripples[1:])
		p.ripples = p.ripples[:len(p.ripples)-1]
	}
	p.ripples = append(p.ripples, r)
}

// Update 每帧更新所有波纹：扩展半径、衰减生命。
func (p *RipplePool) Update(dt float64) {
	n := 0
	for i := range p.ripples {
		r := &p.ripples[i]
		r.Life -= dt
		if r.Life <= 0 {
			continue
		}
		// 半径线性扩展
		progress := 1.0 - r.Life/r.MaxLife
		r.Radius = r.MaxRadius * progress
		p.ripples[n] = p.ripples[i]
		n++
	}
	p.ripples = p.ripples[:n]
}

// Draw 绘制所有存活波纹（扩展圆环 + alpha 渐隐）。
func (p *RipplePool) Draw(screen *ebiten.Image) {
	for i := range p.ripples {
		r := &p.ripples[i]
		// 透明度随生命衰减
		alpha := r.Life / r.MaxLife
		if alpha < 0 {
			alpha = 0
		}
		if alpha > 1 {
			alpha = 1
		}
		a := uint8(float64(r.Color.A) * alpha)
		clr := color.RGBA{R: r.Color.R, G: r.Color.G, B: r.Color.B, A: a}

		// 线宽随扩展递减（从 2.0 到 0.5）
		lineW := float32(2.0 - 1.5*(1.0-alpha))
		if lineW < 0.5 {
			lineW = 0.5
		}

		draw.CircleOutline(screen, float32(r.X), float32(r.Y), float32(r.Radius), lineW, clr)
	}
}

// Count 返回当前存活波纹数量。
func (p *RipplePool) Count() int {
	return len(p.ripples)
}
