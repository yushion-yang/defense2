// build_menu.go — Build tower popup panel.
// Popup grid layout with tower cards, role tags, sprite previews, and hover tooltip.
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

// BuildCardVM 建造菜单中单个塔卡片的展示数据（纯值类型）。
type BuildCardVM struct {
	Key         string
	Label       string
	Cost        int
	Damage      float64
	AttackSpeed float64
	Range       float64
	RoleTag     string        // 预计算的角色标签："输出·减速"/"辅助·光环"/...
	RoleColor   color.RGBA    // 角色标签颜色
	TypeIcon    string        // 塔类型图标名："tower-freeze"/""
	Sprite      *ebiten.Image // 预加载的精灵图
	Buildable   bool          // true=可建造, false=仅展示变体
	AbilityDesc string        // 变体卡的能力描述文本
	Abilities   []AbilityVM   // 预设能力描述（经典模式 hover 时展示）
	Category    string        // 角色分类（经典模式用：dps/aoe/support），空=不分组
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
	bpCols    = 5 // cards per row
	bpCardW   = float32(100)
	bpCardH   = float32(80)
	bpCardGap = float32(8)
	bpCardR   = float32(8)
	bpPadX    = float32(16)
	bpPadY    = float32(12)
	bpHeaderH = float32(32)
	bpTitleH  = float32(28)
)

// buildPanelMetrics computes the panel geometry from the card count.
type buildPanelMetrics struct {
	panelX, panelY, panelW, panelH float32
	gridX, gridY                   float32
	rows                           int
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
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin) - float32(theme.ActionBarH) - 4

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

	m := calcBuildPanelMetrics(len(d.Cards))

	// Panel background + border
	ui.Panel(screen, m.panelX, m.panelY, m.panelW, m.panelH, ui.PanelStyle{
		BgColor: theme.PanelBg, BorderColor: theme.PanelBorder, Radius: float32(theme.CenterPanelRadius),
	})

	// Title
	titleX := float64(m.panelX) + float64(bpPadX)
	titleY := float64(m.panelY) + 8
	titleMaxW := float64(m.panelW) - float64(bpPadX)*2
	ui.Label(screen, i18n.T("hud.build.title"), titleX, titleY, titleMaxW, ui.LabelStyle{
		Font: theme.FontLG, Bold: true, Color: theme.TextTitle,
	})

	// Close hint (top-right, 右对齐，与标题共享整行宽度)
	ui.Label(screen, i18n.T("hud.build.close"), titleX, titleY+2, titleMaxW, ui.LabelStyle{
		Font: theme.FontMD, Color: theme.TextMuted, Align: ui.AlignRight,
	})

	for i, card := range d.Cards {
		col := i % bpCols
		row := i / bpCols
		cx := m.gridX + float32(col)*(bpCardW+bpCardGap)
		cy := m.gridY + float32(row)*(bpCardH+bpCardGap)
		hovered := i == d.HoverIdx

		if card.Buildable {
			drawBuildableCard(screen, card, d, i, cx, cy, hovered)
		} else {
			drawVariantCard(screen, card, cx, cy, hovered)
		}
	}

	// Hover tooltip
	if d.HoverIdx >= 0 && d.HoverIdx < len(d.Cards) {
		drawBuildCardTooltip(screen, d.Cards[d.HoverIdx], m)
	}
}

// drawBuildableCard renders a normal buildable tower card.
func drawBuildableCard(screen *ebiten.Image, card BuildCardVM, d BuildMenuData, i int, cx, cy float32, hovered bool) {
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

	// Card background
	ui.Panel(screen, cx, cy, bpCardW, bpCardH, ui.PanelStyle{
		BgColor: cardBg, Radius: bpCardR,
	})

	// Selected card border
	if selected {
		ui.Panel(screen, cx, cy, bpCardW, bpCardH, ui.PanelStyle{
			BgColor: color.RGBA{A: 0}, BorderColor: theme.BuildCardSelBorder, Radius: bpCardR, BorderWidth: 2,
		})
	}

	nameX := float64(cx) + 8
	nameY := float64(cy) + 6
	nameMaxW := float64(bpCardW) - 16

	// Tower name (bold)
	ui.Label(screen, card.Label, nameX, nameY, 60, ui.LabelStyle{
		Font: theme.FontMD, Bold: true,
	})

	// Cost
	costTxt := strconv.Itoa(card.Cost) + "G"
	costClr := theme.BuildCostColor
	if !affordable {
		costClr = color.RGBA{R: 200, G: 80, B: 80, A: 200}
	}
	ui.Label(screen, costTxt, nameX, nameY+16, 60, ui.LabelStyle{
		Font: theme.FontSM, Color: costClr,
	})

	// Tower-type icon badge
	if card.TypeIcon != "" {
		if im := render.GlobalIcons(); im != nil {
			if img := im.Get(card.TypeIcon); img != nil {
				badgeX := float64(cx) + float64(bpCardW) - 16
				badgeY := float64(cy) + 4
				draw.Sprite(screen, img, badgeX, badgeY+6, 12) //nolint:hud
			}
		}
	}

	// Role tag
	ui.Label(screen, card.RoleTag, nameX, float64(cy)+float64(bpCardH)-16, nameMaxW, ui.LabelStyle{
		Font: theme.FontXS, Color: card.RoleColor,
	})

	// Sprite preview
	if card.Sprite != nil {
		spriteX := float64(cx) + float64(bpCardW) - 26
		spriteY := float64(cy) + float64(bpCardH)/2
		draw.Sprite(screen, card.Sprite, spriteX, spriteY, 36) //nolint:hud
	}
}

