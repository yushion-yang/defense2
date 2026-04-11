// stage_info_vm.go — Builds InfoPanelVM from tower + strength data.
// All data computation that was previously inside hud/info_panel.go
// (fmtAttr, scaledColor, attackStyleLabel, strengthBreakdown, ability template parsing)
// now lives here, keeping the HUD layer purely rendering.
package scene

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/render/hud"
	"defense2/internal/render/theme"
)

// BuildInfoPanelVM constructs an InfoPanelVM from a tower and sell value.
// If t is nil, returns a VM with Visible=false.
func BuildInfoPanelVM(t *tower.Tower, sellValue int, wavesCleared int, testMode bool) hud.InfoPanelVM {
	if t == nil {
		return hud.InfoPanelVM{Visible: false}
	}

	sd := t.Strength
	var effStr float64
	if sd != nil {
		effStr = sd.Effective()
	}

	vm := hud.InfoPanelVM{
		Visible: true,
		Label:   t.Label,
	}

	// Strength title
	if sd != nil {
		vm.StrengthColor = strengthColor(effStr)
		strTxt := fmt.Sprintf("强度%.0f", effStr)
		breakdown := strengthBreakdown(sd)
		if breakdown != "" {
			strTxt += " " + breakdown
		}
		vm.StrengthText = strTxt
	}

	// Attribute segments (colored base + scaled + total + aura bonus from BuffList)
	var pctDamage, pctSpeed, flatRange float64
	if t.Buffs != nil {
		pctDamage = t.Buffs.SumByID("aura:damageAmp")
		pctSpeed = t.Buffs.SumByID("aura:pctSpeed")
		flatRange = t.Buffs.SumByID("aura:flatRange")
	}
	vm.DamageSegs = buildAttrSegsWithMods("%.0f", t.BaseDamage, t.PotentialDamage, effStr, pctDamage, 0)
	vm.SpeedSegs = buildAttrSegsWithMods("%.2f", t.BaseSpeed, t.PotentialSpeed, effStr, pctSpeed, 0)
	vm.RangeSegs = buildAttrSegsWithMods("%.0f", t.BaseRange, t.PotentialRange, effStr, 0, flatRange)

	// Attack style
	style := t.AttackStyleID
	if style == "" {
		style = "projectile"
	}
	vm.AttackStyleText = "攻击: " + attackStyleLabel(style)

	// Slot display — 只显示已选能力和待选的槽位，未解锁/空闲的不占位
	abTable := config.GlobalAbilityTable()
	for i := 0; i < len(t.UnlockOrder); i++ {
		cat := t.UnlockOrder[i]
		abilType := t.AbilitySlots[cat]
		hasPending := t.PendingChoices != nil && len(t.PendingChoices[cat]) > 0 && abilType == ""
		if abilType == "" && !hasPending {
			continue // 跳过未选且无待选的槽位
		}
		slot := hud.SlotVM{
			CategoryIdx:  cat,
			CategoryName: tower.CategoryName(cat),
			Unlocked:     true,
			HasPending:   hasPending,
		}
		if abilType != "" {
			slot.AbilityLabel = abilType
			if def, ok := abTable[abilType]; ok {
				slot.AbilityLabel = def.Label
				slot.AbilityIcon = def.Icon
			}
		}
		vm.Slots = append(vm.Slots, slot)
	}

	// Abilities (已获取的能力详细描述)
	for _, abType := range t.Abilities {
		vm.Abilities = append(vm.Abilities, buildAbilityVM(abType, abTable, effStr))
	}

	// Buffs — skip aura buffs (CatAura) which refresh each frame and would flicker
	for _, b := range t.Buffs.Active() {
		if b.Category == buff.CatAura {
			continue
		}
		vm.Buffs = append(vm.Buffs, hud.BuffVM{
			Source:    b.Source,
			Desc:      buffLabel(b.ID),
			Remaining: b.Remaining,
		})
	}

	// Pending ability count
	if testMode {
		// 测试模式：显示空槽数（不依赖 PendingChoices 缓存）
		for _, a := range t.AbilitySlots {
			if a == "" {
				vm.PendingCount++
			}
		}
	} else {
		vm.PendingCount = tower.PendingCount(t)
	}

	// Buttons
	vm.UpgradeButtonText = fmt.Sprintf("强度+10 $%d", tower.StrengthBuyCost())
	vm.SellButtonText = fmt.Sprintf("卖%d", sellValue)

	return vm
}

// ---------------------------------------------------------------------------
// Data computation helpers (moved from hud/info_panel.go)
// ---------------------------------------------------------------------------

// strengthColor returns the color for the strength display value.
func strengthColor(effStr float64) color.Color {
	if effStr > 100.5 {
		return theme.StatusStrUp
	}
	if effStr < 99.5 {
		return theme.StatusStrDown
	}
	return theme.StatusStrNorm
}

// strengthBreakdown generates the breakdown text, e.g. "↑(永+50 链+20)".
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

