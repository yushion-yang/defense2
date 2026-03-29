// info_panel.go — Bottom-center tower detail panel.
// Shows tower stats, current abilities, upgrade growth, and sell button when a tower is selected.
// Ability display is data-driven from config.AbilityTable with strength scaling.
package hud

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Cached button Rects — written by DrawInfoPanel, read by hit tests.
// ---------------------------------------------------------------------------

var (
	lastUpgradeRect  ui.Rect
	lastSellRect     ui.Rect
	lastPanelRect    ui.Rect // entire info panel bounding box
	lastPanelVisible bool
)

// DrawInfoPanel renders the tower information panel. Passing nil hides it.
func DrawInfoPanel(screen *ebiten.Image, t *tower.Tower, sellValue int) {
	if t == nil {
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
		btnGap    = float32(8)
	)

	currentAbilities := t.Abilities

	// --- Build FlexPanel (content-driven height) ---
	panel := ui.NewFlexPanel(0, 0, panelW, innerPad)
	panel.BgColor = theme.PanelBg
	panel.Border = theme.InfoBorder
	panel.Radius = float32(theme.CenterPanelRadius)
	panel.AddSpace(topPad - innerPad)

	sd := t.Strength
	var effStr float64
	if sd != nil {
		effStr = sd.Effective()
	}

	// Row 1: 塔名 + 战力显示
	panel.AddRow(titleH, func(screen *ebiten.Image, x, y float64, w float64) {
		fm.DrawBoldText(screen, t.Label, x, y, theme.FontLG, theme.TextTitle)
		rightX := x + w

		if sd != nil {
			strClr := theme.StatusStrNorm
			if effStr > 100 {
				strClr = theme.StatusStrUp
			} else if effStr < 100 {
				strClr = theme.StatusStrDown
			}
			strTxt := fmt.Sprintf("强度%.0f", effStr)
			breakdown := strengthBreakdown(sd)
			if breakdown != "" {
				strTxt += " " + breakdown
			}
			fm.DrawRightText(screen, strTxt, rightX, y+2, theme.FontSM, strClr)
		}
	})

	// Row 2: 属性行（伤害/攻速/射程/DPS）
	panel.AddRow(attrH, func(screen *ebiten.Image, x, y float64, w float64) {
		colW := w / 3
		const (
			iconSize = 14.0
			iconGap  = 4.0
		)
		im := render.GlobalIcons()
		textOff := iconSize + iconGap

		// 伤害
		drawStatIcon(screen, im, "stat-damage", x, y, iconSize)
		fm.DrawText(screen, fmtAttr("%.0f", t.BaseDamage, t.PotentialDamage, effStr), x+textOff, y, theme.FontMD, theme.InfoAttrDamage)

		// 攻速（精确到一位小数）
		drawStatIcon(screen, im, "stat-atkspd", x+colW, y, iconSize)
		fm.DrawText(screen, fmtAttr("%.1f", t.BaseSpeed, t.PotentialSpeed, effStr), x+colW+textOff, y, theme.FontMD, theme.InfoAttrAtkSpd)

		// 射程
		drawStatIcon(screen, im, "stat-range", x+colW*2, y, iconSize)
		fm.DrawText(screen, fmtAttr("%.0f", t.BaseRange, t.PotentialRange, effStr), x+colW*2+textOff, y, theme.FontMD, theme.InfoAttrRange)
	})

	// Row 3: 攻击方式
	panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
		style := t.AttackStyleID
		if style == "" {
			style = "projectile"
		}
		fm.DrawText(screen, "攻击: "+attackStyleLabel(style), x, y, theme.FontXS, theme.TextMuted)
	})

	// Row 4: 能力列表（数据驱动）
	abTable := config.GlobalAbilityTable()
	if len(currentAbilities) > 0 {
		for _, ab := range currentAbilities {
			ab := ab
			panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
				drawAbilityRow(screen, fm, ab, abTable, effStr, x, y)
			})
		}
	}

	// 操作按钮：强度+10 / 卖出
	panel.AddSpace(detailGap)
	lastUpgradeRect = ui.Rect{}
	lastSellRect = ui.Rect{}

	panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
		area := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: btnH}

		buyTxt := fmt.Sprintf("强度+10 $%d", tower.StrengthBuyCost)
		sellTxt := fmt.Sprintf("卖%d", sellValue)

		result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
			{Label: buyTxt, Color: theme.TonePrimary},
			{Label: sellTxt, Color: theme.BtnDanger},
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

