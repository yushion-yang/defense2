// describe.go — 从能力描述符生成人类可读的中文描述。
//
// 用于 HUD 展示自定义能力（ca_ 开头）的效果说明。
// 这些能力不在 abilities.json 的 Display 模板中，
// 需要从 Pipeline 的 trigger/condition/selector/effect 组合自动生成。
//
// 格式示例:
//   "每帧: 全范围 → 眩晕 0.1s"
//   "命中时 [是Boss]: 当前目标 → 净化×3 + 沉默"
package descriptor

import (
	"fmt"
	"strings"
)

// GenerateDescription 从描述符的 Pipeline 组合生成人类可读的中文描述。
// 当描述符无 Pipeline（纯 attackStyle 类）时返回 Label。
func GenerateDescription(desc *AbilityDescriptor) string {
	if desc == nil {
		return ""
	}
	if len(desc.Pipelines) == 0 {
		if desc.AttackStyle != "" {
			return desc.Label + " (" + attackStyleLabel(desc.AttackStyle) + ")"
		}
		return desc.Label
	}

	var lines []string
	for _, p := range desc.Pipelines {
		line := describePipeline(p)
		lines = append(lines, line)
	}
	return strings.Join(lines, "; ")
}

// describePipeline 将单条管线描述为一行文本。
// 格式: "触发 [条件]: 选择器 → 效果1 + 效果2"
func describePipeline(p Pipeline) string {
	var b strings.Builder

	// 触发器
	b.WriteString(triggerLabel(p.Trigger))

	// 条件（如果有）
	if len(p.Conditions) > 0 {
		var condParts []string
		for _, c := range p.Conditions {
			condParts = append(condParts, conditionLabel(c))
		}
		b.WriteString(" [")
		b.WriteString(strings.Join(condParts, ", "))
		b.WriteString("]")
	}

	b.WriteString(": ")

	// 选择器
	b.WriteString(selectorLabel(p.Selector))

	// 效果
	if len(p.Effects) > 0 {
		b.WriteString(" → ")
		var effParts []string
		for _, e := range p.Effects {
			effParts = append(effParts, effectLabel(e))
		}
		b.WriteString(strings.Join(effParts, " + "))
	}

	return b.String()
}

// ── 触发器标签 ──────────────────────────────────────────

func triggerLabel(t TriggerType) string {
	switch t {
	case TriggerOnHit:
		return "命中时"
	case TriggerOnTick:
		return "每帧"
	case TriggerOnKill:
		return "击杀时"
	case TriggerOnPlace:
		return "放置时"
	default:
		return t.String()
	}
}

// ── 条件标签 ────────────────────────────────────────────

func conditionLabel(c Condition) string {
	switch v := c.(type) {
	case ChanceCondition:
		return fmt.Sprintf("概率%s", scalerLabel(v.Rate))
	case *CooldownCondition:
		return fmt.Sprintf("CD %.1fs", v.Seconds)
	case HpBelowCondition:
		return fmt.Sprintf("HP<%s", scalerLabel(v.Threshold))
	case HpAboveCondition:
		return fmt.Sprintf("HP>%s", scalerLabel(v.Threshold))
	case DistanceMinCondition:
		return fmt.Sprintf("距离≥%.0f", v.Distance)
	case NoNearbyTowerCondition:
		return fmt.Sprintf("无邻塔(%.0f)", v.Radius)
	case IsBossCondition:
		return "是Boss"
	case NotBossCondition:
		return "非Boss"
	case *EveryCondition:
		return fmt.Sprintf("每%d次", v.N)
	case BuffActiveCondition:
		return fmt.Sprintf("有%s", v.BuffID)
	case BuffAbsentCondition:
		return fmt.Sprintf("无%s", v.BuffID)
	default:
		return "?"
	}
}

// ── 选择器标签 ──────────────────────────────────────────

