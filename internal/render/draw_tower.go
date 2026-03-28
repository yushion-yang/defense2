// draw_tower.go — tower rendering.
// Uses PNG sprites (64px), fallback to geometric shapes. Themed selection ring,
// range indicator, name label, and buff dots for the selected tower.
package render

import (
	"fmt"
	"image/color"

	"defense2/internal/core/tower"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TowerRenderer manages tower PNG sprite rendering.
type TowerRenderer struct {
	cache   *sprite.Cache
	assetFS AssetReader
}

// AssetReader reads embedded asset files.
type AssetReader interface {
	ReadFile(name string) ([]byte, error)
}

// NewTowerRenderer creates a tower renderer.
func NewTowerRenderer(assetFS AssetReader) *TowerRenderer {
	return &TowerRenderer{
		cache:   sprite.NewCache(),
		assetFS: assetFS,
	}
}

const towerSpriteSize = 64 // display size in pixels, matching theme.TowerBaseSize

// DrawTowers renders all placed towers.
func (tr *TowerRenderer) DrawTowers(screen *ebiten.Image, pool *tower.Pool, selectedTower *tower.Tower, animTime float64) {
	pool.Each(func(t *tower.Tower) {
		cx := float32(t.X)
		cy := float32(t.Y)
		selected := selectedTower != nil && t == selectedTower

		// --- Selection ring & range indicator (selected tower only) ---
		if selected {
			draw.CircleOutline(screen, cx, cy,
				theme.TowerSelectionRingR, theme.TowerSelectionWidth, theme.TowerSelectionRing)
			draw.FilledCircle(screen, cx, cy, float32(t.Range), theme.TowerRangeFill)
			draw.CircleOutline(screen, cx, cy,
				float32(t.Range), theme.TowerRangeStrokeWidth, theme.TowerRangeStroke)
		}

		// --- Tower body ---
		img := tr.loadTowerImage(t)
		if img != nil {
			opts := &ebiten.DrawImageOptions{}
			w, h := img.Bounds().Dx(), img.Bounds().Dy()
			scale := float64(towerSpriteSize) / float64(w)
			opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(img, opts)
		} else {
			// Fallback: circle body + barrel rectangle
			bodyClr := theme.TowerFallbackDef
			if selected {
				bodyClr = theme.TowerFallbackSel
			}
			draw.FilledCircle(screen, cx, cy, theme.TowerFallbackRadius, bodyClr)
			// Barrel: 8x14 rectangle pointing upward from center
			barrelW := float32(8)
			barrelH := float32(14)
			vector.DrawFilledRect(screen, cx-barrelW/2, cy-barrelH, barrelW, barrelH, theme.TowerBarrel, true)
		}

		// --- Name label ---
		if fm := GlobalFont(); fm != nil {
			fm.DrawCenteredText(screen, t.Label,
				float64(cx), float64(cy)+theme.TowerNameLabelY,
				theme.FontTowerName, theme.TowerNameLabel)
		}

		// Tower struct has no Buffs field — buff dots rendering skipped.
		// When the Buffs field is added, draw colored dots above the tower:
		// offset = -theme.TowerBuffDotBaseY, spacing = theme.TowerBuffDotSpacing, r = theme.TowerBuffDotRadius
	})
}

// loadTowerImage loads a tower's PNG sprite.
// Convention: assets/towers/core/tower-{key}.png
func (tr *TowerRenderer) loadTowerImage(t *tower.Tower) *ebiten.Image {
	if tr.assetFS == nil {
		return nil
	}
	path := fmt.Sprintf("assets/towers/core/tower-%s.png", t.Key)
	cached := tr.cache.Get(path, towerSpriteSize, towerSpriteSize)
	if cached != nil {
		return cached
	}
	data, err := tr.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := tr.cache.GetOrParse(path, data, towerSpriteSize, towerSpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// DrawTowerRangePreview draws a placement preview (range circle + tower shadow).
func DrawTowerRangePreview(screen *ebiten.Image, cx, cy float32, r float64, valid bool) {
	fr := float32(r)

	var fillClr, strokeClr color.RGBA
	if valid {
		fillClr = color.RGBA{R: 34, G: 197, B: 94, A: 77}  // green 0.3 alpha
		strokeClr = color.RGBA{R: 34, G: 197, B: 94, A: 153} // green 0.6 alpha
	} else {
		fillClr = color.RGBA{R: 239, G: 68, B: 68, A: 77}  // red 0.3 alpha
		strokeClr = color.RGBA{R: 239, G: 68, B: 68, A: 153} // red 0.6 alpha
	}

	// Range circle (filled + outline)
	draw.FilledCircle(screen, cx, cy, fr, fillClr)
	draw.CircleOutline(screen, cx, cy, fr, 1.5, strokeClr)

	// Tower shadow circle
	shadowClr := fillClr
	shadowClr.A = 120
	draw.FilledCircle(screen, cx, cy, theme.TowerFallbackRadius, shadowClr)
}
