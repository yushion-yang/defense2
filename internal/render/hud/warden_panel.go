// warden_panel.go — Bottom-right warden info panel.
// Shows real-time warden stats: name, strength, damage, interval,
// attack pattern, special abilities, and growth info.
// Uses FlexPanel + AnchoredRect for adaptive layout.
package hud

import (
	"fmt"

	"defense2/internal/render"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// WardenPanelData holds pre-formatted display data for the warden info panel.
type WardenPanelData struct {
	Name         string  // e.g. "火灵"
	Type         string  // e.g. "prince", "core", "envoy"
	Strength     float64 // current perceived strength
	PeakStrength float64 // peak (ratchet) strength
	// Display data (pre-formatted by caller).
	Damage      string // e.g. "30"
	Interval    string // e.g. "3s"
	AttackDesc  string // e.g. "冲撞穿透: 向敌群密集方向冲刺..."
	SpecialDesc string // e.g. "火焰轨迹 10 2"
	GrowthDesc  string // e.g. "击杀+1 通波+5"
}

// Warden panel layout constants.
const (
	wardenPanelW       float32 = 350
	wardenPanelRMargin float32 = 20 // right margin from screen edge
	wardenPanelPad     float32 = float32(theme.DetailPad)
)

// DrawWardenPanel renders the warden info panel in the bottom-right area.
func DrawWardenPanel(screen *ebiten.Image, d WardenPanelData) {
	if d.Type == "" {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	// Pre-calculate panel height so AnchoredRect can position it correctly.
	// We use a "dry-run" FlexPanel to accumulate row heights.
	estimatedH := estimateWardenPanelHeight(d)

	// Anchor to bottom-right with margins.
	rect := ui.AnchoredRect(ui.AnchorBottomRight, wardenPanelW, estimatedH,
		0, wardenPanelRMargin, float32(theme.BottomMargin), 0)

	// Build FlexPanel at the anchored position.
	p := ui.NewFlexPanel(rect.X, rect.Y, rect.W, wardenPanelPad)
	p.Radius = float32(theme.PanelRadius)
	p.BgColor = theme.PanelBg
	p.Border = theme.InfoWardenBdr

	// Row 1: Title — "战灵·<Name>" (left) + strength indicator (right).
	p.AddRow(float32(theme.DetailTitleH), func(screen *ebiten.Image, x, y float64, w float64) {
		titleTxt := fmt.Sprintf("战灵·%s", d.Name)
		fm.DrawBoldText(screen, titleTxt, x, y, theme.FontLG, theme.TextTitle)

		strDir := "↓"
		strColor := theme.StatusStrDown
		if d.Strength >= d.PeakStrength && d.PeakStrength > 0 {
			strDir = "↑"
			strColor = theme.StatusStrUp
		} else if d.Strength == 0 && d.PeakStrength == 0 {
			strDir = "↓"
			strColor = theme.StatusStrNorm
		}
		strIndicator := fmt.Sprintf("强度%.0f%s", d.Strength, strDir)
		fm.DrawRightText(screen, strIndicator, x+w, y+2, theme.FontSM, strColor)
	})

	// Row 2: Strength — "强度 X" (left) + "最高 X" (right).
	p.AddRow(float32(theme.DetailAttrH), func(screen *ebiten.Image, x, y float64, w float64) {
		strTxt := fmt.Sprintf("强度 %.0f", d.Strength)
		fm.DrawText(screen, strTxt, x, y, theme.FontSM, theme.StatusWarden)

		peakTxt := fmt.Sprintf("最高 %.0f", d.PeakStrength)
		fm.DrawRightText(screen, peakTxt, x+w, y, theme.FontSM, theme.TextMuted)
	})

	// Row 3: Stats — damage + interval using StatLine.
	if d.Damage != "" || d.Interval != "" {
		p.AddSpace(float32(theme.DetailGap))
		p.AddRow(float32(theme.DetailAttrH), func(screen *ebiten.Image, x, y float64, w float64) {
			var items []ui.StatLineItem
			if d.Damage != "" {
				items = append(items, ui.StatLineItem{
					Label: "↑ ",
					Value: d.Damage,
					Color: theme.InfoAttrDamage,
				})
			}
			if d.Interval != "" {
				items = append(items, ui.StatLineItem{
					Label: "∠ ",
					Value: d.Interval,
					Color: theme.InfoAttrAtkSpd,
				})
			}
			ui.DrawStatLine(screen, x, y, w, items, theme.FontSM, theme.FontSM)
		})
	}

	// Row 4: Attack description.
	if d.AttackDesc != "" {
		p.AddSpace(float32(theme.DetailGap))
		p.AddRow(float32(theme.DetailRowH), func(screen *ebiten.Image, x, y float64, w float64) {
			fm.DrawText(screen, d.AttackDesc, x, y, theme.FontXS, theme.TextBody)
		})
	}

	// Row 5: Special ability.
	if d.SpecialDesc != "" {
		p.AddSpace(float32(theme.DetailGap))
		p.AddRow(float32(theme.DetailRowH), func(screen *ebiten.Image, x, y float64, w float64) {
			fm.DrawText(screen, d.SpecialDesc, x, y, theme.FontXS, theme.StatusSkill)
		})
	}

	// Row 6: Growth info.
	if d.GrowthDesc != "" {
		p.AddSpace(float32(theme.DetailGap))
		p.AddRow(float32(theme.DetailRowH), func(screen *ebiten.Image, x, y float64, w float64) {
			fm.DrawText(screen, d.GrowthDesc, x, y, theme.FontXS, theme.StatusGrowth)
		})
	}

	p.Draw(screen)
}

// estimateWardenPanelHeight pre-calculates the total panel height
// so AnchoredRect can correctly position it from the bottom edge.
func estimateWardenPanelHeight(d WardenPanelData) float32 {
	h := wardenPanelPad*2 + float32(theme.DetailTitleH) + float32(theme.DetailAttrH)
	if d.Damage != "" || d.Interval != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailAttrH)
	}
	if d.AttackDesc != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailRowH)
	}
	if d.SpecialDesc != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailRowH)
	}
	if d.GrowthDesc != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailRowH)
	}
	return h
}