// drawVariantCard renders a display-only attack style variant card.
func drawVariantCard(screen *ebiten.Image, card BuildCardVM, cx, cy float32, hovered bool) {
	cardBg := theme.BuildCardVariant
	if hovered {
		cardBg = color.RGBA{R: 40, G: 55, B: 80, A: 160}
	}

	// Card background + subtle border
	ui.Panel(screen, cx, cy, bpCardW, bpCardH, ui.PanelStyle{
		BgColor: cardBg, BorderColor: color.RGBA{R: 100, G: 120, B: 160, A: 80}, Radius: bpCardR,
	})

	nameX := float64(cx) + 8
	nameY := float64(cy) + 6
	nameMaxW := float64(bpCardW) - 16

	// Tower name (bold)
	ui.Label(screen, card.Label, nameX, nameY, 60, ui.LabelStyle{
		Font: theme.FontMD, Bold: true, Color: color.RGBA{R: 180, G: 200, B: 230, A: 220},
	})

	// Ability type label (short, e.g. "弹射", "散射")
	ui.Label(screen, card.RoleTag, nameX, nameY+18, nameMaxW, ui.LabelStyle{
		Font: theme.FontXS, Color: card.RoleColor,
	})

	// "选择能力后" tag at bottom-left
	ui.Label(screen, i18n.T("hud.build.after_ability"), nameX, float64(cy)+float64(bpCardH)-16, nameMaxW, ui.LabelStyle{
		Font: theme.FontXS, Color: color.RGBA{R: 120, G: 140, B: 170, A: 140},
	})

	// Sprite preview (right side, same as buildable cards)
	if card.Sprite != nil {
		spriteX := float64(cx) + float64(bpCardW) - 26
		spriteY := float64(cy) + float64(bpCardH)/2
		draw.Sprite(screen, card.Sprite, spriteX, spriteY, 36) //nolint:hud
	}
}

