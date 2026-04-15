// descriptor.go — 能力描述符 JSON 模式与解析器。
//
// AbilityDescriptor 是一个能力的完整 JSON 描述，包含一条或多条 Pipeline。
// 每条 Pipeline 是一条 触发→条件→选择器→效果 链。
//
// ParseDescriptor 执行两阶段解析：
//   1. 反序列化顶层字段（id/label/cost/tags/attackStyle/spriteKey）
//   2. 对每条 pipeline 的 conditions/selector/effects，
//      先用 json.RawMessage 提取 type 字段，再分派到对应类型的解析函数。
//
// 所有可变数值参数通过 ParseScaler 解析，使得描述符在不同强度下产生不同效果。
package descriptor

import (
	"encoding/json"
	"fmt"
)

// AbilityDescriptor 一个能力的完整描述。
type AbilityDescriptor struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Icon        string            `json:"icon,omitempty"`
	Cost        int               `json:"cost"`
	Tags        []string          `json:"tags,omitempty"`
	AttackStyle string            `json:"attackStyle,omitempty"`
	SpriteKey   string            `json:"spriteKey,omitempty"`
	AttackParams json.RawMessage  `json:"attackParams,omitempty"` // 攻击参数（原始 JSON，供运行时按 attackStyle 解读）
	Pipelines   []Pipeline        `json:"pipelines"`
}

// Pipeline 一条 触发→条件→目标→效果 管线（已编译形态）。
type Pipeline struct {
	Trigger    TriggerType
	Conditions []Condition
	Selector   Selector
	Effects    []Effect
}

// ── JSON 中间结构 ─────────────────────────────────────────

// rawDescriptor 第一阶段反序列化结构，pipeline 保留为 RawMessage。
type rawDescriptor struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Icon         string            `json:"icon,omitempty"`
	Cost         int               `json:"cost"`
	Tags         []string          `json:"tags,omitempty"`
	AttackStyle  string            `json:"attackStyle,omitempty"`
	SpriteKey    string            `json:"spriteKey,omitempty"`
	AttackParams json.RawMessage   `json:"attackParams,omitempty"`
	Pipelines    []json.RawMessage `json:"pipelines"`
}

// rawPipeline pipeline 的中间 JSON 结构。
type rawPipeline struct {
	Trigger    string              `json:"trigger"`
	Conditions []json.RawMessage   `json:"conditions,omitempty"`
	Selector   json.RawMessage     `json:"selector"`
	Effects    []json.RawMessage   `json:"effects"`
}

// typeEnvelope 用于提取 JSON 对象中的 "type" 字段。
type typeEnvelope struct {
	Type string `json:"type"`
}

// ParseDescriptor 从 JSON 解析为 AbilityDescriptor（含预编译管线）。
//
// 解析流程：
//   1. 反序列化顶层字段
//   2. 校验至少一条 pipeline
//   3. 逐条解析 pipeline（trigger/conditions/selector/effects）
func ParseDescriptor(data []byte) (*AbilityDescriptor, error) {
	var raw rawDescriptor
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor: %w", err)
	}

	// 攻击方式能力（如 scatter/spinAoe）可以没有 pipeline，
	// 其效果完全由 attackStyle + attackParams 描述。
	if len(raw.Pipelines) == 0 && raw.AttackStyle == "" {
		return nil, fmt.Errorf("descriptor %q: at least one pipeline required (or set attackStyle)", raw.ID)
	}

	pipelines := make([]Pipeline, 0, len(raw.Pipelines))
	for i, rpData := range raw.Pipelines {
		p, err := parsePipeline(rpData)
		if err != nil {
			return nil, fmt.Errorf("descriptor %q pipeline[%d]: %w", raw.ID, i, err)
		}
		pipelines = append(pipelines, p)
	}

	return &AbilityDescriptor{
		ID:           raw.ID,
		Label:        raw.Label,
		Icon:         raw.Icon,
		Cost:         raw.Cost,
		Tags:         raw.Tags,
		AttackStyle:  raw.AttackStyle,
		SpriteKey:    raw.SpriteKey,
		AttackParams: raw.AttackParams,
		Pipelines:    pipelines,
	}, nil
}

