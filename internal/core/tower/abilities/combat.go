// combat.go — 战斗类能力实现。
// 包含溅射、暴击、弹射、叠伤、斩杀、距离伤害、百分比伤害等战斗能力。
// 通过 init() 自注册到全局注册表。
package abilities

import (
	"math"
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&SplashAbility{})
	tower.Register(&CritAbility{})
	tower.Register(&BounceAbility{})
	tower.Register(&StackDamage{})
	tower.Register(&ExecutionBonus{})
	tower.Register(&DistanceDamage{})
	tower.Register(&PercentHpDamage{})
	tower.Register(&FlatDamage{})
	tower.Register(&ShieldIgnore{})
	tower.Register(&Overload{})
}

// SplashAbility 溅射能力：命中时对周围敌人造成范围伤害。
type SplashAbility struct{}

func (a *SplashAbility) Name() string { return "splash" }
func (a *SplashAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Splash: &tower.SplashEffect{Radius: 50, Ratio: 0.4}, // 50像素半径，40%伤害
	}
}

// CritAbility 暴击能力：25% 概率造成 80% 额外伤害。
type CritAbility struct{}

func (a *CritAbility) Name() string { return "crit" }
func (a *CritAbility) OnHit(_ *tower.Tower, p *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	if rand.Float64() < 0.25 {
		return &tower.HitResult{
			BonusDamage: p.Damage * 0.8,
			IsCrit:      true,
		}
	}
	return nil
}

// BounceAbility 弹射能力：弹射物命中后跳跃到附近敌人。
type BounceAbility struct{}
func (a *BounceAbility) Name() string { return "bounce" }
func (a *BounceAbility) OnHit(t *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	// 弹射范围 = 塔射程（确保能覆盖到附近敌人）
	bounceRange := t.Range
	if bounceRange < 150 {
		bounceRange = 150
	}
	return &tower.HitResult{
		Bounce: &tower.BounceEffect{MaxBounces: 2, Range: bounceRange, DamageDecay: 0.8},
	}
}

// StackDamage 叠伤能力：连续命中额外 8% 伤害。
type StackDamage struct{}

func (a *StackDamage) Name() string { return "stackDamage" }
func (a *StackDamage) OnHit(_ *tower.Tower, p *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{BonusDamage: p.Damage * 0.08}
}

// ExecutionBonus 斩杀能力：目标 HP ≤50% 时额外 50% 伤害。
type ExecutionBonus struct{}

func (a *ExecutionBonus) Name() string { return "executionBonus" }
func (a *ExecutionBonus) OnHit(_ *tower.Tower, p *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	if e.HP/e.MaxHP <= 0.5 {
		return &tower.HitResult{BonusDamage: p.Damage * 0.5}
	}
	return nil
}

// DistanceDamage 距离伤害能力：目标越远伤害越高。
type DistanceDamage struct{}
func (a *DistanceDamage) Name() string { return "distanceDamage" }
func (a *DistanceDamage) OnHit(t *tower.Tower, p *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	dist := math.Hypot(e.X-t.X, e.Y-t.Y)
	ratio := math.Min(dist/t.Range, 1.0)
	return &tower.HitResult{BonusDamage: p.Damage * ratio * 0.5}
}

// PercentHpDamage 百分比伤害能力：按目标最大血量 20% 造成额外伤害（Boss 上限 5%）。
type PercentHpDamage struct{}

func (a *PercentHpDamage) Name() string { return "percentHpDamage" }
func (a *PercentHpDamage) OnHit(_ *tower.Tower, _ *projectile.Projectile, e *enemy.Enemy) *tower.HitResult {
	ratio := 0.20
	if e.Boss {
		ratio = 0.05
	}
	bonus := e.MaxHP * ratio
	return &tower.HitResult{BonusDamage: bonus}
}

// FlatDamage 固定额外伤害。
type FlatDamage struct{}
func (a *FlatDamage) Name() string { return "flatDamage" }
func (a *FlatDamage) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{BonusDamage: 5}
}

// ShieldIgnore 无视护盾：伤害直接作用于血量。
type ShieldIgnore struct{}
func (a *ShieldIgnore) Name() string { return "shieldIgnore" }
func (a *ShieldIgnore) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 由伤害管线特殊处理
}

// Overload 过载能力：小概率造成双倍伤害。
type Overload struct{}
func (a *Overload) Name() string { return "overload" }
func (a *Overload) OnHit(_ *tower.Tower, p *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	if rand.Float64() < 0.15 {
		return &tower.HitResult{BonusDamage: p.Damage}
	}
	return nil
}
