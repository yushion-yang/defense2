// validate_ability.go — 自定义能力平衡性校验与费用计算。
//
// 职责：
//   - ValidateCustomAbility: 检查自定义能力描述符的平衡性约束
//   - CalcAbilityCost: 根据基元费用和管线数量计算能力总费用
//
// 平衡规则：
//   - Rule 1: onTick + CC 效果（stun/root/slow/silence）必须配 cooldown 条件
//   - Rule 2: 多管线能力有费用加成（每多一条管线 +20%）
//
// 关联：
//   - primitive_meta.go — 各基元的费用定义
//   - descriptor.go — AbilityDescriptor / Pipeline 数据结构
//   - ability_store.go — 保存前调用 ValidateCustomAbility 校验
//   - edit_state.go — EditStateToDescriptor 将编辑状态转为描述符后校验
package descriptor

import (
	"fmt"
	"math"
)

// ccEffectIDs 所有 CC（控制）类效果的 ID 集合。
// 这些效果与 onTick 触发器组合时必须配合冷却条件使用，
// 否则会导致"永久冻结"等 OP 组合。
var ccEffectIDs = map[string]bool{
	"stun":    true,
	"root":    true,
	"slow":    true,
	"silence": true,
}

// spatialSelectorsOnHit 在 onHit 触发时无法正常工作的空间查询选择器。
// 这些选择器依赖 ctx.Enemies 查询范围内敌人，但 onHit 上下文中没有敌人池。
// 特殊处理：aoeRadius 和 chain 会被合成为 HitResult.Splash/Bounce，所以不在此列表中。
var spatialSelectorsOnHit = map[string]bool{
	"allInRange":   true,
	"cone":         true,
	"ring360":      true,
	"random":       true,
	"nearbyAllies": true, // nearbyAllies 在 onHit 时也没有意义（应该用于 onTick）
}

// tickOnlyEffects 仅在 onTick 触发时有意义的效果。
var tickOnlyEffects = map[string]bool{
	"buff":       true,
	"selfBuff":   true,
	"gold":       true,
	"modifyStat": true,
}

// hitOnlyEffects 仅在 onHit 触发时有意义的效果。
var hitOnlyEffects = map[string]bool{
	"crit": true,
}

// ValidateCustomAbility 校验自定义能力的平衡性约束。
//
// 检查规则：
//  1. onTick 触发器 + CC 效果必须有 cooldown 条件
//  2. onHit 触发器 + 空间查询选择器（除 aoeRadius/chain）会返回空目标
//  3. onHit 触发器 + tick 专属效果（buff/gold）无效
//  4. onTick 触发器 + hit 专属效果（crit）无效
//  5. onKill 触发器必须使用空间查询选择器（currentTarget 指向已死敌人）
//
// 返回所有发现的校验错误，空 slice 表示通过校验。
func ValidateCustomAbility(desc *AbilityDescriptor) []ValidationError {
	if desc == nil {
		return nil
	}

	var errs []ValidationError

	for i, p := range desc.Pipelines {
		selectorID := selectorTypeID(p.Selector)

		// Rule 1: onTick + CC 必须有 cooldown
		if p.Trigger == TriggerOnTick && hasCCEffect(p) && !hasCooldownCondition(p) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("pipeline[%d]", i),
				Message: "持续触发的控制效果必须设置冷却时间",
			})
		}

		// Rule 2: onHit + 空间查询选择器（除 aoeRadius/chain 被特殊处理外）
		if p.Trigger == TriggerOnHit && spatialSelectorsOnHit[selectorID] {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("pipeline[%d].selector", i),
				Message: fmt.Sprintf("命中时触发不支持 %s 选择器（无法查询范围内敌人），请改用 currentTarget、aoeRadius 或 chain", selectorID),
			})
		}

		// Rule 3: onHit + tick 专属效果
		for j, eff := range p.Effects {
			effID := effectTypeID(eff)
			if p.Trigger == TriggerOnHit && tickOnlyEffects[effID] {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("pipeline[%d].effects[%d]", i, j),
					Message: fmt.Sprintf("命中时触发不支持 %s 效果（仅适用于每帧触发）", effID),
				})
			}
			// Rule 4: onTick + hit 专属效果
			if p.Trigger == TriggerOnTick && hitOnlyEffects[effID] {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("pipeline[%d].effects[%d]", i, j),
					Message: fmt.Sprintf("每帧触发不支持 %s 效果（仅适用于命中时触发）", effID),
				})
			}
		}

		// Rule 5: onKill + currentTarget（目标已死，应使用空间选择器对周围敌人生效）
		if p.Trigger == TriggerOnKill && selectorID == "currentTarget" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("pipeline[%d].selector", i),
				Message: "击杀时触发不应使用 currentTarget 选择器（目标已死），请改用 aoeRadius 等空间选择器",
			})
		}
	}

	return errs
}

