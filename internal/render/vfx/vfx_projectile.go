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
	// 速度线（所有弹道通用）
	const tailLen = 4.0
	tailX := float64(cx) - math.Cos(angle)*tailLen
	tailY := float64(cy) - math.Sin(angle)*tailLen
	draw.Line(screen, cx, cy, float32(tailX), float32(tailY), 1,
		color.RGBA{R: 255, G: 255, B: 255, A: 80}, true)

	switch {
	case penetrate:
		draw.Glow(screen, cx, cy, 5, 12, color.RGBA{R: 180, G: 100, B: 255, A: 200})
		draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 220, G: 180, B: 255, A: 230})

	case scatter:
		draw.FilledCircle(screen, cx, cy, 3, color.RGBA{R: 100, G: 180, B: 255, A: 200})

	case strings.Contains(style, "sniper"):
		draw.Glow(screen, cx, cy, theme.ProjSniperR, theme.ProjSniperGlow, theme.ProjSniper)

	case strings.Contains(style, "rapid"):
		draw.FilledCircle(screen, cx, cy, theme.ProjDefaultR, theme.ProjRapid)

	case strings.Contains(style, "freeze"):
		draw.DiamondRotated(screen, cx, cy, theme.ProjDefaultR+1, 1.5, angle, theme.ProjFreeze)

	case strings.Contains(style, "wind"):
		draw.FilledCircle(screen, cx, cy, theme.ProjDefaultR, theme.ProjWind)

	default:
		draw.Glow(screen, cx, cy, theme.ProjDefaultR, theme.ProjDefaultGlow, theme.ProjDefault)
	}
}

// DrawProjectileTrail 绘制弹道拖尾。
// trail: 环形缓冲区，cursor: 下一个写入位置，baseClr: 基础颜色。
func DrawProjectileTrail(screen *ebiten.Image, trail []TrailPt, cursor int, baseClr color.RGBA) {
	n := len(trail)
	if n == 0 {
		return
	}
	for i := 0; i < n; i++ {
		idx := (cursor + i) % n
		pt := trail[idx]
		if !pt.Active {
			continue
		}
		frac := float64(i+1) / float64(n)
		alpha := uint8(80 * frac)
		r := float32(theme.ProjDefaultR) * float32(0.3+0.7*frac)
		clr := color.RGBA{R: baseClr.R, G: baseClr.G, B: baseClr.B, A: alpha}
		draw.FilledCircle(screen, float32(pt.X), float32(pt.Y), r, clr)
	}
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
