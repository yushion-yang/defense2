// build_menu.go — Card-style build menu at the bottom of the screen.
// Shows tower cards with selection highlight and affordability indicators.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/core/tower"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BuildMenuData holds the runtime data the build menu needs to render.
type BuildMenuData struct {
	TowerDefs   []tower.TowerDef // buildable tower types
	SelectedIdx int              // currently selected tower index
	Gold        int              // player's current gold
}

// buildMenuMetrics computes the panel geometry from the card count.
type buildMenuMetrics struct {
	panelX, panelY, panelW, panelH float32
	cardStartX, cardStartY         float32
}

func calcBuildMenuMetrics(cardCount int) buildMenuMetrics {
	const (
		cardW   = float32(theme.BuildCardW)
		cardH   = float32(theme.BuildCardH)
		cardGap = float32(theme.BuildCardGap)
		pad     = float32(8)
		radius  = float32(theme.BuildMenuRadius)
	)

	n := float32(cardCount)
	innerW := n*cardW + (n-1)*cardGap
	panelW := innerW + pad*2
	panelH := cardH + pad*2

	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)

	return buildMenuMetrics{
		panelX:     panelX,
		panelY:     panelY,
		panelW:     panelW,
		panelH:     panelH,
		cardStartX: panelX + pad,
		cardStartY: panelY + pad,
	}
}

// DrawBuildMenu renders the card-style build menu panel.
func DrawBuildMenu(screen *ebiten.Image, d BuildMenuData) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	if len(d.TowerDefs) == 0 {
		return
	}

	m := calcBuildMenuMetrics(len(d.TowerDefs))

	const (
		cardW   = float32(theme.BuildCardW)
		cardH   = float32(theme.BuildCardH)
		cardGap = float32(theme.BuildCardGap)
		cardR   = float32(theme.BuildCardRadius)
	)

	// Panel background
	draw.RoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.BuildMenuRadius), theme.BuildMenuBg)

	for i, def := range d.TowerDefs {
		cx := m.cardStartX + float32(i)*(cardW+cardGap)
		cy := m.cardStartY
		affordable := d.Gold >= def.Cost

		// Card background
		cardBg := theme.BuildCardNormal
		if i == d.SelectedIdx {
			cardBg = theme.BuildCardSelected
		}
		if !affordable {
			// Reduce alpha for unaffordable
			cardBg = color.RGBA{R: cardBg.R, G: cardBg.G, B: cardBg.B, A: cardBg.A / 2}
		}
		draw.RoundRect(screen, cx, cy, cardW, cardH, cardR, cardBg)

		// Selection border
		if i == d.SelectedIdx {
			draw.StrokeRoundRect(screen, cx, cy, cardW, cardH, cardR, 1.5, theme.BuildCardSelBorder)
		}

		// Tower color square (12x12) centered horizontally, in upper part
		squareSize := float32(12)
		squareX := cx + (cardW-squareSize)/2
		squareY := cy + 6
		tClr := color.RGBA{R: def.Color[0], G: def.Color[1], B: def.Color[2], A: 255}
		vector.DrawFilledRect(screen, squareX, squareY, squareSize, squareSize, tClr, false)

		// Tower name centered below icon
		nameCX := float64(cx) + float64(cardW)/2
		nameY := float64(squareY) + float64(squareSize) + 3
		fm.DrawCenteredText(screen, def.Label, nameCX, nameY, theme.FontMD, color.White)

		// Cost centered at bottom
		costTxt := fmt.Sprintf("$%d", def.Cost)
		costY := float64(cy) + float64(cardH) - 14
		fm.DrawCenteredText(screen, costTxt, nameCX, costY, theme.FontSM, theme.BuildCostColor)

		// Keyboard hint in top-right
		keyTxt := fmt.Sprintf("[%d]", i+1)
		keyX := float64(cx) + float64(cardW) - 4
		keyY := float64(cy) + 3
		fm.DrawRightText(screen, keyTxt, keyX, keyY, 9, theme.TextMuted)
	}
}

// BuildMenuHitTest returns the tower card index hit by (px, py), or -1.
func BuildMenuHitTest(px, py float32, count int) int {
	if count <= 0 {
		return -1
	}

	m := calcBuildMenuMetrics(count)

	const (
		cardW   = float32(theme.BuildCardW)
		cardH   = float32(theme.BuildCardH)
		cardGap = float32(theme.BuildCardGap)
	)

	if py < m.cardStartY || py > m.cardStartY+cardH {
		return -1
	}

	for i := 0; i < count; i++ {
		cx := m.cardStartX + float32(i)*(cardW+cardGap)
		if px >= cx && px <= cx+cardW {
			return i
		}
	}
	return -1
}
