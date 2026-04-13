// item_panel.go — Item popup panel (2x3 grid), positioned above ActionBar.
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ItemCardVM is the view-model for a single item card (no core imports).
type ItemCardVM struct {
	Name  string
	Desc  string // 描述文字（如 "基础伤害+2"）
	Icon  string // 图标名（如 "item-stone"）
	Count int
	Color color.RGBA
	Kind  int // item.Kind as int (avoid importing item in hud)
}

// ItemPanelData holds the runtime data the item panel needs to render.
type ItemPanelData struct {
	Cards    []ItemCardVM
	Visible  bool
	HoverIdx int // 鼠标悬停卡片索引，-1 表示无悬停
}

// Item panel layout constants
const (
	ipCols    = 2
	ipRows    = 3
	ipCardW   = float32(120)
	ipCardH   = float32(58)
	ipCardGap = float32(8)
	ipCardR   = float32(8)
	ipPadX    = float32(14)
	ipPadY    = float32(12)
	ipTitleH  = float32(28)

	actionBarH = float32(theme.ActionBarH)
)

// itemPanelMetrics computes the panel geometry.
type itemPanelMetrics struct {
	panelX, panelY, panelW, panelH float32
	gridX, gridY                   float32
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
	fm.DrawBoldText(screen, i18n.T("hud.item.title"), titleX, titleY, theme.FontLG, theme.TextTitle)

	// Cards
	for i, card := range d.Cards {
		if i >= ipCols*ipRows {
			break
		}
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)
		drawItemCard(screen, fm, card, cx, cy, i == d.HoverIdx)
	}
}

// itemIconFallback maps item Kind (as int) to a fallback stat icon name.
func itemIconFallback(kind int) string {
	switch kind {
	case 0, 1: // BaseDamage, PotentialDamage
		return "stat-damage"
	case 2, 3: // BaseSpeed, PotentialSpeed
		return "stat-atkspd"
	case 4, 5: // BaseRange, PotentialRange
		return "stat-range"
	default:
		return "stat-damage"
	}
}

// resolveItemIcon returns the icon image for a card, trying card.Icon first then fallback.
func resolveItemIcon(card ItemCardVM) *ebiten.Image {
	im := render.GlobalIcons()
	if im == nil {
		return nil
	}
	if card.Icon != "" {
		if img := im.Get(card.Icon); img != nil {
			return img
		}
	}
	return im.Get(itemIconFallback(card.Kind))
}

// drawItemCard renders a single item card.
func drawItemCard(screen *ebiten.Image, fm *render.FontManager, card ItemCardVM, cx, cy float32, hovered bool) {
	available := card.Count > 0

	// Card background
	cardBg := color.RGBA{R: 30, G: 40, B: 65, A: 220}
	if hovered && available {
		cardBg = color.RGBA{R: 40, G: 55, B: 85, A: 240}
	} else if !available {
		cardBg.A = 120
	}
	draw.RoundRect(screen, cx, cy, ipCardW, ipCardH, ipCardR, cardBg)

	// Item icon (left side)
	iconCX := cx + 18
	iconCY := cy + ipCardH/2 - 2
	if img := resolveItemIcon(card); img != nil {
		if available {
			draw.Sprite(screen, img, float64(iconCX), float64(iconCY), 28)
		} else {
			logScale := 28.0 / float64(img.Bounds().Dx())
			draw.SpriteScaledRotatedAlpha(screen, img, float64(iconCX), float64(iconCY), logScale, 0, 0.35)
		}
	}

	// Name text
	textX := float64(iconCX) + 18
	textY := float64(cy) + 6
	nameClr := color.RGBA{R: 220, G: 230, B: 245, A: 255}
	if !available {
		nameClr = theme.TextLocked
	}
	fm.DrawText(screen, card.Name, textX, textY, theme.FontSM, nameClr)

	// Count text "xN"
	countTxt := "x" + strconv.Itoa(card.Count)
	countClr := color.RGBA{R: 180, G: 200, B: 230, A: 200}
	if !available {
		countClr = theme.TextLocked
	}
	fm.DrawText(screen, countTxt, textX+float64(fm.MeasureText(card.Name, theme.FontSM))+4, textY, theme.FontXS, countClr)

	// Description text
	if card.Desc != "" {
		descClr := theme.TextMuted
		if !available {
			descClr = theme.TextLocked
		}
		fm.DrawText(screen, card.Desc, textX, textY+16, theme.FontXS, descClr)
	}
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

// ItemPanelHoverTest returns the card index the mouse is hovering over, or -1.
// Unlike HitTest, includes unavailable cards for visual hover feedback.
func ItemPanelHoverTest(px, py float32, count int) int {
	if count <= 0 {
		return -1
	}
	m := calcItemPanelMetrics()
	if px < m.panelX || px > m.panelX+m.panelW || py < m.panelY || py > m.panelY+m.panelH {
		return -1
	}
	if count > ipCols*ipRows {
		count = ipCols * ipRows
	}
	for i := 0; i < count; i++ {
		col := i % ipCols
		row := i / ipCols
		cx := m.gridX + float32(col)*(ipCardW+ipCardGap)
		cy := m.gridY + float32(row)*(ipCardH+ipCardGap)
		if px >= cx && px <= cx+ipCardW && py >= cy && py <= cy+ipCardH {
			return i
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
func DrawDragItem(screen *ebiten.Image, x, y float32, clr color.RGBA, name string, kind int, icon string) {
	fm := render.GlobalFont()

	// Colored circle background
	draw.FilledCircle(screen, x, y, 18, color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: 150})
	// White outline
	draw.CircleOutline(screen, x, y, 18, 2, color.RGBA{R: 255, G: 255, B: 255, A: 220})
	// Item icon on top
	vm := ItemCardVM{Icon: icon, Kind: kind}
	if img := resolveItemIcon(vm); img != nil {
		draw.Sprite(screen, img, float64(x), float64(y), 28)
	}

	// Name text below
	if fm != nil {
		fm.DrawCenteredText(screen, name, float64(x), float64(y)+22, theme.FontXS, color.White)
	}
}
