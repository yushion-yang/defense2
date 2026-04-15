// edit_state.go — EditState ↔ AbilityDescriptor 双向转换。
//
// PipelineEditState / ComponentEditState 是 UI 编辑器使用的纯数据结构，
// 所有参数扁平化为 map[string]float64 便于滑条/输入框绑定。
// 本文件提供两个核心函数：
//   - EditStateToDescriptor: 编辑状态 → AbilityDescriptor（用于保存）
//   - DescriptorToEditState: AbilityDescriptor → 编辑状态（用于加载/编辑已有能力）
//
// 关联：
//   - scene/ability_edit.go 中的同名小写类型（跨包时使用本文件的导出类型）
//   - descriptor.go 中的 Pipeline / Condition / Selector / Effect
package descriptor

import "fmt"

// ── 编辑状态类型 ─────────────────────────────────────

// PipelineEditState 管线编辑状态（可跨包共享的纯数据结构）。
type PipelineEditState struct {
	TriggerID      string                 `json:"triggerID"`
	Conditions     []ComponentEditState   `json:"conditions"`
	SelectorID     string                 `json:"selectorID"`
	SelectorParams map[string]float64     `json:"selectorParams"`
	Effects        []ComponentEditState   `json:"effects"`
}

// ComponentEditState 单个条件或效果组件的编辑状态。
type ComponentEditState struct {
	TypeID string             `json:"typeID"`
	Params map[string]float64 `json:"params"`
}

// ── EditState → Descriptor ──────────────────────────

// EditStateToDescriptor 将编辑状态转换为 AbilityDescriptor（用于保存）。
//
// 转换流程（每条 PipelineEditState）：
//   1. 解析 trigger 字符串 → TriggerType
//   2. 构建 conditions: 逐个 dispatch 到对应构建函数
//   3. 构建 selector: dispatch 到对应构建函数
//   4. 构建 effects: 逐个 dispatch 到对应构建函数
func EditStateToDescriptor(id, name string, pipelines []PipelineEditState) (*AbilityDescriptor, error) {
	compiled := make([]Pipeline, 0, len(pipelines))

	for i, ps := range pipelines {
		p, err := buildPipeline(ps)
		if err != nil {
			return nil, fmt.Errorf("pipeline[%d]: %w", i, err)
		}
		compiled = append(compiled, p)
	}

	return &AbilityDescriptor{
		ID:        id,
		Label:     name,
		Pipelines: compiled,
	}, nil
}

