// vfx_warden.go — 战灵视觉效果（解耦版）。
// 所有函数只接受纯值参数，零 core 包依赖。
package vfx

import (
	"image/color"
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 值类型定义 ──────────────────────────────────────

// FireTrailVFX 火灵地面燃烧区参数。
type FireTrailVFX struct {
	X, Y, Life, MaxLife, Radius float64
}

// FireballVFX 火灵飞行火球参数。
type FireballVFX struct {
	X, Y, Radius, Progress     float64
	StartX, StartY, EndX, EndY float64
}

// StrikeVFX 天击打击点参数。
type StrikeVFX struct {
	X, Y  float64
	Timer float64 // 1→0
	Mode  int     // 1=冰锥, 2=水滴, 3=水柱
}

// ── 移动拖尾 ────────────────────────────────────────

// DrawMovementTrail 绘制战灵移动拖尾。
// trail: 环形缓冲区 [N][2]float64, cursor: 下一个写入位置。
func DrawMovementTrail(screen *ebiten.Image, trail [][2]float64, cursor int, clr color.RGBA) {
	n := len(trail)
	for i := 0; i < n; i++ {
		idx := (cursor - 1 - i + n) % n
		tx, ty := trail[idx][0], trail[idx][1]
		if tx == 0 && ty == 0 || tx < -1000 {
			break
		}
		age := float64(i+1) / float64(n)
		alpha := uint8(float64(clr.A) * 0.3 * (1 - age))
		r := float32(6 * (1 - age))
		if r < 1 {
			break
		}
		draw.FilledCircle(screen, float32(tx), float32(ty), r,
			color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: alpha})
	}
}

// ── 火灵效果 ────────────────────────────────────────

// DrawFireTrails 绘制火灵地面燃烧区（多层渐变）。
func DrawFireTrails(screen *ebiten.Image, trails []FireTrailVFX) {
	for _, t := range trails {
		p := t.Life / t.MaxLife
		r := float32(t.Radius)
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r*1.3,
			color.RGBA{R: 180, G: 40, B: 0, A: uint8(40 * p)})
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r,
			color.RGBA{R: 255, G: 100, B: 30, A: uint8(100 * p)})
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), r*0.5,
			color.RGBA{R: 255, G: 200, B: 60, A: uint8(80 * p)})
	}
}

// DrawFireballs 绘制火灵飞行火球（拖尾 + 核心）。
func DrawFireballs(screen *ebiten.Image, fireballs []FireballVFX) {
	for _, fb := range fireballs {
		fx, fy := float32(fb.X), float32(fb.Y)
		r := float32(fb.Radius)
		if fb.Progress > 0.05 {
			dx := fb.EndX - fb.StartX
			dy := fb.EndY - fb.StartY
			dist := math.Hypot(dx, dy)
			if dist > 1 {
				nx, ny := float32(dx/dist), float32(dy/dist)
				for i := 1; i <= 3; i++ {
					off := float32(i) * r * 0.6
					ta := uint8(60 - i*15)
					draw.FilledCircle(screen, fx-nx*off, fy-ny*off, r*0.4,
						color.RGBA{R: 255, G: 120, B: 20, A: ta})
				}
			}
		}
		draw.FilledCircle(screen, fx, fy, r*0.9,
			color.RGBA{R: 255, G: 100, B: 0, A: 50})
		draw.FilledCircle(screen, fx, fy, r*0.45,
			color.RGBA{R: 255, G: 200, B: 60, A: 240})
	}
}

// ── 射击闪光（通用） ────────────────────────────────

// DrawShootFlash 绘制战灵射击闪光。
// shootTimer: 剩余时间（初始 0.15，递减到 0）。
func DrawShootFlash(screen *ebiten.Image, cx, cy float32, shootTimer float64, clr color.RGBA) {
	if shootTimer <= 0 {
		return
	}
	p := shootTimer / 0.15
	alpha := uint8(200 * p)
	r := float32(6 + 4*p)
	draw.FilledCircle(screen, cx, cy, r,
		color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: alpha / 3})
	draw.FilledCircle(screen, cx, cy, r*0.4,
		color.RGBA{R: min(clr.R+100, 255), G: min(clr.G+50, 255), B: 255, A: alpha})
}

