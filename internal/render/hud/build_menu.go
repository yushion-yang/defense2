// build_menu.go — Build tower popup panel.
// Popup grid layout with tower cards, role tags, sprite previews, and hover tooltip.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// BuildCardVM 建造菜单中单个塔卡片的展示数据（纯值类型）。
type BuildCardVM struct {
	Key         string
	Label       string
	Cost        int
	Damage      float64
	AttackSpeed float64
	Range       float64
	RoleTag     string         // 预计算的角色标签："输出·减速"/"辅助·光环"/...
	RoleColor   color.RGBA     // 角色标签颜色
	TypeIcon    string         // 塔类型图标名："tower-freeze"/""
	Sprite      *ebiten.Image  // 预加载的精灵图
	Buildable   bool           // true=可建造, false=仅展示变体
	AbilityDesc string         // 变体卡的能力描述文本
}

// BuildMenuData holds the runtime data the build menu needs to render.
type BuildMenuData struct {
	Cards          []BuildCardVM // 可建造卡 + 变体展示卡
	BuildableCount int           // 前 N 张为可建造卡，之后为展示卡
	SelectedIdx    int
	Gold           int
	HoverIdx       int
	Visible        bool
}

// Build panel constants
const (
	bpCols     = 5     // cards per row
	bpCardW    = float32(100)
	bpCardH    = float32(80)
	bpCardGap  = float32(8)
	bpCardR    = float32(8)
	bpPadX     = float32(16)
	bpPadY     = float32(12)
	bpHeaderH  = float32(32)
	bpTitleH   = float32(28)
)

// buildPanelMetrics computes the panel geometry from the card count.
type buildPanelMetrics struct {
	panelX, panelY, panelW, panelH float32
	gridX, gridY                    float32
	rows                            int
}

func calcBuildPanelMetrics(count int) buildPanelMetrics {
	rows := (count + bpCols - 1) / bpCols
	cols := bpCols
	if count < cols {
		cols = count
	}

	gridW := float32(cols)*bpCardW + float32(cols-1)*bpCardGap
	gridH := float32(rows)*bpCardH + float32(rows-1)*bpCardGap

	panelW := gridW + bpPadX*2
	panelH := bpTitleH + bpHeaderH + gridH + bpPadY*2

	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)

	return buildPanelMetrics{
		panelX: panelX,
		panelY: panelY,
		panelW: panelW,
		panelH: panelH,
		gridX:  panelX + bpPadX,
		gridY:  panelY + bpTitleH + bpHeaderH,
		rows:   rows,
	}
}

// DrawBuildMenu renders the build panel popup.
func DrawBuildMenu(screen *ebiten.Image, d BuildMenuData) {
	if !d.Visible || len(d.Cards) == 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	m := calcBuildPanelMetrics(len(d.Cards))

	draw.RoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.CenterPanelRadius), theme.PanelBg)
	draw.StrokeRoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.CenterPanelRadius), 1, theme.PanelBorder)

	titleX := float64(m.panelX) + float64(bpPadX)
	titleY := float64(m.panelY) + 8
	fm.DrawBoldText(screen, "建造炮塔", titleX, titleY, theme.FontLG, theme.TextTitle)

	closeX := float64(m.panelX) + float64(m.panelW) - float64(bpPadX) - 40
	fm.DrawText(screen, "关闭", closeX, titleY+2, theme.FontMD, theme.TextMuted)

	for i, card := range d.Cards {
		col := i % bpCols
		row := i / bpCols
		cx := m.gridX + float32(col)*(bpCardW+bpCardGap)
		cy := m.gridY + float32(row)*(bpCardH+bpCardGap)
		hovered := i == d.HoverIdx

		if card.Buildable {
			drawBuildableCard(screen, fm, card, d, i, cx, cy, hovered)
		} else {
			drawVariantCard(screen, fm, card, cx, cy, hovered)
		}
	}

	// Hover tooltip
	if d.HoverIdx >= 0 && d.HoverIdx < len(d.Cards) {
		drawBuildCardTooltip(screen, fm, d.Cards[d.HoverIdx], m)
	}
}

