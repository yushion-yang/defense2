// aura.go — 光环类能力实现。
// 光环能力通过 Ticker 接口持续影响周围塔。
// 通过 init() 自注册到全局注册表。
package abilities

import (
	"fmt"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&DamageUpAura{})
	tower.Register(&AttackSpeedAura{})
	tower.Register(&RangeAura{})
	tower.Register(&CritAura{})
	tower.Register(&SoloBoost{})
}

const auraRadius = 150.0   // 光环影响半径（像素）
const soloCheckRadius = 120.0 // 独行检测半径（像素）

// DamageUpAura 伤害光环：提升周围塔 15% 伤害。
type DamageUpAura struct{}

func (a *DamageUpAura) Name() string { return "damageUpAura" }
func (a *DamageUpAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 光环不触发命中效果
}
func (a *DamageUpAura) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	srcKey := fmt.Sprintf("dmgAura_%s_%d_%d", t.Key, t.Row, t.Col)
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= auraRadius {
			// 通过战力系统临时加成（15%基础伤害折算为战力值）
			bonus := other.BaseDamage * 0.15
			ensureTowerStrength(other)
			if sd, ok := other.Strength.(*strength.StrengthData); ok {
				sd.SetTemp(srcKey, bonus)
			}
		}
	})
	return nil
}

// AttackSpeedAura 攻速光环：提升周围塔 10% 攻速。
type AttackSpeedAura struct{}

func (a *AttackSpeedAura) Name() string { return "attackSpeedAura" }
func (a *AttackSpeedAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *AttackSpeedAura) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= auraRadius {
			other.AttackSpeed += other.BaseSpeed * 0.10
		}
	})
	return nil
}

// RangeAura 射程光环：提升周围塔 20px 射程。
type RangeAura struct{}

func (a *RangeAura) Name() string { return "rangeAura" }
func (a *RangeAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *RangeAura) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= auraRadius {
			other.Range += 20
		}
	})
	return nil
}

// CritAura 暴击光环：提升周围塔伤害（简化替代暴击率系统）。
type CritAura struct{}

func (a *CritAura) Name() string { return "critAura" }
func (a *CritAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *CritAura) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	srcKey := fmt.Sprintf("critAura_%s_%d_%d", t.Key, t.Row, t.Col)
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= auraRadius {
			bonus := other.BaseDamage * 0.10
			ensureTowerStrength(other)
			if sd, ok := other.Strength.(*strength.StrengthData); ok {
				sd.SetTemp(srcKey, bonus)
			}
		}
	})
	return nil
}

// SoloBoost 独行增益：周围无友军塔时自身伤害 +30%。
type SoloBoost struct{}

func (a *SoloBoost) Name() string { return "soloBoost" }
func (a *SoloBoost) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
func (a *SoloBoost) OnTick(t *tower.Tower, ctx *tower.TickContext) *tower.TickResult {
	alone := true
	ctx.Towers.Each(func(other *tower.Tower) {
		if other == t {
			return
		}
		if distBetweenTowers(t, other) <= soloCheckRadius {
			alone = false
		}
	})
	srcKey := fmt.Sprintf("soloBoost_%s_%d_%d", t.Key, t.Row, t.Col)
	ensureTowerStrength(t)
	if sd, ok := t.Strength.(*strength.StrengthData); ok {
		if alone {
			sd.SetTemp(srcKey, t.BaseDamage*0.30)
		} else {
			sd.RemoveTemp(srcKey)
		}
	}
	return nil
}

// distBetweenTowers 计算两座塔之间的像素距离。
func distBetweenTowers(a, b *tower.Tower) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

// ensureTowerStrength 确保塔有 StrengthData（懒初始化）。
func ensureTowerStrength(t *tower.Tower) {
	if t.Strength == nil {
		t.Strength = strength.NewStrengthData()
	}
}
