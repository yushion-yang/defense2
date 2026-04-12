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
	Kind  string      // "text", "bold", "base", "scaled", "total"
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
	StrengthText  string      // e.g. "强度120 ↑(永+20)" — empty if no strength data
	StrengthColor color.Color // nil-safe: ignored when StrengthText == ""

	// Attribute row (segment-based for colored rendering)
	DamageSegs []AbilitySegment
	SpeedSegs  []AbilitySegment
	RangeSegs  []AbilitySegment
	Specialty  int // 专精属性 (0=damage, 1=speed, 2=range)，对应行显示星标

	// Attack style row
	AttackStyleText string // e.g. "攻击: 投射物"

	// 6-slot ability display
	Slots []SlotVM // 6 个能力槽（按 UnlockOrder 排列）

	// Abilities (已获取的能力详细描述)
	Abilities []AbilityVM

	// Buffs
	Buffs []BuffVM

	// Pending ability selection
	PendingCount int // 待选能力位数量

	// Unlock ability slot (campaign mode)
	CanUnlockSlot bool // 是否可以解锁下一个能力槽位
	UnlockCost    int  // 解锁下一个槽位的费用
	Gold          int  // 当前金币（用于判断是否买得起）

	// Buttons
	UpgradeButtonText     string // e.g. "强度+10 $10"
	BulkUpgradeButtonText string // e.g. "强度+50 $50"
	SellButtonText        string // e.g. "卖60"
	CanAffordUpgrade      bool   // true if player can afford single upgrade
	CanAffordBulkUpgrade  bool   // true if player can afford bulk upgrade
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
	lastUpgradeRect     ui.Rect
	lastBulkUpgradeRect ui.Rect
	lastSellRect        ui.Rect
	lastAbilityBtnRect  ui.Rect // "选择能力(N)" 按钮
	lastUnlockBtnRect   ui.Rect // "解锁能力 $XX" 按钮
	lastPanelRect       ui.Rect // entire info panel bounding box
	lastPanelVisible    bool
)

// HitTestAbilityBtn 检测点击是否在"选择能力(N)"按钮上。
func HitTestAbilityBtn(mx, my float32) bool {
	if !lastPanelVisible {
		return false
	}
	return lastAbilityBtnRect.Contains(float64(mx), float64(my))
}