// buildAttrSegsWithMods builds attribute display with pct/flat modifier bonus.
func buildAttrSegsWithMods(numFmt string, base, potential, effStr, pctMod, flatMod float64) []hud.AbilitySegment {
	hasMods := pctMod > 0.001 || pctMod < -0.001 || flatMod > 0.005 || flatMod < -0.005
	if potential == 0 && !hasMods {
		return []hud.AbilitySegment{
			{Text: fmt.Sprintf(numFmt, base), Kind: "base"},
		}
	}
	ratio := effStr / 100.0
	scaled := potential * ratio
	baseTotal := base + scaled

	segs := []hud.AbilitySegment{
		{Text: fmt.Sprintf(numFmt+"+", base), Kind: "base"},
		{Text: fmt.Sprintf("("+numFmt+")", scaled), Kind: "scaled", Color: scaledColor(scaled, potential)},
		{Text: fmt.Sprintf("→"+numFmt, baseTotal), Kind: "total"},
	}
	if hasMods {
		// 绿色加成量: 8+(12)→20+3
		bonus := baseTotal*(1+pctMod) + flatMod - baseTotal
		if bonus > 0.005 {
			segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("+"+numFmt, bonus), Kind: "aura"})
		} else if bonus < -0.005 {
			segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf(numFmt, bonus), Kind: "aura"})
		}
	}
	return segs
}

// attackStyleLabel returns the Chinese label for an attack style ID.
func attackStyleLabel(style string) string {
	labels := map[string]string{
		"projectile": "投射物",
		"wideBeam":   "宽光束",
		"scatter":    "散射",
		"spin_aoe":   "旋转AoE",
		"radial":     "环射",
	}
	if l, ok := labels[style]; ok {
		return l
	}
	return style
}

// buffLabels maps raw buff IDs to player-friendly Chinese labels.
var buffLabels = map[string]string{
	"aura:damageAmp": "增伤光环",
	"aura:pctDamage": "百分比伤害光环",
	"aura:pctSpeed":  "攻速光环",
	"aura:flatRange": "射程光环",
	"aura:crit":      "暴击光环",
	"towerStrength":  "强度增益",
	"stealth":        "隐身",
	"berserk":        "狂暴",
	"regen":          "再生",
	"healAura":       "治疗光环",
	"speedAura":      "加速光环",
}

// buffLabel returns a player-friendly label for a buff ID.
func buffLabel(id string) string {
	if label, ok := buffLabels[id]; ok {
		return label
	}
	return id // fallback to raw ID
}

// scaledColor returns the color based on comparison of scaled value vs potential.
func scaledColor(scaled, potential float64) color.Color {
	const eps = 0.01
	diff := scaled - potential
	if diff > eps {
		return theme.StatusStrUp
	}
	if diff < -eps {
		return theme.StatusStrDown
	}
	return theme.TextBody
}

// needsDecimal returns true if the value requires a decimal place for display.
func needsDecimal(v float64) bool {
	return math.Abs(v-math.Round(v)) > 0.05
}