// hasCCEffect 检查管线是否包含控制类效果（stun/root/slow/silence）。
func hasCCEffect(p Pipeline) bool {
	for _, e := range p.Effects {
		switch e.(type) {
		case StunEffect, RootEffect, SlowEffect, SilenceEffect:
			return true
		}
	}
	return false
}

// hasCooldownCondition 检查管线是否包含冷却条件。
func hasCooldownCondition(p Pipeline) bool {
	for _, c := range p.Conditions {
		if _, ok := c.(*CooldownCondition); ok {
			return true
		}
	}
	return false
}

// CalcAbilityCost 根据基元费用和管线数量计算能力总费用。
//
// 计算规则：
//   - 基础费用 = 所有管线中所有基元费用之和
//   - 多管线加成 = 每多一条管线额外 20%（2 管线 1.2x，3 管线 1.4x …）
//
// pipelineMultiplierExtra 为每条额外管线的费用加成比例（0.2 = 20%）。
const pipelineMultiplierExtra = 0.2

func CalcAbilityCost(desc *AbilityDescriptor) int {
	if desc == nil || len(desc.Pipelines) == 0 {
		return 0
	}

	baseCost := sumPrimitiveCosts(desc)

	if len(desc.Pipelines) > 1 {
		multiplier := 1.0 + pipelineMultiplierExtra*float64(len(desc.Pipelines)-1)
		baseCost = int(math.Ceil(float64(baseCost) * multiplier))
	}

	return baseCost
}

// sumPrimitiveCosts 汇总描述符中所有基元的费用。
// 从各类型元数据注册表中查找费用。
func sumPrimitiveCosts(desc *AbilityDescriptor) int {
	// 构建查找表（trigger/condition/selector/effect 各自的 ID→cost 映射）
	triggerCosts := buildCostMap(AllTriggerMeta())
	condCosts := buildCostMap(AllConditionMeta())
	selCosts := buildCostMap(AllSelectorMeta())
	effCosts := buildCostMap(AllEffectMeta())

	total := 0
	for _, p := range desc.Pipelines {
		// 触发器费用
		total += triggerCosts[p.Trigger.String()]

		// 条件费用
		for _, c := range p.Conditions {
			cid := conditionTypeID(c)
			total += condCosts[cid]
		}

		// 选择器费用
		if p.Selector != nil {
			sid := selectorTypeID(p.Selector)
			total += selCosts[sid]
		}

		// 效果费用
		for _, e := range p.Effects {
			eid := effectTypeID(e)
			total += effCosts[eid]
		}
	}

	return total
}

// buildCostMap 从 PrimitiveMeta 切片构建 ID→Cost 映射。
func buildCostMap(metas []PrimitiveMeta) map[string]int {
	m := make(map[string]int, len(metas))
	for _, meta := range metas {
		m[meta.ID] = meta.Cost
	}
	return m
}

// conditionTypeID 从 Condition 实例提取其类型 ID 字符串。
func conditionTypeID(c Condition) string {
	switch c.(type) {
	case ChanceCondition:
		return "chance"
	case *CooldownCondition:
		return "cooldown"
	case HpBelowCondition:
		return "hpBelow"
	case HpAboveCondition:
		return "hpAbove"
	case DistanceMinCondition:
		return "distanceMin"
	case NoNearbyTowerCondition:
		return "noNearbyTower"
	case IsBossCondition:
		return "isBoss"
	case NotBossCondition:
		return "notBoss"
	case *EveryCondition:
		return "every"
	case BuffActiveCondition:
		return "buffActive"
	case BuffAbsentCondition:
		return "buffAbsent"
	default:
		return ""
	}
}

// selectorTypeID 从 Selector 实例提取其类型 ID 字符串。
func selectorTypeID(s Selector) string {
	switch s.(type) {
	case CurrentTargetSelector:
		return "currentTarget"
	case AoeRadiusSelector:
		return "aoeRadius"
	case ChainSelector:
		return "chain"
	case AllInRangeSelector:
		return "allInRange"
	case NearbyAlliesSelector:
		return "nearbyAllies"
	case SelfTowerSelector:
		return "selfTower"
	case ConeSelector:
		return "cone"
	case Ring360Selector:
		return "ring360"
	case RandomSelector:
		return "random"
	default:
		return ""
	}
}

// effectTypeID 从 Effect 实例提取其类型 ID 字符串。
func effectTypeID(e Effect) string {
	switch e.(type) {
	case DamageEffect:
		return "damage"
	case SlowEffect:
		return "slow"
	case StunEffect:
		return "stun"
	case RootEffect:
		return "root"
	case DotEffect:
		return "dot"
	case WeakenEffect:
		return "weaken"
	case SilenceEffect:
		return "silence"
	case BuffEffect:
		return "buff"
	case SelfBuffEffect:
		return "selfBuff"
	case GoldEffect:
		return "gold"
	case ModifyStatEffect:
		return "modifyStat"
	case CritEffect:
		return "crit"
	case PurgeEffect:
		return "purge"
	case TeleportEffect:
		return "teleport"
	default:
		return ""
	}
}
