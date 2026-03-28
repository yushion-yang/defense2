// info_panel.go — Bottom-center tower detail panel.
// Shows tower stats, abilities, and sell button when a tower is selected.
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

// DrawInfoPanel renders the tower information panel. Passing nil hides it.
func DrawInfoPanel(screen *ebiten.Image, t *tower.Tower, sellValue int) {
	if t == nil {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		panelX     = float32(theme.CenterPanelX)
		panelW     = float32(theme.CenterPanelW)
		panelR     = float32(theme.CenterPanelRadius)
		innerPad   = float32(theme.CenterPanelInnerPad)
		titleH     = float32(theme.DetailTitleH)
		attrH      = float32(theme.DetailAttrH)
		abilityH   = float32(theme.DetailRowH)
		btnH       = float32(theme.DetailBtnH)
		topPad     = float32(theme.DetailTopPad)
		botPad     = float32(theme.DetailBotPad)
		detailGap  = float32(theme.DetailGap)
	)

	// Calculate panel height based on content.
	contentH := topPad + titleH + attrH
	if len(t.Abilities) > 0 {
		contentH += detailGap + float32(len(t.Abilities))*abilityH
	}
	contentH += detailGap + btnH + botPad
	panelH := contentH

	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)

	// Panel background + border
	draw.RoundRect(screen, panelX, panelY, panelW, panelH, panelR, theme.PanelBg)
	draw.StrokeRoundRect(screen, panelX, panelY, panelW, panelH, panelR, 1, theme.InfoBorder)

	// Content area
	ix := float64(panelX) + float64(innerPad)
	iy := float64(panelY) + float64(topPad)

	// Title row: tower label
	fm.DrawText(screen, t.Label, ix, iy, theme.FontLG, theme.TextTitle)
	iy += float64(titleH)

	// Attribute row: three columns evenly spaced
	colW := (float64(panelW) - float64(innerPad)*2) / 3

	dmgTxt := fmt.Sprintf("攻击: %.0f", t.Damage)
	fm.DrawText(screen, dmgTxt, ix, iy, theme.FontMD, theme.InfoAttrDamage)

	spdTxt := fmt.Sprintf("攻速: %.2f", t.AttackSpeed)
	fm.DrawText(screen, spdTxt, ix+colW, iy, theme.FontMD, theme.InfoAttrAtkSpd)

	rngTxt := fmt.Sprintf("射程: %.0f", t.Range)
	fm.DrawText(screen, rngTxt, ix+colW*2, iy, theme.FontMD, theme.InfoAttrRange)
	iy += float64(attrH)

	// Ability list
	if len(t.Abilities) > 0 {
		iy += float64(detailGap)
		for _, ab := range t.Abilities {
			fm.DrawText(screen, ab, ix, iy, theme.FontSM, theme.TextBody)
			iy += float64(abilityH)
		}
	}

	// Sell button
	iy += float64(detailGap)
	sellBtnW := panelW - innerPad*2
	sellBtnX := panelX + innerPad
	sellBtnY := float32(iy)
	sellBtnR := float32(theme.ButtonRadius)
	draw.RoundRect(screen, sellBtnX, sellBtnY, sellBtnW, btnH, sellBtnR, theme.BtnDanger)

	sellTxt := fmt.Sprintf("卖出 $%d", sellValue)
	sellCX := float64(sellBtnX) + float64(sellBtnW)/2
	sellCY := float64(sellBtnY) + float64(btnH)/2 - 6
	fm.DrawCenteredText(screen, sellTxt, sellCX, sellCY, theme.FontMD, color.White)
}

// InfoPanelSellHitTest checks if (px, py) hits the sell button.
// Returns true when the panel is visible and the click is on the button.
func InfoPanelSellHitTest(px, py float32, t *tower.Tower) bool {
	if t == nil {
		return false
	}

	const (
		panelX   = float32(theme.CenterPanelX)
		panelW   = float32(theme.CenterPanelW)
		innerPad = float32(theme.CenterPanelInnerPad)
		titleH   = float32(theme.DetailTitleH)
		attrH    = float32(theme.DetailAttrH)
		abilityH = float32(theme.DetailRowH)
		btnH     = float32(theme.DetailBtnH)
		topPad   = float32(theme.DetailTopPad)
		botPad   = float32(theme.DetailBotPad)
		gap      = float32(theme.DetailGap)
	)

	contentH := topPad + titleH + attrH
	if len(t.Abilities) > 0 {
		contentH += gap + float32(len(t.Abilities))*abilityH
	}
	contentH += gap + btnH + botPad
	panelH := contentH
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)

	btnX := panelX + innerPad
	btnY := panelY + contentH - botPad - btnH
	btnW := panelW - innerPad*2

	return px >= btnX && px <= btnX+btnW && py >= btnY && py <= btnY+btnH
}
