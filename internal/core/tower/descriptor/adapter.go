// adapter.go — EffectResult → HitResult/TickResult 适配层。
//
// 描述符引擎的 Interpreter 产出 []EffectResult，
// 而现有战斗管线 (combat/apply_hit.go) 消费 *tower.HitResult，
// tick 系统 (pipeline/tick_abilities.go) 消费 *tower.TickResult。
// 本文件负责两者之间的桥接转换。
//
// 映射策略：
//   - Flat/HpPercent 伤害 → SeparateDamage（独立走 damageCap 管线）
//   - Ratio 伤害 → BonusDamage（已由 Effect.Apply 预乘 TowerDamage）
//   - 同类效果累加（多个 damage 求和）
//   - CC/DoT 取最后一个（last wins）
//   - Root/Weaken/Silence/Poison → HitResult 新字段（combat 管线施加）
//   - Buff/SelfBuff → tick 管线由 applyTickBuffEffects 直接处理（不走 TickResult）
//   - Gold → TickResult.GoldEarned
package descriptor

import "defense2/internal/core/tower"

// AdaptToHitResult 将描述符引擎的效果列表转为战斗管线的 HitResult。
// 多个同类效果合并（如多个 damage 效果累加）。
// 返回 nil 如果没有任何 onHit 相关效果。
func AdaptToHitResult(results []EffectResult) *tower.HitResult {
	if len(results) == 0 {
		return nil
	}

	var hr tower.HitResult
	hasEffect := false

	for i := range results {
		r := &results[i]
		switch r.Type {
		case EffTypeDamage:
			hasEffect = true
			switch r.DamageMode {
			case DmgFlat, DmgHpPercent:
				hr.SeparateDamage += r.Damage
			case DmgRatio:
				hr.BonusDamage += r.Damage
			}
			if r.IsCrit {
				hr.IsCrit = true
			}

		case EffTypeSlow:
			hasEffect = true
			hr.Slow = &tower.SlowEffect{
				Factor:   r.SlowFactor,
				Duration: r.Duration,
			}

		case EffTypeStun:
			hasEffect = true
			hr.Stun = &tower.StunEffect{
				Duration: r.StunDur,
			}

		case EffTypeDot:
			eff := &tower.BleedEffect{
				DPS:      r.DotValue,
				Duration: r.DotDuration,
			}
			switch r.DotSubtype {
			case "bleed":
				hasEffect = true
				hr.Bleed = eff
			case "burn":
				hasEffect = true
				hr.Burn = eff
			case "poison":
				hasEffect = true
				hr.Poison = eff
			}

		case EffTypeCrit:
			hasEffect = true
			hr.IsCrit = true
			if r.Damage > 0 {
				hr.BonusDamage += r.Damage
			}

		case EffTypeRoot:
			hasEffect = true
			hr.Root = &tower.RootEffect{Duration: r.RootDur}

		case EffTypeWeaken:
			hasEffect = true
			hr.Weaken = &tower.WeakenEffect{
				Amplify:  r.WeakenAmp,
				Duration: r.WeakenDur,
			}

		case EffTypeSilence:
			hasEffect = true
			hr.Silence = true

		// Purge 暂时不进入 HitResult（需要战斗管线后续支持），
		// 但标记 hasEffect 以确保结果不被丢弃。
		case EffTypePurge:
			hasEffect = true

		// Teleport 暂时不进入 HitResult（实际路径回推由战斗管线后续实现），
		// 标记 hasEffect 以确保结果不被丢弃。
		case EffTypeTeleport:
			hasEffect = true

		// 以下类型不进入 HitResult，由 tick 系统单独处理
		case EffTypeBuff, EffTypeSelfBuff:
		case EffTypeGold:
		case EffTypeModifyStat:
		}
	}

	if !hasEffect {
		return nil
	}
	return &hr
}

// TrySynthSplash 检测描述符是否为 splash 模式（onHit + aoeRadius + damage ratio），
// 如果是则直接合成 HitResult.Splash，不走 interpreter 的 per-target 路径。
// 返回 nil 表示不是 splash 模式。
//
// 设计原因：onHit 上下文没有 Enemies 池（只有被命中的单个敌人），
// aoeRadius selector 无法查询范围内敌人。而战斗管线的 applyHitEffectsUnified
// 已有完整的溅射逻辑（递归 ApplyHit、VFX、击杀统计），
// 合成 HitResult.Splash 复用现有管线是最安全的做法。
func TrySynthSplash(desc *AbilityDescriptor, strength float64) *tower.HitResult {
	for _, p := range desc.Pipelines {
		if p.Trigger != TriggerOnHit {
			continue
		}
		aoe, ok := p.Selector.(AoeRadiusSelector)
		if !ok {
			continue
		}
		// 找第一个 DamageEffect(ratio) 作为溅射比例
		for _, eff := range p.Effects {
			dmg, ok := eff.(DamageEffect)
			if !ok || dmg.Mode != DmgRatio {
				continue
			}
			radius := aoe.Radius.Calc(strength)
			ratio := dmg.Value.Calc(strength)
			return &tower.HitResult{
				Splash: &tower.SplashEffect{
					Radius: radius,
					Ratio:  ratio,
				},
			}
		}
	}
	return nil
}

// TrySynthBounce 检测描述符是否为 bounce 模式（onHit + chain selector + damage ratio），
// 如果是则直接合成 HitResult.Bounce。与 TrySynthSplash 同理：
// onHit 上下文没有 Enemies 池，ChainSelector 无法查询附近敌人。
func TrySynthBounce(desc *AbilityDescriptor, strength, towerDamage float64) *tower.HitResult {
	for _, p := range desc.Pipelines {
		if p.Trigger != TriggerOnHit {
			continue
		}
		chain, ok := p.Selector.(ChainSelector)
		if !ok {
			continue
		}
		maxBounces := int(chain.MaxBounce.Calc(strength))
		if maxBounces < 1 {
			maxBounces = 1
		}
		return &tower.HitResult{
			Bounce: &tower.BounceEffect{
				MaxBounces:  maxBounces,
				Range:       chain.ChainRange,
				DamageRatio: chain.DecayRatio,
				SrcDamage:   towerDamage,
			},
		}
	}
	return nil
}

// AdaptToTickResult 将描述符引擎的效果列表转为 tick 管线的 TickResult。
// Gold 通过 TickResult 传递给管线。
// Buff/SelfBuff/Silence/Weaken/Root 等效果由 applyTickBuffEffects 直接应用，
// 不走 TickResult 适配（避免目标信息丢失）。
// 返回 nil 如果没有任何 tick 相关效果。
func AdaptToTickResult(results []EffectResult) *tower.TickResult {
	if len(results) == 0 {
		return nil
	}

	var tr tower.TickResult
	hasEffect := false

	for i := range results {
		r := &results[i]
		switch r.Type {
		case EffTypeGold:
			hasEffect = true
			tr.GoldEarned += int(r.GoldAmount)
		// Buff/SelfBuff/Silence/Weaken/Root 由 OnTick 的 applyTickBuffEffects 直接处理
		case EffTypeBuff, EffTypeSelfBuff, EffTypeSilence, EffTypeWeaken, EffTypeRoot:
			hasEffect = true
		}
	}

	if !hasEffect {
		return nil
	}
	return &tr
}
