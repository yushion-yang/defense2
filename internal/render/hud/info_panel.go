// info_panel.go — Bottom-center tower detail panel.
// Shows tower stats, current abilities, upgrade growth, and sell button when a tower is selected.
// All data is provided via InfoPanelVM — no direct dependency on core/tower or core/strength.
//
// 布局分为两部分：
//   - 固定头部（FlexPanel）：塔名 + 属性行 + 攻击方式
//   - 可滚动区域（DrawScrollRegion）：能力详情 + buff 列表
//   - 固定尾部（FlexPanel）：按钮区域
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/i18n"
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
	SellButtonText        string // e.g. "卖60"
	CanAffordUpgrade      bool   // true if player can afford single upgrade

	// 所有权（AI 塔只显示信息，不显示操作按钮）
	HideActions bool   // true=隐藏升级/卖出/能力按钮（非自己的塔）
	OwnerLabel  string // 非空时显示在标题旁（如 "AI 的塔"）
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
	lastSellRect        ui.Rect
	lastAbilityBtnRect  ui.Rect // "选择能力(N)" 按钮
	lastUnlockBtnRect   ui.Rect // "解锁能力 $XX" 按钮
	lastPanelRect       ui.Rect // entire info panel bounding box
	lastPanelVisible    bool
)

// ---------------------------------------------------------------------------
// Scroll state — 包级变量，跨帧保持滚动位置。
// ---------------------------------------------------------------------------

var infoPanelScroll ui.ScrollState

// InfoPanelScroll 接收鼠标滚轮增量，更新能力/buff 区域的滚动偏移。
// 由 stage_input.go 在 modeTowerSel 时调用。
func InfoPanelScroll(deltaY float64) {
	infoPanelScroll.Scroll(deltaY)
}

