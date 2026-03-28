// draw_hero.go — hero rendering.
// Themed body with stroke, dashed leash circle, FontManager level label,
// and pill-shaped XP bar.
package render

import (
	"fmt"

	"defense2/internal/core/hero"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawHero renders the hero entity.
func DrawHero(screen *ebiten.Image, h *hero.Hero) {
	if h == nil || !h.Active {
		return
	}
	cx := float32(h.X)
	cy := float32(h.Y)
	r := float32(h.BodyRadius)

	// --- Leash circle (dashed) ---
	leashClr := theme.HeroLeashDef
	if h.State == hero.StateEngage {
		leashClr = theme.HeroLeashSel
	}
	draw.DashedCircle(screen, float32(h.BaseX), float32(h.BaseY),
		float32(h.LeashRadius), 1, 8, 8, leashClr)

	// --- Body: stroke ring + filled circle ---
	draw.FilledCircle(screen, cx, cy, r+2, theme.HeroStroke)
	draw.FilledCircle(screen, cx, cy, r, theme.HeroBodyFB)

	// --- Level label via FontManager ---
	if fm := GlobalFont(); fm != nil {
		label := fmt.Sprintf("Lv%d", h.Level)
		fm.DrawCenteredText(screen, label,
			float64(cx), float64(cy-r-16), theme.FontXS, theme.TextTitle)
	}

	// --- XP bar (pill shape) ---
	if h.Level < 10 && h.XPToNext > 0 {
		barW := r * 2.5
		barH := float32(4)
		barX := cx - barW/2
		barY := cy - r - 6

		// Background pill
		draw.Pill(screen, barX, barY, barW, barH, theme.HeroXPBarBg)

		// Fill pill
		ratio := float32(h.XP) / float32(h.XPToNext)
		if ratio > 1 {
			ratio = 1
		}
		fillW := barW * ratio
		if fillW > 0 {
			draw.Pill(screen, barX, barY, fillW, barH, theme.HeroXPBarFill)
		}
	}
}