// drawBuildCardTooltip renders a small stats tooltip above the build panel.
// 有预设能力时动态增高，展示每个能力的图标+标签+数值描述。
func drawBuildCardTooltip(screen *ebiten.Image, card BuildCardVM, m buildPanelMetrics) {
	if !card.Buildable {
		drawVariantTooltip(screen, card, m)
		return
	}

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		tipW       = float32(280)
		tipBaseH   = float32(70) // 基础高度（名称+属性行）
		tipAbilGap = float32(6)  // 能力区域上方间距
		tipR       = float32(8)
		tipPadX    = float32(12)
	)

	// 动态计算高度：每个能力行用 measureAbilityRowHeight 获取实际换行高度
	tipH := tipBaseH
	contentW := float64(tipW) - float64(tipPadX)*2
	if len(card.Abilities) > 0 {
		tipH += tipAbilGap
		for _, ab := range card.Abilities {
			tipH += float32(measureAbilityRowHeight(fm, ab, contentW))
		}
	}

	tipX := (float32(theme.CanvasW) - tipW) / 2
	tipY := m.panelY - tipH - 6

	// Tooltip background + border
	ui.Panel(screen, tipX, tipY, tipW, tipH, ui.PanelStyle{
		BgColor: theme.PanelBg, BorderColor: theme.PanelBorder, Radius: tipR,
	})

	tx := float64(tipX) + float64(tipPadX)
	ty := float64(tipY) + 8

	// Tower name
	ui.Label(screen, card.Label, tx, ty, 240, ui.LabelStyle{
		Font: theme.FontLG, Bold: true, Color: theme.TextTitle,
	})

	// Role tag + cost
	ui.Label(screen, card.RoleTag+" · "+strconv.Itoa(card.Cost)+"G", tx, ty+18, contentW, ui.LabelStyle{
		Font: theme.FontSM, Color: card.RoleColor,
	})

	ty += 38
	im := render.GlobalIcons()
	const tipIconSz = 12.0
	const tipIconGap = 4.0

	// Stat icons + values
	drawStatIcon(screen, im, "stat-damage", tx, ty, tipIconSz)
	ui.Label(screen, strconv.FormatFloat(card.Damage, 'f', 0, 64), tx+tipIconSz+tipIconGap, ty, 40, ui.LabelStyle{
		Font: theme.FontSM, Color: theme.InfoAttrDamage,
	})

	drawStatIcon(screen, im, "stat-atkspd", tx+60, ty, tipIconSz)
	ui.Label(screen, strconv.FormatFloat(1.0/card.AttackSpeed, 'f', 2, 64)+"s", tx+60+tipIconSz+tipIconGap, ty, 60, ui.LabelStyle{
		Font: theme.FontSM, Color: theme.InfoAttrAtkSpd,
	})

	drawStatIcon(screen, im, "stat-range", tx+140, ty, tipIconSz)
	ui.Label(screen, strconv.FormatFloat(card.Range, 'f', 0, 64), tx+140+tipIconSz+tipIconGap, ty, 40, ui.LabelStyle{
		Font: theme.FontSM, Color: theme.InfoAttrRange,
	})

	// 预设能力描述（使用 DrawSegmentsWrapped 自动换行，防止溢出右边界）
	if len(card.Abilities) > 0 {
		ty += float64(tipAbilGap) + 14
		for _, ab := range card.Abilities {
			// Icon
			if ab.Icon != "" {
				drawStatIcon(screen, im, ab.Icon, tx, ty, 14)
			}
			abX := tx + 19.0

			// Label (bold, 自动缩放)
			ui.Label(screen, ab.Label, abX, ty, 120, ui.LabelStyle{
				Font: theme.FontSM, Color: theme.TextBody, Bold: true,
			})
			labelW := fm.MeasureText(ab.Label, theme.FontSM) + 4
			if labelW > 124 {
				labelW = 124
			}
			segX := abX + labelW

			// Segments — 换行渲染
			if len(ab.Segments) > 0 {
				segs := abilitySegsToTextSegs(ab.Segments)
				remainW := contentW - 19.0 - labelW
				if remainW < 40 {
					remainW = 40
				}
				ui.DrawSegmentsWrapped(screen, fm, segs, segX, ty, remainW, theme.FontSM)
			} else if ab.Fallback != "" {
				ui.Label(screen, ab.Fallback, segX, ty+1, 0, ui.LabelStyle{
					Font: theme.FontSM, Color: theme.TextMuted,
				})
			}

			ty += measureAbilityRowHeight(fm, ab, contentW)
		}
	}
}

// drawVariantTooltip renders a tooltip for display-only variant cards.
func drawVariantTooltip(screen *ebiten.Image, card BuildCardVM, m buildPanelMetrics) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	const (
		tipPad = float32(12)
		tipR   = float32(8)
	)

	// 测量描述文本宽度，自适应 tooltip 宽度
	descW := fm.MeasureText(card.AbilityDesc, theme.FontXS)
	labelW := fm.MeasureText(card.Label, theme.FontLG)
	contentW := descW
	if labelW > contentW {
		contentW = labelW
	}
	tipW := float32(contentW) + tipPad*2 + 4
	if tipW < 200 {
		tipW = 200
	}
	if tipW > float32(theme.CanvasW)-40 {
		tipW = float32(theme.CanvasW) - 40
	}
	tipH := float32(48)
	tipX := (float32(theme.CanvasW) - tipW) / 2
	tipY := m.panelY - tipH - 6

	// Tooltip background + border
	ui.Panel(screen, tipX, tipY, tipW, tipH, ui.PanelStyle{
		BgColor: theme.PanelBg, BorderColor: theme.PanelBorder, Radius: tipR,
	})

	tx := float64(tipX) + float64(tipPad)
	ty := float64(tipY) + 8
	maxW := float64(tipW) - float64(tipPad)*2

	// Title
	ui.Label(screen, card.Label, tx, ty, 200, ui.LabelStyle{
		Font: theme.FontLG, Bold: true, Color: theme.TextTitle,
	})

	// Description
	ui.Label(screen, card.AbilityDesc, tx, ty+20, maxW, ui.LabelStyle{
		Font: theme.FontXS, Color: theme.TextMuted,
	})
}

// BuildMenuHitTest returns the buildable tower card index hit by (px, py).
// totalCount is total cards (for panel sizing), buildableCount is the clickable subset.
// Returns: ≥0 = buildable card index, -1 = outside panel, -2 = close button, -3 = inside panel but no card hit.
func BuildMenuHitTest(px, py float32, totalCount, buildableCount int) int {
	if totalCount <= 0 {
		return -1
	}

	m := calcBuildPanelMetrics(totalCount)

	// Quick panel bounds check
	if px < m.panelX || px > m.panelX+m.panelW || py < m.panelY || py > m.panelY+m.panelH {
		return -1 // outside panel
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
	return -3 // inside panel but no buildable card hit (e.g. variant card, padding)
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
