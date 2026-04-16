// marshal.go — AbilityDescriptor 的 JSON 序列化/反序列化。
//
// Pipeline 包含接口字段（Condition/Selector/Effect），标准 json.Marshal
// 无法正确序列化这些接口。本文件提供自定义 MarshalJSON/UnmarshalJSON，
// 将编译后的 Pipeline 转换回 rawDescriptor/rawPipeline JSON 中间结构。
//
// MarshalJSON: AbilityDescriptor → rawDescriptor → JSON bytes
// UnmarshalJSON: JSON bytes → ParseDescriptor → AbilityDescriptor
//
// 关联：
//   - descriptor.go 中的 rawDescriptor/rawPipeline/ParseDescriptor（解析方向）
//   - scaler.go 中的 Scaler 类型层次
//   - condition.go / selector.go / effect.go 中的具体类型
package descriptor

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON 将 AbilityDescriptor 序列化为 JSON。
// 核心任务：将编译后的 Pipeline（含接口字段）转换回可序列化的 JSON 结构。
func (d AbilityDescriptor) MarshalJSON() ([]byte, error) {
	// 构建 rawDescriptor 用于序列化
	raw := rawDescriptor{
		ID:           d.ID,
		Label:        d.Label,
		Icon:         d.Icon,
		Cost:         d.Cost,
		Tags:         d.Tags,
		AttackStyle:  d.AttackStyle,
		SpriteKey:    d.SpriteKey,
		AttackParams: d.AttackParams,
	}

	// 逐条 Pipeline 转换为 JSON
	raw.Pipelines = make([]json.RawMessage, 0, len(d.Pipelines))
	for i, p := range d.Pipelines {
		rpJSON, err := marshalPipeline(p)
		if err != nil {
			return nil, fmt.Errorf("pipeline[%d]: %w", i, err)
		}
		raw.Pipelines = append(raw.Pipelines, rpJSON)
	}

	return json.Marshal(raw)
}

// UnmarshalJSON 从 JSON 反序列化 AbilityDescriptor。
// 委托给 ParseDescriptor 完成两阶段解析。
func (d *AbilityDescriptor) UnmarshalJSON(data []byte) error {
	parsed, err := ParseDescriptor(data)
	if err != nil {
		return err
	}
	*d = *parsed
	return nil
}

// marshalPipeline 将编译后的 Pipeline 转换为 JSON bytes。
func marshalPipeline(p Pipeline) (json.RawMessage, error) {
	rp := rawPipeline{
		Trigger: p.Trigger.String(),
	}

	// 条件
	rp.Conditions = make([]json.RawMessage, 0, len(p.Conditions))
	for _, c := range p.Conditions {
		raw, err := conditionToRawJSON(c)
		if err != nil {
			return nil, fmt.Errorf("condition: %w", err)
		}
		rp.Conditions = append(rp.Conditions, raw)
	}

	// 选择器
	if p.Selector != nil {
		raw, err := selectorToRawJSON(p.Selector)
		if err != nil {
			return nil, fmt.Errorf("selector: %w", err)
		}
		rp.Selector = raw
	}

	// 效果
	rp.Effects = make([]json.RawMessage, 0, len(p.Effects))
	for _, e := range p.Effects {
		raw, err := effectToRawJSON(e)
		if err != nil {
			return nil, fmt.Errorf("effect: %w", err)
		}
		rp.Effects = append(rp.Effects, raw)
	}

	return json.Marshal(rp)
}

// ── Scaler → JSON ─────────────────────────────────────────

// scalerToJSON 将 Scaler 实例转换为可 JSON 序列化的 map。
// 包含 "scaler" 类型标记字段，与 ParseScaler 输入格式一致。
func scalerToJSON(s Scaler) map[string]any {
	switch v := s.(type) {
	case FixedScaler:
		return map[string]any{"scaler": "fixed", "value": v.Value}
	case LinearScaler:
		return map[string]any{"scaler": "linear", "base": v.Base, "potential": v.Potential}
	case DiminishingScaler:
		return map[string]any{"scaler": "diminishing", "base": v.Base, "potential": v.Potential, "k": v.K}
	case CappedScaler:
		return map[string]any{"scaler": "capped", "base": v.Base, "potential": v.Potential, "cap": v.Cap}
	case SteppedScaler:
		return map[string]any{"scaler": "stepped", "steps": v.Steps}
	default:
		return map[string]any{"scaler": "fixed", "value": 0}
	}
}

// ── Condition → JSON ──────────────────────────────────────

// conditionToRawJSON 将 Condition 实例转换为 JSON bytes。
// 输出格式与 parseCondition 的输入格式一致。
func conditionToRawJSON(c Condition) (json.RawMessage, error) {
	var m map[string]any

	switch v := c.(type) {
	case ChanceCondition:
		m = map[string]any{"type": "chance", "rate": scalerToJSON(v.Rate)}

	case *CooldownCondition:
		m = map[string]any{"type": "cooldown", "seconds": v.Seconds}

	case HpBelowCondition:
		m = map[string]any{"type": "hpBelow", "threshold": scalerToJSON(v.Threshold)}

	case HpAboveCondition:
		m = map[string]any{"type": "hpAbove", "threshold": scalerToJSON(v.Threshold)}

	case DistanceMinCondition:
		m = map[string]any{"type": "distanceMin", "distance": v.Distance}

	case NoNearbyTowerCondition:
		m = map[string]any{"type": "noNearbyTower", "radius": v.Radius}

	case IsBossCondition:
		m = map[string]any{"type": "isBoss"}

	case NotBossCondition:
		m = map[string]any{"type": "notBoss"}

	case *EveryCondition:
		m = map[string]any{"type": "every", "n": v.N}

	case BuffActiveCondition:
		m = map[string]any{"type": "buffActive", "buffId": v.BuffID}

	case BuffAbsentCondition:
		m = map[string]any{"type": "buffAbsent", "buffId": v.BuffID}

	default:
		return nil, fmt.Errorf("unknown condition type: %T", c)
	}

	return json.Marshal(m)
}