// buildPipeline 从编辑状态构建单条 Pipeline。
func buildPipeline(ps PipelineEditState) (Pipeline, error) {
	// 触发器
	if ps.TriggerID == "" {
		return Pipeline{}, fmt.Errorf("trigger is required")
	}
	trigger, err := ParseTrigger(ps.TriggerID)
	if err != nil {
		return Pipeline{}, fmt.Errorf("trigger: %w", err)
	}

	// 条件
	conditions := make([]Condition, 0, len(ps.Conditions))
	for j, cs := range ps.Conditions {
		c, err := buildCondition(cs)
		if err != nil {
			return Pipeline{}, fmt.Errorf("conditions[%d]: %w", j, err)
		}
		conditions = append(conditions, c)
	}

	// 选择器
	sel, err := buildSelector(ps.SelectorID, ps.SelectorParams)
	if err != nil {
		return Pipeline{}, fmt.Errorf("selector: %w", err)
	}

	// 效果
	effects := make([]Effect, 0, len(ps.Effects))
	for j, es := range ps.Effects {
		e, err := buildEffect(es)
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

// ── 条件构建 ─────────────────────────────────────────

// buildCondition 从 ComponentEditState 构建 Condition 实例。
//
// 支持的 TypeID：
//   chance, cooldown, hpBelow, hpAbove, distanceMin, noNearbyTower,
//   isBoss, notBoss, every, buffActive, buffAbsent
func buildCondition(cs ComponentEditState) (Condition, error) {
	switch cs.TypeID {
	case "chance":
		return ChanceCondition{Rate: buildScaler(cs.Params)}, nil

	case "cooldown":
		return NewCooldownCondition(cs.Params["seconds"]), nil

	case "hpBelow":
		return HpBelowCondition{Threshold: buildScaler(cs.Params)}, nil

	case "hpAbove":
		return HpAboveCondition{Threshold: buildScaler(cs.Params)}, nil

	case "distanceMin":
		return DistanceMinCondition{Distance: cs.Params["distance"]}, nil

	case "noNearbyTower":
		return NoNearbyTowerCondition{Radius: cs.Params["radius"]}, nil

	case "isBoss":
		return IsBossCondition{}, nil

	case "notBoss":
		return NotBossCondition{}, nil

	case "every":
		return NewEveryCondition(int(cs.Params["n"])), nil

	case "buffActive":
		// buffId 是字符串，float map 无法直接存储；由 UI 层另行管理
		return BuffActiveCondition{}, nil

	case "buffAbsent":
		return BuffAbsentCondition{}, nil

	default:
		return nil, fmt.Errorf("unknown condition type: %q", cs.TypeID)
	}
}

// ── 选择器构建 ───────────────────────────────────────

// buildSelector 从类型 ID 和参数构建 Selector 实例。
//
// 支持的类型：
//   currentTarget, aoeRadius, chain, allInRange, nearbyAllies,
//   selfTower, cone, ring360, random
func buildSelector(selectorID string, params map[string]float64) (Selector, error) {
	switch selectorID {
	case "currentTarget":
		return CurrentTargetSelector{}, nil

	case "aoeRadius":
		return AoeRadiusSelector{Radius: buildScaler(params)}, nil

	case "chain":
		// maxBounce 使用 base/potential（或 value），range 和 decayRatio 为固定值
		bounceParams := extractSubParams(params, "base", "potential", "value")
		return ChainSelector{
			MaxBounce:  buildScaler(bounceParams),
			ChainRange: params["range"],
			DecayRatio: params["decayRatio"],
		}, nil

	case "allInRange":
		return AllInRangeSelector{}, nil

	case "nearbyAllies":
		return NearbyAlliesSelector{Radius: params["radius"]}, nil

	case "selfTower":
		return SelfTowerSelector{}, nil

	case "cone":
		radiusParams := extractSubParams(params, "base", "potential", "value")
		return ConeSelector{
			Angle:  params["angle"],
			Radius: buildScaler(radiusParams),
		}, nil

	case "ring360":
		return Ring360Selector{Count: buildScaler(params)}, nil

	case "random":
		countParams := extractSubParams(params, "base", "potential", "value")
		return RandomSelector{
			Count:  buildScaler(countParams),
			Radius: params["radius"],
		}, nil

	case "":
		// 空选择器：默认当前目标
		return CurrentTargetSelector{}, nil

	default:
		return nil, fmt.Errorf("unknown selector type: %q", selectorID)
	}
}

// ── 效果构建 ─────────────────────────────────────────

// buildEffect 从 ComponentEditState 构建 Effect 实例。
//
// 支持的 TypeID：
//   damage, slow, stun, root, dot, weaken, silence,
//   buff, selfBuff, gold, modifyStat, crit, purge, teleport
func buildEffect(es ComponentEditState) (Effect, error) {
	switch es.TypeID {
	case "damage":
		mode := DamageMode(int(es.Params["mode"]))
		// value 参数使用前缀过滤
		valParams := extractScalerParams(es.Params)
		return DamageEffect{Mode: mode, Value: buildScaler(valParams)}, nil

	case "slow":
		factorParams := extractPrefixParams(es.Params, "factor_")
		durParams := extractPrefixParams(es.Params, "duration_")
		return SlowEffect{
			Factor:   buildScaler(factorParams),
			Duration: buildScaler(durParams),
		}, nil

	case "stun":
		return StunEffect{Duration: buildScaler(es.Params)}, nil

	case "root":
		return RootEffect{Duration: buildScaler(es.Params)}, nil

	case "dot":
		subtype := dotSubtypeFromFloat(es.Params["subtype"])
		mode := DamageMode(int(es.Params["mode"]))
		valParams := extractPrefixParams(es.Params, "value_")
		durParams := extractPrefixParams(es.Params, "duration_")
		return DotEffect{
			Subtype:  subtype,
			Mode:     mode,
			Value:    buildScaler(valParams),
			Duration: buildScaler(durParams),
		}, nil

	case "weaken":
		ampParams := extractPrefixParams(es.Params, "amplify_")
		durParams := extractPrefixParams(es.Params, "duration_")
		return WeakenEffect{
			Amplify:  buildScaler(ampParams),
			Duration: buildScaler(durParams),
		}, nil

	case "silence":
		return SilenceEffect{}, nil

	case "buff":
		stat := "" // string 字段需外部设置，float map 无法表示
		return BuffEffect{Stat: stat, Bonus: buildScaler(es.Params)}, nil

	case "selfBuff":
		stat := ""
		return SelfBuffEffect{Stat: stat, Bonus: buildScaler(es.Params)}, nil

	case "gold":
		return GoldEffect{Amount: buildScaler(es.Params)}, nil

	case "modifyStat":
		stat := ""
		return ModifyStatEffect{Stat: stat, Multiplier: es.Params["multiplier"]}, nil

	case "crit":
		return CritEffect{Multiplier: es.Params["multiplier"]}, nil

	case "purge":
		return PurgeEffect{Count: int(es.Params["count"])}, nil

	case "teleport":
		return TeleportEffect{Distance: buildScaler(es.Params)}, nil

	default:
		return nil, fmt.Errorf("unknown effect type: %q", es.TypeID)
	}
}

// ── Descriptor → EditState ──────────────────────────

// DescriptorToEditState 将 AbilityDescriptor 转换为编辑状态（用于加载到 UI）。
//
// 对每条 Pipeline：
//   1. 触发器 → TriggerType.String()
//   2. 条件 → type switch 提取 TypeID + Params
//   3. 选择器 → type switch 提取 SelectorID + SelectorParams
//   4. 效果 → type switch 提取 TypeID + Params
func DescriptorToEditState(desc *AbilityDescriptor) []PipelineEditState {
	result := make([]PipelineEditState, len(desc.Pipelines))

	for i, p := range desc.Pipelines {
		pe := PipelineEditState{
			TriggerID:      p.Trigger.String(),
			SelectorParams: map[string]float64{},
		}

		// 条件
		pe.Conditions = make([]ComponentEditState, 0, len(p.Conditions))
		for _, c := range p.Conditions {
			pe.Conditions = append(pe.Conditions, conditionToEditState(c))
		}

		// 选择器
		if p.Selector != nil {
			sid, params := selectorToEditState(p.Selector)
			pe.SelectorID = sid
			pe.SelectorParams = params
		}

		// 效果
		pe.Effects = make([]ComponentEditState, 0, len(p.Effects))
		for _, e := range p.Effects {
			pe.Effects = append(pe.Effects, effectToEditState(e))
		}

		result[i] = pe
	}

	return result
}

// ── 条件 → EditState ────────────────────────────────

// conditionToEditState 将 Condition 实例转换为 ComponentEditState。
func conditionToEditState(c Condition) ComponentEditState {
	switch v := c.(type) {
	case ChanceCondition:
		return ComponentEditState{TypeID: "chance", Params: scalerToParams(v.Rate)}

	case *CooldownCondition:
		return ComponentEditState{TypeID: "cooldown", Params: map[string]float64{"seconds": v.Seconds}}

	case HpBelowCondition:
		return ComponentEditState{TypeID: "hpBelow", Params: scalerToParams(v.Threshold)}

	case HpAboveCondition:
		return ComponentEditState{TypeID: "hpAbove", Params: scalerToParams(v.Threshold)}

	case DistanceMinCondition:
		return ComponentEditState{TypeID: "distanceMin", Params: map[string]float64{"distance": v.Distance}}

	case NoNearbyTowerCondition:
		return ComponentEditState{TypeID: "noNearbyTower", Params: map[string]float64{"radius": v.Radius}}

	case IsBossCondition:
		return ComponentEditState{TypeID: "isBoss", Params: map[string]float64{}}

	case NotBossCondition:
		return ComponentEditState{TypeID: "notBoss", Params: map[string]float64{}}

	case *EveryCondition:
		return ComponentEditState{TypeID: "every", Params: map[string]float64{"n": float64(v.N)}}

	case BuffActiveCondition:
		return ComponentEditState{TypeID: "buffActive", Params: map[string]float64{}}

	case BuffAbsentCondition:
		return ComponentEditState{TypeID: "buffAbsent", Params: map[string]float64{}}

	default:
		return ComponentEditState{TypeID: fmt.Sprintf("unknown:%T", c), Params: map[string]float64{}}
	}
}

// ── 选择器 → EditState ──────────────────────────────

// selectorToEditState 将 Selector 实例转换为 (selectorID, params)。
func selectorToEditState(s Selector) (string, map[string]float64) {
	switch v := s.(type) {
	case CurrentTargetSelector:
		return "currentTarget", map[string]float64{}

	case AoeRadiusSelector:
		return "aoeRadius", scalerToParams(v.Radius)

	case ChainSelector:
		params := scalerToParams(v.MaxBounce)
		params["range"] = v.ChainRange
		params["decayRatio"] = v.DecayRatio
		return "chain", params

	case AllInRangeSelector:
		return "allInRange", map[string]float64{}

	case NearbyAlliesSelector:
		return "nearbyAllies", map[string]float64{"radius": v.Radius}

	case SelfTowerSelector:
		return "selfTower", map[string]float64{}

	case ConeSelector:
		params := scalerToParams(v.Radius)
		params["angle"] = v.Angle
		return "cone", params

	case Ring360Selector:
		return "ring360", scalerToParams(v.Count)

	case RandomSelector:
		params := scalerToParams(v.Count)
		params["radius"] = v.Radius
		return "random", params

	default:
		return fmt.Sprintf("unknown:%T", s), map[string]float64{}
	}
}

// ── 效果 → EditState ────────────────────────────────

// effectToEditState 将 Effect 实例转换为 ComponentEditState。
func effectToEditState(e Effect) ComponentEditState {
	switch v := e.(type) {
	case DamageEffect:
		params := extractScalerParams4(v.Value)
		params["mode"] = float64(v.Mode)
		return ComponentEditState{TypeID: "damage", Params: params}

	case SlowEffect:
		params := map[string]float64{}
		mergePrefixParams(params, "factor_", v.Factor)
		mergePrefixParams(params, "duration_", v.Duration)
		return ComponentEditState{TypeID: "slow", Params: params}

	case StunEffect:
		return ComponentEditState{TypeID: "stun", Params: scalerToParams(v.Duration)}

	case RootEffect:
		return ComponentEditState{TypeID: "root", Params: scalerToParams(v.Duration)}

	case DotEffect:
		params := map[string]float64{}
		params["subtype"] = dotSubtypeToFloat(v.Subtype)
		params["mode"] = float64(v.Mode)
		mergePrefixParams(params, "value_", v.Value)
		mergePrefixParams(params, "duration_", v.Duration)
		return ComponentEditState{TypeID: "dot", Params: params}

	case WeakenEffect:
		params := map[string]float64{}
		mergePrefixParams(params, "amplify_", v.Amplify)
		mergePrefixParams(params, "duration_", v.Duration)
		return ComponentEditState{TypeID: "weaken", Params: params}

	case SilenceEffect:
		return ComponentEditState{TypeID: "silence", Params: map[string]float64{}}

	case BuffEffect:
		return ComponentEditState{TypeID: "buff", Params: scalerToParams(v.Bonus)}

	case SelfBuffEffect:
		return ComponentEditState{TypeID: "selfBuff", Params: scalerToParams(v.Bonus)}

	case GoldEffect:
		return ComponentEditState{TypeID: "gold", Params: scalerToParams(v.Amount)}

	case ModifyStatEffect:
		return ComponentEditState{TypeID: "modifyStat", Params: map[string]float64{"multiplier": v.Multiplier}}

	case CritEffect:
		return ComponentEditState{TypeID: "crit", Params: map[string]float64{"multiplier": v.Multiplier}}

	case PurgeEffect:
		return ComponentEditState{TypeID: "purge", Params: map[string]float64{"count": float64(v.Count)}}

	case TeleportEffect:
		return ComponentEditState{TypeID: "teleport", Params: scalerToParams(v.Distance)}

	default:
		return ComponentEditState{TypeID: fmt.Sprintf("unknown:%T", e), Params: map[string]float64{}}
	}
}

// ── Scaler ↔ Params 转换辅助 ────────────────────────

// buildScaler 从 params 推断 Scaler 类型并构建。
//
// 推断规则：
//   - 仅含 "value" → FixedScaler
//   - 含 "base" + "potential" + "k" → DiminishingScaler
//   - 含 "base" + "potential" + "cap" → CappedScaler
//   - 含 "base" + "potential" → LinearScaler
//   - 其他 → FixedScaler{Value: 0}
func buildScaler(params map[string]float64) Scaler {
	if params == nil {
		return FixedScaler{Value: 0}
	}

	_, hasValue := params["value"]
	_, hasBase := params["base"]
	_, hasPotential := params["potential"]
	_, hasK := params["k"]
	_, hasCap := params["cap"]

	switch {
	case hasValue && !hasBase:
		return FixedScaler{Value: params["value"]}
	case hasBase && hasPotential && hasK:
		return DiminishingScaler{Base: params["base"], Potential: params["potential"], K: params["k"]}
	case hasBase && hasPotential && hasCap:
		return CappedScaler{Base: params["base"], Potential: params["potential"], Cap: params["cap"]}
	case hasBase && hasPotential:
		return LinearScaler{Base: params["base"], Potential: params["potential"]}
	case hasBase:
		return FixedScaler{Value: params["base"]}
	default:
		return FixedScaler{Value: 0}
	}
}

// scalerToParams 将 Scaler 实例拆解为 map[string]float64。
func scalerToParams(s Scaler) map[string]float64 {
	switch v := s.(type) {
	case FixedScaler:
		return map[string]float64{"value": v.Value}
	case LinearScaler:
		return map[string]float64{"base": v.Base, "potential": v.Potential}
	case DiminishingScaler:
		return map[string]float64{"base": v.Base, "potential": v.Potential, "k": v.K}
	case CappedScaler:
		return map[string]float64{"base": v.Base, "potential": v.Potential, "cap": v.Cap}
	case SteppedScaler:
		// SteppedScaler 无法简洁地映射到 flat params，返回空
		return map[string]float64{}
	default:
		return map[string]float64{}
	}
}

// extractScalerParams4 是 scalerToParams 的别名，用于多参数效果中提取 scaler 部分。
func extractScalerParams4(s Scaler) map[string]float64 {
	return scalerToParams(s)
}

// ── 前缀参数辅助 ────────────────────────────────────

// extractPrefixParams 从 params 中提取带指定前缀的参数，去掉前缀后返回新 map。
// 例如 extractPrefixParams({"factor_base":0.3, "factor_potential":0.1, "mode":0}, "factor_")
// 返回 {"base":0.3, "potential":0.1}
func extractPrefixParams(params map[string]float64, prefix string) map[string]float64 {
	result := map[string]float64{}
	pLen := len(prefix)
	for k, v := range params {
		if len(k) > pLen && k[:pLen] == prefix {
			result[k[pLen:]] = v
		}
	}
	return result
}

// mergePrefixParams 将 Scaler 拆解后的 params 以指定前缀合并到 target 中。
func mergePrefixParams(target map[string]float64, prefix string, s Scaler) {
	sub := scalerToParams(s)
	for k, v := range sub {
		target[prefix+k] = v
	}
}

// extractSubParams 从 params 中仅提取指定的 key，构成新 map。
func extractSubParams(params map[string]float64, keys ...string) map[string]float64 {
	result := map[string]float64{}
	for _, k := range keys {
		if v, ok := params[k]; ok {
			result[k] = v
		}
	}
	return result
}

// extractScalerParams 从效果 params 中提取 scaler 相关的 key（排除 mode 等元数据）。
func extractScalerParams(params map[string]float64) map[string]float64 {
	result := map[string]float64{}
	for k, v := range params {
		if k == "mode" || k == "subtype" {
			continue
		}
		result[k] = v
	}
	return result
}

// ── DoT subtype 编码 ────────────────────────────────

// dotSubtypeToFloat 将 DoT 子类型字符串编码为 float64。
// burn=0, bleed=1, poison=2
func dotSubtypeToFloat(s string) float64 {
	switch s {
	case "burn":
		return 0
	case "bleed":
		return 1
	case "poison":
		return 2
	default:
		return 0
	}
}

// dotSubtypeFromFloat 将 float64 解码为 DoT 子类型字符串。
func dotSubtypeFromFloat(f float64) string {
	switch int(f) {
	case 0:
		return "burn"
	case 1:
		return "bleed"
	case 2:
		return "poison"
	default:
		return "burn"
	}
}

