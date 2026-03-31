// info_panel.go — Bottom-center tower detail panel.
// Shows tower stats, current abilities, upgrade growth, and sell button when a tower is selected.
// All data is provided via InfoPanelVM — no direct dependency on core/tower or core/strength.
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// View-model structs — built by the caller (scene package), consumed here.
// ---------------------------------------------------------------------------

// AbilitySegment is one rendering segment inside an ability display line.
// Kind: "text" (plain muted), "bold" (bold body), "base" (body color),
// "scaled" (custom color), "total" (body color).
type AbilitySegment struct {
	Text  string
	Kind  string     // "text", "bold", "base", "scaled", "total"
	Color color.Color // used when Kind == "scaled"
}

// AbilityVM holds pre-computed display data for one ability row.
type AbilityVM struct {
	Icon     string           // icon name (empty = no icon)
	Label    string           // bold label text
	Segments []AbilitySegment // display segments after label (may be nil for fallback-only abilities)
	Fallback string           // shown when Segments is nil (fallback desc)
}

// BuffVM holds display data for one buff row.
type BuffVM struct {
	Source    string  // source display name
	Desc      string  // effect description
	Remaining float64 // remaining seconds (-1 = permanent)
}

// InfoPanelVM contains all display data needed by DrawInfoPanel.
// Built by scene.BuildInfoPanelVM; consumed by hud.DrawInfoPanel.
type InfoPanelVM struct {
	Visible bool // false → panel is hidden

	// Title row
	Label         string
	StrengthText  string     // e.g. "强度120 ↑(永+20)" — empty if no strength data
	StrengthColor color.Color // nil-safe: ignored when StrengthText == ""

	// Attribute row (segment-based for colored rendering)
	DamageSegs []AbilitySegment
	SpeedSegs  []AbilitySegment
	RangeSegs  []AbilitySegment

	// Attack style row
	AttackStyleText string // e.g. "攻击: 投射物"

	// Abilities
	Abilities []AbilityVM

	// Buffs
	Buffs []BuffVM

	// Upgrade ability choices (empty = no pending upgrade)
	UpgradeChoices []UpgradeChoiceVM

	// Buttons
	UpgradeButtonText string // e.g. "强度+10 $10"
	SellButtonText    string // e.g. "卖60"
}

// UpgradeChoiceVM 单个候选能力按钮的展示数据。
type UpgradeChoiceVM struct {
	Type  string // 能力类型标识（用于回调）
	Label string // 显示名
	Desc  string // 描述文本
}

// ---------------------------------------------------------------------------
// Cached button Rects — written by DrawInfoPanel, read by hit tests.
// ---------------------------------------------------------------------------

var (
	lastUpgradeRect      ui.Rect
	lastSellRect         ui.Rect
	lastPanelRect        ui.Rect // entire info panel bounding box
	lastPanelVisible     bool
	lastChoiceRects      []ui.Rect // 候选能力按钮 rects
)

// HitTestUpgradeChoice 检测点击是否在候选能力按钮上，返回索引(-1=未命中)。
func HitTestUpgradeChoice(mx, my float32) int {
	for i, r := range lastChoiceRects {
		if mx >= r.X && mx <= r.X+r.W && my >= r.Y && my <= r.Y+r.H {
			return i
		}
	}
	return -1
}

