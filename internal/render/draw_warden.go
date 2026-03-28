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
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawWarden renders a single warden entity.
func DrawWarden(screen *ebiten.Image, w *warden.Warden) {
	if w == nil || !w.Active {
		return
	}

	switch s := w.State.(type) {
	case *wardenTypes.EnvoyState:
		drawEnvoy(screen, s)
	case *wardenTypes.PrinceState:
		drawPrince(screen, s)
	case *wardenTypes.CoreState:
		drawCoreMech(screen, s)
	case *wardenTypes.ChainState:
		_ = s // no visual entity, state indicator only
	case *wardenTypes.SkystrikeState:
		drawSkystrike(screen, s)
	}
}

// drawEnvoy renders the envoy warden (purple circle / gold when possessing).
func drawEnvoy(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	cx := float32(s.X)
	cy := float32(s.Y)
	if cx == 0 && cy == 0 {
		return
	}
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
	if cx == 0 && cy == 0 {
		return
	}

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
		// Dash trail line
		vector.StrokeLine(screen, float32(s.DashStartX), float32(s.DashStartY), cx, cy, 3,
			color.RGBA{R: 255, G: 120, B: 20, A: 100}, true)
	}

	// Diamond outline
	draw.Diamond(screen, cx, cy, r, 2, bodyClr)
	// Filled center
	draw.FilledCircle(screen, cx, cy, r*0.5, bodyClr)
}

// drawCoreMech renders the core mech warden (blue triangle + shoot line).
func drawCoreMech(screen *ebiten.Image, s *wardenTypes.CoreState) {
	cx := float32(s.X)
	cy := float32(s.Y)
	if cx == 0 && cy == 0 {
		return
	}

	// Shoot line (briefly visible)
	if s.ShootTimer > 0 {
		alpha := uint8(200 * (s.ShootTimer / 0.15))
		vector.StrokeLine(screen, cx, cy, float32(s.LastTargetX), float32(s.LastTargetY), 2,
			color.RGBA{R: 100, G: 180, B: 255, A: alpha}, true)
	}

	// Body (blue triangle)
	r := float32(10)
	bodyClr := color.RGBA{R: 60, G: 140, B: 255, A: 230}
	angle := s.OrbitAngle
	x1 := cx + r*float32(math.Cos(angle))
	y1 := cy + r*float32(math.Sin(angle))
	x2 := cx + r*float32(math.Cos(angle+2.4))
	y2 := cy + r*float32(math.Sin(angle+2.4))
	x3 := cx + r*float32(math.Cos(angle-2.4))
	y3 := cy + r*float32(math.Sin(angle-2.4))
	vector.StrokeLine(screen, x1, y1, x2, y2, 2, bodyClr, true)
	vector.StrokeLine(screen, x2, y2, x3, y3, 2, bodyClr, true)
	vector.StrokeLine(screen, x3, y3, x1, y1, 2, bodyClr, true)

	// Center glow dot
	draw.FilledCircle(screen, cx, cy, 3,
		color.RGBA{R: 100, G: 200, B: 255, A: 200})
}

// drawSkystrike renders the skystrike AoE effect (sky-blue fading circle).
func drawSkystrike(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
	if s.StrikeTimer <= 0 {
		return
	}

	progress := s.StrikeTimer / 0.5
	alpha := uint8(150 * progress)
	r := float32(s.AoERadius) * float32(1.2-0.2*progress)

	// Fading fill
	draw.FilledCircle(screen, float32(s.StrikeX), float32(s.StrikeY), r,
		color.RGBA{R: 80, G: 200, B: 255, A: alpha / 3})
	// Outline ring
	draw.CircleOutline(screen, float32(s.StrikeX), float32(s.StrikeY), r, 2,
		color.RGBA{R: 100, G: 220, B: 255, A: alpha})
}
