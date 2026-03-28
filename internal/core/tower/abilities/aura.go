// aura.go — 光环类能力实现。
// 光环能力不在命中时触发，而是通过 tick 持续影响周围塔/敌人。
// 此处注册占位，实际 tick 逻辑由战斗管线中的光环子系统调用。
// 通过 init() 自注册到全局注册表。
package abilities

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&DamageUpAura{})
	tower.Register(&AttackSpeedAura{})
	tower.Register(&RangeAura{})
	tower.Register(&CritAura{})
	tower.Register(&SoloBoost{})
}

// DamageUpAura 伤害光环：提升周围塔伤害。
type DamageUpAura struct{}
func (a *DamageUpAura) Name() string { return "damageUpAura" }
func (a *DamageUpAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil // 光环不触发命中效果
}

// AttackSpeedAura 攻速光环：提升周围塔攻速。
type AttackSpeedAura struct{}
func (a *AttackSpeedAura) Name() string { return "attackSpeedAura" }
func (a *AttackSpeedAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// RangeAura 射程光环：提升周围塔射程。
type RangeAura struct{}
func (a *RangeAura) Name() string { return "rangeAura" }
func (a *RangeAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// CritAura 暴击光环：提升周围塔暴击率。
type CritAura struct{}
func (a *CritAura) Name() string { return "critAura" }
func (a *CritAura) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}

// SoloBoost 独行增益：周围无友军塔时自身伤害大幅提升。
type SoloBoost struct{}
func (a *SoloBoost) Name() string { return "soloBoost" }
func (a *SoloBoost) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return nil
}