// fmtNum formats a number: integers without decimal, non-integers with one decimal.
func fmtNum(v float64) string {
	if needsDecimal(v) {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.0f", v)
}

// isPercentCapped 判断维度是否应在展示层 clamp 到 100%。
// chance（概率）和 factor（减速因子）超过 100% 语义不成立。
func isPercentCapped(scaleDim string) bool {
	return scaleDim == "chance" || scaleDim == "factor"
}

// ---------------------------------------------------------------------------
// Ability VM builder
// ---------------------------------------------------------------------------

// fallback maps for abilities not in AbilityTable.
var (
	fallbackIconMap = map[string]string{
		"multishot":   "multishot",
		"pulse":       "pulse",
		"multiTarget": "multishot",
	}
	fallbackLabelMap = map[string]string{
		"multishot": "多重射击",
		"pulse":     "脉冲",
	}
	fallbackDescMap = map[string]string{
		"multishot": "多重射击",
		"pulse":     "脉冲",
	}
)

// buildAbilityVM builds an AbilityVM for one ability type.
func buildAbilityVM(abilityType string, abTable config.AbilityTable, effStr float64) hud.AbilityVM {
	def := abTable[abilityType]

	// Resolve icon
	iconName := ""
	if def != nil {
		iconName = def.Icon
	} else if name, ok := fallbackIconMap[abilityType]; ok {
		iconName = name
	}

	// Resolve label
	label := abilityType
	if def != nil {
		label = def.Label
	} else if l, ok := fallbackLabelMap[abilityType]; ok {
		label = l
	}

	vm := hud.AbilityVM{
		Icon:  iconName,
		Label: label,
	}

	// No AbilityDef → use fallback description
	if def == nil {
		if desc, ok := fallbackDescMap[abilityType]; ok {
			vm.Fallback = desc
		}
		return vm
	}

	// Parse Display template into segments
	vm.Segments = buildAbilitySegments(def, effStr)
	return vm
}

// FormatAbilityDisplay replaces template placeholders with actual values for plain text display.
// Used by the choice panel to show player-friendly descriptions (total values only, no base+scaled breakdown).
func FormatAbilityDisplay(def *config.AbilityDef, effStr float64) string {
	tpl := def.Display
	if tpl == "" {
		return ""
	}
	scaled := def.Potential * (effStr / 100.0)
	total := def.Base + scaled
	// 概率/减速类展示 clamp 到 100%（实际运行时由 ApplyXxx 各自处理上限）
	displayTotal := total
	if isPercentCapped(def.ScaleDim) && displayTotal > 1 {
		displayTotal = 1
	}

	var b strings.Builder
	i := 0
	for i < len(tpl) {
		next := strings.Index(tpl[i:], "{")
		if next < 0 {
			b.WriteString(tpl[i:])
			break
		}
		b.WriteString(tpl[i : i+next])
		i += next
		end := strings.Index(tpl[i:], "}")
		if end < 0 {
			break
		}
		ph := tpl[i+1 : i+end]
		i += end + 1
		switch ph {
		case "s%":
			b.WriteString(fmt.Sprintf("%.0f%%", displayTotal*100))
		case "s":
			b.WriteString(fmtNum(total))
		case "si":
			b.WriteString(fmt.Sprintf("%.0f", math.Floor(total)))
		case "p":
			b.WriteString(fmtNum(def.Param))
		case "p%":
			b.WriteString(fmtNum(def.Param*100) + "%")
		}
	}
	return b.String()
}

// buildAbilitySegments parses a Display template and produces rendering segments.
// Template placeholders: {s}, {s%}, {si}, {sh%}, {p}, {p%}
func buildAbilitySegments(def *config.AbilityDef, effStr float64) []hud.AbilitySegment {
	tpl := def.Display
	if tpl == "" {
		return nil
	}

	scaled := def.Potential * (effStr / 100.0)
	total := def.Base + scaled
	// 概率/减速类展示 clamp 到 100%
	displayTotal := total
	if isPercentCapped(def.ScaleDim) && displayTotal > 1 {
		displayTotal = 1
	}
	sClr := scaledColor(scaled, def.Potential)

	var segs []hud.AbilitySegment
	i := 0
	for i < len(tpl) {
		next := strings.Index(tpl[i:], "{")
		if next < 0 {
			segs = append(segs, hud.AbilitySegment{Text: tpl[i:], Kind: "text"})
			break
		}
		if next > 0 {
			segs = append(segs, hud.AbilitySegment{Text: tpl[i : i+next], Kind: "text"})
		}
		i += next

		end := strings.Index(tpl[i:], "}")
		if end < 0 {
			break
		}
		ph := tpl[i+1 : i+end]
		i += end + 1

		switch ph {
		case "s%":
			if def.Base == 0 {
				// base=0 时只显示缩放值
				dv := scaled
				if isPercentCapped(def.ScaleDim) && dv > 1 {
					dv = 1
				}
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f%%", dv*100), Kind: "scaled", Color: sClr})
			} else {
				segs = append(segs,
					hud.AbilitySegment{Text: fmt.Sprintf("%.0f%%+", def.Base*100), Kind: "base"},
					hud.AbilitySegment{Text: fmt.Sprintf("(%.0f%%)", scaled*100), Kind: "scaled", Color: sClr},
					hud.AbilitySegment{Text: fmt.Sprintf("→%.0f%%", displayTotal*100), Kind: "total"},
				)
			}
		case "s":
			nf := "%.0f"
			if needsDecimal(def.Base) || needsDecimal(scaled) || needsDecimal(total) {
				nf = "%.1f"
			}
			if def.Base == 0 {
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf(nf, scaled), Kind: "scaled", Color: sClr})
			} else {
				segs = append(segs,
					hud.AbilitySegment{Text: fmt.Sprintf(nf+"+", def.Base), Kind: "base"},
					hud.AbilitySegment{Text: fmt.Sprintf("("+nf+")", scaled), Kind: "scaled", Color: sClr},
					hud.AbilitySegment{Text: fmt.Sprintf("→"+nf, total), Kind: "total"},
				)
			}
		case "si":
			if def.Base == 0 {
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f", math.Floor(scaled)), Kind: "scaled", Color: sClr})
			} else {
				segs = append(segs,
					hud.AbilitySegment{Text: fmt.Sprintf("%.0f+", math.Floor(def.Base)), Kind: "base"},
					hud.AbilitySegment{Text: fmt.Sprintf("(%.0f)", math.Floor(scaled)), Kind: "scaled", Color: sClr},
					hud.AbilitySegment{Text: fmt.Sprintf("→%.0f", math.Floor(total)), Kind: "total"},
				)
			}
		case "sh%":
			// {sh%} = 缩放值减半的百分比（用于 enhance 射程减半提升）
			half := total / 2
			segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f%%", half*100), Kind: "base"})
		case "p":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param), Kind: "text"})
		case "p%":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param*100) + "%", Kind: "text"})
		}
	}
	return segs
}
