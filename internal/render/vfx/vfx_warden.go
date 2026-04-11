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

// DrawFireTrails 绘制火灵地面燃烧区。
// 多层渐变 + 边缘火焰粒子 + 呼吸脉冲 + 热浪扭曲环。
func DrawFireTrails(screen *ebiten.Image, trails []FireTrailVFX, animTime float64) {
	for idx, t := range trails {
		p := t.Life / t.MaxLife
		cx, cy := float32(t.X), float32(t.Y)
		r := float32(t.Radius)
		seed := float64(idx) * 1.7

		// 呼吸脉冲（半径微调）
		breathe := float32(1.0 + 0.08*math.Sin(animTime*6+seed))

		// 1. 外层暗红光晕（脉冲）
		draw.Glow(screen, cx, cy, r*0.8*breathe, r*1.5*breathe,
			color.RGBA{R: 180, G: 40, B: 0, A: uint8(35 * p)})

		// 2. 中层橙色火焰
		draw.FilledCircle(screen, cx, cy, r*breathe,
			color.RGBA{R: 255, G: 100, B: 30, A: uint8(90 * p)})

		// 3. 内核亮黄（脉动更快）
		coreBreath := float32(1.0 + 0.15*math.Sin(animTime*10+seed))
		draw.FilledCircle(screen, cx, cy, r*0.45*coreBreath,
			color.RGBA{R: 255, G: 210, B: 80, A: uint8(100 * p)})

		// 4. 边缘火焰粒子（6 颗沿圆周随机跳动）
		for i := 0; i < 6; i++ {
			angle := float64(i)*math.Pi/3 + math.Sin(animTime*3+seed+float64(i)*2.1)*0.5
			dist := float64(r) * (0.8 + 0.3*math.Sin(animTime*5+float64(i)*1.3))
			fx := cx + float32(math.Cos(angle)*dist)
			fy := cy + float32(math.Sin(angle)*dist)
			// 火舌向上飘动
			fy -= float32(3 * p * math.Abs(math.Sin(animTime*7+float64(i))))
			fa := uint8(float64(120) * p * (0.5 + 0.5*math.Sin(animTime*8+float64(i)*1.7)))
			fr := float32(1.5 + 0.8*math.Sin(animTime*6+float64(i)))
			draw.FilledCircle(screen, fx, fy, fr,
				color.RGBA{R: 255, G: 160, B: 40, A: fa})
		}

		// 5. 热浪扩散环（周期性向外扩散）
		waveT := math.Mod(animTime+seed, 1.5) / 1.5
		waveR := r * (0.6 + 1.0*float32(waveT))
		waveA := uint8(float64(60) * p * (1 - waveT))
		if waveA > 2 {
			draw.CircleOutline(screen, cx, cy, waveR, float32(0.8*(1-waveT)+0.3),
				color.RGBA{R: 255, G: 120, B: 30, A: waveA})
		}
	}
}

