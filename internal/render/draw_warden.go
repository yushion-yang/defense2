// draw_warden.go — warden rendering.
// Uses PNG sprites for warden bodies, keeps dynamic effects (shoot lines,
// fireballs, flame trails, AoE strikes, buff connections) as code-rendered.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/warden"
	wardenTypes "defense2/internal/core/warden/types"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

// WardenRenderer manages warden PNG sprite rendering.
type WardenRenderer struct {
	cache   *sprite.Cache
	assetFS AssetReader
}

// NewWardenRenderer creates a warden renderer.
func NewWardenRenderer(assetFS AssetReader) *WardenRenderer {
	return &WardenRenderer{
		cache:   sprite.NewCache(),
		assetFS: assetFS,
	}
}

const wardenSpriteSize = 24 // display size in logical pixels

// loadSprite loads and caches a warden's PNG sprite.
// Convention: assets/wardens/warden-{type}.png
func (wr *WardenRenderer) loadSprite(typ string) *ebiten.Image {
	if wr.assetFS == nil || typ == "" {
		return nil
	}
	path := fmt.Sprintf("assets/wardens/warden-%s.png", typ)
	if cached := wr.cache.Get(path, wardenSpriteSize, wardenSpriteSize); cached != nil {
		return cached
	}
	data, err := wr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := wr.cache.GetOrParse(path, data, wardenSpriteSize, wardenSpriteSize)
	return img
}

// GetSprite returns the cached sprite image for a warden type (for UI preview).
func (wr *WardenRenderer) GetSprite(typ string) *ebiten.Image {
	return wr.loadSprite(typ)
}

// DrawWarden renders a single warden entity.
func (wr *WardenRenderer) DrawWarden(screen *ebiten.Image, w *warden.Warden) {
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

	// Type-specific effects (drawn BEFORE body so body renders on top)
	switch s := w.State.(type) {
	case *wardenTypes.PrinceState:
		drawPrinceEffects(screen, s)
	case *wardenTypes.EnvoyState:
		drawEnvoyEffects(screen, s)
	case *wardenTypes.SkystrikeState:
		drawSkystrikeEffects(screen, s)
	}

	// Body sprite（根据精灵原始朝向校正旋转角度）
	img := wr.loadSprite(w.Type)
	if img != nil {
		rotation := wardenSpriteRotation(w.Type, base.FacingAngle)
		draw.SpriteRotated(screen, img, float64(base.X), float64(base.Y),
			wardenSpriteSize, rotation, 0)
	} else {
		// Fallback: simple colored circle
		draw.FilledCircle(screen, float32(base.X), float32(base.Y), 8,
			wardenShootColor(w.Type))
	}
}

// wardenSpriteRotation 根据精灵原始朝向计算实际旋转角度。
// rotation = facingAngle - spriteNativeAngle
// 对称精灵（chain, envoy）不旋转。
func wardenSpriteRotation(typ string, facingAngle float64) float64 {
	switch typ {
	case "prince", "core":
		// 精灵朝上（-π/2），移动方向 facingAngle=0 时需旋转 +π/2
		return facingAngle + math.Pi/2
	case "skystrike":
		// 精灵朝下（+π/2），移动方向 facingAngle=0 时需旋转 -π/2
		return facingAngle - math.Pi/2
	default:
		// chain, envoy: 对称精灵不旋转
		return 0
	}
}

// wardenShootColor returns the shoot-line color for each warden type.
func wardenShootColor(typ string) color.RGBA {
	switch typ {
	case "prince":
		return color.RGBA{R: 255, G: 140, B: 30, A: 200}
	case "core":
		return color.RGBA{R: 100, G: 180, B: 255, A: 200}
	case "chain":
		return color.RGBA{R: 160, G: 80, B: 255, A: 200}
	case "skystrike":
		return color.RGBA{R: 80, G: 200, B: 255, A: 200}
	case "envoy":
		return color.RGBA{R: 255, G: 200, B: 100, A: 200}
	default:
		return color.RGBA{R: 200, G: 200, B: 200, A: 200}
	}
}

// drawPrinceEffects renders fire trails and fireballs (drawn under body).
func drawPrinceEffects(screen *ebiten.Image, s *wardenTypes.PrinceState) {
	// Flame trails (ground fire)
	for _, t := range s.Trails {
		alpha := uint8(120 * (t.Life / t.MaxLife))
		draw.FilledCircle(screen, float32(t.X), float32(t.Y), float32(t.Radius),
			color.RGBA{R: 255, G: 100, B: 30, A: alpha})
	}

	// Flying fireballs
	for _, fb := range s.Fireballs {
		draw.FilledCircle(screen, float32(fb.X), float32(fb.Y), float32(fb.Radius)*0.8,
			color.RGBA{R: 255, G: 120, B: 20, A: 60})
		draw.FilledCircle(screen, float32(fb.X), float32(fb.Y), float32(fb.Radius)*0.5,
			color.RGBA{R: 255, G: 180, B: 40, A: 230})
	}
}

// drawEnvoyEffects renders buff connection line.
func drawEnvoyEffects(screen *ebiten.Image, s *wardenTypes.EnvoyState) {
	if s.BuffExpiry <= 0 || s.BuffedTower == nil {
		return
	}
	alpha := uint8(120 * (s.BuffExpiry / 4.0))
	if alpha > 120 {
		alpha = 120
	}
	draw.Line(screen, float32(s.X), float32(s.Y),
		float32(s.BuffedTower.X), float32(s.BuffedTower.Y),
		1, color.RGBA{R: 255, G: 200, B: 100, A: alpha}, true)
}

// drawSkystrikeEffects renders AoE strike visual.
func drawSkystrikeEffects(screen *ebiten.Image, s *wardenTypes.SkystrikeState) {
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
