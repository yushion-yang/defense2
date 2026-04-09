// config_ability.go — 数据驱动的统一能力实现。
// 所有能力共用 ConfigAbility 结构，按 AbilityDef.Type 分发 OnHit/OnTick 逻辑。
// 参数从 config.AbilityDef 读取：scaledValue = base + potential * (strength/100)。
package abilities

import (
	"fmt"
	"math"
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
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
	case "splash":
		// scaleDim=ratio, param=radius
		return &tower.HitResult{
			Splash: &tower.SplashEffect{Radius: pm, Ratio: sv},
		}

	case "crit":
		// scaleDim=chance — 概率 = sv + CritBonus(critAura), param=multiplier(1.8)
		if rand.Float64() < sv+t.CritBonus {
			return &tower.HitResult{
				BonusDamage: p.Damage * (pm - 1), // pm=1.8 → +0.8x = 1.8 倍
				IsCrit:      true,
			}
		}
		return nil

	case "bounce":
		// scaleDim=maxBounces, param=damageRatio（弹射伤害 = 塔伤害 * ratio）
		bounceRange := t.Range
		if bounceRange < 150 {
			bounceRange = 150
		}
		return &tower.HitResult{
			Bounce: &tower.BounceEffect{
				MaxBounces:  int(math.Floor(sv)),
				Range:       bounceRange,
				DamageRatio: pm,
				SrcDamage:   t.Damage,
			},
		}

	case "momentum":
		// scaleDim=bonusRatio — 每次攻击附加 sv% 额外伤害
		return &tower.HitResult{BonusDamage: p.Damage * sv}

	case "executionBonus":
		// scaleDim=damageBonus, param=hpThreshold
		if e.MaxHP > 0 && e.HP/e.MaxHP <= pm {
			return &tower.HitResult{BonusDamage: p.Damage * sv}
		}
		return nil

	case "flatDamage":
		// scaleDim=damage, 无固定参数
		return &tower.HitResult{BonusDamage: sv}

	case "distanceDamage":
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

	case "slowPower":
		// scaleDim=factor(强度提升减速值), param=duration(固定时长)
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - sv, Duration: pm},
		}

	case "slowDuration":
		// scaleDim=duration(强度提升持续时间), param=factor(固定减速值)
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - pm, Duration: sv},
		}

	case "stun", "stunChance":
		// scaleDim=chance(强度提升概率), param=duration(固定时长)
		if rand.Float64() < sv {
			return &tower.HitResult{
				Stun: &tower.StunEffect{Duration: pm},
			}
		}
		return nil

	case "stunDuration":
		// scaleDim=duration(强度提升时长), param=chance(固定概率)
		if rand.Float64() < pm {
			return &tower.HitResult{
				Stun: &tower.StunEffect{Duration: sv},
			}
		}
		return nil

	case "bleedDot":
		// scaleDim=hpPercent — 每秒失去 sv% 最大生命值，对 Boss 无效
		if e.Boss {
			return nil
		}
		dps := e.MaxHP * sv // sv=0.01 → 1%HP/s
		return &tower.HitResult{
			Bleed: &tower.BleedEffect{DPS: dps, Duration: pm},
		}

	case "burn":
		// scaleDim=ratio, param=duration
		return &tower.HitResult{
			Burn: &tower.BleedEffect{DPS: p.Damage * sv, Duration: pm},
		}

	case "poison":
		// scaleDim=dps, param=duration — 固定 DPS 中毒（独立于 bleed）
		e.PoisonTimer = pm
		e.PoisonDPS = sv
		return nil

	case "weaken":
		// scaleDim=amplify, param=duration — 命中后受伤增加（取较强效果）
		if sv > e.DamageAmplify {
			e.DamageAmplify = sv
		}
		e.DamageAmplifyTimer = pm
		return nil

	case "deathMark":
		// 不在 OnHit 中处理 — 击杀时由 pipeline 检查 srcTower 是否有此能力
		return nil

	case "enhance":
		// 强化能力 — 选择时一次性提升基础属性，OnHit 无额外效果
		return nil

	case "multiTarget":
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
	case "damageUpAura":
		// scaleDim=bonus(比例), param=radius — 写入 DamageAmp（伤害管线统一乘算）
		srcKey := fmt.Sprintf("dmgAura_%s_%d_%d", t.Key, t.Row, t.Col)
		desc := fmt.Sprintf("+%.0f%%全伤害", sv*100)
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.DamageAmp += sv
				applyBuffDisplay(other, srcKey, "伤害光环", desc)
			} else {
				removeBuffDisplay(other, srcKey)
			}
		})

	case "attackSpeedAura":
		// scaleDim=bonus(比例), param=radius — 写入 Mods.PctSpeed
		srcKey := fmt.Sprintf("spdAura_%s_%d_%d", t.Key, t.Row, t.Col)
		desc := fmt.Sprintf("+%.0f%%攻速", sv*100)
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Mods.PctSpeed += sv
				other.RecalcStats()
				applyBuffDisplay(other, srcKey, "攻速光环", desc)
			} else {
				removeBuffDisplay(other, srcKey)
			}
		})

	case "rangeAura":
		// scaleDim=bonus(像素), param=radius — 写入 Mods.FlatRange
		srcKey := fmt.Sprintf("rngAura_%s_%d_%d", t.Key, t.Row, t.Col)
		desc := fmt.Sprintf("+%.0f射程", sv)
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.Mods.FlatRange += sv
				other.RecalcStats()
				applyBuffDisplay(other, srcKey, "射程光环", desc)
			} else {
				removeBuffDisplay(other, srcKey)
			}
		})

	case "critAura":
		// scaleDim=bonus(暴击率加成), param=radius — CritBonus 每帧重置
		srcKey := fmt.Sprintf("critAura_%s_%d_%d", t.Key, t.Row, t.Col)
		desc := fmt.Sprintf("+%.0f%%暴击", sv*100)
		ctx.Towers.Each(func(other *tower.Tower) {
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				other.CritBonus += sv
				applyBuffDisplay(other, srcKey, "暴击光环", desc)
			} else {
				removeBuffDisplay(other, srcKey)
			}
		})

	case "soloBoost":
		// scaleDim=bonus, param=checkRadius — 写入 Mods.PctDamage
		alone := true
		ctx.Towers.Each(func(other *tower.Tower) {
			if other != t && math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				alone = false
			}
		})
		srcKey := fmt.Sprintf("solo_%s_%d_%d", t.Key, t.Row, t.Col)
		if alone {
			t.Mods.PctDamage += sv
			t.RecalcStats()
			applyBuffDisplay(t, srcKey, "独行加成", fmt.Sprintf("+%.0f%%伤害", sv*100))
		} else {
			removeBuffDisplay(t, srcKey)
		}

	case "poisonZone":
		// scaleDim=dps — 对射程内敌人造成毒伤（累积到 ZoneDmgAccum，按 DotTick 结算）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.ZoneDmgAccum += sv * ctx.DT
			}
		})

	case "silenceZone":
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

	case "curseZone":
		// scaleDim=hpPercentPerSec — 对射程内敌人百分比扣血（累积到 ZoneDmgAccum）
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.ZoneDmgAccum += e.MaxHP * sv * ctx.DT
			}
		})

	case "weakenZone":
		// scaleDim=amplify — 射程内敌人受伤增加
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				if sv > e.DamageAmplify {
					e.DamageAmplify = sv // 取最强的一个 zone 效果，不叠加
				}
				e.DamageAmplifyTimer = 0.2 // 短 timer，每帧在 zone 内刷新；离开后自然过期
			}
		})

	case "goldPassive":
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