// ── 聚能连线 ────────────────────────────────────────

// DrawChainLinks 绘制聚能战灵的串联线段 + 流动粒子 + 端点光晕。
// links: 每条线 [x1, y1, x2, y2]。
func DrawChainLinks(screen *ebiten.Image, links [][4]float64, animTime float64) {
	for _, l := range links {
		x1, y1, x2, y2 := float32(l[0]), float32(l[1]), float32(l[2]), float32(l[3])
		// ThickLine at alpha 100
		draw.ThickLine(screen, x1, y1, x2, y2, 2, color.RGBA{160, 80, 255, 100})
		// 3 flowing particles per link
		for i := 0; i < 3; i++ {
			phase := math.Mod(animTime*1.2+float64(i)/3.0, 1.0)
			px := float32(float64(x1) + float64(x2-x1)*phase)
			py := float32(float64(y1) + float64(y2-y1)*phase)
			pAlpha := uint8(120 + phase*100)
			draw.FilledCircle(screen, px, py, 2, color.RGBA{200, 140, 255, pAlpha})
		}
		// Glow at endpoints
		draw.Glow(screen, x1, y1, 3, 8, color.RGBA{160, 80, 255, 30})
		draw.Glow(screen, x2, y2, 3, 8, color.RGBA{160, 80, 255, 30})
	}
}

// ── 天击三式 ────────────────────────────────────────

// DrawSkyStrike 绘制天击打击效果（含预警圈）。
func DrawSkyStrike(screen *ebiten.Image, x, y float32, timer float64, mode int) {
	p := float32(timer / 0.8)

	// 预警瞄准圈
	if p > 0.3 {
		circleA := uint8(120 * (p - 0.3) / 0.7)
		circleR := float32(12 * (2 - p))
		draw.DashedCircle(screen, x, y, circleR, 1, 4, 3,
			color.RGBA{R: 120, G: 200, B: 255, A: circleA})
	}

	switch mode {
	case 1:
		drawIceCone(screen, x, y, p)
	case 2:
		drawWaterDrop(screen, x, y, p)
	case 3:
		drawGeyser(screen, x, y, p)
	default:
		drawIceCone(screen, x, y, p)
	}
}