func selectorLabel(s Selector) string {
	if s == nil {
		return "?"
	}
	switch v := s.(type) {
	case CurrentTargetSelector:
		return "当前目标"
	case AoeRadiusSelector:
		return fmt.Sprintf("范围(%s)", scalerLabel(v.Radius))
	case ChainSelector:
		return fmt.Sprintf("弹跳×%s", scalerLabel(v.MaxBounce))
	case AllInRangeSelector:
		return "全范围"
	case NearbyAlliesSelector:
		return fmt.Sprintf("友方塔(%.0f)", v.Radius)
	case SelfTowerSelector:
		return "自身"
	case ConeSelector:
		return fmt.Sprintf("扇形(%.0f°)", v.Angle)
	case Ring360Selector:
		return fmt.Sprintf("环形×%s", scalerLabel(v.Count))
	case RandomSelector:
		return fmt.Sprintf("随机×%s", scalerLabel(v.Count))
	default:
		return "?"
	}
}

// ── 效果标签 ────────────────────────────────────────────

func effectLabel(e Effect) string {
	switch v := e.(type) {
	case DamageEffect:
		return fmt.Sprintf("伤害%s(%s)", damageModeLabel(v.Mode), scalerLabel(v.Value))
	case SlowEffect:
		return fmt.Sprintf("减速%s/%s", scalerLabel(v.Factor), scalerLabel(v.Duration))
	case StunEffect:
		return fmt.Sprintf("眩晕%s", scalerLabel(v.Duration))
	case RootEffect:
		return fmt.Sprintf("定身%s", scalerLabel(v.Duration))
	case DotEffect:
		return fmt.Sprintf("%s%s/%s", dotSubtypeLabel(v.Subtype), scalerLabel(v.Value), scalerLabel(v.Duration))
	case WeakenEffect:
		return fmt.Sprintf("易伤%s/%s", scalerLabel(v.Amplify), scalerLabel(v.Duration))
	case SilenceEffect:
		return "沉默"
	case BuffEffect:
		return fmt.Sprintf("增益%s+%s", v.Stat, scalerLabel(v.Bonus))
	case SelfBuffEffect:
		return fmt.Sprintf("自增%s+%s", v.Stat, scalerLabel(v.Bonus))
	case GoldEffect:
		return fmt.Sprintf("产金%s", scalerLabel(v.Amount))
	case ModifyStatEffect:
		return fmt.Sprintf("改%s×%.1f", v.Stat, v.Multiplier)
	case CritEffect:
		return fmt.Sprintf("暴击×%.1f", v.Multiplier)
	case PurgeEffect:
		return fmt.Sprintf("净化×%d", v.Count)
	case TeleportEffect:
		return fmt.Sprintf("传送%s", scalerLabel(v.Distance))
	default:
		return "?"
	}
}

// ── 辅助格式化 ──────────────────────────────────────────

// scalerLabel 将 Scaler 显示为简短文本。
// 固定值显示数值，线性缩放显示 "base+p*str" 格式。
func scalerLabel(s Scaler) string {
	if s == nil {
		return "?"
	}
	switch v := s.(type) {
	case FixedScaler:
		return fmtVal(v.Value)
	case LinearScaler:
		if v.Potential == 0 {
			return fmtVal(v.Base)
		}
		return fmt.Sprintf("%s+%s*str", fmtVal(v.Base), fmtVal(v.Potential))
	default:
		return "?"
	}
}

// fmtVal 格式化数值：小于 1 的显示为百分比，否则显示最多 1 位小数。
func fmtVal(v float64) string {
	// 明确的百分比值（0~1 之间且非整数）
	if v > 0 && v < 1 {
		pct := v * 100
		if pct == float64(int(pct)) {
			return fmt.Sprintf("%.0f%%", pct)
		}
		return fmt.Sprintf("%.1f%%", pct)
	}
	// 整数或大数
	if v == float64(int(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func damageModeLabel(m DamageMode) string {
	switch m {
	case DmgFlat:
		return "固定"
	case DmgRatio:
		return "比例"
	case DmgHpPercent:
		return "%HP"
	default:
		return ""
	}
}

func dotSubtypeLabel(sub string) string {
	switch sub {
	case "bleed":
		return "流血"
	case "burn":
		return "灼烧"
	case "poison":
		return "中毒"
	default:
		return "DoT"
	}
}

func attackStyleLabel(style string) string {
	switch style {
	case "projectile":
		return "投射物"
	case "wideBeam":
		return "宽光束"
	case "scatter":
		return "散射"
	case "spin_aoe":
		return "旋转AOE"
	case "radial":
		return "径向"
	case "barrage":
		return "连击"
	default:
		return style
	}
}
