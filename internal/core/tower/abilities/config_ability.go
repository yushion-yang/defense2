// config_ability.go — 数据驱动的统一能力实现。
// 所有能力共用 ConfigAbility 结构，按 AbilityDef.Type 分发 OnHit/OnTick 逻辑。
// 参数从 config.AbilityDef 读取：scaledValue = base + potential * (strength/100)。
package abilities

import (
	"math"
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
	"defense2/internal/i18n"
)

// InitConfigAbilities 加载能力配置表并注册所有数据驱动的能力。
// 必须在 config.SetDataFS() 之后调用。
func InitConfigAbilities() error {
	table, err := config.LoadAbilityTable()
	if err != nil {
		return err
	}
	RegisterConfigAbilities(table)
	return nil
}

// ConfigAbility 数据驱动的统一能力实现。
// 一个实例对应一个能力定义，按 Def.Type 分发行为。
type ConfigAbility struct {
	Def *config.AbilityDef // 能力配置定义
}

func (a *ConfigAbility) Name() string { return a.Def.Type }

// towerStrength 获取塔的有效战力（nil 安全，默认100）。
func towerStrength(t *tower.Tower) float64 {
	if t.Strength != nil {
		return t.Strength.Effective()
	}
	return 100
}

// OnHit 命中时按 type 分发处理。
func (a *ConfigAbility) OnHit(t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	str := towerStrength(t)
	sv := a.Def.CalcScale(str) // 缩放维度实际值
	pm := a.Def.Param          // 固定常量参数

	switch a.Def.Type {
	case tower.AbilitySplash:
		// scaleDim=ratio, param=radius
		return &tower.HitResult{
			Splash: &tower.SplashEffect{Radius: pm, Ratio: sv},
		}

	case tower.AbilityCrit:
		// scaleDim=chance — 概率 = sv + CritBonus(critAura), param=multiplier(1.8)
		if rand.Float64() < sv+t.CritBonus {
			return &tower.HitResult{
				BonusDamage: p.Damage * (pm - 1), // pm=1.8 → +0.8x = 1.8 倍
				IsCrit:      true,
			}
		}
		return nil

	case tower.AbilityBounce:
		// scaleDim=maxBounces, param=damageRatio, param2=bounceRange
		bounceRange := t.Range
		if a.Def.Param2 > 0 {
			bounceRange = a.Def.Param2
		}
		return &tower.HitResult{
			Bounce: &tower.BounceEffect{
				MaxBounces:  int(math.Floor(sv)),
				Range:       bounceRange,
				DamageRatio: pm,
				SrcDamage:   t.Damage,
			},
		}

	case tower.AbilityMomentum:
		// scaleDim=bonusRatio — 每次攻击附加 sv% 额外伤害
		return &tower.HitResult{BonusDamage: p.Damage * sv}

	case tower.AbilityExecutionBonus:
		// scaleDim=damageBonus, param=hpThreshold
		if e.MaxHP > 0 && e.HP/e.MaxHP <= pm {
			return &tower.HitResult{BonusDamage: p.Damage * sv}
		}
		return nil

	case tower.AbilityFlatDamage:
		// scaleDim=damage, 独立二段伤害（对抗 damageCap）
		return &tower.HitResult{SeparateDamage: sv}

	case tower.AbilityDistanceDamage:
		// scaleDim=maxBonus, 无固定参数
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		if t.Range <= 0 {
			return nil
		}
		ratio := dist / t.Range
		if ratio > 1 {
			ratio = 1
		}
		bonus := p.Damage * ratio * sv
		return &tower.HitResult{BonusDamage: bonus}

	case tower.AbilitySlowPower:
		// scaleDim=factor(强度提升减速值), param=duration(固定时长)
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - sv, Duration: pm},
		}

	case tower.AbilitySlowDuration:
		// scaleDim=duration(强度提升持续时间), param=factor(固定减速值)
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - pm, Duration: sv},
		}

	case tower.AbilityStunChance:
		// scaleDim=chance(强度提升概率), param=duration(固定时长)
		if rand.Float64() < sv {
			return &tower.HitResult{
				Stun: &tower.StunEffect{Duration: pm},
			}
		}
		return nil

	case tower.AbilityStunDuration:
		// scaleDim=duration(强度提升时长), param=chance(固定概率)
		if rand.Float64() < pm {
			return &tower.HitResult{
				Stun: &tower.StunEffect{Duration: sv},
			}
		}
		return nil

	case tower.AbilityBleedDot:
		// scaleDim=hpPercent — 每秒失去 sv% 最大生命值，对 Boss 无效
		if e.Boss {
			return nil
		}
		dps := e.MaxHP * sv // sv=0.01 → 1%HP/s
		return &tower.HitResult{
			Bleed: &tower.BleedEffect{DPS: dps, Duration: pm},
		}

	case tower.AbilityBurn:
		// scaleDim=ratio, param=duration
		return &tower.HitResult{
			Burn: &tower.BleedEffect{DPS: p.Damage * sv, Duration: pm},
		}

	case tower.AbilityPoison:
		// scaleDim=dps, param=duration — 固定 DPS 中毒（独立于 bleed）
		wasPoisoned := e.Buffs.Has(buff.IDPoison)
		e.Buffs.Add(buff.Buff{
			ID:        buff.IDPoison,
			Category:  buff.CatDoT,
			Source:    t.InstanceKey,
			Value:     sv,
			Duration:  pm,
			Remaining: pm,
		})
		if !wasPoisoned {
			e.SetFloatText(i18n.T("combat.poison"), 100, 200, 60)
		}
		return nil

	case tower.AbilityWeaken:
		// scaleDim=amplify, param=duration — 命中后受伤增加
		wasWeakened := e.Buffs.Has(buff.IDWeaken)
		e.Buffs.Add(buff.Buff{
			ID:        buff.IDWeaken,
			Category:  buff.CatDebuff,
			Source:    t.InstanceKey,
			Value:     sv,
			Duration:  pm,
			Remaining: pm,
		})
		if !wasWeakened {
			e.SetFloatText(i18n.T("combat.weaken"), 180, 100, 220)
		}
		return nil

	case tower.AbilityStackDamage:
		// scaleDim=bonusPerStack, param=maxStacks — 连续命中同一目标叠加伤害
		stacks := e.HitStacks[t.InstanceKey]
		if stacks >= int(pm) {
			stacks = int(pm)
		}
		bonus := p.Damage * sv * float64(stacks)
		e.IncHitStack(t.InstanceKey, int(pm))
		return &tower.HitResult{BonusDamage: bonus}

	case tower.AbilityPercentHp:
		// scaleDim=hpPercent — 命中时额外造成目标百分比生命的伤害，Boss 减半
		dmg := e.MaxHP * sv
		if e.Boss {
			dmg *= 0.5
		}
		return &tower.HitResult{BonusDamage: dmg}

	case tower.AbilityPercentHpMinor:
		// scaleDim=hpPercent — 微量百分比生命伤害（无 Boss 减免）
		return &tower.HitResult{BonusDamage: e.MaxHP * sv}

	case tower.AbilityOnHitSlow:
		// scaleDim=factor, param=duration — 命中减速（不占 CC 槽位，归类 damage）
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - sv, Duration: pm},
		}

	case tower.AbilityDeathMark:
		// 不在 OnHit 中处理 — 击杀时由 pipeline 检查 srcTower 是否有此能力
		return nil

	case tower.AbilityEnhance:
		// 强化能力 — 选择时一次性提升基础属性，OnHit 无额外效果
		return nil

	case tower.AbilityMultiTarget:
		// 无缩放，param=targets — 逻辑在 pipeline 层处理
		return nil

	}

	return nil
}