// ResetInfoPanelScroll 重置滚动到顶部。
// 在塔选中变更时调用，防止新塔继承旧塔的滚动位置。
func ResetInfoPanelScroll() {
	infoPanelScroll.OffsetY = 0
}

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
		ui.Label(screen, vm.Label, x, y, w, ui.LabelStyle{
			Font: theme.FontXL, Color: theme.TextTitle, Bold: true,
		})
		if vm.StrengthText != "" {
			ui.Label(screen, vm.StrengthText, x, y+4, w, ui.LabelStyle{
				Font: theme.FontSM, Color: vm.StrengthColor, Align: ui.AlignRight,
			})
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
		{"stat-damage", i18n.T("hud.stat.damage"), vm.DamageSegs, vm.Specialty == 0},
		{"stat-atkspd", i18n.T("hud.stat.atkspd"), vm.SpeedSegs, vm.Specialty == 1},
		{"stat-range", i18n.T("hud.stat.range"), vm.RangeSegs, vm.Specialty == 2},
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
				ui.Label(screen, prefix+ar.label, rx, ry, rw, ui.LabelStyle{
					Font: theme.FontSM, Color: labelClr,
				})
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
					ui.Label(screen, seg.Text, sx, ry, 0, ui.LabelStyle{
						Font: theme.FontLG, Color: clr,
					})
					sx += segW
				}
			})
			row.Draw(screen, float64(attrH))
		})
	}

	// Row 5: 攻击方式
	panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
		ui.Label(screen, vm.AttackStyleText, x, y, w, ui.LabelStyle{
			Font: theme.FontSM, Color: theme.TextMuted,
		})
	})

	// --- 计算可滚动区域高度预算 ---
	// infoPanelMaxH 限制面板总高度，scrollViewH 是能力+buff 可用的可视高度
	const infoPanelMaxH float32 = 420

	// 尾部固定高度：按钮组（含间距）
	// tailH 包括：detailGap + 按钮们 + botPad
	var tailH float32
	tailH += detailGap + btnH // 强度/卖出按钮
	tailH += botPad - innerPad
	if vm.PendingCount > 0 {
		tailH += detailGap + 32 // "选择能力(N)" 按钮
	}
	if vm.CanUnlockSlot {
		tailH += detailGap + 32 // "解锁能力 $XX" 按钮
	}

	headerH := panel.Height() // 头部已用高度
	scrollViewH := infoPanelMaxH - headerH - tailH
	if scrollViewH < 40 {
		scrollViewH = 40
	}

	// 内容区宽度（面板内边距内）
	contentW := float64(panel.ContentWidth())

	// --- 预算量能力+buff 内容总高度，判断是否需要滚动 ---
	var scrollContentH float32
	if len(vm.Abilities) > 0 {
		scrollContentH += 2 // 顶部间距
		for _, ab := range vm.Abilities {
			h := measureAbilityRowHeight(fm, ab, contentW)
			scrollContentH += float32(h)
		}
	}
	if len(vm.Buffs) > 0 {
		scrollContentH += 2 // 间距
		scrollContentH += float32(len(vm.Buffs)) * 14
	}

	// 如果内容不需要滚动，缩小可视区域到内容高度，避免多余空白
	actualViewH := scrollViewH
	if scrollContentH < scrollViewH {
		actualViewH = scrollContentH
	}
	if actualViewH < 0 {
		actualViewH = 0
	}

	// 更新 ScrollState 的内容高度
	infoPanelScroll.ContentH = scrollContentH

	// Row 6: 可滚动的能力+buff 区域
	if scrollContentH > 0 {
		panel.AddRow(actualViewH, func(screen *ebiten.Image, x, y float64, w float64) {
			ui.DrawScrollRegion(screen, float32(x), float32(y), float32(w), actualViewH,
				&infoPanelScroll,
				func(screen *ebiten.Image, cx, cy float64, cw float64) {
					drawScrollableContent(screen, fm, vm, cx, cy, cw)
				})
		})
	}

	// "选择能力(N)" 按钮（AI 的塔不显示）
	lastAbilityBtnRect = ui.Rect{}
	if vm.PendingCount > 0 && !vm.HideActions {
		panel.AddSpace(detailGap)
		panel.AddRow(32, func(screen *ebiten.Image, x, y float64, w float64) {
			btnRect := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: 30}
			btnClr := color.RGBA{R: 200, G: 160, B: 40, A: 255} // gold
			mx, my, hov := draw.HoverPos()
			if hov && float32(mx) >= btnRect.X && float32(mx) <= btnRect.X+btnRect.W &&
				float32(my) >= btnRect.Y && float32(my) <= btnRect.Y+btnRect.H {
				btnClr = color.RGBA{R: 230, G: 190, B: 60, A: 255}
			}
			ui.Panel(screen, btnRect.X, btnRect.Y, btnRect.W, btnRect.H, ui.PanelStyle{
				BgColor: btnClr, Radius: 6,
			})
			label := i18n.TF("hud.info.select_ability", vm.PendingCount)
			ui.LabelV(screen, label,
				btnRect.CenterX(), btnRect.CenterY(), float64(btnRect.W)-12, ui.LabelStyle{
					Font: theme.FontSM, Color: theme.TextTitle, Bold: true,
				})
			lastAbilityBtnRect = btnRect
		})
	}

	// "解锁能力 $XX" 按钮（战役模式，AI 的塔不显示）
	lastUnlockBtnRect = ui.Rect{}
	if vm.CanUnlockSlot && !vm.HideActions {
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
			ui.Panel(screen, btnRect.X, btnRect.Y, btnRect.W, btnRect.H, ui.PanelStyle{
				BgColor: btnClr, Radius: 6,
			})
			label := i18n.TF("hud.info.unlock_ability", vm.UnlockCost)
			textClr := theme.TextTitle
			if !affordable {
				textClr = color.RGBA{R: 160, G: 160, B: 160, A: 255}
			}
			ui.LabelV(screen, label,
				btnRect.CenterX(), btnRect.CenterY(), float64(btnRect.W)-12, ui.LabelStyle{
					Font: theme.FontSM, Color: textClr, Bold: true,
				})
			lastUnlockBtnRect = btnRect
		})
	}

	// 操作按钮：强度+10 / 卖出（AI 的塔不显示）
	lastUpgradeRect = ui.Rect{}
	lastSellRect = ui.Rect{}

	if !vm.HideActions {
		panel.AddSpace(detailGap)
		panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
			area := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: btnH}

			upgradeClr := theme.ToneDisabled
			if vm.CanAffordUpgrade {
				upgradeClr = theme.TonePrimary
			}

			result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
				{Label: vm.UpgradeButtonText, Color: upgradeClr},
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
	} else if vm.OwnerLabel != "" {
		// AI 的塔：显示所有者标签替代按钮
		panel.AddSpace(detailGap)
		panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
			ui.LabelV(screen, vm.OwnerLabel, x+w/2, y+float64(btnH)/2, w, ui.LabelStyle{
				Font: theme.FontSM, Color: theme.TextMuted, Align: ui.AlignCenter,
			})
		})
	}

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

// drawScrollableContent 渲染可滚动区域内的全部能力+buff 内容。
// 由 DrawScrollRegion 的 renderContent 回调调用，y 已减去滚动偏移。
// 渲染所有条目，不做截断（"还有N个..."已移除）。
func drawScrollableContent(screen *ebiten.Image, fm *render.FontManager, vm InfoPanelVM, x, y, w float64) {
	curY := y

	// 能力详情列表
	if len(vm.Abilities) > 0 {
		curY += 2 // 顶部间距
		for _, ab := range vm.Abilities {
			drawAbilityRowVM(screen, fm, ab, x, curY, w)
			curY += measureAbilityRowHeight(fm, ab, w)
		}
	}

	// Buff 列表
	if len(vm.Buffs) > 0 {
		curY += 2 // 间距
		const buffH = 14.0
		for _, b := range vm.Buffs {
			drawBuffRow(screen, fm, b, x, curY, w)
			curY += buffH
		}
	}
}

