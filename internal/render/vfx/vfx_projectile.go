// vfx_projectile.go — 弹道视觉效果（解耦版）。
// 所有函数只接受纯值参数，零 core 包依赖。
package vfx

import (
	"image/color"
	"math"
	"strings"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TrailPt 弹道拖尾的单个历史点。
type TrailPt struct {
	X, Y   float64
	Active bool
}

// DrawProjectileBody 绘制弹体（含速度线）。
// style: SourceTowerKey，用于视觉类型判定。
// angle: 飞行方向弧度。
func DrawProjectileBody(screen *ebiten.Image, cx, cy float32, angle float64, style string, penetrate, scatter bool) {
	// Velocity tail (all projectiles) — thicker, brighter
	const tailLen = 8.0
	tailX := float64(cx) - math.Cos(angle)*tailLen
	tailY := float64(cy) - math.Sin(angle)*tailLen
	draw.ThickLine(screen, cx, cy, float32(tailX), float32(tailY), 1.5,
		color.RGBA{R: 255, G: 255, B: 255, A: 120})

	switch {
	case penetrate:
		draw.Glow(screen, cx, cy, 5, 12, color.RGBA{R: 180, G: 100, B: 255, A: 200})
		draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 220, G: 180, B: 255, A: 230})

	case scatter:
		// Core circle + short trailing line
		draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 100, G: 180, B: 255, A: 200})
		scTailX := float64(cx) - math.Cos(angle)*5
		scTailY := float64(cy) - math.Sin(angle)*5
		draw.ThickLine(screen, cx, cy, float32(scTailX), float32(scTailY), 1,
			color.RGBA{R: 100, G: 180, B: 255, A: 140})

	case strings.Contains(style, "sniper"):
		draw.Glow(screen, cx, cy, theme.ProjSniperR, theme.ProjSniperGlow, theme.ProjSniper)

	case strings.Contains(style, "rapid"):
		// Front circle + trailing circle + halo
		draw.FilledCircle(screen, cx, cy, theme.ProjDefaultR, theme.ProjRapid)
		trailCx := cx - float32(math.Cos(angle)*3)
		trailCy := cy - float32(math.Sin(angle)*3)
		draw.FilledCircle(screen, trailCx, trailCy, theme.ProjDefaultR*0.6,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 160})
		draw.Glow(screen, cx, cy, theme.ProjDefaultR, theme.ProjDefaultR+4,
			color.RGBA{R: theme.ProjRapid.R, G: theme.ProjRapid.G, B: theme.ProjRapid.B, A: 60})

	case strings.Contains(style, "freeze"):
		draw.DiamondRotated(screen, cx, cy, theme.ProjDefaultR+1, 1.5, angle, theme.ProjFreeze)

	case strings.Contains(style, "wind"):
		// Center circle + 2 static arc swirls
		draw.FilledCircle(screen, cx, cy, theme.ProjDefaultR, theme.ProjWind)
		arcR := float32(theme.ProjDefaultR + 2)
		sweepHalf := float32(0.2) // half of 0.4 rad sweep
		a1 := float32(angle) + math.Pi/3
		draw.Arc(screen, cx, cy, arcR, a1-sweepHalf, a1+sweepHalf, 1,
			color.RGBA{R: theme.ProjWind.R, G: theme.ProjWind.G, B: theme.ProjWind.B, A: 140})
		a2 := float32(angle) + math.Pi + math.Pi/3
		draw.Arc(screen, cx, cy, arcR, a2-sweepHalf, a2+sweepHalf, 1,
			color.RGBA{R: theme.ProjWind.R, G: theme.ProjWind.G, B: theme.ProjWind.B, A: 140})

	default:
		draw.Glow(screen, cx, cy, 3, 10, theme.ProjDefault)
	}
}

// DrawProjectileTrail 绘制弹道拖尾。
// trail: 环形缓冲区，cursor: 下一个写入位置，baseClr: 基础颜色。
func DrawProjectileTrail(screen *ebiten.Image, trail []TrailPt, cursor int, baseClr color.RGBA) {
	n := len(trail)
	if n == 0 {
		return
	}

	// Track previous active point for connecting lines
	var prevX, prevY float32
	var prevActive bool
	var prevAlpha uint8
	var prevR float32

	for i := 0; i < n; i++ {
		idx := (cursor + i) % n
		pt := trail[idx]
		if !pt.Active {
			prevActive = false
			continue
		}
		frac := float64(i+1) / float64(n)
		alpha := uint8(140 * frac)
		r := float32(theme.ProjDefaultR) * float32(0.3+0.7*frac)
		clr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: alpha}

		curX := float32(pt.X)
		curY := float32(pt.Y)

		// Connect adjacent active points with a thick line
		if prevActive {
			lineAlpha := prevAlpha
			if alpha < lineAlpha {
				lineAlpha = alpha
			}
			lineClr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: lineAlpha}
			lineW := prevR * 0.8
			if lineW < 0.5 {
				lineW = 0.5
			}
			draw.ThickLine(screen, prevX, prevY, curX, curY, lineW, lineClr)
		}

		draw.FilledCircle(screen, curX, curY, r, clr)

		// Newest point highlight glow
		if i == n-1 {
			draw.Glow(screen, curX, curY, r*0.5, r*1.5, clr)
		}

		prevX, prevY = curX, curY
		prevAlpha = alpha
		prevR = r
		prevActive = true
	}
}

// DrawWindArcs 绘制风系弹道的两个弧线装饰（从批量化弹体中拆出的仅 arc 部分）。
func DrawWindArcs(screen *ebiten.Image, cx, cy float32, angle float64) {
	arcR := float32(theme.ProjDefaultR + 2)
	sweepHalf := float32(0.2)
	a1 := float32(angle) + math.Pi/3
	draw.Arc(screen, cx, cy, arcR, a1-sweepHalf, a1+sweepHalf, 1,
		color.RGBA{R: theme.ProjWind.R, G: theme.ProjWind.G, B: theme.ProjWind.B, A: 140})
	a2 := float32(angle) + math.Pi + math.Pi/3
	draw.Arc(screen, cx, cy, arcR, a2-sweepHalf, a2+sweepHalf, 1,
		color.RGBA{R: theme.ProjWind.R, G: theme.ProjWind.G, B: theme.ProjWind.B, A: 140})
}

// ProjectileTrailColor 根据塔类型返回拖尾基础颜色。
func ProjectileTrailColor(towerKey string) color.RGBA {
	switch {
	case strings.Contains(towerKey, "sniper"):
		return theme.ProjSniper
	case strings.Contains(towerKey, "rapid"):
		return theme.ProjRapid
	case strings.Contains(towerKey, "freeze"):
		return theme.ProjFreeze
	case strings.Contains(towerKey, "wind"):
		return theme.ProjWindTrail
	default:
		return theme.ProjDefault
	}
}
