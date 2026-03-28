// draw_enemy.go — enemy rendering.
// Uses PNG sprites with fallback circles. Themed HP bar, boss pulsing rings,
// runner/tank/flying visuals, and status effect dots.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// EnemyRenderer manages enemy PNG sprite rendering.
type EnemyRenderer struct {
	cache   *sprite.Cache
	assetFS AssetReader
}

// NewEnemyRenderer creates an enemy renderer.
func NewEnemyRenderer(assetFS AssetReader) *EnemyRenderer {
	return &EnemyRenderer{
		cache:   sprite.NewCache(),
		assetFS: assetFS,
	}
}

const enemySpriteSize = 24

// DrawEnemies renders all alive enemies.
func (er *EnemyRenderer) DrawEnemies(screen *ebiten.Image, pool *enemy.Pool, animTime float64) {
	pool.Each(func(e *enemy.Enemy) {
		cx := float32(e.X)
		cy := float32(e.Y)
		r := float32(e.Radius)

		// --- Flying enemy ground shadow ---
		if e.Archetype == "flying" {
			draw.FilledCircle(screen, cx+2, cy+8, r*1.3, theme.EnemyFlyShadow)
		}

		// --- Boss pulsing rings ---
		if e.Boss {
			innerAlpha := uint8(clampF(float64(theme.EnemyBossInner.A)*
				(0.5+0.5*math.Sin(animTime*2.5)), 0, 255))
			innerClr := color.RGBA{R: theme.EnemyBossInner.R, G: theme.EnemyBossInner.G,
				B: theme.EnemyBossInner.B, A: innerAlpha}
			draw.CircleOutline(screen, cx, cy, r+6, 2, innerClr)

			outerAlpha := uint8(clampF(float64(theme.EnemyBossOuter.A)*
				(0.5+0.5*math.Sin(animTime*2.5+1.5)), 0, 255))
			outerClr := color.RGBA{R: theme.EnemyBossOuter.R, G: theme.EnemyBossOuter.G,
				B: theme.EnemyBossOuter.B, A: outerAlpha}
			draw.CircleOutline(screen, cx, cy, r+10, 1.5, outerClr)
		}

		// --- Runner pulsing ring ---
		if e.Archetype == "runner" {
			pulseAlpha := uint8(clampF(float64(theme.EnemyRunnerPulse.A)*
				(0.5+0.5*math.Sin(animTime*3)), 0, 255))
			pulseClr := color.RGBA{R: theme.EnemyRunnerPulse.R, G: theme.EnemyRunnerPulse.G,
				B: theme.EnemyRunnerPulse.B, A: pulseAlpha}
			draw.CircleOutline(screen, cx, cy, r+4, 1.5, pulseClr)
		}

		// --- Enemy body ---
		img := er.loadEnemyImage(e)
		if img != nil {
			opts := &ebiten.DrawImageOptions{}
			w, h := img.Bounds().Dx(), img.Bounds().Dy()
			scale := float64(enemySpriteSize) / float64(w)
			opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(img, opts)
		} else {
			bodyColor := color.RGBA{R: 200, G: 60, B: 60, A: 255}
			if e.Boss {
				bodyColor = color.RGBA{R: 220, G: 160, B: 40, A: 255}
			}
			draw.FilledCircle(screen, cx, cy, r, bodyColor)
		}

		// --- Tank overlay ---
		if e.Archetype == "tank" {
			size := float32(14)
			vector.DrawFilledRect(screen, cx-size/2, cy-size/2, size, size,
				color.RGBA{R: 255, G: 255, B: 255, A: 40}, true)
		}

		// --- HP bar dimensions ---
		var barW, barH, barOffY float32
		if e.Boss {
			barW = theme.EnemyBossHPBarW
			barH = theme.EnemyBossHPBarH
			barOffY = theme.EnemyBossHPOffsetY
		} else {
			barW = theme.EnemyHPBarW
			barH = theme.EnemyHPBarH
			barOffY = theme.EnemyHPBarOffsetY
		}

		// --- Shield bar (above HP bar) ---
		if e.ShieldHP > 0 {
			shieldH := float32(theme.EnemyShieldBarH)
			shieldX := cx - barW/2
			shieldY := cy - barOffY - shieldH - 1
			shieldRatio := float32(e.ShieldHP / e.MaxHP)
			if shieldRatio > 1 {
				shieldRatio = 1
			}
			vector.DrawFilledRect(screen, shieldX, shieldY, barW*shieldRatio, shieldH,
				color.RGBA{R: 255, G: 255, B: 255, A: 200}, true)
		}

		// --- HP bar (only when damaged) ---
		if e.HP < e.MaxHP {
			barX := cx - barW/2
			barY := cy - barOffY

			// Border
			vector.DrawFilledRect(screen, barX-1, barY-1, barW+2, barH+2,
				theme.EnemyHPBarBorder, true)
			// Background
			vector.DrawFilledRect(screen, barX, barY, barW, barH,
				theme.EnemyHPBarBg, true)

			// HP fill
			ratio := float32(e.HP / e.MaxHP)
			if ratio < 0 {
				ratio = 0
			}
			fillW := barW * ratio

			var fillClr color.RGBA
			switch {
			case ratio > 0.6:
				fillClr = theme.EnemyHPFillHigh
			case ratio > 0.3:
				fillClr = theme.EnemyHPFillMid
			default:
				fillClr = theme.EnemyHPFillLow
			}
			vector.DrawFilledRect(screen, barX, barY, fillW, barH, fillClr, true)

			// Boss HP segment dividers (every 20%)
			if e.Boss {
				for i := 1; i < 5; i++ {
					divX := barX + barW*float32(i)*0.2
					vector.DrawFilledRect(screen, divX, barY, 1, barH, theme.EnemyHPSegDiv, true)
				}
			}
		}

		// --- Status effect dots ---
		dotY := cy - barOffY - 4
		dotX := cx - 8.0
		dotR := float32(2.5)

		if e.SlowTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 125, G: 211, B: 252, A: 235})
			dotX += 6
		}
		if e.StunTimer > 0 || e.RootTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 245, G: 208, B: 254, A: 235})
			dotX += 6
		}
		if e.BleedTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 239, G: 68, B: 68, A: 255})
			dotX += 6
		}
		if e.BurnTimer > 0 {
			draw.FilledCircle(screen, dotX, dotY, dotR,
				color.RGBA{R: 34, G: 197, B: 94, A: 255})
			dotX += 6
		}
	})
}

// loadEnemyImage loads an enemy's PNG sprite.
// Convention: assets/enemies/{archetype}.png
func (er *EnemyRenderer) loadEnemyImage(e *enemy.Enemy) *ebiten.Image {
	if er.assetFS == nil || e.Archetype == "" {
		return nil
	}
	path := fmt.Sprintf("assets/enemies/%s.png", e.Archetype)
	cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize)
	if cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	if err != nil {
		return nil
	}
	return img
}

// clampF clamps a float64 value between min and max.
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
