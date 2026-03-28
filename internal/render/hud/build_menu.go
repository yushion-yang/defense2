// build_menu.go — Build tower popup panel.
// Popup grid layout with tower cards, role tags, sprite previews, and hover tooltip.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/core/tower"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// BuildMenuData holds the runtime data the build menu needs to render.
type BuildMenuData struct {
	TowerDefs   []tower.TowerDef               // buildable tower types
	SelectedIdx int                             // currently selected tower index
	Gold        int                             // player's current gold
	HoverIdx    int                             // mouse hover index (-1 = none)
	SpriteFunc  func(key string) *ebiten.Image  // optional: returns tower sprite by key
	Visible     bool                            // whether the build panel is open
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

// towerTypeIcon returns the tower-type icon name for the given tower key.
func towerTypeIcon(key string) string {
	switch key {
	case "freeze":
		return "tower-freeze"
	case "electric":
		return "tower-electric"
	case "hunter":
		return "tower-hunter"
	case "laser":
		return "tower-laser"
	default:
		return ""
	}
}

// towerRoleTags returns a role description for a tower based on its abilities.
func towerRoleTags(def tower.TowerDef) (string, color.RGBA) {
	for _, ab := range def.Abilities {
		switch ab {
		case "onHitSlow":
			return "控制·减速", color.RGBA{R: 80, G: 180, B: 220, A: 255}
		case "stun":
			return "输出·眩晕", color.RGBA{R: 180, G: 120, B: 220, A: 255}
		case "bounce":
			return "输出·连锁", color.RGBA{R: 220, G: 180, B: 80, A: 255}
		case "splash":
			return "输出·溅射", color.RGBA{R: 220, G: 120, B: 80, A: 255}
		case "bleedDot", "burn":
			return "输出·持续", color.RGBA{R: 220, G: 80, B: 80, A: 255}
		case "executionBonus", "percentHpDamage":
			return "输出·斩杀", color.RGBA{R: 180, G: 60, B: 60, A: 255}
		case "damageUpAura", "attackSpeedAura":
			return "辅助·光环", color.RGBA{R: 80, G: 200, B: 120, A: 255}
		case "poisonZone", "silenceZone":
			return "控制·区域", color.RGBA{R: 100, G: 160, B: 200, A: 255}
		case "goldOnKill", "goldPassive":
			return "经济", color.RGBA{R: 220, G: 200, B: 80, A: 255}
		}
	}
	return "输出", color.RGBA{R: 200, G: 200, B: 200, A: 200}
}

// DrawBuildMenu renders the build panel popup.
func DrawBuildMenu(screen *ebiten.Image, d BuildMenuData) {
	if !d.Visible || len(d.TowerDefs) == 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	m := calcBuildPanelMetrics(len(d.TowerDefs))

	// Panel background + border
	draw.RoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.CenterPanelRadius), theme.PanelBg)
	draw.StrokeRoundRect(screen, m.panelX, m.panelY, m.panelW, m.panelH,
		float32(theme.CenterPanelRadius), 1, theme.PanelBorder)

	// Title: "建造炮塔"
	titleX := float64(m.panelX) + float64(bpPadX)
	titleY := float64(m.panelY) + 8
	fm.DrawBoldText(screen, "建造炮塔", titleX, titleY, theme.FontLG, theme.TextTitle)

	// Close button (right side)
	closeX := float64(m.panelX) + float64(m.panelW) - float64(bpPadX) - 40
	fm.DrawText(screen, "关闭", closeX, titleY+2, theme.FontMD, theme.TextMuted)

	// Tower cards grid
	for i, def := range d.TowerDefs {
		col := i % bpCols
		row := i / bpCols
		cx := m.gridX + float32(col)*(bpCardW+bpCardGap)
		cy := m.gridY + float32(row)*(bpCardH+bpCardGap)
		affordable := d.Gold >= def.Cost
		selected := i == d.SelectedIdx
		hovered := i == d.HoverIdx

		// Card background
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

		// Selection border
		if selected {
			draw.StrokeRoundRect(screen, cx, cy, bpCardW, bpCardH, bpCardR, 2, theme.BuildCardSelBorder)
		}

		// Tower name (left) + cost (left below)
		nameX := float64(cx) + 8
		nameY := float64(cy) + 6
		fm.DrawBoldText(screen, def.Label, nameX, nameY, theme.FontMD, color.White)

		costTxt := fmt.Sprintf("%dG", def.Cost)
		costClr := theme.BuildCostColor
		if !affordable {
			costClr = color.RGBA{R: 200, G: 80, B: 80, A: 200}
		}
		fm.DrawText(screen, costTxt, nameX, nameY+16, theme.FontSM, costClr)

		// Tower-type icon badge (top-right corner)
		if towerIcon := towerTypeIcon(def.Key); towerIcon != "" {
			if im := render.GlobalIcons(); im != nil {
				if img := im.Get(towerIcon); img != nil {
					badgeX := float64(cx) + float64(bpCardW) - 16
					badgeY := float64(cy) + 4
					draw.Sprite(screen, img, badgeX, badgeY+6, 12)
				}
			}
		}

		// Role tag
		role, roleClr := towerRoleTags(def)
		fm.DrawText(screen, role, nameX, float64(cy)+float64(bpCardH)-16, theme.FontXS, roleClr)

		// Sprite preview (right side of card)
		if d.SpriteFunc != nil {
			if img := d.SpriteFunc(def.Key); img != nil {
				spriteX := float64(cx) + float64(bpCardW) - 26
				spriteY := float64(cy) + float64(bpCardH)/2
				draw.Sprite(screen, img, spriteX, spriteY, 36)
			}
		}
	}

	// Hover tooltip
	if d.HoverIdx >= 0 && d.HoverIdx < len(d.TowerDefs) {
		drawBuildTooltip(screen, fm, d.TowerDefs[d.HoverIdx], m)
	}
}

// drawBuildTooltip renders a small stats tooltip above the build panel.
func drawBuildTooltip(screen *ebiten.Image, fm *render.FontManager, def tower.TowerDef, m buildPanelMetrics) {
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

	// Tower name + cost
	role, roleClr := towerRoleTags(def)
	fm.DrawBoldText(screen, def.Label, tx, ty, theme.FontLG, theme.TextTitle)
	fm.DrawText(screen, fmt.Sprintf("%s · %dG", role, def.Cost), tx, ty+18, theme.FontSM, roleClr)

	// Stats row with icons
	ty += 38
	im := render.GlobalIcons()
	const tipIconSz = 12.0
	const tipIconGap = 4.0

	drawStatIcon(screen, im, "stat-damage", tx, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.0f", def.Damage), tx+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrDamage)

	drawStatIcon(screen, im, "stat-atkspd", tx+60, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.2fs", 1.0/def.AttackSpeed), tx+60+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrAtkSpd)

	drawStatIcon(screen, im, "stat-range", tx+140, ty, tipIconSz)
	fm.DrawText(screen, fmt.Sprintf("%.0f", def.Range), tx+140+tipIconSz+tipIconGap, ty, theme.FontSM, theme.InfoAttrRange)
}

// BuildMenuHitTest returns the tower card index hit by (px, py), or -1.
// Also handles close button detection (returns -2 for close).
func BuildMenuHitTest(px, py float32, count int) int {
	if count <= 0 {
		return -1
	}

	m := calcBuildPanelMetrics(count)

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

	// Card hit test
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

// BuildMenuHoverTest returns the tower card index the mouse is hovering over.
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