// parsePipeline 解析单条 pipeline JSON。
func parsePipeline(data json.RawMessage) (Pipeline, error) {
	var rp rawPipeline
	if err := json.Unmarshal(data, &rp); err != nil {
		return Pipeline{}, fmt.Errorf("unmarshal: %w", err)
	}

	// 解析 trigger
	trigger, err := ParseTrigger(rp.Trigger)
	if err != nil {
		return Pipeline{}, err
	}

	// 解析 conditions（可为空）
	conditions := make([]Condition, 0, len(rp.Conditions))
	for j, cd := range rp.Conditions {
		c, err := parseCondition(cd)
		if err != nil {
			return Pipeline{}, fmt.Errorf("conditions[%d]: %w", j, err)
		}
		conditions = append(conditions, c)
	}

	// 解析 selector
	sel, err := parseSelector(rp.Selector)
	if err != nil {
		return Pipeline{}, fmt.Errorf("selector: %w", err)
	}

	// 解析 effects
	effects := make([]Effect, 0, len(rp.Effects))
	for j, ed := range rp.Effects {
		e, err := parseEffect(ed)
		if err != nil {
			return Pipeline{}, fmt.Errorf("effects[%d]: %w", j, err)
		}
		effects = append(effects, e)
	}

	return Pipeline{
		Trigger:    trigger,
		Conditions: conditions,
		Selector:   sel,
		Effects:    effects,
	}, nil
}

// ── Condition 解析 ──────────────────────────────────────────

// conditionChanceRaw chance 条件的 JSON 参数。
type conditionChanceRaw struct {
	Rate json.RawMessage `json:"rate"`
}

// conditionCooldownRaw cooldown 条件的 JSON 参数。
type conditionCooldownRaw struct {
	Seconds float64 `json:"seconds"`
}

// conditionHpThresholdRaw hpBelow/hpAbove 条件的 JSON 参数。
type conditionHpThresholdRaw struct {
	Threshold json.RawMessage `json:"threshold"`
}

// conditionDistanceMinRaw distanceMin 条件的 JSON 参数。
type conditionDistanceMinRaw struct {
	Distance float64 `json:"distance"`
}

// conditionNoNearbyTowerRaw noNearbyTower 条件的 JSON 参数。
type conditionNoNearbyTowerRaw struct {
	Radius float64 `json:"radius"`
}

func parseCondition(data json.RawMessage) (Condition, error) {
	var env typeEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal condition type: %w", err)
	}

	switch env.Type {
	case "chance":
		var raw conditionChanceRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse chance condition: %w", err)
		}
		rate, err := ParseScaler(raw.Rate)
		if err != nil {
			return nil, fmt.Errorf("chance.rate: %w", err)
		}
		return ChanceCondition{Rate: rate}, nil

	case "cooldown":
		var raw conditionCooldownRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse cooldown condition: %w", err)
		}
		return NewCooldownCondition(raw.Seconds), nil

	case "hpBelow":
		var raw conditionHpThresholdRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse hpBelow condition: %w", err)
		}
		threshold, err := ParseScaler(raw.Threshold)
		if err != nil {
			return nil, fmt.Errorf("hpBelow.threshold: %w", err)
		}
		return HpBelowCondition{Threshold: threshold}, nil

	case "hpAbove":
		var raw conditionHpThresholdRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse hpAbove condition: %w", err)
		}
		threshold, err := ParseScaler(raw.Threshold)
		if err != nil {
			return nil, fmt.Errorf("hpAbove.threshold: %w", err)
		}
		return HpAboveCondition{Threshold: threshold}, nil

	case "distanceMin":
		var raw conditionDistanceMinRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse distanceMin condition: %w", err)
		}
		return DistanceMinCondition{Distance: raw.Distance}, nil

	case "noNearbyTower":
		var raw conditionNoNearbyTowerRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse noNearbyTower condition: %w", err)
		}
		return NoNearbyTowerCondition{Radius: raw.Radius}, nil

	default:
		return nil, fmt.Errorf("unknown condition type: %q", env.Type)
	}
}

// ── Selector 解析 ──────────────────────────────────────────

// selectorAoeRaw aoeRadius 选择器的 JSON 参数。
type selectorAoeRaw struct {
	Radius json.RawMessage `json:"radius"`
}

// selectorChainRaw chain 选择器的 JSON 参数。
type selectorChainRaw struct {
	MaxBounce  json.RawMessage `json:"maxBounce"`
	Range      float64         `json:"range"`
	DecayRatio float64         `json:"decayRatio"`
}