// drawAbilityRow 渲染一行能力：图标 + 标签（粗体）+ 缩放值（带颜色）+ 固定参数。
func drawAbilityRow(screen *ebiten.Image, fm *render.FontManager, abilityType string, abTable config.AbilityTable, effStr float64, x, y float64) {
	im := render.GlobalIcons()
	def := abTable[abilityType]

	// 图标
	iconName := ""
	if def != nil {
		iconName = def.Icon
	} else if name, ok := fallbackIconMap[abilityType]; ok {
		iconName = name
	}
	if iconName != "" {
		drawStatIcon(screen, im, iconName, x, y, 12)
	}
	abX := x + 16.0

	// 标签（粗体）
	label := abilityType
	if def != nil {
		label = def.Label
	} else if l, ok := fallbackLabelMap[abilityType]; ok {
		label = l
	}
	fm.DrawBoldText(screen, label, abX, y, theme.FontXS, theme.TextBody)
	abX += fm.MeasureText(label, theme.FontXS) + 6

	// 无 AbilityDef 时显示 fallback 描述
	if def == nil {
		if desc, ok := fallbackDescMap[abilityType]; ok {
			fm.DrawText(screen, desc, abX, y+1, theme.FontXS, theme.TextMuted)
		}
		return
	}

	// 用 Display 模板渲染
	drawAbilityDisplay(screen, fm, def, effStr, abX, y+1)
}

// drawAbilityDisplay 按 Display 模板渲染能力描述。
// {s} → 缩放值 base+(scaled)=total（带颜色），{s%} → 百分比格式
// {p} → 参数原值，{p%} → 参数百分比
// 其余文本原样渲染。
func drawAbilityDisplay(screen *ebiten.Image, fm *render.FontManager, def *config.AbilityDef, effStr float64, x, y float64) {
	tpl := def.Display
	if tpl == "" {
		return
	}

	scaled := def.Potential * (effStr / 100.0)
	total := def.Base + scaled
	scaledClr := scaledColor(scaled, def.Potential)

	i := 0
	for i < len(tpl) {
		// 查找下一个占位符
		next := strings.Index(tpl[i:], "{")
		if next < 0 {
			// 剩余纯文本
			fm.DrawText(screen, tpl[i:], x, y, theme.FontXS, theme.TextMuted)
			x += fm.MeasureText(tpl[i:], theme.FontXS)
			break
		}

		// 输出占位符前的纯文本
		if next > 0 {
			seg := tpl[i : i+next]
			fm.DrawText(screen, seg, x, y, theme.FontXS, theme.TextMuted)
			x += fm.MeasureText(seg, theme.FontXS)
		}
		i += next

		// 解析占位符
		end := strings.Index(tpl[i:], "}")
		if end < 0 {
			break
		}
		ph := tpl[i+1 : i+end] // 占位符内容（如 "s%", "p"）
		i += end + 1

		switch ph {
		case "s%":
			// 缩放百分比：10%+(22%)=32%
			x = drawScaleSegment(screen, fm, x, y, scaledClr,
				fmt.Sprintf("%.0f%%+", def.Base*100),
				fmt.Sprintf("(%.0f%%)", scaled*100),
				fmt.Sprintf("=%.0f%%", total*100))
		case "s":
			// 缩放绝对值：2+(1)=3
			nf := "%.0f"
			if needsDecimal(def.Base) || needsDecimal(scaled) || needsDecimal(total) {
				nf = "%.1f"
			}
			x = drawScaleSegment(screen, fm, x, y, scaledClr,
				fmt.Sprintf(nf+"+", def.Base),
				fmt.Sprintf("("+nf+")", scaled),
				fmt.Sprintf("="+nf, total))
		case "p":
			// 参数原值
			txt := fmtNum(def.Param)
			fm.DrawText(screen, txt, x, y, theme.FontXS, theme.TextMuted)
			x += fm.MeasureText(txt, theme.FontXS)
		case "p%":
			// 参数百分比
			txt := fmtNum(def.Param*100) + "%"
			fm.DrawText(screen, txt, x, y, theme.FontXS, theme.TextMuted)
			x += fm.MeasureText(txt, theme.FontXS)
		}
	}
}

// drawScaleSegment 渲染 "base+(scaled)=total" 三段文本，中间段带颜色。
func drawScaleSegment(screen *ebiten.Image, fm *render.FontManager, x, y float64, scaledClr color.Color, baseTxt, scaledTxt, totalTxt string) float64 {
	fm.DrawText(screen, baseTxt, x, y, theme.FontXS, theme.TextBody)
	x += fm.MeasureText(baseTxt, theme.FontXS)
	fm.DrawText(screen, scaledTxt, x, y, theme.FontXS, scaledClr)
	x += fm.MeasureText(scaledTxt, theme.FontXS)
	fm.DrawText(screen, totalTxt, x, y, theme.FontXS, theme.TextBody)
	x += fm.MeasureText(totalTxt, theme.FontXS)
	return x
}