// drawIceCone 绘制冰锥坠落效果。p: 1→0 进度。
func drawIceCone(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(200 * p)
	drop := 80 * p * p
	tipY := sy - drop
	draw.Line(screen, sx, tipY-8, sx-4, tipY, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx, tipY-8, sx+4, tipY, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx-4, tipY, sx, tipY+5, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.Line(screen, sx+4, tipY, sx, tipY+5, 1.5, color.RGBA{R: 120, G: 210, B: 255, A: a}, true)
	draw.FilledCircle(screen, sx, tipY-2, 2, color.RGBA{R: 220, G: 245, B: 255, A: a})

	if p < 0.4 {
		ip := (0.4 - p) / 0.4
		ia := uint8(180 * ip)
		draw.FilledCircle(screen, sx, sy, float32(6*ip), color.RGBA{R: 160, G: 230, B: 255, A: ia / 2})
		draw.FilledCircle(screen, sx, sy, float32(3*ip), color.RGBA{R: 220, G: 245, B: 255, A: ia})
		sp := float32(8 * ip)
		draw.FilledCircle(screen, sx-sp, sy-sp*0.4, 1.5, color.RGBA{R: 200, G: 235, B: 255, A: ia / 2})
		draw.FilledCircle(screen, sx+sp*0.8, sy+sp*0.3, 1.5, color.RGBA{R: 200, G: 235, B: 255, A: ia / 2})
	}
}

// drawWaterDrop 绘制水滴坠落溅射效果。p: 1→0 进度。
func drawWaterDrop(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(220 * p)
	drop := 60 * p * p
	dropY := sy - drop
	draw.FilledCircle(screen, sx, dropY, 3, color.RGBA{R: 60, G: 140, B: 255, A: a})
	draw.FilledCircle(screen, sx, dropY-2, 1.5, color.RGBA{R: 150, G: 200, B: 255, A: a})

	if p < 0.5 {
		ip := (0.5 - p) / 0.5
		ia := uint8(160 * ip)
		sp := float32(6 * ip)
		draw.FilledCircle(screen, sx-sp, sy-sp*0.3, 2, color.RGBA{R: 80, G: 160, B: 255, A: ia})
		draw.FilledCircle(screen, sx+sp, sy-sp*0.2, 2, color.RGBA{R: 80, G: 160, B: 255, A: ia})
		draw.FilledCircle(screen, sx, sy, float32(4*ip), color.RGBA{R: 120, G: 190, B: 255, A: ia / 2})
	}
}

// drawGeyser 绘制水柱喷涌效果。p: 1→0 进度。
func drawGeyser(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(180 * p)
	height := float32(35 * p)
	draw.FilledCircle(screen, sx, sy, float32(8*p), color.RGBA{R: 140, G: 210, B: 255, A: a / 3})
	draw.Line(screen, sx-3, sy, sx-2, sy-height, 3, color.RGBA{R: 100, G: 190, B: 255, A: a}, true)
	draw.Line(screen, sx+3, sy, sx+2, sy-height, 3, color.RGBA{R: 100, G: 190, B: 255, A: a}, true)
	draw.Line(screen, sx, sy, sx, sy-height-5, 2, color.RGBA{R: 200, G: 235, B: 255, A: a}, true)
	draw.FilledCircle(screen, sx, sy-height-3, float32(3*p), color.RGBA{R: 220, G: 240, B: 255, A: a})

	if p > 0.3 {
		sp := float32((p - 0.3) / 0.7)
		spr := float32(12 * sp)
		sa := uint8(120 * sp)
		draw.FilledCircle(screen, sx-spr, sy-height*0.6, 2, color.RGBA{R: 160, G: 220, B: 255, A: sa})
		draw.FilledCircle(screen, sx+spr*0.8, sy-height*0.5, 1.5, color.RGBA{R: 160, G: 220, B: 255, A: sa})
		draw.FilledCircle(screen, sx-spr*0.5, sy-height*0.3, 1.5, color.RGBA{R: 160, G: 220, B: 255, A: sa})
	}
}

// ── 金灵光束 ────────────────────────────────────────

// DrawGoldBeam 绘制金灵施 buff 的金色光束 + 流动粒子 + 端点光晕。
// progress: 1（刚施放）→0（衰减完毕）。animTime: 全局动画时间。
func DrawGoldBeam(screen *ebiten.Image, wx, wy, tx, ty float32, progress float64, animTime float64) {
	if progress <= 0 {
		return
	}
	p := progress
	alpha := uint8(200 * p)
	draw.Line(screen, wx, wy, tx, ty, float32(2*p),
		color.RGBA{255, 220, 80, alpha}, true)
	// 2 flowing particles along beam
	for i := 0; i < 2; i++ {
		phase := math.Mod(animTime*1.5+float64(i)*0.5, 1.0)
		px := float32(float64(wx) + float64(tx-wx)*phase)
		py := float32(float64(wy) + float64(ty-wy)*phase)
		pAlpha := uint8(float64(alpha) * (0.5 + 0.5*phase))
		draw.FilledCircle(screen, px, py, float32(2.5*p), color.RGBA{255, 240, 140, pAlpha})
	}
	// Glow at endpoints
	draw.Glow(screen, wx, wy, 3, 8, color.RGBA{255, 220, 80, uint8(float64(30) * p)})
	draw.Glow(screen, tx, ty, 4, 12, color.RGBA{255, 240, 130, uint8(float64(40) * p)})
	// Target flash
	draw.FilledCircle(screen, tx, ty, float32(10*p),
		color.RGBA{255, 240, 130, alpha / 2})
}
