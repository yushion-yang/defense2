// draw_enemy.go — enemy rendering.
// Uses PNG sprites with fallback circles. Themed HP bar, boss pulsing rings,
// runner/tank/flying visuals, and status effect dots.
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/render/anim"
	"defense2/internal/render/draw"
	"defense2/internal/render/sprite"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// EnemyRenderer manages enemy PNG sprite rendering with optional frame animation.
type EnemyRenderer struct {
	cache     *sprite.Cache
	assetFS   AssetReader
	animators map[string]*anim.Animator // per archetype, lazy initialized
}

// NewEnemyRenderer creates an enemy renderer.
func NewEnemyRenderer(assetFS AssetReader) *EnemyRenderer {
	return &EnemyRenderer{
		cache:     sprite.NewCache(),
		assetFS:   assetFS,
		animators: make(map[string]*anim.Animator),
	}
}

const enemySpriteSize = 24

// DrawEnemies renders all alive enemies.
func (er *EnemyRenderer) DrawEnemies(screen *ebiten.Image, pool *enemy.Pool, animTime float64) {
	pool.Each(func(e *enemy.Enemy) {
		cx := float32(e.X)
		cy := float32(e.Y)

		// --- Dying animation: fade + shrink + float upward ---
		if e.IsDying() {
			progress := 1.0 - e.DyingTimer/e.DyingDuration // 0→1 (0=just died, 1=gone)
			scale := 1.0 - progress                         // shrink from 1 to 0
			alpha := float32(1.0 - progress)                // fade from 1 to 0
			offsetY := -progress * 8                         // float up 8px

			img := er.getEnemyFrame(e, 1.0/60.0)
			if img != nil {
				displaySize := float64(enemySpriteSize) * scale
				if displaySize < 0.5 {
					return // too small to see
				}
				w := float64(img.Bounds().Dx())
				h := float64(img.Bounds().Dy())
				s := displaySize / w * draw.Scale
				var op ebiten.DrawImageOptions
				op.GeoM.Translate(-w/2, -h/2)
				op.GeoM.Scale(s, s)
				op.GeoM.Translate(float64(cx)*draw.Scale, (float64(cy)+offsetY)*draw.Scale)
				op.ColorScale.ScaleAlpha(alpha)
				screen.DrawImage(img, &op)
			}
			return // skip normal rendering for dying enemies
		}

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

		// --- Enemy body (animated or static) ---
		img := er.getEnemyFrame(e, 1.0/60.0)
		if img != nil {
			// 行走摆动：用 X 位置作为相位，产生微小的上下浮动和旋转
			wobblePhase := e.X*0.05 + animTime*4
			wobbleY := math.Sin(wobblePhase) * 1.5
			wobbleRot := math.Sin(wobblePhase) * 0.05 // ~3 degrees
			if e.StunTimer > 0 || e.RootTimer > 0 {
				wobbleY = 0
				wobbleRot = 0
			}
			draw.SpriteRotated(screen, img, float64(cx), float64(cy), enemySpriteSize, wobbleRot, wobbleY)
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
			draw.FilledRect(screen, cx-size/2, cy-size/2, size, size,
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

		// --- Shield bar (above HP bar, 4px gap) ---
		if e.ShieldHP > 0 {
			shieldH := float32(theme.EnemyShieldBarH)
			shieldX := cx - barW/2
			shieldY := cy - barOffY - shieldH - 4
			shieldRatio := float32(e.ShieldHP / e.MaxHP)
			if shieldRatio > 1 {
				shieldRatio = 1
			}
			// Shield border + bg + fill
			draw.FilledRect(screen, shieldX-1, shieldY-1, barW+2, shieldH+2,
				theme.EnemyHPBarBorder, true)
			draw.FilledRect(screen, shieldX, shieldY, barW, shieldH,
				theme.EnemyHPBarBg, true)
			draw.FilledRect(screen, shieldX, shieldY, barW*shieldRatio, shieldH,
				color.RGBA{R: 255, G: 255, B: 255, A: 220}, true)
		}

		// --- HP bar (always visible) ---
		{
			barX := cx - barW/2
			barY := cy - barOffY

			// Border (#0f172a)
			draw.FilledRect(screen, barX-1, barY-1, barW+2, barH+2,
				theme.EnemyHPBarBorder, true)
			// Background (#1e293b)
			draw.FilledRect(screen, barX, barY, barW, barH,
				theme.EnemyHPBarBg, true)

			// Damage trail (orange, behind HP fill)
			if e.DisplayHP > e.HP && e.DisplayHP > 0 {
				trailRatio := float32(e.DisplayHP / e.MaxHP)
				if trailRatio > 1 {
					trailRatio = 1
				}
				draw.FilledRect(screen, barX, barY, barW*trailRatio, barH,
					color.RGBA{R: 251, G: 146, B: 60, A: 255}, true) // #fb923c
			}

			// HP fill (red-only color scheme)
			ratio := float32(e.HP / e.MaxHP)
			if ratio < 0 {
				ratio = 0
			}
			fillW := barW * ratio

			var fillClr color.RGBA
			switch {
			case ratio > 0.6:
				fillClr = color.RGBA{R: 239, G: 68, B: 68, A: 255}  // #ef4444 bright red
			case ratio > 0.3:
				fillClr = color.RGBA{R: 220, G: 38, B: 38, A: 255}  // #dc2626 darker red
			default:
				fillClr = color.RGBA{R: 153, G: 27, B: 27, A: 255}  // #991b1b deep red
			}
			draw.FilledRect(screen, barX, barY, fillW, barH, fillClr, true)

			// Boss HP segment dividers (5 segments, 20% each)
			if e.Boss {
				for i := 1; i < 5; i++ {
					divX := barX + barW*float32(i)*0.2
					draw.FilledRect(screen, divX, barY, 1, barH, theme.EnemyHPSegDiv, true)
				}
			}

			// Elite center tick (single 50% divider)
			if e.Elite && !e.Boss {
				divX := barX + barW*0.5
				draw.FilledRect(screen, divX, barY, 1, barH,
					color.RGBA{R: 15, G: 23, B: 42, A: 128}, true)
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

// GetSprite 按原型名加载敌人静态精灵（供造怪菜单等外部使用）。
func (er *EnemyRenderer) GetSprite(archetype string) *ebiten.Image {
	if er.assetFS == nil || archetype == "" {
		return nil
	}
	path := fmt.Sprintf("assets/enemies/%s.png", archetype)
	if cached := er.cache.Get(path, enemySpriteSize, enemySpriteSize); cached != nil {
		return cached
	}
	data, err := er.assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, _ := er.cache.GetOrParse(path, data, enemySpriteSize, enemySpriteSize)
	return img
}

// getEnemyFrame returns the current animation frame for an enemy, falling back to static sprite.
func (er *EnemyRenderer) getEnemyFrame(e *enemy.Enemy, dt float64) *ebiten.Image {
	if e.Archetype == "" {
		return nil
	}

	a, ok := er.animators[e.Archetype]
	if !ok {
		a = anim.LoadEnemyAnimator(er.assetFS, e.Archetype)
		er.animators[e.Archetype] = a
	}

	// Choose animation state
	if a.HasAnim("walk") {
		a.Play("walk")
	}
	a.Update(dt)

	img := a.CurrentImage()
	if img != nil {
		return img
	}
	return er.loadEnemyImage(e)
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