// DrawInfoPanel renders the tower information panel using pre-built view data.
// Pass a VM with Visible=false to hide the panel.
func DrawInfoPanel(screen *ebiten.Image, vm InfoPanelVM) {
	if !vm.Visible {
		lastPanelVisible = false
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		lastPanelVisible = false
		return
	}

	const (
		panelW    = float32(theme.CenterPanelW)
		innerPad  = float32(theme.CenterPanelInnerPad)
		titleH    = float32(theme.DetailTitleH)
		attrH     = float32(theme.DetailAttrH)
		abilityH  = float32(theme.DetailRowH)
		btnH      = float32(theme.DetailBtnH)
		topPad    = float32(theme.DetailTopPad)
		botPad    = float32(theme.DetailBotPad)
		detailGap = float32(theme.DetailGap)
		btnGap    = float32(12)
	)

	// --- Build FlexPanel (content-driven height) ---
	panel := ui.NewFlexPanel(0, 0, panelW, innerPad)
	panel.BgColor = theme.PanelBg
	panel.Border = theme.InfoBorder
	panel.Radius = float32(theme.CenterPanelRadius)
	panel.AddSpace(topPad - innerPad)

	// Row 1: 塔名 + 战力显示
	panel.AddRow(titleH, func(screen *ebiten.Image, x, y float64, w float64) {
		fm.DrawBoldText(screen, vm.Label, x, y, theme.FontXL, theme.TextTitle)
		if vm.StrengthText != "" {
			fm.DrawRightText(screen, vm.StrengthText, x+w, y+4, theme.FontSM, vm.StrengthColor)
		}
	})

	// Row 2: 属性行（伤害/攻速/射程）— 三段式着色
	panel.AddRow(attrH, func(screen *ebiten.Image, x, y float64, w float64) {
		colW := w / 3
		const (
			iconSize = 16.0
			iconGap  = 5.0
		)
		im := render.GlobalIcons()
		textOff := iconSize + iconGap

		drawAttrSegs := func(segs []AbilitySegment, sx, sy float64) {
			for _, seg := range segs {
				var clr color.Color = theme.TextBody
				if seg.Kind == "scaled" && seg.Color != nil {
					clr = seg.Color
				}
				fm.DrawText(screen, seg.Text, sx, sy, theme.FontLG, clr)
				sx += fm.MeasureText(seg.Text, theme.FontLG)
			}
		}

		// 伤害
		drawStatIcon(screen, im, "stat-damage", x, y, iconSize)
		drawAttrSegs(vm.DamageSegs, x+textOff, y)

		// 攻速
		drawStatIcon(screen, im, "stat-atkspd", x+colW, y, iconSize)
		drawAttrSegs(vm.SpeedSegs, x+colW+textOff, y)

		// 射程
		drawStatIcon(screen, im, "stat-range", x+colW*2, y, iconSize)
		drawAttrSegs(vm.RangeSegs, x+colW*2+textOff, y)
	})

	// Row 3: 攻击方式
	panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
		fm.DrawText(screen, vm.AttackStyleText, x, y, theme.FontSM, theme.TextMuted)
	})

	// Row 4: 能力列表
	for _, ab := range vm.Abilities {
		ab := ab
		panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
			drawAbilityRowVM(screen, fm, ab, x, y)
		})
	}

	// Row 5: Buff 列表
	if len(vm.Buffs) > 0 {
		panel.AddSpace(2)
		for _, b := range vm.Buffs {
			b := b
			panel.AddRow(14, func(screen *ebiten.Image, x, y float64, w float64) {
				srcClr := color.RGBA{R: 180, G: 140, B: 255, A: 220}
				fm.DrawText(screen, b.Source, x, y, theme.FontXS, srcClr)
				srcW := fm.MeasureText(b.Source, theme.FontXS)
				fm.DrawText(screen, b.Desc, x+srcW+6, y, theme.FontXS, theme.TextMuted)
				if b.Remaining >= 0 {
					timeStr := fmt.Sprintf("%.0fs", b.Remaining)
					fm.DrawRightText(screen, timeStr, x+w, y, theme.FontXS, theme.TextMuted)
				}
			})
		}
	}

	// 候选能力选择（有 pending upgrade 时显示）
	lastChoiceRects = lastChoiceRects[:0]
	if len(vm.UpgradeChoices) > 0 {
		panel.AddSpace(detailGap)
		panel.AddRow(14, func(screen *ebiten.Image, x, y float64, _ float64) {
			fm.DrawBoldText(screen, "选择能力:", x, y, theme.FontSM, color.RGBA{R: 250, G: 200, B: 50, A: 255})
		})
		panel.AddSpace(4)
		for idx, ch := range vm.UpgradeChoices {
			ch := ch
			idx := idx
			panel.AddRow(32, func(screen *ebiten.Image, x, y float64, w float64) {
				btnRect := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: 30}
				btnClr := theme.TonePrimary
				// 简单 hover 检测
				mx, my := draw.CursorPos()
				if float32(mx) >= btnRect.X && float32(mx) <= btnRect.X+btnRect.W &&
					float32(my) >= btnRect.Y && float32(my) <= btnRect.Y+btnRect.H {
					btnClr = color.RGBA{R: 60, G: 120, B: 200, A: 255}
				}
				draw.RoundRect(screen, btnRect.X, btnRect.Y, btnRect.W, btnRect.H, 6, btnClr)
				fm.DrawBoldText(screen, ch.Label, x+10, y+7, theme.FontSM, theme.TextTitle)
				fm.DrawText(screen, ch.Desc, x+10+fm.MeasureText(ch.Label, theme.FontSM)+8, y+8, theme.FontXS, theme.TextBody)
				// 扩展 rects 到正确索引
				for len(lastChoiceRects) <= idx {
					lastChoiceRects = append(lastChoiceRects, ui.Rect{})
				}
				lastChoiceRects[idx] = btnRect
			})
		}
	}

	// 操作按钮：强度+10 / 卖出
	panel.AddSpace(detailGap)
	lastUpgradeRect = ui.Rect{}
	lastSellRect = ui.Rect{}

	panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
		area := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: btnH}

		result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
			{Label: vm.UpgradeButtonText, Color: theme.TonePrimary},
			{Label: vm.SellButtonText, Color: theme.BtnDanger},
		}, ui.ButtonRowStyle{
			Height:   btnH,
			Gap:      btnGap,
			Radius:   float32(theme.ButtonRadius),
			FontSize: theme.FontSM,
		})
		if len(result.Rects) == 2 {
			lastUpgradeRect = result.Rects[0]
			lastSellRect = result.Rects[1]
		}
	})

	panel.AddSpace(botPad - innerPad)

	// --- Position panel at bottom-center ---
	totalH := panel.Height()
	anchor := ui.AnchoredRect(ui.AnchorBottomCenter, panelW, totalH,
		0, 0, float32(theme.BottomMargin), 0)
	panel.X = anchor.X
	panel.Y = anchor.Y

	panel.Draw(screen)
	lastPanelRect = ui.Rect{X: panel.X, Y: panel.Y, W: panelW, H: totalH}
	lastPanelVisible = true
}