// drawBuildableCard renders a normal buildable tower card.
func drawBuildableCard(screen *ebiten.Image, fm *render.FontManager, card BuildCardVM, d BuildMenuData, i int, cx, cy float32, hovered bool) {
	affordable := d.Gold >= card.Cost
	selected := i == d.SelectedIdx

	cardBg := theme.BuildCardNormal
	if selected {
		cardBg = theme.BuildCardSelected
	} else if hovered {
		cardBg = color.RGBA{R: 35, G: 45, B: 70, A: 240}
	}
	if !affordable {
		cardBg.A = cardBg.A / 2
	}
	draw.RoundRect(screen, cx, cy, bpCardW, bpCardH, bpCardR, cardBg)

	if selected {
		draw.StrokeRoundRect(screen, cx, cy, bpCardW, bpCardH, bpCardR, 2, theme.BuildCardSelBorder)
	}

	nameX := float64(cx) + 8
	nameY := float64(cy) + 6
	fm.DrawBoldText(screen, card.Label, nameX, nameY, theme.FontMD, color.White)

	costTxt := fmt.Sprintf("%dG", card.Cost)
	costClr := theme.BuildCostColor
	if !affordable {
		costClr = color.RGBA{R: 200, G: 80, B: 80, A: 200}
	}
	fm.DrawText(screen, costTxt, nameX, nameY+16, theme.FontSM, costClr)

	// Tower-type icon badge
	if card.TypeIcon != "" {
		if im := render.GlobalIcons(); im != nil {
			if img := im.Get(card.TypeIcon); img != nil {
				badgeX := float64(cx) + float64(bpCardW) - 16
				badgeY := float64(cy) + 4
				draw.Sprite(screen, img, badgeX, badgeY+6, 12)
			}
		}
	}

	// Role tag
	fm.DrawText(screen, card.RoleTag, nameX, float64(cy)+float64(bpCardH)-16, theme.FontXS, card.RoleColor)

	// Sprite preview
	if card.Sprite != nil {
		spriteX := float64(cx) + float64(bpCardW) - 26
		spriteY := float64(cy) + float64(bpCardH)/2
		draw.Sprite(screen, card.Sprite, spriteX, spriteY, 36)
	}
}

// drawVariantCard renders a display-only attack style variant card.
func drawVariantCard(screen *ebiten.Image, fm *render.FontManager, card BuildCardVM, cx, cy float32, hovered bool) {
	cardBg := theme.BuildCardVariant
	if hovered {
		cardBg = color.RGBA{R: 40, G: 55, B: 80, A: 160}
	}
	draw.RoundRect(screen, cx, cy, bpCardW, bpCardH, bpCardR, cardBg)

	// Subtle border
	draw.StrokeRoundRect(screen, cx, cy, bpCardW, bpCardH, bpCardR, 1,
		color.RGBA{R: 100, G: 120, B: 160, A: 80})

	nameX := float64(cx) + 8
	nameY := float64(cy) + 6

	// Tower name (bold)
	fm.DrawBoldText(screen, card.Label, nameX, nameY, theme.FontMD,
		color.RGBA{R: 180, G: 200, B: 230, A: 220})

	// Ability type label (short, e.g. "弹射", "散射")
	fm.DrawText(screen, card.RoleTag, nameX, nameY+18, theme.FontXS, card.RoleColor)

	// "选择能力后" tag at bottom-left
	fm.DrawText(screen, "选择能力后", nameX, float64(cy)+float64(bpCardH)-16, theme.FontXS,
		color.RGBA{R: 120, G: 140, B: 170, A: 140})

	// Sprite preview (right side, same as buildable cards)
	if card.Sprite != nil {
		spriteX := float64(cx) + float64(bpCardW) - 26
		spriteY := float64(cy) + float64(bpCardH)/2
		draw.Sprite(screen, card.Sprite, spriteX, spriteY, 36)
	}
}

