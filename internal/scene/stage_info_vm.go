// stage_info_vm.go — 塔信息面板的 ViewModel 构建器。
//
// 设计模式: ViewModel（视图模型）
//
// 本文件是 scene 层与 hud 渲染层之间的数据桥梁。核心思路：
//   - hud 包只负责渲染（零 core 依赖），接收纯值 struct（InfoPanelVM）画面板
//   - 本文件负责从 core 游戏对象（Tower/Strength/Buff）提取数据，计算显示值，
//     格式化文本，最终组装成 InfoPanelVM 交给 hud 渲染
//
// 职责划分：
//
//	scene 层(本文件)          hud 层(info_panel.go)
//	──────────────           ─────────────────
//	读取 Tower 属性            接收 InfoPanelVM
//	计算 base+scaled→total    按 Segment 着色渲染
//	格式化 Strength 分解文本   绘制文字和按钮
//	解析能力模板 {s%}/{p}     布局排版
//	聚合光环 buff 摘要        处理 hitTest
//
// 这样做的好处：
//  1. hud 包可独立预览/测试（不需要真实的 Tower 对象）
//  2. 所有数据转换逻辑集中在一处，避免渲染代码中掺杂业务逻辑
//  3. 修改显示格式只需改本文件，不影响渲染层
package scene

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/i18n"
	"defense2/internal/render/hud"
	"defense2/internal/render/theme"
)