// DrawFireballs 绘制火灵飞行火球。
// 外层热浪光晕 + 旋转火翼 + 渐变拖尾 + 明亮核心 + 前方冲击波。
func DrawFireballs(screen *ebiten.Image, fireballs []FireballVFX, animTime float64) {
	for idx, fb := range fireballs {
		fx, fy := float32(fb.X), float32(fb.Y)
		r := float32(fb.Radius)
		seed := float64(idx) * 2.3

		dx := fb.EndX - fb.StartX
		dy := fb.EndY - fb.StartY
		dist := math.Hypot(dx, dy)
		var nx, ny float32
		if dist > 1 {
			nx, ny = float32(dx/dist), float32(dy/dist)
		}

		// 1. 外层热浪光晕（呼吸脉冲）
		glowPulse := float32(1.0 + 0.2*math.Sin(animTime*12+seed))
		draw.Glow(screen, fx, fy, r*0.5, r*1.4*glowPulse,
			color.RGBA{R: 255, G: 80, B: 0, A: 35})

		// 2. 渐变拖尾（5 段，越远越暗越小，带横向抖动）
		if fb.Progress > 0.05 && dist > 1 {
			for i := 1; i <= 5; i++ {
				off := float32(i) * r * 0.5
				t := float64(i) / 5.0
				// 横向抖动模拟火焰飘动
				jitter := float32(math.Sin(animTime*15+seed+float64(i)*3) * 1.5)
				tx := fx - nx*off + ny*jitter
				ty := fy - ny*off - nx*jitter
				ta := uint8(float64(140) * (1 - t*0.8))
				tr := r * float32(0.5-t*0.3)
				// 颜色从橙黄渐变到暗红
				rr := uint8(255)
				gg := uint8(float64(180) * (1 - t*0.7))
				bb := uint8(float64(40) * (1 - t))
				draw.FilledCircle(screen, tx, ty, tr,
					color.RGBA{R: rr, G: gg, B: bb, A: ta})
			}
		}

		// 3. 旋转火翼（2 条弧形火焰臂绕核心旋转）
		rot := animTime*8 + seed
		for i := 0; i < 2; i++ {
			a := rot + float64(i)*math.Pi
			wingR := float64(r) * 0.7
			wx := fx + float32(math.Cos(a)*wingR)
			wy := fy + float32(math.Sin(a)*wingR)
			wa := uint8(160 + 50*math.Sin(animTime*10+float64(i)))
			draw.FilledCircle(screen, wx, wy, r*0.3,
				color.RGBA{R: 255, G: 150, B: 30, A: wa})
			// 翼尖小火星
			tipX := fx + float32(math.Cos(a)*wingR*1.3)
			tipY := fy + float32(math.Sin(a)*wingR*1.3)
			draw.FilledCircle(screen, tipX, tipY, 1.2,
				color.RGBA{R: 255, G: 220, B: 100, A: wa / 2})
		}

		// 4. 明亮核心（白黄色，高亮度脉冲）
		corePulse := float32(1.0 + 0.1*math.Sin(animTime*14+seed))
		draw.FilledCircle(screen, fx, fy, r*0.5*corePulse,
			color.RGBA{R: 255, G: 210, B: 60, A: 250})
		draw.FilledCircle(screen, fx, fy, r*0.25*corePulse,
			color.RGBA{R: 255, G: 250, B: 200, A: 220})

		// 5. 前方冲击波弧线（飞行方向前方的微弱弧线）
		if dist > 1 && fb.Progress > 0.1 && fb.Progress < 0.95 {
			bowA := uint8(60 + 40*math.Sin(animTime*6+seed))
			bowR := r * 0.8
			// 前方半弧
			headAngle := math.Atan2(float64(ny), float64(nx))
			for j := -3; j <= 3; j++ {
				a := headAngle + float64(j)*0.2
				bx := fx + nx*r*0.7 + float32(math.Cos(a))*bowR
				by := fy + ny*r*0.7 + float32(math.Sin(a))*bowR
				draw.FilledCircle(screen, bx, by, 0.8,
					color.RGBA{R: 255, G: 180, B: 60, A: bowA})
			}
		}
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

// DrawSkyStrike 绘制天击打击效果（含预警标记）。
func DrawSkyStrike(screen *ebiten.Image, x, y float32, timer float64, mode int) {
	p := float32(timer / 0.8)

	// 预警标记（按模式差异化）
	if p > 0.3 {
		warningP := (p - 0.3) / 0.7
		switch mode {
		case 1: // 冰锥：收缩十字准星
			drawIceWarning(screen, x, y, warningP)
		case 2: // 连击：闪烁 X 标记
			drawBurstWarning(screen, x, y, warningP)
		case 3: // 间歇泉：地面裂缝
			drawGeyserWarning(screen, x, y, warningP)
		default:
			drawIceWarning(screen, x, y, warningP)
		}
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

// drawIceWarning 冰锥预警：四角向中心收缩的准星线段。
func drawIceWarning(screen *ebiten.Image, x, y, p float32) {
	a := uint8(140 * p)
	spread := 14 * (2 - p) // 由远到近收缩
	lineLen := float32(5)
	clr := color.RGBA{R: 140, G: 220, B: 255, A: a}
	// 四条向心短线
	draw.Line(screen, x-spread, y, x-spread+lineLen, y, 1.2, clr, true)
	draw.Line(screen, x+spread, y, x+spread-lineLen, y, 1.2, clr, true)
	draw.Line(screen, x, y-spread, x, y-spread+lineLen, 1.2, clr, true)
	draw.Line(screen, x, y+spread, x, y+spread-lineLen, 1.2, clr, true)
}

// drawBurstWarning 连击预警：旋转 X 标记。
func drawBurstWarning(screen *ebiten.Image, x, y, p float32) {
	a := uint8(160 * p)
	r := 8 * (2 - p)
	clr := color.RGBA{R: 80, G: 170, B: 255, A: a}
	draw.Line(screen, x-r, y-r, x+r, y+r, 1.5, clr, true)
	draw.Line(screen, x+r, y-r, x-r, y+r, 1.5, clr, true)
}

// drawGeyserWarning 间歇泉预警：地面裂缝纹路。
func drawGeyserWarning(screen *ebiten.Image, x, y, p float32) {
	a := uint8(120 * p)
	spread := 10 * p
	clr := color.RGBA{R: 100, G: 200, B: 255, A: a}
	// 从中心向外的 3 条裂缝线
	draw.Line(screen, x, y, x-spread, y+spread*0.6, 1, clr, true)
	draw.Line(screen, x, y, x+spread*0.8, y+spread*0.4, 1, clr, true)
	draw.Line(screen, x, y, x+spread*0.3, y-spread*0.7, 1, clr, true)
}

// drawIceCone 冰锥坠落效果（模式 1：多目标）。
// 尖锐冰刺从天而降 → 菱形碎片扩散 + 冰霜地面纹路。
func drawIceCone(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(220 * p)
	drop := 80 * p * p
	tipY := sy - drop

	// 冰锥主体（菱形轮廓，比纯线条更厚实）
	coneClr := color.RGBA{R: 130, G: 215, B: 255, A: a}
	draw.Diamond(screen, sx, tipY, 6, 1.8, coneClr)
	// 冰锥内部高光竖线
	draw.Line(screen, sx, tipY-5, sx, tipY+4, 1.0, color.RGBA{R: 220, G: 245, B: 255, A: a}, true)

	// 落地后效果
	if p < 0.4 {
		ip := (0.4 - p) / 0.4
		ia := uint8(200 * ip)
		// 冰霜地面纹路：从中心向外 4 条辐射裂纹
		crackLen := float32(12 * ip)
		crackClr := color.RGBA{R: 180, G: 230, B: 255, A: ia}
		draw.Line(screen, sx, sy, sx-crackLen, sy+crackLen*0.4, 1.2, crackClr, true)
		draw.Line(screen, sx, sy, sx+crackLen*0.9, sy+crackLen*0.3, 1.2, crackClr, true)
		draw.Line(screen, sx, sy, sx-crackLen*0.5, sy-crackLen*0.7, 1.0, crackClr, true)
		draw.Line(screen, sx, sy, sx+crackLen*0.6, sy-crackLen*0.5, 1.0, crackClr, true)
		// 3 颗菱形碎片向外飞散
		for i := 0; i < 3; i++ {
			angle := float64(i)*2.1 + 0.5
			dist := float64(crackLen) * 0.8
			dx := sx + float32(math.Cos(angle)*dist)
			dy := sy + float32(math.Sin(angle)*dist)
			shardA := uint8(float64(ia) * 0.7)
			draw.Diamond(screen, dx, dy, 2.5*ip, 1.0, color.RGBA{R: 200, G: 240, B: 255, A: shardA})
		}
	}
}

// drawWaterDrop 连击打击效果（模式 2：单目标连击）。
// 快速锤击 → X 形冲击标记 + 放射状冲击线。
func drawWaterDrop(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(230 * p)
	drop := 60 * p * p
	dropY := sy - drop

	// 下落的楔形打击标记（倒三角）
	w := float32(3.5)
	hitClr := color.RGBA{R: 70, G: 150, B: 255, A: a}
	draw.Line(screen, sx-w, dropY-w, sx, dropY+w, 2.0, hitClr, true)
	draw.Line(screen, sx+w, dropY-w, sx, dropY+w, 2.0, hitClr, true)
	draw.Line(screen, sx-w, dropY-w, sx+w, dropY-w, 1.5, hitClr, true)
	// 内部高光
	draw.Line(screen, sx, dropY-w+1, sx, dropY+w-1, 1.0,
		color.RGBA{R: 170, G: 220, B: 255, A: a}, true)

	// 命中后效果
	if p < 0.5 {
		ip := (0.5 - p) / 0.5
		ia := uint8(180 * ip)
		// X 形冲击标记
		xr := float32(8 * ip)
		xClr := color.RGBA{R: 100, G: 180, B: 255, A: ia}
		draw.Line(screen, sx-xr, sy-xr, sx+xr, sy+xr, 1.8, xClr, true)
		draw.Line(screen, sx+xr, sy-xr, sx-xr, sy+xr, 1.8, xClr, true)
		// 4 条放射冲击线（向外扩张后消失）
		lineLen := float32(5 * ip)
		lineR := float32(6 + 8*(1-ip))
		lineClr := color.RGBA{R: 140, G: 210, B: 255, A: uint8(float64(ia) * 0.6)}
		draw.Line(screen, sx-lineR, sy, sx-lineR-lineLen, sy, 1.0, lineClr, true)
		draw.Line(screen, sx+lineR, sy, sx+lineR+lineLen, sy, 1.0, lineClr, true)
		draw.Line(screen, sx, sy-lineR, sx, sy-lineR-lineLen, 1.0, lineClr, true)
		draw.Line(screen, sx, sy+lineR, sx, sy+lineR+lineLen, 1.0, lineClr, true)
	}
}

// drawGeyser 间歇泉喷涌效果（模式 3：百分比生命值）。
// 地面裂开 → 水柱喷涌（粗线条）→ 弧形水花抛物线。
func drawGeyser(screen *ebiten.Image, sx, sy, p float32) {
	a := uint8(200 * p)
	height := float32(40 * p)

	// 水柱主体（粗线条 + 内部高光，不用圆点）
	pillarClr := color.RGBA{R: 80, G: 180, B: 255, A: a}
	draw.ThickLine(screen, sx-2, sy, sx-1, sy-height, 3, pillarClr)
	draw.ThickLine(screen, sx+2, sy, sx+1, sy-height, 3, pillarClr)
	// 中心高亮线
	draw.Line(screen, sx, sy, sx, sy-height-3, 1.5,
		color.RGBA{R: 200, G: 240, B: 255, A: a}, true)

	// 顶部菱形水花标记
	if p > 0.2 {
		topP := (p - 0.2) / 0.8
		topA := uint8(180 * topP)
		draw.Diamond(screen, sx, sy-height-4, float32(4*topP), 1.5,
			color.RGBA{R: 180, G: 230, B: 255, A: topA})
	}

	// 弧形水花抛物线（从顶部向两侧抛出的弧线）
	if p > 0.3 {
		sp := (p - 0.3) / 0.7
		sa := uint8(140 * sp)
		splashClr := color.RGBA{R: 140, G: 215, B: 255, A: sa}
		// 左侧抛物线弧（3 段折线近似弧线）
		arcW := float32(16 * sp)
		arcH := float32(10 * sp)
		draw.Line(screen, sx, sy-height*0.8, sx-arcW*0.5, sy-height*0.8-arcH, 1.2, splashClr, true)
		draw.Line(screen, sx-arcW*0.5, sy-height*0.8-arcH, sx-arcW, sy-height*0.4, 1.0, splashClr, true)
		// 右侧抛物线弧
		draw.Line(screen, sx, sy-height*0.8, sx+arcW*0.4, sy-height*0.8-arcH*0.8, 1.2, splashClr, true)
		draw.Line(screen, sx+arcW*0.4, sy-height*0.8-arcH*0.8, sx+arcW*0.9, sy-height*0.3, 1.0, splashClr, true)
	}

	// 底部地面裂缝（贯穿始终，代替原来的圆形地面标记）
	crackA := uint8(100 * p)
	crackClr := color.RGBA{R: 120, G: 200, B: 255, A: crackA}
	draw.Line(screen, sx-8, sy+2, sx+8, sy+2, 1.0, crackClr, true)
	draw.Line(screen, sx-5, sy+1, sx-10, sy+4, 0.8, crackClr, true)
	draw.Line(screen, sx+5, sy+1, sx+10, sy+4, 0.8, crackClr, true)
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