// ensureStr 确保塔有战力数据。
func ensureStr(t *tower.Tower) {
	if t.Strength == nil {
		t.Strength = strength.NewStrengthData()
	}
}

// RegisterConfigAbilities 从能力表注册所有数据驱动的能力。
// 替代各文件 init() 中的硬编码注册。
func RegisterConfigAbilities(table config.AbilityTable) {
	for _, def := range table {
		tower.Register(&ConfigAbility{Def: def})
	}
}

// applyAura 通用光环施加：对范围内塔（含自身）ApplyBuff，范围外 RemoveBuff。
func applyAura(src *tower.Tower, ctx *tower.TickContext, srcKey string, radius float64,
	calcValue func(other *tower.Tower) float64, source, desc string) {
	ctx.Towers.Each(func(other *tower.Tower) {
		ensureStr(other)
		if math.Hypot(src.X-other.X, src.Y-other.Y) <= radius {
			val := calcValue(other)
			other.ApplyBuff(tower.TowerBuff{
				Key: srcKey, Source: source, Desc: desc,
				Value: val, Duration: -1, Remaining: -1,
			})
		} else {
			other.RemoveBuff(srcKey)
		}
	})
}

// applyBuffDisplay 仅在 Tower.Buffs 中添加/更新一条显示记录（不走 Strength）。
// 用于 critAura 等不通过 Strength 系统的光环。
func applyBuffDisplay(t *tower.Tower, key, source, desc string) {
	for i := range t.Buffs {
		if t.Buffs[i].Key == key {
			t.Buffs[i].Source = source
			t.Buffs[i].Desc = desc
			return
		}
	}
	t.Buffs = append(t.Buffs, tower.TowerBuff{
		Key: key, Source: source, Desc: desc,
		Duration: -1, Remaining: -1,
	})
}

// removeBuffDisplay 移除 Tower.Buffs 中的显示记录（不动 Strength）。
func removeBuffDisplay(t *tower.Tower, key string) {
	for i := range t.Buffs {
		if t.Buffs[i].Key == key {
			t.Buffs = append(t.Buffs[:i], t.Buffs[i+1:]...)
			return
		}
	}
}
