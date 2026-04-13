// warden_panel.go — Bottom-right warden info panel.
// Shows real-time warden stats: name, strength, damage, interval,
// attack pattern, special abilities, and growth info.
// Uses FlexPanel + AnchoredRect for adaptive layout.
package hud

import (
	"image/color"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// WardenPanelData holds pre-formatted display data for the warden info panel.
type WardenPanelData struct {
	Name         string        // e.g. "火灵"
	Type         string        // e.g. "prince", "core", "envoy"
	Icon         *ebiten.Image // 战灵精灵图标（可为 nil）
	Strength     float64       // current perceived strength
	PeakStrength float64       // peak (ratchet) strength
	// Display data (pre-formatted by caller).
	Damage      string // e.g. "30"
	Interval    string // e.g. "3s"
	AttackDesc  string // e.g. "冲撞穿透: 向敌群密集方向冲刺..."
	SpecialDesc string // e.g. "火焰轨迹 10 2"
	GrowthDesc  string // e.g. "击杀+1 通波+5"
}

// Warden panel layout constants (与炮塔面板同位 — 底部中央).
const (
	wardenPanelW   float32 = float32(theme.CenterPanelW)
	wardenPanelPad float32 = float32(theme.DetailPad)
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

	// Anchor to bottom-center (与炮塔面板相同位置).
	rect := ui.AnchoredRect(ui.AnchorBottomCenter, wardenPanelW, estimatedH,
		0, 0, float32(theme.BottomMargin), 0)

	// Build FlexPanel at the anchored position.
	p := ui.NewFlexPanel(rect.X, rect.Y, rect.W, wardenPanelPad)
	p.Radius = float32(theme.PanelRadius)
	p.BgColor = theme.PanelBg
	p.Border = theme.InfoWardenBdr

	// Row 1: Title — icon + "战灵·<Name>" (left) + strength indicator (right).
	p.AddRow(float32(theme.DetailTitleH), func(screen *ebiten.Image, x, y float64, w float64) {
		textX := x
		if d.Icon != nil {
			iconSize := float64(theme.DetailTitleH)
			draw.Sprite(screen, d.Icon, x+iconSize/2, y+iconSize/2, iconSize)
			textX += iconSize + 4
		}
		titleTxt := i18n.TF("hud.warden.title", d.Name)
		fm.DrawBoldText(screen, titleTxt, textX, y, theme.FontLG, theme.TextTitle)

		strIndicator := i18n.TF("hud.warden.strength_short", d.Strength)
		fm.DrawRightText(screen, strIndicator, x+w, y+2, theme.FontSM, theme.TextBody)
	})

	// Row 2: Strength — "强度 X" (left) + "最高 X" (right).
	p.AddRow(float32(theme.DetailAttrH), func(screen *ebiten.Image, x, y float64, w float64) {
		strTxt := i18n.TF("hud.warden.strength", d.Strength)
		fm.DrawText(screen, strTxt, x, y, theme.FontSM, theme.TextBody)

		peakTxt := i18n.TF("hud.warden.peak_strength", d.PeakStrength)
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

	// Row 4: Attack description (自适应多行).
	if d.AttackDesc != "" {
		lines := wrapText(fm, d.AttackDesc, float64(wardenPanelW)-float64(wardenPanelPad)*2, theme.FontXS)
		addDescLines(p, fm, lines, theme.TextBody)
	}

	// Row 5: Special ability (自适应多行).
	if d.SpecialDesc != "" {
		lines := wrapText(fm, d.SpecialDesc, float64(wardenPanelW)-float64(wardenPanelPad)*2, theme.FontXS)
		addDescLines(p, fm, lines, theme.StatusStrUp)
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

// addDescLines 添加多行描述文本到面板。
func addDescLines(p *ui.FlexPanel, fm *render.FontManager, lines []string, clr color.RGBA) {
	if len(lines) == 0 {
		return
	}
	p.AddSpace(float32(theme.DetailGap))
	for _, line := range lines {
		line := line
		p.AddRow(14, func(screen *ebiten.Image, x, y float64, w float64) {
			fm.DrawText(screen, line, x, y, theme.FontXS, clr)
		})
	}
}

// wrapText 按像素宽度拆行（逐字符，对中英文混排友好）。
func wrapText(fm *render.FontManager, text string, maxW float64, fontSize float64) []string {
	if fm == nil || text == "" {
		return nil
	}
	runes := []rune(text)
	var lines []string
	start := 0
	for start < len(runes) {
		end := start
		for end < len(runes) {
			w := fm.MeasureText(string(runes[start:end+1]), fontSize)
			if w > maxW && end > start {
				break
			}
			end++
		}
		lines = append(lines, string(runes[start:end]))
		start = end
	}
	return lines
}

// estimateWardenPanelHeight pre-calculates the total panel height.
func estimateWardenPanelHeight(d WardenPanelData) float32 {
	fm := render.GlobalFont()
	contentW := float64(wardenPanelW) - float64(wardenPanelPad)*2
	h := wardenPanelPad*2 + float32(theme.DetailTitleH) + float32(theme.DetailAttrH)
	if d.Damage != "" || d.Interval != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailAttrH)
	}
	if d.AttackDesc != "" {
		n := len(wrapText(fm, d.AttackDesc, contentW, theme.FontXS))
		if n < 1 {
			n = 1
		}
		h += float32(theme.DetailGap) + float32(n)*14
	}
	if d.SpecialDesc != "" {
		n := len(wrapText(fm, d.SpecialDesc, contentW, theme.FontXS))
		if n < 1 {
			n = 1
		}
		h += float32(theme.DetailGap) + float32(n)*14
	}
	if d.GrowthDesc != "" {
		h += float32(theme.DetailGap) + float32(theme.DetailRowH)
	}
	return h
}