// drawBuildCardTooltip renders a small stats tooltip above the build panel.
func drawBuildCardTooltip(screen *ebiten.Image, fm *render.FontManager, card BuildCardVM, m buildPanelMetrics) {
	if !card.Buildable {
		drawVariantTooltip(screen, fm, card, m)
		return
	}

	const (
		tipW = float32(240)
		tipH = float32(70)
		tipR = float32(8)
	)
	tipX := (float32(theme.CanvasW) - tipW) / 2
	tipY := m.panelY - tipH - 6

	draw.RoundRect(screen, tipX, tipY, tipW, tipH, tipR, theme.PanelBg)
	draw.StrokeRoundRect(screen, tipX, tipY, tipW, tipH, tipR, 1, theme.PanelBorder)

	tx := float64(tipX) + 12
	ty := float64(tipY) + 8

	fm.DrawBoldText(screen, card.Label, tx, ty, theme.FontLG, theme.TextTitle)
	fm.DrawText(screen, fmt.Sprintf("%s · %dG", card.RoleTag, card.Cost), tx, ty+18, theme.FontSM, card.RoleColor)

	ty += 38
	im := render.GlobalIcons()
	const tipIconSz = 12.0
	const tipIconGap = 4.0

	drawStatIcon(screen, im, "stat-damage", tx, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.0f", card.Damage), tx+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrDamage)

	drawStatIcon(screen, im, "stat-atkspd", tx+60, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.2fs", 1.0/card.AttackSpeed), tx+60+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrAtkSpd)

	drawStatIcon(screen, im, "stat-range", tx+140, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.0f", card.Range), tx+140+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrRange)
}

// drawVariantTooltip renders a tooltip for display-only variant cards.
func drawVariantTooltip(screen *ebiten.Image, fm *render.FontManager, card BuildCardVM, m buildPanelMetrics) {
	const (
		tipW = float32(300)
		tipH = float32(52)
		tipR = float32(8)
	)
	tipX := (float32(theme.CanvasW) - tipW) / 2
	tipY := m.panelY - tipH - 6

	draw.RoundRect(screen, tipX, tipY, tipW, tipH, tipR, theme.PanelBg)
	draw.StrokeRoundRect(screen, tipX, tipY, tipW, tipH, tipR, 1, theme.PanelBorder)

	tx := float64(tipX) + 12
	ty := float64(tipY) + 8

	fm.DrawBoldText(screen, card.Label, tx, ty, theme.FontLG, theme.TextTitle)
	fm.DrawText(screen, card.AbilityDesc, tx, ty+20, theme.FontSM, theme.TextMuted)
}

// BuildMenuHitTest returns the buildable tower card index hit by (px, py), or -1.
// totalCount is total cards (for panel sizing), buildableCount is the clickable subset.
// Returns -2 for close button, -1 for miss/non-buildable card.
func BuildMenuHitTest(px, py float32, totalCount, buildableCount int) int {
	if totalCount <= 0 {
		return -1
	}

	m := calcBuildPanelMetrics(totalCount)

	// Quick panel bounds check
	if px < m.panelX || px > m.panelX+m.panelW || py < m.panelY || py > m.panelY+m.panelH {
		return -1
	}

	// Close button area (top-right corner)
	closeX := m.panelX + m.panelW - bpPadX - 40
	closeY := m.panelY + 4
	if px >= closeX && px <= closeX+50 && py >= closeY && py <= closeY+24 {
		return -2 // close signal
	}

	// Card hit test — only buildable cards are clickable
	for i := 0; i < buildableCount; i++ {
		col := i % bpCols
		row := i / bpCols
		cx := m.gridX + float32(col)*(bpCardW+bpCardGap)
		cy := m.gridY + float32(row)*(bpCardH+bpCardGap)
		if px >= cx && px <= cx+bpCardW && py >= cy && py <= cy+bpCardH {
			return i
		}
	}
	return -1
}

// BuildMenuHoverTest returns the card index the mouse is hovering over (all cards, including variants).
func BuildMenuHoverTest(px, py float32, count int) int {
	if count <= 0 {
		return -1
	}
	m := calcBuildPanelMetrics(count)

	for i := 0; i < count; i++ {
		col := i % bpCols
		row := i / bpCols
		cx := m.gridX + float32(col)*(bpCardW+bpCardGap)
		cy := m.gridY + float32(row)*(bpCardH+bpCardGap)
		if px >= cx && px <= cx+bpCardW && py >= cy && py <= cy+bpCardH {
			return i
		}
	}
	return -1
}
