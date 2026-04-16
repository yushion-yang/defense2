// cooperation.go — 协作意识系统。
//
// 分析人类玩家的塔配置，识别队伍整体缺口，AI 主动补位。
// 职责：观察全队（人类+AI）塔配置，推导"谁缺什么"，生成补位建议。
// 关联：由 decision.go Evaluate() 调用，输出 CoopAnalysis 影响决策偏好。
//       与 awareness.go 的 StrategicAdvice 合并，协作建议优先于个人判断。
package aiplayer

import "strings"

// CoopAnalysis 协作分析结果。
type CoopAnalysis struct {
	HumanHasCC  bool // 人类玩家有 CC 塔
	HumanHasDPS bool // 人类玩家有高 DPS 塔
	HumanHasAOE bool // 人类玩家有 AOE 塔
	HumanTowers int  // 人类塔总数
	AITowers    int  // AI 塔总数

	TeamNeedsCC  bool // 全队缺 CC
	TeamNeedsDPS bool // 全队缺 DPS
	TeamNeedsAOE bool // 全队缺 AOE

	Recommendation string // "fill_cc" / "fill_dps" / "fill_aoe" / "balanced"
	Reason         string // 可读原因（用于气泡文案和 LLM prompt）
}

// dpsAbilities 高 DPS 类能力关键字。
// 与 aoeAbilities/ccAbilities 互补，检测纯输出型塔。
var dpsAbilities = map[string]bool{
	"multiTarget": true, "bounce": true, "barrage": true,
	"critical": true, "armorBreak": true, "execute": true,
}

// dpsMinDamage DPS 塔的最低伤害阈值。
// 低于此值的塔即使有 DPS 能力也不视为高 DPS。
const dpsMinDamage = 30.0

// AnalyzeCooperation 分析全队塔配置，生成协作补位建议。
//
// 流程：
//  1. 按 Owner 分割塔（human=Owner==0, AI=Owner==aiOwnerID）
//  2. 检查人类玩家的 CC/AOE/DPS 覆盖
//  3. 检查全队（双方合计）是否存在缺口
//  4. 生成补位建议：优先填补人类玩家缺少且全队也缺的方向
func AnalyzeCooperation(allTowers []AITower, aiOwnerID int) CoopAnalysis {
	ca := CoopAnalysis{
		Recommendation: "balanced",
		Reason:         "队伍配置还行",
	}

	// ── 1. 按 Owner 分割 ──
	var humanTowers, aiTowers []AITower
	for _, t := range allTowers {
		if t.Owner == 0 {
			humanTowers = append(humanTowers, t)
		} else if t.Owner == aiOwnerID {
			aiTowers = append(aiTowers, t)
		}
	}
	ca.HumanTowers = len(humanTowers)
	ca.AITowers = len(aiTowers)

	// ── 2. 检查人类玩家的能力覆盖 ──
	ca.HumanHasCC = towersHaveCC(humanTowers)
	ca.HumanHasAOE = towersHaveAOE(humanTowers)
	ca.HumanHasDPS = towersHaveDPS(humanTowers)

	// ── 3. 检查全队缺口 ──
	allCC := ca.HumanHasCC || towersHaveCC(aiTowers)
	allAOE := ca.HumanHasAOE || towersHaveAOE(aiTowers)
	allDPS := ca.HumanHasDPS || towersHaveDPS(aiTowers)

	ca.TeamNeedsCC = !allCC
	ca.TeamNeedsAOE = !allAOE
	ca.TeamNeedsDPS = !allDPS

	// ── 4. 生成补位建议 ──
	// 前置条件：人类至少有 1 座塔时才分析补位（否则无意义）
	// 优先级：CC > DPS > AOE（CC 是基础防御，DPS 是核心输出）
	// 关键逻辑：只在"人类缺且全队缺"时触发补位，
	//           如果 AI 已有某方向覆盖则不算全队缺。
	if ca.HumanTowers == 0 {
		// 人类还没建塔，AI 无法判断补位方向
		ca.Recommendation = "balanced"
		ca.Reason = "队伍配置还行"
		return ca
	}

	switch {
	case !ca.HumanHasCC && ca.TeamNeedsCC:
		ca.Recommendation = "fill_cc"
		ca.Reason = "队友全输出，我来补控制"
	case !ca.HumanHasDPS && ca.TeamNeedsDPS:
		ca.Recommendation = "fill_dps"
		ca.Reason = "队友有控制了，我加点伤害"
	case !ca.HumanHasAOE && ca.TeamNeedsAOE:
		ca.Recommendation = "fill_aoe"
		ca.Reason = "怪太密集了，我补个范围"
	default:
		ca.Recommendation = "balanced"
		ca.Reason = "队伍配置还行"
	}

	return ca
}