// HitTestUnlockBtn 检测点击是否在"解锁能力"按钮上。
func HitTestUnlockBtn(mx, my float32) bool {
	if !lastPanelVisible {
		return false
	}
	return lastUnlockBtnRect.Contains(float64(mx), float64(my))
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

	// Row 2~4: 属性行（每行一个属性，FlexRow 布局：图标 | 标签 | 数值段）
	const (
		iconSize = 16.0
		iconGap  = 4.0
		labelW   = 36.0
	)
	im := render.GlobalIcons()
	attrRows := []struct {
		icon      string
		label     string
		segs      []AbilitySegment
		specialty bool
	}{
		{"stat-damage", "伤害", vm.DamageSegs, vm.Specialty == 0},
		{"stat-atkspd", "攻速", vm.SpeedSegs, vm.Specialty == 1},
		{"stat-range", "射程", vm.RangeSegs, vm.Specialty == 2},
	}
	for _, ar := range attrRows {
		ar := ar
		panel.AddRow(attrH, func(screen *ebiten.Image, x, y float64, w float64) {
			row := ui.NewFlexRow(x, y, w)
			row.AddFixed(iconSize+iconGap, func(screen *ebiten.Image, rx, ry, rw, rh float64) {
				drawStatIcon(screen, im, ar.icon, rx, ry, iconSize)
			})
			row.AddFixed(labelW, func(screen *ebiten.Image, rx, ry, rw, rh float64) {
				labelClr := theme.TextMuted
				prefix := ""
				if ar.specialty {
					prefix = "\u2605"                                    // ★
					labelClr = color.RGBA{R: 255, G: 200, B: 50, A: 255} // gold
				}
				fm.DrawText(screen, prefix+ar.label, rx, ry, theme.FontSM, labelClr)
			})
			row.AddFill(func(screen *ebiten.Image, rx, ry, rw, rh float64) {
				sx := rx
				for _, seg := range ar.segs {
					var clr color.Color = theme.TextBody
					switch {
					case seg.Kind == "scaled" && seg.Color != nil:
						clr = seg.Color
					case seg.Kind == "aura":
						clr = color.RGBA{R: 80, G: 220, B: 120, A: 255}
					}
					segW := fm.MeasureText(seg.Text, theme.FontLG)
					if sx+segW > rx+rw {
						break // 超出可用宽度则截断
					}
					fm.DrawText(screen, seg.Text, sx, ry, theme.FontLG, clr)
					sx += segW
				}
			})
			row.Draw(screen, float64(attrH))
		})
	}

	// Row 3: 攻击方式
	panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
		fm.DrawText(screen, vm.AttackStyleText, x, y, theme.FontSM, theme.TextMuted)
	})

	// 高度预算：预估按钮区域高度，限制能力+buff 可用空间
	const infoPanelMaxH float32 = 420
	fixedH := panel.Height() + detailGap + btnH + botPad // 当前高度 + 按钮 + 底部
	budgetH := infoPanelMaxH - fixedH                    // 能力+buff 可用高度

	// Row 4: 已获取能力详细描述（受高度预算限制）
	var usedH float32
	if len(vm.Abilities) > 0 {
		panel.AddSpace(2)
		usedH += 2
		maxAbil := len(vm.Abilities)
		for i, ab := range vm.Abilities {
			if usedH+abilityH > budgetH-14 { // 预留一行给截断指示
				remaining := maxAbil - i
				if remaining > 0 {
					panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
						fm.DrawText(screen, fmt.Sprintf("...+%d个能力", remaining), x, y, theme.FontXS, theme.TextMuted)
					})
					usedH += abilityH
				}
				break
			}
			ab := ab
			panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
				drawAbilityRowVM(screen, fm, ab, x, y)
			})
			usedH += abilityH
		}
	}

	// Row 6: Buff 列表（受剩余高度预算限制）
	if len(vm.Buffs) > 0 {
		panel.AddSpace(2)
		usedH += 2
		const buffH float32 = 14
		for i, b := range vm.Buffs {
			if usedH+buffH > budgetH-14 {
				remaining := len(vm.Buffs) - i
				if remaining > 0 {
					panel.AddRow(buffH, func(screen *ebiten.Image, x, y float64, _ float64) {
						fm.DrawText(screen, fmt.Sprintf("...+%d个buff", remaining), x, y, theme.FontXS, theme.TextMuted)
					})
					usedH += buffH
				}
				break
			}
			b := b
			panel.AddRow(buffH, func(screen *ebiten.Image, x, y float64, w float64) {
				srcClr := color.RGBA{R: 180, G: 140, B: 255, A: 220}
				fm.DrawText(screen, b.Source, x, y, theme.FontXS, srcClr)
				srcW := fm.MeasureText(b.Source, theme.FontXS)
				fm.DrawText(screen, b.Desc, x+srcW+6, y, theme.FontXS, theme.TextMuted)
				if b.Remaining >= 0 {
					timeStr := fmt.Sprintf("%.0fs", b.Remaining)
					fm.DrawRightText(screen, timeStr, x+w, y, theme.FontXS, theme.TextMuted)
				}
			})
			usedH += buffH
		}
	}

	// "选择能力(N)" 按钮
	lastAbilityBtnRect = ui.Rect{}
	if vm.PendingCount > 0 {
		panel.AddSpace(detailGap)
		panel.AddRow(32, func(screen *ebiten.Image, x, y float64, w float64) {
			btnRect := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: 30}
			btnClr := color.RGBA{R: 200, G: 160, B: 40, A: 255} // gold
			mx, my, hov := draw.HoverPos()
			if hov && float32(mx) >= btnRect.X && float32(mx) <= btnRect.X+btnRect.W &&
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

	// "解锁能力 $XX" 按钮（战役模式）
	lastUnlockBtnRect = ui.Rect{}
	if vm.CanUnlockSlot {
		panel.AddSpace(detailGap)
		panel.AddRow(32, func(screen *ebiten.Image, x, y float64, w float64) {
			btnRect := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: 30}
			affordable := vm.Gold >= vm.UnlockCost
			var btnClr color.RGBA
			if affordable {
				btnClr = color.RGBA{R: 60, G: 160, B: 200, A: 255} // blue
				mx, my, hov := draw.HoverPos()
				if hov && float32(mx) >= btnRect.X && float32(mx) <= btnRect.X+btnRect.W &&
					float32(my) >= btnRect.Y && float32(my) <= btnRect.Y+btnRect.H {
					btnClr = color.RGBA{R: 80, G: 190, B: 230, A: 255}
				}
			} else {
				btnClr = color.RGBA{R: 80, G: 80, B: 80, A: 200} // gray
			}
			draw.RoundRect(screen, btnRect.X, btnRect.Y, btnRect.W, btnRect.H, 6, btnClr)
			label := fmt.Sprintf("解锁能力 $%d", vm.UnlockCost)
			textClr := theme.TextTitle
			if !affordable {
				textClr = color.RGBA{R: 160, G: 160, B: 160, A: 255}
			}
			fm.DrawCenteredBoldText(screen, label,
				float64(btnRect.X)+float64(btnRect.W)/2, y+7, theme.FontSM, textClr)
			lastUnlockBtnRect = btnRect
		})
	}

	// 操作按钮：强度+10 / 卖出
	panel.AddSpace(detailGap)
	lastUpgradeRect = ui.Rect{}
	lastBulkUpgradeRect = ui.Rect{}
	lastSellRect = ui.Rect{}

	panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
		area := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: btnH}

		upgradeClr := theme.ToneDisabled
		if vm.CanAffordUpgrade {
			upgradeClr = theme.TonePrimary
		}
		bulkClr := theme.ToneDisabled
		if vm.CanAffordBulkUpgrade {
			bulkClr = theme.TonePrimary
		}

		result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
			{Label: vm.UpgradeButtonText, Color: upgradeClr},
			{Label: vm.BulkUpgradeButtonText, Color: bulkClr},
			{Label: vm.SellButtonText, Color: theme.BtnDanger},
		}, ui.ButtonRowStyle{
			Height:   btnH,
			Gap:      btnGap,
			Radius:   float32(theme.ButtonRadius),
			FontSize: theme.FontSM,
		})
		if len(result.Rects) == 3 {
			lastUpgradeRect = result.Rects[0]
			lastBulkUpgradeRect = result.Rects[1]
			lastSellRect = result.Rects[2]
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

// InfoPanelBulkUpgradeHitTest 检查是否点击了大额购买强度按钮。
func InfoPanelBulkUpgradeHitTest(px, py float32, visible bool) bool {
	if !visible || !lastPanelVisible {
		return false
	}
	return lastBulkUpgradeRect.Contains(float64(px), float64(py))
}

// InfoPanelSellHitTest 检查是否点击了卖出按钮。
func InfoPanelSellHitTest(px, py float32, visible bool) bool {
	if !visible || !lastPanelVisible {
		return false
	}
	return lastSellRect.Contains(float64(px), float64(py))
}
