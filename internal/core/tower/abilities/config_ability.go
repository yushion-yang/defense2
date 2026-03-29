// config_ability.go — 数据驱动的统一能力实现。
// 所有能力共用 ConfigAbility 结构，按 AbilityDef.Type 分发 OnHit/OnTick 逻辑。
// 参数从 config.AbilityDef 读取：scaledValue = base + potential * (strength/100)。
package abilities

import (
	"fmt"
	"math"
	"math/rand"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

// BossPercentHpCap Boss 百分比伤害全局上限（游戏规则常量）。
const BossPercentHpCap = 0.05

// lastPercentHpTarget 记录每座塔上次触发 percentHpDamage 的敌人 ID。
// 用于实现"切换目标时首击触发"的机制。
var lastPercentHpTarget = map[string]int{}

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
		// scaleDim=chance, param=multiplier
		if rand.Float64() < sv {
			return &tower.HitResult{
				BonusDamage: p.Damage * (pm - 1), // multiplier 1.8 → +0.8x
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

	case "stackDamage":
		// scaleDim=bonusPerStack, 无固定参数
		return &tower.HitResult{BonusDamage: p.Damage * sv}

	case "executionBonus":
		// scaleDim=damageBonus, param=hpThreshold
		if e.MaxHP > 0 && e.HP/e.MaxHP <= pm {
			return &tower.HitResult{BonusDamage: p.Damage * sv}
		}
		return nil

	case "percentHpDamage":
		// 切换目标时首击触发（猎手印记），同一目标不重复触发
		key := towerAccKey(t)
		if lastID, ok := lastPercentHpTarget[key]; ok && lastID == e.ID {
			return nil // 同一目标，不触发
		}
		lastPercentHpTarget[key] = e.ID
		ratio := sv
		if e.Boss && ratio > BossPercentHpCap {
			ratio = BossPercentHpCap
		}
		return &tower.HitResult{BonusDamage: e.MaxHP * ratio}

	case "percentHpMinor":
		// 每次命中触发的小额百分比伤害
		ratio := sv
		if e.Boss && ratio > BossPercentHpCap {
			ratio = BossPercentHpCap
		}
		return &tower.HitResult{BonusDamage: e.MaxHP * ratio}

	case "flatDamage":
		// scaleDim=damage, 无固定参数
		return &tower.HitResult{BonusDamage: sv}

	case "distanceDamage":
		// scaleDim=maxBonus, 无固定参数
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		ratio := math.Min(dist/t.Range, 1.0)
		return &tower.HitResult{BonusDamage: p.Damage * ratio * sv}

	case "overload":
		// scaleDim=chance, param=multiplier
		if rand.Float64() < sv {
			return &tower.HitResult{BonusDamage: p.Damage * (pm - 1)}
		}
		return nil

	case "onHitSlow":
		// scaleDim=factor, param=duration
		return &tower.HitResult{
			Slow: &tower.SlowEffect{Factor: 1 - sv, Duration: pm},
		}

	case "stun":
		// scaleDim=chance, param=duration
		if rand.Float64() < sv {
			return &tower.HitResult{
				Stun: &tower.StunEffect{Duration: pm},
			}
		}
		return nil

	case "bleedDot":
		// scaleDim=dps, param=duration
		return &tower.HitResult{
			Bleed: &tower.BleedEffect{DPS: sv, Duration: pm},
		}

	case "burn":
		// scaleDim=ratio, param=duration
		return &tower.HitResult{
			Burn: &tower.BleedEffect{DPS: p.Damage * sv, Duration: pm},
		}

	case "root":
		// scaleDim=duration, 无固定参数
		return &tower.HitResult{
			Stun: &tower.StunEffect{Duration: sv}, // 定身复用眩晕效果
		}

	case "judgmentMark":
		// scaleDim=damageAmp, param=duration — 标记目标（简化为额外伤害）
		return &tower.HitResult{BonusDamage: p.Damage * sv}

	case "deathMark":
		// scaleDim=explosionDamage, param=radius — 简化为额外伤害
		return &tower.HitResult{BonusDamage: sv}

	case "buffPurge":
		// scaleDim=shieldDrainPerSec — 命中时剥离护盾
		if e.ShieldHP > 0 {
			drain := sv * 0.5 // 每次命中剥离半秒量
			if drain > e.ShieldHP {
				drain = e.ShieldHP
			}
			e.ShieldHP -= drain
		}
		return nil

	case "multiTarget":
		// 无缩放，param=targets — 逻辑在 pipeline 层处理
		return nil

	case "shieldIgnore":
		// 无参数 — 逻辑在伤害管线层处理
		return nil

	case "goldOnKill":
		// 击杀产金 — 在 OnHit 中无效果
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
		// scaleDim=bonus, param=radius
		srcKey := fmt.Sprintf("dmgAura_%s_%d_%d", t.Key, t.Row, t.Col)
		ctx.Towers.Each(func(other *tower.Tower) {
			if other == t {
				return
			}
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				ensureStr(other)
				other.Strength.SetTemp(srcKey, other.BaseDamage*sv)
			}
		})

	case "attackSpeedAura":
		// scaleDim=bonus, param=radius
		srcKey := fmt.Sprintf("spdAura_%s_%d_%d", t.Key, t.Row, t.Col)
		ctx.Towers.Each(func(other *tower.Tower) {
			if other == t {
				return
			}
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				ensureStr(other)
				other.Strength.SetTemp(srcKey, other.BaseSpeed*sv)
			}
		})

	case "rangeAura":
		// scaleDim=bonus(像素), param=radius
		srcKey := fmt.Sprintf("rngAura_%s_%d_%d", t.Key, t.Row, t.Col)
		ctx.Towers.Each(func(other *tower.Tower) {
			if other == t {
				return
			}
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				ensureStr(other)
				other.Strength.SetTemp(srcKey, sv)
			}
		})

	case "critAura":
		// scaleDim=bonus, param=radius
		srcKey := fmt.Sprintf("critAura_%s_%d_%d", t.Key, t.Row, t.Col)
		ctx.Towers.Each(func(other *tower.Tower) {
			if other == t {
				return
			}
			if math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				ensureStr(other)
				other.Strength.SetTemp(srcKey, other.BaseDamage*sv)
			}
		})

	case "soloBoost":
		// scaleDim=bonus, param=checkRadius
		alone := true
		ctx.Towers.Each(func(other *tower.Tower) {
			if other != t && math.Hypot(t.X-other.X, t.Y-other.Y) <= pm {
				alone = false
			}
		})
		srcKey := fmt.Sprintf("solo_%s_%d_%d", t.Key, t.Row, t.Col)
		ensureStr(t)
		if alone {
			t.Strength.SetTemp(srcKey, t.BaseDamage*sv)
		} else {
			t.Strength.RemoveTemp(srcKey)
		}

	case "poisonZone":
		// scaleDim=dps — 对射程内敌人造成毒伤
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.HP -= sv * ctx.DT
			}
		})

	case "silenceZone":
		// scaleDim=slowFactor — 对射程内敌人减速（受全局减速下限约束）
		factor := 1 - sv
		if factor < combat.MinSpeedRatio {
			factor = combat.MinSpeedRatio
		}
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.Speed = e.BaseSpeed * factor
			}
		})

	case "curseZone":
		// scaleDim=hpPercentPerSec — 对射程内敌人百分比扣血
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.HP -= e.MaxHP * sv * ctx.DT
			}
		})

	case "channelLaser":
		// scaleDim=dps — 对射程内最近敌人造成持续伤害
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) <= t.Range {
				e.HP -= sv * ctx.DT
			}
		})

	case "goldPassive":
		// scaleDim=amount, param=interval
		// 需要冷却计时器，简化为每帧按比例产金
		gold := sv * ctx.DT / pm
		if gold > 0 {
			return &tower.TickResult{GoldEarned: int(gold * 60)} // 近似为整数
		}

	case "goldOnKill":
		// 击杀产金 — 由管线在击杀时处理，tick 无操作
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