// MergeCoopWithAdvice 合并协作建议和个人局势建议。
// 协作建议（团队平衡）优先于个人判断。
//
// 合并规则：
//   - 两者一致 → urgency 加成 20%（双重确认更紧急）
//   - 协作有明确补位方向，个人无特别偏好 → 采用协作建议
//   - 协作有明确补位方向，个人有不同方向 → 协作胜出（团队 > 个人）
//   - 协作为 balanced → 保持个人建议不变
func MergeCoopWithAdvice(coop CoopAnalysis, advice StrategicAdvice) StrategicAdvice {
	// 协作建议是 balanced → 不修改个人建议
	if coop.Recommendation == "balanced" {
		return advice
	}

	// 将协作建议映射到 StrategicAdvice 的 Priority 格式
	coopPriority := mapCoopToPriority(coop.Recommendation)

	// 两者一致 → urgency 加成
	if advice.Priority == coopPriority {
		advice.Urgency = clampFloat(advice.Urgency*1.2, 0, 1.0)
		advice.Reason = coop.Reason
		return advice
	}

	// 协作有明确方向 → 覆盖个人建议（团队平衡优先）
	advice.Priority = coopPriority
	advice.Reason = coop.Reason
	// 保持 urgency 不低于 0.6（协作补位有基础紧迫性）
	if advice.Urgency < 0.6 {
		advice.Urgency = 0.6
	}
	return advice
}

// ── 内部辅助函数 ──

// towersHaveCC 检查塔列表中是否有至少一座拥有 CC 能力的塔。
func towersHaveCC(towers []AITower) bool {
	for _, t := range towers {
		for _, ab := range t.Abilities {
			if ccAbilities[ab] {
				return true
			}
		}
	}
	return false
}

// towersHaveAOE 检查塔列表中是否有至少一座拥有 AOE 能力的塔。
func towersHaveAOE(towers []AITower) bool {
	for _, t := range towers {
		for _, ab := range t.Abilities {
			if aoeAbilities[ab] {
				return true
			}
		}
	}
	return false
}

// towersHaveDPS 检查塔列表中是否有高 DPS 塔。
// 判定条件：拥有 DPS 类能力 且 伤害值 >= dpsMinDamage，
// 或者伤害值 >= 50（纯靠数值碾压也算 DPS 塔）。
func towersHaveDPS(towers []AITower) bool {
	for _, t := range towers {
		// 纯数值 DPS
		if t.Damage >= 50 {
			return true
		}
		// 能力 + 伤害组合
		if t.Damage >= dpsMinDamage {
			for _, ab := range t.Abilities {
				if dpsAbilities[ab] {
					return true
				}
			}
		}
	}
	return false
}

// mapCoopToPriority 将协作建议映射到 StrategicAdvice 的 Priority 格式。
func mapCoopToPriority(recommendation string) string {
	switch recommendation {
	case "fill_cc":
		return "build_cc"
	case "fill_dps":
		return "build_dps"
	case "fill_aoe":
		return "build_aoe"
	default:
		return "balanced"
	}
}

// clampFloat 钳制浮点数到 [min, max] 范围。
func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// CoopDescription 生成协作分析的可读描述（用于 LLM prompt）。
func CoopDescription(ca CoopAnalysis) string {
	var parts []string
	if ca.HumanTowers == 0 {
		parts = append(parts, "队友还没建塔")
	} else {
		if ca.HumanHasCC {
			parts = append(parts, "队友有控制")
		}
		if ca.HumanHasDPS {
			parts = append(parts, "队友有输出")
		}
		if ca.HumanHasAOE {
			parts = append(parts, "队友有范围")
		}
		if !ca.HumanHasCC && !ca.HumanHasDPS && !ca.HumanHasAOE {
			parts = append(parts, "队友塔配置不明确")
		}
	}
	if ca.TeamNeedsCC {
		parts = append(parts, "全队缺控制")
	}
	if ca.TeamNeedsDPS {
		parts = append(parts, "全队缺输出")
	}
	if ca.TeamNeedsAOE {
		parts = append(parts, "全队缺范围")
	}
	if len(parts) == 0 {
		return "队伍配置均衡"
	}
	return strings.Join(parts, "，")
}
