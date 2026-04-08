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

	// 6-slot ability display
	Slots []SlotVM // 6 个能力槽（按 UnlockOrder 排列）

	// Abilities (已获取的能力详细描述)
	Abilities []AbilityVM

	// Buffs
	Buffs []BuffVM

	// Pending ability selection
	PendingCount int               // 待选能力位数量
	UpgradeChoices []UpgradeChoiceVM // 兼容旧代码（废弃路径）

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

// SlotVM 能力槽展示数据。
type SlotVM struct {
	CategoryIdx  int    // 类别索引 (0-5)
	CategoryName string // 类别中文名
	AbilityLabel string // 已选能力名（空=未选）
	AbilityIcon  string // 已选能力图标（空=未选）
	Unlocked     bool   // 是否已解锁
	HasPending   bool   // 是否有待选缓存选项
}

// ---------------------------------------------------------------------------
// Cached button Rects — written by DrawInfoPanel, read by hit tests.
// ---------------------------------------------------------------------------

var (
	lastUpgradeRect      ui.Rect
	lastSellRect         ui.Rect
	lastAbilityBtnRect   ui.Rect   // "选择能力(N)" 按钮
	lastPanelRect        ui.Rect   // entire info panel bounding box
	lastPanelVisible     bool
	lastChoiceRects      []ui.Rect // 候选能力按钮 rects (legacy)
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

// HitTestAbilityBtn 检测点击是否在"选择能力(N)"按钮上。
func HitTestAbilityBtn(mx, my float32) bool {
	if !lastPanelVisible {
		return false
	}
	return lastAbilityBtnRect.Contains(float64(mx), float64(my))
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
				switch {
				case seg.Kind == "scaled" && seg.Color != nil:
					clr = seg.Color
				case seg.Kind == "aura":
					clr = color.RGBA{R: 80, G: 220, B: 120, A: 255} // green for aura buff
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

	// Row 4: 已获取能力详细描述
	if len(vm.Abilities) > 0 {
		panel.AddSpace(2)
		for _, ab := range vm.Abilities {
			ab := ab
			panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
				drawAbilityRowVM(screen, fm, ab, x, y)
			})
		}
	}

	// Row 6: Buff 列表
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

	// "选择能力(N)" 按钮
	lastChoiceRects = lastChoiceRects[:0]
	lastAbilityBtnRect = ui.Rect{}
	if vm.PendingCount > 0 {
		panel.AddSpace(detailGap)
		panel.AddRow(32, func(screen *ebiten.Image, x, y float64, w float64) {
			btnRect := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: 30}
			btnClr := color.RGBA{R: 200, G: 160, B: 40, A: 255} // gold
			mx, my := draw.CursorPos()
			if float32(mx) >= btnRect.X && float32(mx) <= btnRect.X+btnRect.W &&
				float32(my) >= btnRect.Y && float32(my) <= btnRect.Y+btnRect.H {
				btnClr = color.RGBA{R: 230, G: 190, B: 60, A: 255}
			}
			draw.RoundRect(screen, btnRect.X, btnRect.Y, btnRect.W, btnRect.H, 6, btnClr)
			label := fmt.Sprintf("选择能力 (%d)", vm.PendingCount)
			fm.DrawCenteredBoldText(screen, label,
				float64(btnRect.X)+float64(btnRect.W)/2, y+7, theme.FontSM, theme.TextTitle)
			lastAbilityBtnRect = btnRect
		})
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
		0, 0, float32(theme.BottomMargin)+float32(theme.ActionBarH)+4, 0)
	panel.X = anchor.X
	panel.Y = anchor.Y

	panel.Draw(screen)
	lastPanelRect = ui.Rect{X: panel.X, Y: panel.Y, W: panelW, H: totalH}
	lastPanelVisible = true
}

// drawSlotRow renders one ability slot row.
func drawSlotRow(screen *ebiten.Image, fm *render.FontManager, slot SlotVM, x, y, w float64) {
	im := render.GlobalIcons()

	if !slot.Unlocked {
		// Locked slot: grey text
		fm.DrawText(screen, "🔒 "+slot.CategoryName, x, y, theme.FontXS, color.RGBA{R: 80, G: 90, B: 110, A: 140})
		return
	}

	if slot.AbilityLabel != "" {
		// Filled slot: icon + ability name
		if slot.AbilityIcon != "" {
			drawStatIcon(screen, im, slot.AbilityIcon, x, y, 14)
		}
		fm.DrawBoldText(screen, slot.AbilityLabel, x+19, y, theme.FontSM, theme.TextBody)
		fm.DrawText(screen, " ("+slot.CategoryName+")", x+19+fm.MeasureText(slot.AbilityLabel, theme.FontSM), y, theme.FontXS, theme.TextMuted)
		return
	}

	if slot.HasPending {
		// Pending slot: gold flash
		fm.DrawText(screen, "⚡ "+slot.CategoryName+" — 待选择", x, y, theme.FontSM,
			color.RGBA{R: 250, G: 200, B: 50, A: 230})
		return
	}

	// Unlocked but not yet cached (shouldn't happen)
	fm.DrawText(screen, slot.CategoryName, x, y, theme.FontXS, theme.TextMuted)
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

