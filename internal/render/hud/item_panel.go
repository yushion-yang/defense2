// item_panel.go — Item popup panel (2x3 grid), positioned above ActionBar.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ItemCardVM is the view-model for a single item card (no core imports).
type ItemCardVM struct {
	Name  string
	Count int
	Color color.RGBA
	Kind  int // item.Kind as int (avoid importing item in hud)
}

// ItemPanelData holds the runtime data the item panel needs to render.
type ItemPanelData struct {
	Cards   []ItemCardVM
	Visible bool
}

// Item panel layout constants
const (
	ipCols    = 2
	ipRows    = 3
	ipCardW   = float32(110)
	ipCardH   = float32(50)
	ipCardGap = float32(8)
	ipCardR   = float32(8)
	ipPadX    = float32(14)
	ipPadY    = float32(12)
	ipTitleH  = float32(28)

	// ActionBar height — defined locally since theme.ActionBarH may not exist yet.
	actionBarH = float32(40)
)

// itemPanelMetrics computes the panel geometry.
type itemPanelMetrics struct {
	panelX, panelY, panelW, panelH float32
	gridX, gridY                    float32
}

func calcItemPanelMetrics() itemPanelMetrics {
	gridW := float32(ipCols)*ipCardW + float32(ipCols-1)*ipCardGap
	gridH := float32(ipRows)*ipCardH + float32(ipRows-1)*ipCardGap

	panelW := gridW + ipPadX*2
	panelH := ipTitleH + gridH + ipPadY*2

	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - float32(theme.BottomMargin) - actionBarH - panelH - 4

	return itemPanelMetrics{
		panelX: panelX,
		panelY: panelY,
		panelW: panelW,
		panelH: panelH,
		gridX:  panelX + ipPadX,
		gridY:  panelY + ipTitleH + ipPadY,
	}
}

// DrawItemPanel renders the item popup panel if visible.
func DrawItemPanel(screen *ebiten.Image, d ItemPanelData) {
	if !d.Visible || len(d.Cards) == 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	m := calcItemPanelMetrics()

	// Panel background
	draw.RoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.PanelRadius), theme.PanelBg)
	draw.StrokeRoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.PanelRadius), 1, theme.PanelBorder)

	// Title
	titleX := float64(m.panelX) + float64(ipPadX)
	titleY := float64(m.panelY) + 6
	fm.DrawBoldText(screen, "道具", titleX, titleY, theme.FontLG, theme.TextTitle)

	// Cards
	for i, card := range d.Cards {
		if i >= ipCols*ipRows {
			break
		}
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)
		drawItemCard(screen, fm, card, cx, cy)
	}
}

// drawItemCard renders a single item card.
func drawItemCard(screen *ebiten.Image, fm *render.FontManager, card ItemCardVM, cx, cy float32) {
	available := card.Count > 0

	// Card background
	cardBg := color.RGBA{R: 30, G: 40, B: 65, A: 220}
	if !available {
		cardBg.A = 120
	}
	draw.RoundRect(screen, cx, cy, ipCardW, ipCardH, ipCardR, cardBg)

	// Colored circle indicator (left side)
	circleR := float32(8)
	circleCX := cx + 16
	circleCY := cy + ipCardH/2
	circleClr := card.Color
	if !available {
		circleClr.A = 80
	}
	draw.FilledCircle(screen, circleCX, circleCY, circleR, circleClr)

	// Name text
	textX := float64(circleCX) + float64(circleR) + 8
	textY := float64(cy) + 10
	nameClr := color.RGBA{R: 220, G: 230, B: 245, A: 255}
	if !available {
		nameClr = theme.TextLocked
	}
	fm.DrawText(screen, card.Name, textX, textY, theme.FontSM, nameClr)

	// Count text "xN"
	countTxt := fmt.Sprintf("x%d", card.Count)
	countClr := color.RGBA{R: 180, G: 200, B: 230, A: 200}
	if !available {
		countClr = theme.TextLocked
	}
	fm.DrawText(screen, countTxt, textX, textY+16, theme.FontXS, countClr)
}

// ItemPanelHitTest returns the card index hit by (px, py), or -1.
// Only returns cards with count > 0.
func ItemPanelHitTest(px, py float32, cards []ItemCardVM) int {
	if len(cards) == 0 {
		return -1
	}

	m := calcItemPanelMetrics()

	// Quick panel bounds check
	if px < m.panelX || px > m.panelX+m.panelW || py < m.panelY || py > m.panelY+m.panelH {
		return -1
	}

	count := len(cards)
	if count > ipCols*ipRows {
		count = ipCols * ipRows
	}

	for i := 0; i < count; i++ {
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)
		if px >= cx && px <= cx+ipCardW && py >= cy && py <= cy+ipCardH {
			if cards[i].Count > 0 {
				return i
			}
			return -1
		}
	}
	return -1
}

// ItemPanelContains returns true if (px, py) is inside the item panel area.
func ItemPanelContains(px, py float32) bool {
	m := calcItemPanelMetrics()
	return px >= m.panelX && px <= m.panelX+m.panelW &&
		py >= m.panelY && py <= m.panelY+m.panelH
}

// DrawDragItem renders a floating item at cursor position during drag.
func DrawDragItem(screen *ebiten.Image, x, y float32, clr color.RGBA, name string) {
	fm := render.GlobalFont()

	// Filled circle
	draw.FilledCircle(screen, x, y, 14, clr)
	// White outline
	draw.CircleOutline(screen, x, y, 14, 2, color.RGBA{R: 255, G: 255, B: 255, A: 220})

	// Name text below
	if fm != nil {
		fm.DrawText(screen, name, float64(x)-20, float64(y)+18, theme.FontXS, color.White)
	}
}