// drawBuffRow 渲染单个 buff 行，正确计算 maxW 防止溢出。
func drawBuffRow(screen *ebiten.Image, fm *render.FontManager, b BuffVM, x, y, w float64) {
	srcClr := color.RGBA{R: 180, G: 140, B: 255, A: 220}

	// 如果有剩余时间，预留右侧空间给时间文本
	var timeW float64
	var timeStr string
	if b.Remaining >= 0 {
		timeStr = strconv.FormatFloat(b.Remaining, 'f', 0, 64) + "s"
		timeW = fm.MeasureText(timeStr, theme.FontXS) + 4
	}

	// Source 文本（带宽度约束）
	srcMaxW := w * 0.4 // source 最多占 40% 宽度
	ui.Label(screen, b.Source, x, y, srcMaxW, ui.LabelStyle{
		Font: theme.FontXS, Color: srcClr,
	})
	srcW := fm.MeasureText(b.Source, theme.FontXS)
	if srcW > srcMaxW {
		srcW = srcMaxW
	}

	// Desc 文本（受 Source 和 Time 约束）
	descX := x + srcW + 6
	descMaxW := w - srcW - 6 - timeW
	if descMaxW < 20 {
		descMaxW = 20
	}
	ui.Label(screen, b.Desc, descX, y, descMaxW, ui.LabelStyle{
		Font: theme.FontXS, Color: theme.TextMuted,
	})

	// 剩余时间（右对齐）
	if timeStr != "" {
		ui.Label(screen, timeStr, x, y, w, ui.LabelStyle{
			Font: theme.FontXS, Color: theme.TextMuted, Align: ui.AlignRight,
		})
	}
}

// drawSlotRow renders one ability slot row.
func drawSlotRow(screen *ebiten.Image, fm *render.FontManager, slot SlotVM, x, y, w float64) {
	im := render.GlobalIcons()

	if !slot.Unlocked {
		// Locked slot: grey text
		ui.Label(screen, "\U0001F512 "+slot.CategoryName, x, y, w, ui.LabelStyle{
			Font: theme.FontXS, Color: color.RGBA{R: 80, G: 90, B: 110, A: 140},
		})
		return
	}

	if slot.AbilityLabel != "" {
		// Filled slot: icon + ability name (Label 内置 ShrinkFontSize 自动缩放)
		if slot.AbilityIcon != "" {
			drawStatIcon(screen, im, slot.AbilityIcon, x, y, 14)
		}
		labelMaxW := w - 19 // 留出图标空间
		catSuffix := " (" + slot.CategoryName + ")"
		fullText := slot.AbilityLabel + catSuffix
		ui.Label(screen, fullText, x+19, y, labelMaxW, ui.LabelStyle{
			Font: theme.FontSM, Color: theme.TextBody, Bold: true,
		})
		return
	}

	if slot.HasPending {
		// Pending slot: gold flash
		ui.Label(screen, "\u26A1 "+slot.CategoryName+" \u2014 "+i18n.T("hud.info.pending"), x, y, w, ui.LabelStyle{
			Font: theme.FontSM, Color: color.RGBA{R: 250, G: 200, B: 50, A: 230},
		})
		return
	}

	// Unlocked but not yet cached (shouldn't happen)
	ui.Label(screen, slot.CategoryName, x, y, w, ui.LabelStyle{
		Font: theme.FontXS, Color: theme.TextMuted,
	})
}

// drawAbilityRowVM 渲染一个能力行，支持段落自动换行。
// 使用 abilitySegsToTextSegs + DrawSegmentsWrapped 替代单行内联渲染。
func drawAbilityRowVM(screen *ebiten.Image, fm *render.FontManager, ab AbilityVM, x, y, w float64) {
	im := render.GlobalIcons()

	// Icon
	if ab.Icon != "" {
		drawStatIcon(screen, im, ab.Icon, x, y, 14)
	}
	abX := x + 19.0

	// Label (bold, Label 内置 ShrinkFontSize 自动缩放)
	labelMaxW := 120.0
	ui.Label(screen, ab.Label, abX, y, labelMaxW, ui.LabelStyle{
		Font: theme.FontSM, Color: theme.TextBody, Bold: true,
	})
	labelW := fm.MeasureText(ab.Label, theme.FontSM)
	if labelW > labelMaxW {
		labelW = labelMaxW
	}
	abX += labelW + 6

	// Segments（自动换行）or fallback（带宽度约束）
	remainW := w - 19.0 - labelW - 6
	if remainW < 40 {
		remainW = 40
	}

	if len(ab.Segments) == 0 {
		if ab.Fallback != "" {
			ui.Label(screen, ab.Fallback, abX, y+1, remainW, ui.LabelStyle{
				Font: theme.FontSM, Color: theme.TextMuted,
			})
		}
		return
	}

	segs := abilitySegsToTextSegs(ab.Segments)
	ui.DrawSegmentsWrapped(screen, fm, segs, abX, y+1, remainW, theme.FontSM)
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
	draw.Sprite(screen, img, x+size/2, y+size/2, size) //nolint:hud
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