// fmtNum 格式化数字：整数不带小数点，非整数保留一位。
func fmtNum(v float64) string {
	if needsDecimal(v) {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.0f", v)
}

// scaledColor 根据缩放值与潜力值的比较返回颜色。
func scaledColor(scaled, potential float64) color.Color {
	const eps = 0.001
	diff := scaled - potential
	if diff > eps {
		return theme.StatusStrUp // 绿色：强度>100
	}
	if diff < -eps {
		return theme.StatusStrDown // 红色：强度<100
	}
	return theme.TextBody // 白色：强度=100
}

// needsDecimal 判断数值是否需要小数位显示。
func needsDecimal(v float64) bool {
	return math.Abs(v-math.Round(v)) > 0.05
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

// fallbackIconMap 不在 AbilityTable 中的能力的图标映射。
var fallbackIconMap = map[string]string{
	"shieldIgnore": "armorPen",
	"multishot":    "multishot",
	"pulse":        "pulse",
	"multiTarget":  "multishot",
}

// fallbackLabelMap 不在 AbilityTable 中的能力的显示标签。
var fallbackLabelMap = map[string]string{
	"shieldIgnore": "无视护盾",
	"multishot":    "多重射击",
	"pulse":        "脉冲",
}

// fallbackDescMap 不在 AbilityTable 中的能力的描述。
var fallbackDescMap = map[string]string{
	"shieldIgnore": "伤害无视护盾",
	"multishot":    "多重射击",
	"pulse":        "脉冲",
}

// ── 战力系统 HUD 辅助函数 ──

// strengthBreakdown 生成战力加成分解文本，如 "↑(永+50 链+20)"。
func strengthBreakdown(sd *strength.StrengthData) string {
	if sd == nil {
		return ""
	}
	var parts []string
	if sd.Permanent > 0 {
		parts = append(parts, fmt.Sprintf("永+%.0f", sd.Permanent))
	}
	if chain, ok := sd.Temp["chain"]; ok && chain > 0 {
		parts = append(parts, fmt.Sprintf("链+%.0f", chain))
	}
	for key, val := range sd.Temp {
		if key == "chain" || val <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("+%.0f", val))
	}
	if len(parts) == 0 {
		return ""
	}
	result := "↑("
	for i, p := range parts {
		if i > 0 {
			result += " "
		}
		result += p
	}
	result += ")"
	return result
}

// fmtAttr 格式化塔属性: "base+(scaled)=total"。
// potential 为 0 时只显示总值。
func fmtAttr(numFmt string, base, potential, effStr float64) string {
	if potential == 0 {
		return fmt.Sprintf(numFmt, base)
	}
	ratio := effStr / 100.0
	if effStr == 0 {
		ratio = 1.0 // 无强度数据时按100
	}
	scaled := potential * ratio
	total := base + scaled
	return fmt.Sprintf(numFmt+"+("+numFmt+")="+numFmt, base, scaled, total)
}

// attackStyleLabel 攻击方式中文标签。
func attackStyleLabel(style string) string {
	labels := map[string]string{
		"projectile": "投射物",
		"laser":      "激光",
		"wideBeam":   "宽光束",
		"scatter":    "散射",
		"charge":     "蓄力",
		"spin_aoe":   "旋转AoE",
		"pierce":     "穿刺",
		"aura_dot":   "范围毒伤",
	}
	if l, ok := labels[style]; ok {
		return l
	}
	return style
}

// DrawInfoPanelHoverTooltip 悬停面板时的提示（已无等级系统，保留接口兼容）。
func DrawInfoPanelHoverTooltip(screen *ebiten.Image, t *tower.Tower, mx, my float32) {
}

// InfoPanelUpgradeHitTest 检查是否点击了购买强度按钮。
func InfoPanelUpgradeHitTest(px, py float32, t *tower.Tower) bool {
	if t == nil || !lastPanelVisible {
		return false
	}
	return lastUpgradeRect.Contains(float64(px), float64(py))
}

// InfoPanelSellHitTest 检查是否点击了卖出按钮。
func InfoPanelSellHitTest(px, py float32, t *tower.Tower) bool {
	if t == nil || !lastPanelVisible {
		return false
	}
	return lastSellRect.Contains(float64(px), float64(py))
}
