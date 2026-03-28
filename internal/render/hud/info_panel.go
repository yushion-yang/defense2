// info_panel.go — Bottom-center tower detail panel.
// Shows tower stats, current abilities, upgrade growth, and sell button when a tower is selected.
// Future ability unlocks are shown in a hover tooltip.
package hud

import (
	"fmt"
	"math"

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
		panelW   = float32(theme.CenterPanelW)
		innerPad = float32(theme.CenterPanelInnerPad)
		titleH   = float32(theme.DetailTitleH)
		attrH    = float32(theme.DetailAttrH)
		abilityH = float32(theme.DetailRowH)
		btnH     = float32(theme.DetailBtnH)
		topPad   = float32(theme.DetailTopPad)
		botPad   = float32(theme.DetailBotPad)
		detailGap = float32(theme.DetailGap)
		btnGap   = float32(8)
	)

	hasUpgrade := t.Level < tower.MaxTowerLevel

	// Collect current (unlocked) abilities
	currentAbilities, _ := splitAbilities(t)

	// --- Build FlexPanel (content-driven height) ---
	panel := ui.NewFlexPanel(0, 0, panelW, innerPad)
	panel.BgColor = theme.PanelBg
	panel.Border = theme.InfoBorder
	panel.Radius = float32(theme.CenterPanelRadius)
	panel.AddSpace(topPad - innerPad)

	// 提取战力数据（从 Tower 的 interface{} 字段做类型断言）
	sd, _ := t.Strength.(*strength.StrengthData)
	cfg, _ := t.StrengthCfg.(*strength.StrengthConfig)
	var effStr float64
	if sd != nil {
		effStr = sd.Effective()
	}

	// Row 1: 塔名 + 等级 + 战力显示
	panel.AddRow(titleH, func(screen *ebiten.Image, x, y float64, w float64) {
		fm.DrawBoldText(screen, t.Label, x, y, theme.FontLG, theme.TextTitle)
		afterName := x + fm.MeasureText(t.Label, theme.FontLG) + 8
		levelTxt := fmt.Sprintf("Lv.%d", t.Level)
		fm.DrawText(screen, levelTxt, afterName, y+2, theme.FontSM, theme.StatusSkill)

		rightX := x + w

		// 右侧：战力数值 + 加成分解
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
		} else if hasUpgrade {
			upgCost := t.UpgradeCost()
			upgTxt := fmt.Sprintf("[U] 升级 $%d", upgCost)
			fm.DrawRightText(screen, upgTxt, rightX, y+2, theme.FontSM, theme.StatusWarden)
		} else {
			fm.DrawRightText(screen, "MAX", rightX, y+2, theme.FontSM, theme.StatusStrUp)
		}
	})

	// Row 2: 属性行（伤害/攻速/射程/DPS）
	panel.AddRow(attrH, func(screen *ebiten.Image, x, y float64, w float64) {
		colW := w / 4
		const (
			iconSize = 14.0
			iconGap  = 4.0
		)
		im := render.GlobalIcons()
		textOff := iconSize + iconGap

		// 伤害
		drawStatIcon(screen, im, "stat-damage", x, y, iconSize)
		if cfg != nil {
			fm.DrawText(screen, formatStrengthStat(cfg, "attackDamage", effStr, t.Damage), x+textOff, y, theme.FontMD, theme.InfoAttrDamage)
		} else {
			fm.DrawText(screen, formatStatShort(t.BaseDamage, t.Damage), x+textOff, y, theme.FontMD, theme.InfoAttrDamage)
		}

		// 攻速
		drawStatIcon(screen, im, "stat-atkspd", x+colW, y, iconSize)
		if cfg != nil {
			fm.DrawText(screen, formatStrengthSpeed(cfg, effStr, t.AttackSpeed), x+colW+textOff, y, theme.FontMD, theme.InfoAttrAtkSpd)
		} else {
			fm.DrawText(screen, formatStatShort(t.BaseSpeed, t.AttackSpeed), x+colW+textOff, y, theme.FontMD, theme.InfoAttrAtkSpd)
		}

		// 射程
		drawStatIcon(screen, im, "stat-range", x+colW*2, y, iconSize)
		if cfg != nil {
			fm.DrawText(screen, formatStrengthStat(cfg, "range", effStr, t.Range), x+colW*2+textOff, y, theme.FontMD, theme.InfoAttrRange)
		} else {
			fm.DrawText(screen, fmt.Sprintf("%.0f", t.Range), x+colW*2+textOff, y, theme.FontMD, theme.InfoAttrRange)
		}

		// DPS
		drawStatIcon(screen, im, "stat-dps", x+colW*3, y, iconSize)
		fm.DrawText(screen, fmt.Sprintf("%.1f", t.DPS()), x+colW*3+textOff, y, theme.FontMD, theme.StatusStrUp)
	})

	// Row 2b: Remaining upgrade growth (show all remaining levels' stat growth)
	if hasUpgrade {
		panel.AddRow(14, func(screen *ebiten.Image, x, y float64, w float64) {
			// Show max level stats as upgrade ceiling
			maxDmg := t.BaseDamage * (1 + 0.25*float64(tower.MaxTowerLevel-1))
			maxSpd := t.BaseSpeed * (1 + 0.10*float64(tower.MaxTowerLevel-1))
			maxRng := t.BaseRange * (1 + 0.08*float64(tower.MaxTowerLevel-1))

			colW := w / 4
			previewClr := theme.StatusStrUp
			im := render.GlobalIcons()
			const sz = 10.0
			off := sz + 3.0

			dmgGrowth := maxDmg - t.Damage
			spdGrowth := maxSpd - t.AttackSpeed
			rngGrowth := maxRng - t.Range

			drawStatIcon(screen, im, "stat-damage", x, y, sz)
			fm.DrawText(screen, fmt.Sprintf("+%.0f", dmgGrowth), x+off, y, theme.FontXS, previewClr)

			drawStatIcon(screen, im, "stat-atkspd", x+colW, y, sz)
			fm.DrawText(screen, fmt.Sprintf("+%.2f", spdGrowth), x+colW+off, y, theme.FontXS, previewClr)

			drawStatIcon(screen, im, "stat-range", x+colW*2, y, sz)
			fm.DrawText(screen, fmt.Sprintf("+%.0f", rngGrowth), x+colW*2+off, y, theme.FontXS, previewClr)

			maxDPS := maxDmg * maxSpd
			dpsGrowth := maxDPS - t.DPS()
			drawStatIcon(screen, im, "stat-dps", x+colW*3, y, sz)
			fm.DrawText(screen, fmt.Sprintf("+%.1f", dpsGrowth), x+colW*3+off, y, theme.FontXS, previewClr)
		})
	}

	// Row 3: 攻击方式
	panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
		style := t.AttackStyleID
		if style == "" {
			style = "projectile"
		}
		fm.DrawText(screen, "攻击: "+attackStyleLabel(style), x, y, theme.FontXS, theme.TextMuted)
	})

	// Row 4: 当前（已解锁）能力
	if len(currentAbilities) > 0 {
		for _, ab := range currentAbilities {
			ab := ab
			panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, w float64) {
				im := render.GlobalIcons()
				if iconName, ok := abilityIconMap[ab]; ok {
					drawStatIcon(screen, im, iconName, x, y, 12)
				}
				abX := x + 16.0
				fm.DrawBoldText(screen, abilityLabel(ab), abX, y, theme.FontXS, theme.TextBody)
				abX += fm.MeasureText(abilityLabel(ab), theme.FontXS) + 6

				// 战力缩放参数（如 "比率 20%+(60%)=80%"）
				if cfg != nil && sd != nil {
					if strDesc := abilityStrengthDesc(ab, cfg, effStr); strDesc != "" {
						fm.DrawText(screen, strDesc, abX, y+1, theme.FontXS, theme.StatusStrUp)
						abX += fm.MeasureText(strDesc, theme.FontXS) + 4
					} else if desc, ok := abilityDescMap[ab]; ok {
						fm.DrawText(screen, desc, abX, y+1, theme.FontXS, theme.TextMuted)
					}
				} else if desc, ok := abilityDescMap[ab]; ok {
					fm.DrawText(screen, desc, abX, y+1, theme.FontXS, theme.TextMuted)
				}
			})
		}
	}

	// Row 4b: 待解锁能力预览
	_, futureAbilities := splitAbilities(t)
	for _, fu := range futureAbilities {
		fu := fu
		panel.AddRow(abilityH, func(screen *ebiten.Image, x, y float64, _ float64) {
			fm.DrawText(screen, fmt.Sprintf("Lv%d: %s", fu.Level, fu.Name), x, y, theme.FontXS, theme.TextMuted)
		})
	}

	// Row 4: Action buttons
	panel.AddSpace(detailGap)
	lastUpgradeRect = ui.Rect{}
	lastSellRect = ui.Rect{}

	panel.AddRow(btnH, func(screen *ebiten.Image, x, y float64, w float64) {
		area := ui.Rect{X: float32(x), Y: float32(y), W: float32(w), H: btnH}

		if hasUpgrade {
			upgCost := t.UpgradeCost()
			upgTxt := fmt.Sprintf("升级 Lv.%d $%d", t.Level+1, upgCost)
			sellTxt := fmt.Sprintf("卖出 $%d", sellValue)

			result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
				{Label: upgTxt, Color: theme.TonePrimary},
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
		} else {
			sellTxt := fmt.Sprintf("卖出 $%d", sellValue)
			result := ui.DrawButtonRow(screen, area, []ui.ButtonRowItem{
				{Label: sellTxt, Color: theme.BtnDanger},
			}, ui.ButtonRowStyle{
				Height:   btnH,
				Gap:      btnGap,
				Radius:   float32(theme.ButtonRadius),
				FontSize: theme.FontMD,
			})
			if len(result.Rects) == 1 {
				lastSellRect = result.Rects[0]
			}
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

// splitAbilities separates tower abilities into current (unlocked) and future (locked) based on tower level.
func splitAbilities(t *tower.Tower) (current []string, future []tower.AbilityUnlock) {
	// If no unlock data, all abilities are current
	if len(t.AbilityUnlocks) == 0 {
		return t.Abilities, nil
	}

	// Build set of unlocked ability types (level <= current)
	unlockedSet := make(map[string]bool)
	for _, u := range t.AbilityUnlocks {
		if u.Level <= t.Level {
			unlockedSet[u.Type] = true
		} else {
			future = append(future, u)
		}
	}

	// Filter current abilities: keep those in unlocked set OR not in any unlock list
	// (abilities from bounceConfig etc. that aren't in abilityUnlocks are always shown)
	unlockTypes := make(map[string]bool)
	for _, u := range t.AbilityUnlocks {
		unlockTypes[u.Type] = true
	}
	for _, ab := range t.Abilities {
		if !unlockTypes[ab] || unlockedSet[ab] {
			current = append(current, ab)
		}
	}

	return current, future
}

// formatStatShort formats a stat as "base+(bonus)=total" or just "value".
func formatStatShort(base, current float64) string {
	bonus := current - base
	if bonus < 0.5 && bonus > -0.5 {
		return fmt.Sprintf("%.0f", current)
	}
	return fmt.Sprintf("%.0f+(%.0f)=%.0f", base, bonus, current)
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

// abilityIconMap 能力代码名 -> 图标名映射。
var abilityIconMap = map[string]string{
	"onHitSlow":       "slow",
	"stun":            "stun",
	"bounce":          "bounce",
	"burn":            "burn",
	"splash":          "stat-splash",
	"executionBonus":  "execute",
	"percentHpDamage": "hunterInstinct",
	"crit":            "heavyHit",
	"bleedDot":        "burn",
	"stackDamage":     "armorPen",
	"overload":        "thunder",
	"flatDamage":      "stat-damage",
	"distanceDamage":  "stat-range",
	"damageUpAura":    "tower-aura",
	"attackSpeedAura": "tower-aura",
	"rangeAura":       "tower-aura",
	"critAura":        "tower-aura",
	"soloBoost":       "heavyHit",
	"poisonZone":      "tower-poison",
	"silenceZone":     "slow",
	"curseZone":       "tower-poison",
	"channelLaser":    "tower-laser",
	"judgmentMark":    "tower-judicator",
	"deathMark":       "tower-summon",
	"buffPurge":       "stun",
	"root":            "stun",
	"shieldIgnore":    "armorPen",
	"goldPassive":     "stat-dps",
	"multishot":       "multishot",
	"pulse":           "pulse",
}

// abilityLabelMap 能力代码名 -> 显示标签。
var abilityLabelMap = map[string]string{
	"onHitSlow":       "减速",
	"stun":            "眩晕",
	"bounce":          "弹射",
	"burn":            "灼烧",
	"splash":          "溅射",
	"executionBonus":  "斩杀",
	"percentHpDamage": "百分比伤害",
	"crit":            "暴击",
	"bleedDot":        "流血",
	"stackDamage":     "叠伤",
	"overload":        "过载",
	"flatDamage":      "固伤",
	"distanceDamage":  "距离伤害",
	"damageUpAura":    "伤害光环",
	"attackSpeedAura": "攻速光环",
	"rangeAura":       "射程光环",
	"critAura":        "暴击光环",
	"soloBoost":       "独行加成",
	"poisonZone":      "毒区",
	"silenceZone":     "沉默区",
	"curseZone":       "诅咒区",
	"channelLaser":    "引导激光",
	"buffPurge":       "净化",
	"root":            "定身",
	"shieldIgnore":    "无视护盾",
	"judgmentMark":    "审判标记",
	"deathMark":       "死亡标记",
	"goldPassive":     "被动产金",
	"multishot":       "多重射击",
	"pulse":           "脉冲",
}

// abilityDescMap 能力简短描述。
var abilityDescMap = map[string]string{
	"onHitSlow":       "命中减速32%持续1.4s",
	"stun":            "12%概率眩晕0.4s",
	"bounce":          "弹射2次,衰减80%",
	"burn":            "灼烧30%伤害/2s",
	"splash":          "50px范围40%溅射",
	"executionBonus":  "≤50%HP时+50%伤害",
	"percentHpDamage": "额外20%最大HP(Boss5%)",
	"crit":            "25%概率+80%暴击",
	"bleedDot":        "流血5dps/3s",
	"stackDamage":     "每次命中+8%伤害",
	"overload":        "15%概率双倍伤害",
	"flatDamage":      "+5固定伤害",
	"distanceDamage":  "越远伤害越高+50%",
	"damageUpAura":    "周围塔+15%伤害",
	"attackSpeedAura": "周围塔+10%攻速",
	"rangeAura":       "周围塔+20px射程",
	"critAura":        "周围塔+10%暴击",
	"soloBoost":       "无邻塔时+30%伤害",
	"poisonZone":      "范围内3dps毒伤",
	"silenceZone":     "范围内减速20%",
	"curseZone":       "范围内1.5%HP/s",
	"channelLaser":    "持续光束10dps",
	"buffPurge":       "剥离护盾5/s",
	"root":            "定身0.5s",
	"shieldIgnore":    "伤害无视护盾",
	"goldPassive":     "每3s产1金币",
}

// abilityLabel 返回能力的显示标签，未找到则返回原始代码名。
func abilityLabel(code string) string {
	if label, ok := abilityLabelMap[code]; ok {
		return label
	}
	return code
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
	// 检查链加成
	if chain, ok := sd.Temp["chain"]; ok && chain > 0 {
		parts = append(parts, fmt.Sprintf("链+%.0f", chain))
	}
	// 其他临时加成
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

// formatStrengthStat 用战力绑定格式化普通属性: "base+(scaled)=total"。
// 如果该属性无绑定或 potential 为0，返回简单数值。
func formatStrengthStat(cfg *strength.StrengthConfig, path string, eff, fallback float64) string {
	if cfg == nil {
		return fmt.Sprintf("%.0f", fallback)
	}
	b := cfg.ResolveBinding(path)
	if b == nil || b.Potential == 0 {
		return fmt.Sprintf("%.0f", fallback)
	}
	scaled := b.Potential * (eff / 100.0)
	total := b.Base + scaled
	return fmt.Sprintf("%.0f+(%.0f)=%.0f", b.Base, scaled, total)
}

// formatStrengthSpeed 用战力绑定格式化攻速（反向公式）: "base+(scaled)=total s"。
// 攻速公式: base + potential * (100 / max(1, eff))，越高战力间隔越短。
func formatStrengthSpeed(cfg *strength.StrengthConfig, eff, fallback float64) string {
	if cfg == nil {
		return fmt.Sprintf("%.2f", fallback)
	}
	b := cfg.ResolveBinding("attackSpeed")
	if b == nil || b.Potential == 0 {
		return fmt.Sprintf("%.2f", fallback)
	}
	e := eff
	if e < 1 {
		e = 1
	}
	scaled := b.Potential * (100.0 / e)
	total := b.Base + scaled
	if total < 0.1 {
		total = 0.1
	}
	return fmt.Sprintf("%.2f+(%.2f)=%.2fs", b.Base, scaled, total)
}

// abilityStrengthDesc 返回能力的战力缩放参数描述。
// 对有 effects.* 绑定的能力，显示 "参数名 base%+(scaled%)=total%"。
func abilityStrengthDesc(abilityType string, cfg *strength.StrengthConfig, eff float64) string {
	// 能力类型 → (绑定路径, 参数显示名, 是否百分比)
	mapping, ok := abilityStrengthBindings[abilityType]
	if !ok {
		return ""
	}
	b := cfg.ResolveBinding(mapping.path)
	if b == nil || b.Potential == 0 {
		return ""
	}
	scaled := b.Potential * (eff / 100.0)
	total := b.Base + scaled
	if mapping.percent {
		return fmt.Sprintf("%s %.0f%%+(%.0f%%)=%.0f%%",
			mapping.label,
			b.Base*100, scaled*100, total*100)
	}
	return fmt.Sprintf("%s %.1f+(%.1f)=%.1f",
		mapping.label, b.Base, scaled, total)
}

// abilityStrengthBinding 能力战力绑定映射。
type abilityStrengthBinding struct {
	path    string // StrengthConfig 中的绑定路径
	label   string // 参数显示名
	percent bool   // 是否以百分比显示
}

// abilityStrengthBindings 已知的能力-战力绑定映射表。
var abilityStrengthBindings = map[string]abilityStrengthBinding{
	"percentHpDamage": {"effects.percentHp", "比率", true},
	"onHitSlow":       {"effects.slowFactor", "减速", true},
	"executionBonus":  {"effects.executionThreshold", "阈值", true},
	"splash":          {"effects.splashRadius", "范围", false},
	"burn":            {"effects.burnDps", "伤害", false},
	"bleedDot":        {"effects.bleedDps", "伤害", false},
	"stun":            {"effects.stunDuration", "时长", false},
	"bounce":          {"effects.bounceRange", "范围", false},
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

// 确保 math 包被使用（formatStrengthSpeed 中的计算）
var _ = math.Max

// DrawInfoPanelHoverTooltip draws the upgrade detail tooltip above the info panel
// when the mouse is hovering over the panel. Shows per-level stat growth and future ability unlocks.
func DrawInfoPanelHoverTooltip(screen *ebiten.Image, t *tower.Tower, mx, my float32) {
	if t == nil || !lastPanelVisible {
		return
	}
	// Only show when hovering the panel area
	if !lastPanelRect.Contains(float64(mx), float64(my)) {
		return
	}
	// Nothing to show if already max level
	if t.Level >= tower.MaxTowerLevel && len(t.AbilityUnlocks) == 0 {
		return
	}
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	_, futureUnlocks := splitAbilities(t)
	hasUpgrade := t.Level < tower.MaxTowerLevel

	// Nothing to show
	if !hasUpgrade && len(futureUnlocks) == 0 {
		return
	}

	const (
		tipW     = float32(280)
		tipR     = float32(8)
		tipPad   = float32(10)
		rowH     = float32(16)
		titleH   = float32(20)
		sectionGap = float32(6)
	)

	// Calculate content height
	rows := 0
	if hasUpgrade {
		rows++ // title "升级路径"
		remaining := tower.MaxTowerLevel - t.Level
		rows += remaining // one row per level
	}
	if len(futureUnlocks) > 0 {
		rows++ // title "待解锁能力"
		rows += len(futureUnlocks)
	}
	tipH := tipPad*2 + float32(rows)*rowH
	if hasUpgrade && len(futureUnlocks) > 0 {
		tipH += sectionGap // gap between sections
	}

	// Position: centered above the info panel
	tipX := lastPanelRect.X + (lastPanelRect.W-tipW)/2
	tipY := lastPanelRect.Y - tipH - 6

	// Background
	draw.RoundRect(screen, tipX, tipY, tipW, tipH, tipR, theme.PanelBg)
	draw.StrokeRoundRect(screen, tipX, tipY, tipW, tipH, tipR, 1, theme.PanelBorder)

	cx := float64(tipX) + float64(tipPad)
	cy := float64(tipY) + float64(tipPad)

	// Section 1: Per-level upgrade stats
	if hasUpgrade {
		fm.DrawBoldText(screen, "升级路径", cx, cy, theme.FontSM, theme.TextTitle)
		cy += float64(rowH)

		im := render.GlobalIcons()
		for lv := t.Level + 1; lv <= tower.MaxTowerLevel; lv++ {
			lvDmg := t.BaseDamage * (1 + 0.25*float64(lv-1))
			lvSpd := t.BaseSpeed * (1 + 0.10*float64(lv-1))
			lvRng := t.BaseRange * (1 + 0.08*float64(lv-1))
			lvDPS := lvDmg * lvSpd

			// Level label
			lvTxt := fmt.Sprintf("Lv.%d", lv)
			fm.DrawBoldText(screen, lvTxt, cx, cy, theme.FontXS, theme.StatusSkill)

			// Stats after this level
			off := cx + 36
			colW := (float64(tipW) - float64(tipPad)*2 - 36) / 4
			const sz = 9.0
			iconOff := sz + 2.0

			drawStatIcon(screen, im, "stat-damage", off, cy, sz)
			fm.DrawText(screen, fmt.Sprintf("%.0f", lvDmg), off+iconOff, cy, theme.FontXS, theme.InfoAttrDamage)

			drawStatIcon(screen, im, "stat-atkspd", off+colW, cy, sz)
			fm.DrawText(screen, fmt.Sprintf("%.2f", lvSpd), off+colW+iconOff, cy, theme.FontXS, theme.InfoAttrAtkSpd)

			drawStatIcon(screen, im, "stat-range", off+colW*2, cy, sz)
			fm.DrawText(screen, fmt.Sprintf("%.0f", lvRng), off+colW*2+iconOff, cy, theme.FontXS, theme.InfoAttrRange)

			drawStatIcon(screen, im, "stat-dps", off+colW*3, cy, sz)
			fm.DrawText(screen, fmt.Sprintf("%.1f", lvDPS), off+colW*3+iconOff, cy, theme.FontXS, theme.StatusStrUp)

			cy += float64(rowH)
		}
	}

	// Section 2: Future ability unlocks
	if len(futureUnlocks) > 0 {
		if hasUpgrade {
			cy += float64(sectionGap)
		}
		fm.DrawBoldText(screen, "待解锁能力", cx, cy, theme.FontSM, theme.TextTitle)
		cy += float64(rowH)

		for _, fu := range futureUnlocks {
			lvTxt := fmt.Sprintf("Lv.%d", fu.Level)
			fm.DrawText(screen, lvTxt, cx, cy, theme.FontXS, theme.StatusSkill)

			im := render.GlobalIcons()
			abX := cx + 36
			if iconName, ok := abilityIconMap[fu.Type]; ok {
				drawStatIcon(screen, im, iconName, abX, cy, 10)
				abX += 14
			}
			fm.DrawBoldText(screen, fu.Name, abX, cy, theme.FontXS, theme.TextBody)
			if desc, ok := abilityDescMap[fu.Type]; ok {
				fm.DrawText(screen, desc, abX+fm.MeasureText(fu.Name, theme.FontXS)+6, cy+1, theme.FontXS, theme.TextMuted)
			}
			cy += float64(rowH)
		}
	}
}

// InfoPanelUpgradeHitTest 检查是否点击了升级按钮。
func InfoPanelUpgradeHitTest(px, py float32, t *tower.Tower) bool {
	if t == nil || !lastPanelVisible || t.Level >= tower.MaxTowerLevel {
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