// ── Selector → JSON ───────────────────────────────────────

// selectorToRawJSON 将 Selector 实例转换为 JSON bytes。
// 输出格式与 parseSelector 的输入格式一致。
func selectorToRawJSON(s Selector) (json.RawMessage, error) {
	var m map[string]any

	switch v := s.(type) {
	case CurrentTargetSelector:
		m = map[string]any{"type": "currentTarget"}

	case AoeRadiusSelector:
		m = map[string]any{"type": "aoeRadius", "radius": scalerToJSON(v.Radius)}

	case ChainSelector:
		m = map[string]any{
			"type":       "chain",
			"maxBounce":  scalerToJSON(v.MaxBounce),
			"range":      v.ChainRange,
			"decayRatio": v.DecayRatio,
		}

	case AllInRangeSelector:
		m = map[string]any{"type": "allInRange"}

	case NearbyAlliesSelector:
		m = map[string]any{"type": "nearbyAllies", "radius": v.Radius}

	case SelfTowerSelector:
		m = map[string]any{"type": "selfTower"}

	case ConeSelector:
		m = map[string]any{"type": "cone", "angle": v.Angle, "radius": scalerToJSON(v.Radius)}

	case Ring360Selector:
		m = map[string]any{"type": "ring360", "count": scalerToJSON(v.Count)}

	case RandomSelector:
		m = map[string]any{"type": "random", "count": scalerToJSON(v.Count), "radius": v.Radius}

	default:
		return nil, fmt.Errorf("unknown selector type: %T", s)
	}

	return json.Marshal(m)
}

// ── Effect → JSON ─────────────────────────────────────────

// effectToRawJSON 将 Effect 实例转换为 JSON bytes。
// 输出格式与 parseEffect 的输入格式一致。
func effectToRawJSON(e Effect) (json.RawMessage, error) {
	var m map[string]any

	switch v := e.(type) {
	case DamageEffect:
		m = map[string]any{
			"type":  "damage",
			"mode":  damageModeToString(v.Mode),
			"value": scalerToJSON(v.Value),
		}

	case SlowEffect:
		m = map[string]any{
			"type":     "slow",
			"factor":   scalerToJSON(v.Factor),
			"duration": scalerToJSON(v.Duration),
		}

	case StunEffect:
		m = map[string]any{
			"type":     "stun",
			"duration": scalerToJSON(v.Duration),
		}

	case RootEffect:
		m = map[string]any{
			"type":     "root",
			"duration": scalerToJSON(v.Duration),
		}

	case DotEffect:
		m = map[string]any{
			"type":     "dot",
			"subtype":  v.Subtype,
			"mode":     damageModeToString(v.Mode),
			"value":    scalerToJSON(v.Value),
			"duration": scalerToJSON(v.Duration),
		}

	case WeakenEffect:
		m = map[string]any{
			"type":     "weaken",
			"amplify":  scalerToJSON(v.Amplify),
			"duration": scalerToJSON(v.Duration),
		}

	case SilenceEffect:
		m = map[string]any{"type": "silence"}

	case BuffEffect:
		m = map[string]any{
			"type":  "buff",
			"stat":  v.Stat,
			"bonus": scalerToJSON(v.Bonus),
		}

	case SelfBuffEffect:
		m = map[string]any{
			"type":  "selfBuff",
			"stat":  v.Stat,
			"bonus": scalerToJSON(v.Bonus),
		}

	case GoldEffect:
		m = map[string]any{
			"type":   "gold",
			"amount": scalerToJSON(v.Amount),
		}

	case ModifyStatEffect:
		m = map[string]any{
			"type":       "modifyStat",
			"stat":       v.Stat,
			"multiplier": v.Multiplier,
		}

	case CritEffect:
		m = map[string]any{
			"type":       "crit",
			"multiplier": v.Multiplier,
		}

	case PurgeEffect:
		m = map[string]any{
			"type":  "purge",
			"count": v.Count,
		}

	case TeleportEffect:
		m = map[string]any{
			"type":     "teleport",
			"distance": scalerToJSON(v.Distance),
		}

	default:
		return nil, fmt.Errorf("unknown effect type: %T", e)
	}

	return json.Marshal(m)
}

// ── DamageMode 反向映射 ───────────────────────────────────

// damageModeToString 将 DamageMode 枚举转换为 JSON 字符串。
// 与 parseDamageMode 互为逆操作。
func damageModeToString(m DamageMode) string {
	switch m {
	case DmgFlat:
		return "flat"
	case DmgRatio:
		return "ratio"
	case DmgHpPercent:
		return "hpPercent"
	default:
		return "flat"
	}
}