// selectorNearbyAlliesRaw nearbyAllies 选择器的 JSON 参数。
type selectorNearbyAlliesRaw struct {
	Radius float64 `json:"radius"`
}

func parseSelector(data json.RawMessage) (Selector, error) {
	var env typeEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal selector type: %w", err)
	}

	switch env.Type {
	case "currentTarget":
		return CurrentTargetSelector{}, nil

	case "aoeRadius":
		var raw selectorAoeRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse aoeRadius selector: %w", err)
		}
		radius, err := ParseScaler(raw.Radius)
		if err != nil {
			return nil, fmt.Errorf("aoeRadius.radius: %w", err)
		}
		return AoeRadiusSelector{Radius: radius}, nil

	case "chain":
		var raw selectorChainRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse chain selector: %w", err)
		}
		maxBounce, err := ParseScaler(raw.MaxBounce)
		if err != nil {
			return nil, fmt.Errorf("chain.maxBounce: %w", err)
		}
		return ChainSelector{
			MaxBounce:  maxBounce,
			ChainRange: raw.Range,
			DecayRatio: raw.DecayRatio,
		}, nil

	case "allInRange":
		return AllInRangeSelector{}, nil

	case "nearbyAllies":
		var raw selectorNearbyAlliesRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse nearbyAllies selector: %w", err)
		}
		return NearbyAlliesSelector{Radius: raw.Radius}, nil

	case "selfTower":
		return SelfTowerSelector{}, nil

	default:
		return nil, fmt.Errorf("unknown selector type: %q", env.Type)
	}
}

// ── Effect 解析 ─────────────────────────────────────────────

// effectDamageRaw damage 效果的 JSON 参数。
type effectDamageRaw struct {
	Mode  string          `json:"mode"`
	Value json.RawMessage `json:"value"`
}

// effectSlowRaw slow 效果的 JSON 参数。
type effectSlowRaw struct {
	Factor   json.RawMessage `json:"factor"`
	Duration json.RawMessage `json:"duration"`
}

// effectDurationRaw 仅含 duration 的效果（stun/root）JSON 参数。
type effectDurationRaw struct {
	Duration json.RawMessage `json:"duration"`
}

// effectDotRaw dot 效果的 JSON 参数。
type effectDotRaw struct {
	Subtype  string          `json:"subtype"`
	Mode     string          `json:"mode"`
	Value    json.RawMessage `json:"value"`
	Duration json.RawMessage `json:"duration"`
}

// effectWeakenRaw weaken 效果的 JSON 参数。
type effectWeakenRaw struct {
	Amplify  json.RawMessage `json:"amplify"`
	Duration json.RawMessage `json:"duration"`
}

// effectBuffRaw buff/selfBuff 效果的 JSON 参数。
type effectBuffRaw struct {
	Stat  string          `json:"stat"`
	Bonus json.RawMessage `json:"bonus"`
}

// effectGoldRaw gold 效果的 JSON 参数。
type effectGoldRaw struct {
	Amount json.RawMessage `json:"amount"`
}

// effectModifyStatRaw modifyStat 效果的 JSON 参数。
type effectModifyStatRaw struct {
	Stat       string  `json:"stat"`
	Multiplier float64 `json:"multiplier"`
}

func parseEffect(data json.RawMessage) (Effect, error) {
	var env typeEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal effect type: %w", err)
	}

	switch env.Type {
	case "damage":
		return parseDamageEffect(data)
	case "slow":
		return parseSlowEffect(data)
	case "stun":
		return parseStunEffect(data)
	case "root":
		return parseRootEffect(data)
	case "dot":
		return parseDotEffect(data)
	case "weaken":
		return parseWeakenEffect(data)
	case "silence":
		return SilenceEffect{}, nil
	case "buff":
		return parseBuffEffect(data)
	case "selfBuff":
		return parseSelfBuffEffect(data)
	case "gold":
		return parseGoldEffect(data)
	case "modifyStat":
		return parseModifyStatEffect(data)
	default:
		return nil, fmt.Errorf("unknown effect type: %q", env.Type)
	}
}

func parseDamageEffect(data json.RawMessage) (Effect, error) {
	var raw effectDamageRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse damage effect: %w", err)
	}
	mode, err := parseDamageMode(raw.Mode)
	if err != nil {
		return nil, err
	}
	value, err := ParseScaler(raw.Value)
	if err != nil {
		return nil, fmt.Errorf("damage.value: %w", err)
	}
	return DamageEffect{Mode: mode, Value: value}, nil
}

