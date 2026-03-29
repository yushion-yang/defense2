// draw_warden.go — warden rendering.
// Dispatches by warden type to render geometric representations.
package render

import (
	"image/color"
	"math"

	"defense2/internal/core/warden"
	wardenTypes "defense2/internal/core/warden/types"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawWarden renders a single warden entity.
func DrawWarden(screen *ebiten.Image, w *warden.Warden) {
	if w == nil || !w.Active {
		return
	}

	base := w.BaseState()
	if base == nil {
		return
	}
	if base.X == 0 && base.Y == 0 {
		return
	}

	// 公共：射击线（所有能攻击的战灵共享）
	if base.ShootTimer > 0 {
		alpha := uint8(200 * (base.ShootTimer / 0.15))
		clr := wardenShootColor(w.Type)
		draw.Line(screen,
			float32(base.X), float32(base.Y),
			float32(base.LastTargetX), float32(base.LastTargetY),
			2, color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: alpha}, true)
	}

	// 类型特有：本体形状
	switch s := w.State.(type) {
	case *wardenTypes.EnvoyState:
		drawEnvoy(screen, s)
	case *wardenTypes.PrinceState:
		drawPrince(screen, s)
	case *wardenTypes.CoreState:
		drawCoreMechBody(screen, &s.WardenState)
	case *wardenTypes.ChainState:
		drawChainBody(screen, &s.WardenState)
	case *wardenTypes.SkystrikeState:
		drawSkystrikeBody(screen, s)
	}
}

// wardenShootColor 返回各类型战灵的射击线颜色。
func wardenShootColor(typ string) color.RGBA {
	switch typ {
	case "core":
		return color.RGBA{R: 100, G: 180, B: 255, A: 200}
	case "chain":
		return color.RGBA{R: 80, G: 255, B: 120, A: 200}
	case "skystrike":
		return color.RGBA{R: 80, G: 200, B: 255, A: 200}
	default:
		return color.RGBA{R: 200, G: 200, B: 200, A: 200}
	}
}

// drawEnvoy renders the envoy warden (purple circle / gold when possessing).
func drawEnvoy(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	cx := float32(s.X)
	cy := float32(s.Y)
	r := float32(8)

	// Aura
	draw.FilledCircle(screen, cx, cy, r+4,
		color.RGBA{R: 180, G: 120, B: 255, A: 40})

	// Body
	bodyClr := color.RGBA{R: 180, G: 120, B: 255, A: 200}
	if s.Phase == "possessing" {
		bodyClr = color.RGBA{R: 255, G: 200, B: 100, A: 230}
	}
	draw.FilledCircle(screen, cx, cy, r, bodyClr)
}

// drawPrince renders the prince warden (orange diamond + dash trail + flame traces).
func drawPrince(screen *ebiten.Image, s *wardenTypes.PrinceState) {
	cx := float32(s.X)
	cy := float32(s.Y)

	// Flame trails
	for _, t := range s.Trails {
		alpha := uint8(120 * (t.Life / t.MaxLife))
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), float32(t.Radius),
			color.RGBA{R: 255, G: 100, B: 30, A: alpha})
	}

	// Body (orange diamond)
	r := float32(10)
	bodyClr := color.RGBA{R: 255, G: 140, B: 30, A: 230}
	if s.Phase == "dashing" {
		bodyClr = color.RGBA{R: 255, G: 200, B: 80, A: 255}
		draw.Line(screen, float32(s.DashStartX), float32(s.DashStartY), cx, cy, 3,
			color.RGBA{R: 255, G: 120, B: 20, A: 100}, true)
	}

	draw.Diamond(screen, cx, cy, r, 2, bodyClr)
	draw.FilledCircle(screen, cx, cy, r*0.5, bodyClr)
}

// drawCoreMechBody renders the core mech body (blue rotating triangle).
func drawCoreMechBody(screen *ebiten.Image, s *warden.WardenState) {
	cx := float32(s.X)
	cy := float32(s.Y)
	r := float32(10)
	bodyClr := color.RGBA{R: 60, G: 140, B: 255, A: 230}
	angle := s.OrbitAngle

	x1 := cx + r*float32(math.Cos(angle))
	y1 := cy + r*float32(math.Sin(angle))
	x2 := cx + r*float32(math.Cos(angle+2.4))
	y2 := cy + r*float32(math.Sin(angle+2.4))
	x3 := cx + r*float32(math.Cos(angle-2.4))
	y3 := cy + r*float32(math.Sin(angle-2.4))
	draw.Line(screen, x1, y1, x2, y2, 2, bodyClr, true)
	draw.Line(screen, x2, y2, x3, y3, 2, bodyClr, true)
	draw.Line(screen, x3, y3, x1, y1, 2, bodyClr, true)

	draw.FilledCircle(screen, cx, cy, 3,
		color.RGBA{R: 100, G: 200, B: 255, A: 200})
}

// drawChainBody 渲染能量串联战灵本体（绿色菱形）。
func drawChainBody(screen *ebiten.Image, s *warden.WardenState) {
	cx := float32(s.X)
	cy := float32(s.Y)

	// 光环
	draw.FilledCircle(screen, cx, cy, 12,
		color.RGBA{R: 80, G: 255, B: 120, A: 30})

	// 菱形本体
	bodyClr := color.RGBA{R: 80, G: 230, B: 120, A: 220}
	draw.Diamond(screen, cx, cy, 8, 2, bodyClr)
	draw.FilledCircle(screen, cx, cy, 4, bodyClr)
}

// drawSkystrikeBody 渲染天降战灵本体（天蓝色三角 + AoE 效果）。
func drawSkystrikeBody(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
	cx := float32(s.X)
	cy := float32(s.Y)

	// 三角本体
	bodyClr := color.RGBA{R: 80, G: 200, B: 255, A: 220}
	r := float32(9)
	angle := s.OrbitAngle
	x1 := cx + r*float32(math.Cos(angle))
	y1 := cy + r*float32(math.Sin(angle))
	x2 := cx + r*float32(math.Cos(angle+2.4))
	y2 := cy + r*float32(math.Sin(angle+2.4))
	x3 := cx + r*float32(math.Cos(angle-2.4))
	y3 := cy + r*float32(math.Sin(angle-2.4))
	draw.Line(screen, x1, y1, x2, y2, 2, bodyClr, true)
	draw.Line(screen, x2, y2, x3, y3, 2, bodyClr, true)
	draw.Line(screen, x3, y3, x1, y1, 2, bodyClr, true)
	draw.FilledCircle(screen, cx, cy, 3,
		color.RGBA{R: 120, G: 220, B: 255, A: 200})

	// AoE 打击视觉效果
	if s.StrikeTimer <= 0 {
		return
	}
	progress := s.StrikeTimer / 0.5
	alpha := uint8(150 * progress)
	aoeR := float32(s.AoERadius) * float32(1.2-0.2*progress)

	draw.FilledCircle(screen, float32(s.StrikeX), float32(s.StrikeY), aoeR,
		color.RGBA{R: 80, G: 200, B: 255, A: alpha / 3})
	draw.CircleOutline(screen, float32(s.StrikeX), float32(s.StrikeY), aoeR, 2,
		color.RGBA{R: 100, G: 220, B: 255, A: alpha})
}