// BuildInfoPanelVM 从塔实例构建完整的信息面板 ViewModel。
//
// 这是本文件的主入口函数，由 StageScene 每帧（modeTowerSel 时）调用。
// 构建流程：
//  1. 计算有效强度（effStr）并生成强度分解文本（如 "↑(永+50 链+20)"）
//  2. 构建三大属性段（伤害/攻速/射程），每段包含 base+scaled→total+光环加成
//  3. 生成攻击方式标签（如 "攻击方式: 散射"）
//  4. 构建已选能力槽位列表（图标+标签+待选标记）
//  5. 构建已获取能力的详细描述（含分段着色的数值模板）
//  6. 聚合光环 buff 摘要（同 ID 的多来源 buff 合并显示）
//  7. 设置升级/卖出按钮的文本和可用状态
//
// 参数说明：
//   - t: 选中的塔（nil 时返回 Visible=false 的空 VM）
//   - sellValue: 卖出退款金额（由调用方根据经济规则计算）
//   - gold/upgradeCosts: 用于战役模式解锁槽位按钮的费用判断
func BuildInfoPanelVM(t *tower.Tower, sellValue int, wavesCleared int, testMode bool, gold int, upgradeCosts []int) hud.InfoPanelVM {
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

	// 强度标题行：颜色 + 数值 + 分解文本
	// 强度 > 100 显示绿色（增益），< 100 显示红色（减益），= 100 显示默认色
	if sd != nil {
		vm.StrengthColor = strengthColor(effStr)
		strTxt := i18n.TF("game.tower.strength_val", effStr)
		breakdown := strengthBreakdown(sd)
		if breakdown != "" {
			strTxt += " " + breakdown
		}
		vm.StrengthText = strTxt
	}

	// Specialty
	vm.Specialty = t.Specialty

	// ── 三大属性段构建 ──
	// 属性公式: 最终值 = (Base + Potential × Str/100) × (1 + pctMod) + flatMod
	// 显示格式: "8+(12)→20+3" = base+(scaled)→baseTotal+auraBonus
	// 颜色规则: base=白, scaled=绿(>potential)/红(<)/白(=), total=白, aura=绿
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
	vm.AttackStyleText = i18n.T("game.tower.attack_prefix") + attackStyleLabel(style)

	// ── 能力槽位显示 ──
	// 只显示已选能力和有待选项的槽位，未解锁/空闲的不占位。
	// UnlockOrder 决定了槽位的显示顺序（按解锁先后，非固定类别顺序）。
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

	// ── 已获取能力的详细描述 ──
	// 每个能力生成 AbilityVM，包含图标、标签和分段着色的数值描述。
	// 数值随 effStr 动态变化（如 "30%+(20%)→50%"），让玩家看到强度的实际影响。
	for _, abType := range t.Abilities {
		vm.Abilities = append(vm.Abilities, buildAbilityVM(abType, abTable, effStr))
	}

	// ── Buff 显示 ──
	// 光环 buff 每帧刷新（0.3s 短 buff），同 ID 的多来源 buff 聚合显示。
	// 例如 3 座塔的 damageAmp 光环合并为 "伤害提升 +45%（光环 ×3）"。
	// 非光环 buff（CC/DoT/行为等）直接列出，显示来源和剩余时间。
	type auraAgg struct {
		totalValue  float64
		sourceCount int
		id          string
	}
	auraMap := map[string]*auraAgg{}
	for _, b := range t.Buffs.Active() {
		if b.Category == buff.CatAura {
			agg, ok := auraMap[b.ID]
			if !ok {
				agg = &auraAgg{id: b.ID}
				auraMap[b.ID] = agg
			}
			agg.totalValue += b.Value
			agg.sourceCount++
			continue
		}
		vm.Buffs = append(vm.Buffs, hud.BuffVM{
			Source:    b.Source,
			Desc:      buffLabel(b.ID),
			Remaining: b.Remaining,
		})
	}
	// Append merged aura summaries (sorted by ID for stable display order)
	auraIDs := make([]string, 0, len(auraMap))
	for id := range auraMap {
		auraIDs = append(auraIDs, id)
	}
	slices.Sort(auraIDs)
	for _, id := range auraIDs {
		agg := auraMap[id]
		desc := buffLabel(agg.id)
		if isPercentBuff(agg.id) {
			desc += fmt.Sprintf(" +%.0f%%", agg.totalValue*100)
		} else {
			desc += fmt.Sprintf(" +%s", fmtNum(agg.totalValue))
		}
		src := i18n.T("game.buff.aura")
		if agg.sourceCount > 1 {
			src = i18n.TF("game.buff.aura_multi", agg.sourceCount)
		}
		vm.Buffs = append(vm.Buffs, hud.BuffVM{
			Source:    src,
			Desc:      desc,
			Remaining: -1, // permanent (refreshed each frame)
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

	// 战役模式：解锁能力槽位按钮
	if !testMode && tower.CanUnlockMore(t) {
		tempDef := tower.TowerDef{UpgradeCosts: upgradeCosts}
		vm.CanUnlockSlot = true
		vm.UnlockCost = tower.NextUpgradeCost(t, tempDef)
		vm.Gold = gold
	}

	// Buttons
	upgCost := tower.StrengthBuyCost()
	vm.UpgradeButtonText = i18n.TF("game.tower.btn_upgrade", upgCost)
	vm.BulkUpgradeButtonText = i18n.TF("game.tower.btn_bulk_upgrade", upgCost*5)
	vm.SellButtonText = i18n.TF("game.tower.btn_sell", sellValue)
	vm.CanAffordUpgrade = gold >= upgCost
	vm.CanAffordBulkUpgrade = gold >= upgCost*5

	return vm
}

// ─── 数据计算辅助函数 ────────────────────────────────────────────────
// 从 hud/info_panel.go 迁移至此，保持 hud 层零业务逻辑。

// strengthColor 根据有效强度值返回显示颜色。
// > 100 = 绿色（有增益），< 100 = 红色（有减益），≈ 100 = 默认色。
// 使用 ±0.5 的阈值避免浮点精度导致的颜色跳变。
func strengthColor(effStr float64) color.Color {
	if effStr > 100.5 {
		return theme.StatusStrUp
	}
	if effStr < 99.5 {
		return theme.StatusStrDown
	}
	return theme.StatusStrNorm
}

// strengthBreakdown 生成强度分解文本，如 "↑(永+50 链+20)"。
// 展示永久强度（来自金币购买）和临时强度（来自战灵连锁等）的明细，
// 帮助玩家理解强度数值的组成来源。
func strengthBreakdown(sd *strength.StrengthData) string {
	if sd == nil {
		return ""
	}
	var parts []string
	if sd.Permanent > 0 {
		parts = append(parts, fmt.Sprintf("%s+%.0f", i18n.T("game.str.perm"), sd.Permanent))
	}
	if chain, ok := sd.Temp["chain"]; ok && chain > 0 {
		parts = append(parts, fmt.Sprintf("%s+%.0f", i18n.T("game.str.chain"), chain))
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

// buildAttrSegsWithMods 构建带光环加成的属性显示段。
//
// 输出格式示例: "8+(12)→20+3" 含义为:
//
//	"8"   = base（基础值，白色）
//	"(12)" = scaled（潜力 × 强度/100，颜色随强度变化）
//	"→20" = baseTotal（base + scaled，白色）
//	"+3"  = auraBonus（来自光环的额外加成，绿色）
//
// 当 potential=0 且无光环时，简化为只显示 base 值。
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

// attackStyleLabel 通过 i18n 将攻击方式 ID 转为玩家可读的标签。
// 如 "scatter" → "散射"，"spin_aoe" → "旋转AOE"。
// 若 i18n key 不存在（返回原 key），则 fallback 到原始 ID。
func attackStyleLabel(style string) string {
	key := "game.attack_style." + style
	label := i18n.T(key)
	if label == key {
		return style
	}
	return label
}

// buffLabelKeys 将 buff 的内部 ID 映射到 i18n 翻译 key。
// 按 buff 类别分组：光环(CatAura) / CC / DoT / 防御 / 减益 / 行为。
var buffLabelKeys = map[string]string{
	// ── 塔光环 buff (CatAura) ──
	"aura:damageAmp": "game.buff.damage_up",
	"aura:pctDamage": "game.buff.pct_damage",
	"aura:pctSpeed":  "game.buff.atk_speed",
	"aura:flatRange": "game.buff.range",
	"aura:crit":      "game.buff.crit",
	"towerStrength":  "game.buff.str_boost",
	// ── CC 控制效果 (CatCC) ──
	"stun": "game.buff.stun",
	"slow": "game.buff.slow",
	"root": "game.buff.root",
	// ── 持续伤害 (CatDoT) ──
	"bleed":  "game.buff.bleed",
	"burn":   "game.buff.burn",
	"poison": "game.buff.poison",
	// ── 防御 (CatDefense) ──
	"controlImmune": "game.buff.control_immune",
	"damageReduce":  "game.buff.damage_reduce",
	// ── 减益 (CatDebuff) ──
	"weaken": "game.buff.weaken",
	// ── 行为 buff (CatBehavior) ──
	"stealth":    "game.buff.stealth",
	"berserk":    "game.buff.berserk",
	"regen":      "game.buff.regen",
	"healAura":   "game.buff.heal_aura",
	"bufferAura": "game.buff.speed_aura",
	"speedUp":    "game.buff.speed_up",
	"phaseShift": "game.buff.phase_shift",
}

// buffLabel 将 buff ID 转为玩家可读的标签。
// 支持三种来源：静态映射表 → 动态前缀匹配（战灵 buff）→ 原始 ID（兜底）。
func buffLabel(id string) string {
	if key, ok := buffLabelKeys[id]; ok {
		return i18n.T(key)
	}
	// Dynamic warden buff IDs: "chain_warden_<N>", "envoy_buff_<N>"
	if strings.HasPrefix(id, "chain_warden_") {
		return i18n.T("game.buff.chain_link")
	}
	if strings.HasPrefix(id, "envoy_buff_") {
		return i18n.T("game.buff.envoy_boost")
	}
	return id // fallback to raw ID
}

// isPercentBuff 判断 buff 值是否为百分比语义（内部 0-1，显示为 0%-100%）。
// 百分比 buff 聚合后以 "+45%" 格式显示，非百分比 buff 以 "+30" 格式显示。
func isPercentBuff(id string) bool {
	switch id {
	case "aura:damageAmp", "aura:pctDamage", "aura:pctSpeed", "aura:crit":
		return true
	}
	return false
}

// scaledColor 根据实际缩放值与潜力值的比较返回显示颜色。
// scaled > potential（强度 > 100）= 绿色，< potential = 红色，≈ potential = 默认。
// 这让玩家直观看到当前强度对属性的正面/负面影响。
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

// ─── 能力 VM 构建器 ─────────────────────────────────────────────────
// 将 AbilityDef（配置层数据）转为 AbilityVM（渲染层数据），核心是解析 Display 模板。
//
// Display 模板语法（定义在 abilities.json 每个能力的 display 字段）：
//   {s}   = base + potential×(str/100)，绝对值格式
//   {s%}  = 同上，百分比格式
//   {si}  = 同上，取整（向下）
//   {sh%} = total/2 的百分比（用于"增强"类减半效果）
//   {p}   = Param 参数值
//   {p%}  = Param 百分比格式
//   {p2}  = Param2 参数值
//   {p2%} = Param2 百分比格式
//
// 模板被解析为 AbilitySegment 切片，每段有 Kind（text/base/scaled/total/aura）和可选颜色，
// 由 hud 层按 Kind 着色渲染——base 白色、scaled 动态色、total 白色。

// fallback 映射：用于不在 AbilityTable 中的遗留能力（multishot/pulse）。
var (
	fallbackIconMap = map[string]string{
		"multishot": "multishot",
		"pulse":     "pulse",
	}
	fallbackLabelKeys = map[string]string{
		"multishot": "game.ability.multishot",
		"pulse":     "game.ability.pulse",
	}
	fallbackDescKeys = map[string]string{
		"multishot": "game.ability.multishot",
		"pulse":     "game.ability.pulse",
	}
)

// buildAbilityVM 为单个能力类型构建 AbilityVM。
// 优先从 AbilityTable 取定义（图标/标签/Display模板），不存在则走 fallback 映射。
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
	} else if key, ok := fallbackLabelKeys[abilityType]; ok {
		label = i18n.T(key)
	}

	vm := hud.AbilityVM{
		Icon:  iconName,
		Label: label,
	}

	// No AbilityDef → use fallback description
	if def == nil {
		if key, ok := fallbackDescKeys[abilityType]; ok {
			vm.Fallback = i18n.T(key)
		}
		return vm
	}

	// Parse Display template into segments
	vm.Segments = buildAbilitySegments(def, effStr)
	return vm
}

// FormatAbilityDisplay 将 Display 模板解析为纯文本字符串（不含分段着色）。
// 用于 ChoicePanel（3选1面板）的简化描述——只显示最终值，不展示 base+scaled 分解。
// 例如 "造成{s%}暴击率" → "造成50%暴击率"。
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

// buildAbilitySegments 解析 Display 模板并生成分段着色的渲染段。
//
// 与 FormatAbilityDisplay 不同，此函数生成 AbilitySegment 切片用于信息面板的富文本渲染：
//   - 纯文本部分 → Kind="text"（白色）
//   - {s%} 展开为最多 3 段: base(白) + scaled(动态色) + total(白)
//   - 当 base=0 时只显示 scaled（无 base+total 分解）
//   - 当 potential=0 时只显示 base（无成长的固定值）
//
// 这种设计让玩家在信息面板上直观看到能力数值的组成，
// 理解 "升级强度" 具体会提升多少。
func buildAbilitySegments(def *config.AbilityDef, effStr float64) []hud.AbilitySegment {
	tpl := def.Display
	if tpl == "" {
		return nil
	}

	scaled := def.Potential * (effStr / 100.0) // 潜力按强度比例缩放
	total := def.Base + scaled                 // 最终数值 = 基础 + 缩放
	// 概率/减速类展示值 clamp 到 100%（显示层限制，实际运行时由各 Apply 函数自行处理上限）
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
			} else if def.Potential == 0 {
				// potential=0 时只显示 base（无成长，如 bleedDot 1%）
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f%%", def.Base*100), Kind: "base"})
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
			} else if def.Potential == 0 {
				// potential=0 时只显示 base（无成长）
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf(nf, def.Base), Kind: "base"})
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
			} else if def.Potential == 0 {
				// potential=0 时只显示 base（无成长）
				segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f", math.Floor(def.Base)), Kind: "base"})
			} else {
				segs = append(segs,
					hud.AbilitySegment{Text: fmt.Sprintf("%.0f+", math.Floor(def.Base)), Kind: "base"},
					hud.AbilitySegment{Text: fmt.Sprintf("(%.0f)", math.Floor(scaled)), Kind: "scaled", Color: sClr},
					hud.AbilitySegment{Text: fmt.Sprintf("→%.0f", math.Floor(total)), Kind: "total"},
				)
			}
		case "sh%":
			// {sh%} = total 减半后的百分比（用于 enhance 类效果："射程提升一半"的描述）
			half := total / 2
			segs = append(segs, hud.AbilitySegment{Text: fmt.Sprintf("%.0f%%", half*100), Kind: "base"})
		case "p":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param), Kind: "text"})
		case "p%":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param*100) + "%", Kind: "text"})
		case "ph%":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param*50) + "%", Kind: "text"})
		case "p2":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param2), Kind: "text"})
		case "p2%":
			segs = append(segs, hud.AbilitySegment{Text: fmtNum(def.Param2*100) + "%", Kind: "text"})
		}
	}
	return segs
}