func parseSlowEffect(data json.RawMessage) (Effect, error) {
	var raw effectSlowRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse slow effect: %w", err)
	}
	factor, err := ParseScaler(raw.Factor)
	if err != nil {
		return nil, fmt.Errorf("slow.factor: %w", err)
	}
	duration, err := ParseScaler(raw.Duration)
	if err != nil {
		return nil, fmt.Errorf("slow.duration: %w", err)
	}
	return SlowEffect{Factor: factor, Duration: duration}, nil
}

func parseStunEffect(data json.RawMessage) (Effect, error) {
	var raw effectDurationRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse stun effect: %w", err)
	}
	duration, err := ParseScaler(raw.Duration)
	if err != nil {
		return nil, fmt.Errorf("stun.duration: %w", err)
	}
	return StunEffect{Duration: duration}, nil
}

func parseRootEffect(data json.RawMessage) (Effect, error) {
	var raw effectDurationRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse root effect: %w", err)
	}
	duration, err := ParseScaler(raw.Duration)
	if err != nil {
		return nil, fmt.Errorf("root.duration: %w", err)
	}
	return RootEffect{Duration: duration}, nil
}

func parseDotEffect(data json.RawMessage) (Effect, error) {
	var raw effectDotRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse dot effect: %w", err)
	}
	mode, err := parseDamageMode(raw.Mode)
	if err != nil {
		return nil, err
	}
	value, err := ParseScaler(raw.Value)
	if err != nil {
		return nil, fmt.Errorf("dot.value: %w", err)
	}
	duration, err := ParseScaler(raw.Duration)
	if err != nil {
		return nil, fmt.Errorf("dot.duration: %w", err)
	}
	return DotEffect{
		Subtype:  raw.Subtype,
		Mode:     mode,
		Value:    value,
		Duration: duration,
	}, nil
}

func parseWeakenEffect(data json.RawMessage) (Effect, error) {
	var raw effectWeakenRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse weaken effect: %w", err)
	}
	amplify, err := ParseScaler(raw.Amplify)
	if err != nil {
		return nil, fmt.Errorf("weaken.amplify: %w", err)
	}
	duration, err := ParseScaler(raw.Duration)
	if err != nil {
		return nil, fmt.Errorf("weaken.duration: %w", err)
	}
	return WeakenEffect{Amplify: amplify, Duration: duration}, nil
}

func parseBuffEffect(data json.RawMessage) (Effect, error) {
	var raw effectBuffRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse buff effect: %w", err)
	}
	bonus, err := ParseScaler(raw.Bonus)
	if err != nil {
		return nil, fmt.Errorf("buff.bonus: %w", err)
	}
	return BuffEffect{Stat: raw.Stat, Bonus: bonus}, nil
}

func parseSelfBuffEffect(data json.RawMessage) (Effect, error) {
	var raw effectBuffRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse selfBuff effect: %w", err)
	}
	bonus, err := ParseScaler(raw.Bonus)
	if err != nil {
		return nil, fmt.Errorf("selfBuff.bonus: %w", err)
	}
	return SelfBuffEffect{Stat: raw.Stat, Bonus: bonus}, nil
}

func parseGoldEffect(data json.RawMessage) (Effect, error) {
	var raw effectGoldRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gold effect: %w", err)
	}
	amount, err := ParseScaler(raw.Amount)
	if err != nil {
		return nil, fmt.Errorf("gold.amount: %w", err)
	}
	return GoldEffect{Amount: amount}, nil
}

func parseModifyStatEffect(data json.RawMessage) (Effect, error) {
	var raw effectModifyStatRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse modifyStat effect: %w", err)
	}
	return ModifyStatEffect{Stat: raw.Stat, Multiplier: raw.Multiplier}, nil
}

// ── DamageMode 解析 ─────────────────────────────────────────

// parseDamageMode 将字符串解析为 DamageMode 枚举。
func parseDamageMode(s string) (DamageMode, error) {
	switch s {
	case "flat":
		return DmgFlat, nil
	case "ratio":
		return DmgRatio, nil
	case "hpPercent":
		return DmgHpPercent, nil
	default:
		return 0, fmt.Errorf("unknown damage mode: %q", s)
	}
}