// OnTick 每帧 tick 按 type 分发处理（实现 Ticker 接口）。
func (a *ConfigAbility) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	str := towerStrength(t)
	sv := a.Def.CalcScale(str)
	pm := a.Def.Param

	switch a.Def.Type {
	case tower.AbilityDamageUpAura:
		// scaleDim=bonus(比例), param=radius — via BuffList aura:damageAmp
		srcKey := "dmgAura_" + t.InstanceKey
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Buffs.Add(buff.Buff{
					ID: buff.IDAuraDamageAmp, Category: buff.CatAura,
					Source: srcKey, Value: sv,
					Duration: 0.3, Remaining: 0.3,
				})
			}
		})

	case tower.AbilityAttackSpeedAura:
		// scaleDim=bonus(比例), param=radius — via BuffList aura:pctSpeed
		srcKey := "spdAura_" + t.InstanceKey
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Buffs.Add(buff.Buff{
					ID: buff.IDAuraPctSpeed, Category: buff.CatAura,
					Source: srcKey, Value: sv,
					Duration: 0.3, Remaining: 0.3,
				})
			}
		})

	case tower.AbilityRangeAura:
		// scaleDim=bonus(像素), param=radius — via BuffList aura:flatRange
		srcKey := "rngAura_" + t.InstanceKey
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Buffs.Add(buff.Buff{
					ID: buff.IDAuraFlatRange, Category: buff.CatAura,
					Source: srcKey, Value: sv,
					Duration: 0.3, Remaining: 0.3,
				})
			}
		})

	case tower.AbilityCritAura:
		// scaleDim=bonus(暴击率加成), param=radius — via BuffList aura:crit
		srcKey := "critAura_" + t.InstanceKey
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Buffs.Add(buff.Buff{
					ID: buff.IDAuraCrit, Category: buff.CatAura,
					Source: srcKey, Value: sv,
					Duration: 0.3, Remaining: 0.3,
				})
			}
		})

	case tower.AbilitySoloBoost:
		// scaleDim=bonus, param=checkRadius — via BuffList aura:damageAmp (self)
		alone := true
		ctx.Towers.Each(func(other *tower.Tower) {
			if other != t && math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				alone = false
			}
		})
		if alone {
			srcKey := "solo_" + t.InstanceKey
			t.Buffs.Add(buff.Buff{
				ID: buff.IDAuraDamageAmp, Category: buff.CatAura,
				Source: srcKey, Value: sv,
				Duration: 0.3, Remaining: 0.3,
			})
		}

	case tower.AbilityPoisonZone:
		// scaleDim=dps — 对射程内敌人造成毒伤（累积到 ZoneDmgAccum，按 DotTick 结算）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.ZoneDmgAccum += sv * ctx.DT
			}
		})

	case tower.AbilitySilenceZone:
		// 射程内敌人沉默（禁用 DamageCap + 禁用怪物可沉默能力）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if e.IsDying() || e.IsSpawning() {
				return
			}
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.Silenced = true        // 禁用 DamageCap
				e.AbilitySilenced = true // 禁用可沉默的怪物能力
			}
		})

	case tower.AbilityCurseZone:
		// scaleDim=hpPercentPerSec — 对射程内敌人百分比扣血（累积到 ZoneDmgAccum）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.ZoneDmgAccum += e.MaxHP * sv * ctx.DT
			}
		})

	case tower.AbilityWeakenZone:
		// scaleDim=amplify — 射程内敌人受伤增加 (per-frame, short duration)
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.Buffs.Add(buff.Buff{
					ID:        buff.IDWeaken,
					Category:  buff.CatDebuff,
					Source:    "zone_" + t.InstanceKey,
					Value:     sv,
					Duration:  0.2,
					Remaining: 0.2,
				})
			}
		})

	case tower.AbilityGoldPassive:
		// scaleDim=amount, param=interval — 每 interval 秒产生 floor(amount) 金币
		t.GoldCooldown -= ctx.DT
		if t.GoldCooldown <= 0 {
			t.GoldCooldown += pm // 重置冷却（用 += 而非 = 避免累积误差）
			if t.GoldCooldown <= 0 {
				t.GoldCooldown = pm
			}
			gold := int(sv) // 取整
			if gold < 1 {
				gold = 1
			}
			return &tower.TickResult{GoldEarned: gold}
		}

	}

	return nil
}

// RegisterConfigAbilities 从能力表注册所有数据驱动的能力。
// 替代各文件 init() 中的硬编码注册。
func RegisterConfigAbilities(table config.AbilityTable) {
	for _, def := range table {
		tower.Register(&ConfigAbility{Def: def})
	}
}