// drawAbilityRowVM renders one ability row from pre-computed VM data.
func drawAbilityRowVM(screen *ebiten.Image, fm *render.FontManager, ab AbilityVM, x, y float64) {
	im := render.GlobalIcons()

	// Icon
	if ab.Icon != "" {
		drawStatIcon(screen, im, ab.Icon, x, y, 14)
	}
	abX := x + 19.0

	// Label (bold)
	fm.DrawBoldText(screen, ab.Label, abX, y, theme.FontSM, theme.TextBody)
	abX += fm.MeasureText(ab.Label, theme.FontSM) + 6

	// Segments or fallback
	if len(ab.Segments) == 0 {
		if ab.Fallback != "" {
			fm.DrawText(screen, ab.Fallback, abX, y+1, theme.FontSM, theme.TextMuted)
		}
		return
	}

	segX := abX
	for _, seg := range ab.Segments {
		switch seg.Kind {
		case "text":
			fm.DrawText(screen, seg.Text, segX, y+1, theme.FontSM, theme.TextMuted)
			segX += fm.MeasureText(seg.Text, theme.FontSM)
		case "base", "total":
			fm.DrawText(screen, seg.Text, segX, y+1, theme.FontSM, theme.TextBody)
			segX += fm.MeasureText(seg.Text, theme.FontSM)
		case "scaled":
			clr := seg.Color
			if clr == nil {
				clr = theme.TextBody
			}
			fm.DrawText(screen, seg.Text, segX, y+1, theme.FontSM, clr)
			segX += fm.MeasureText(seg.Text, theme.FontSM)
		}
	}
}

// drawStatIcon draws a stat icon at (x, y) with the given logical display size.
func drawStatIcon(screen *ebiten.Image, im *render.IconManager, name string, x, y, size float64) {
	if im == nil {
		return
	}
	img := im.Get(name)
	if img == nil {
		return
	}
	draw.Sprite(screen, img, x+size/2, y+size/2, size)
}

// DrawInfoPanelHoverTooltip 悬停面板时的提示（已无等级系统，保留接口兼容）。
func DrawInfoPanelHoverTooltip(screen *ebiten.Image, visible bool, mx, my float32) {
}

// InfoPanelUpgradeHitTest 检查是否点击了购买强度按钮。
func InfoPanelUpgradeHitTest(px, py float32, visible bool) bool {
	if !visible || !lastPanelVisible {
		return false
	}
	return lastUpgradeRect.Contains(float64(px), float64(py))
}

// InfoPanelSellHitTest 检查是否点击了卖出按钮。
func InfoPanelSellHitTest(px, py float32, visible bool) bool {
	if !visible || !lastPanelVisible {
		return false
	}
	return lastSellRect.Contains(float64(px), float64(py))
}

