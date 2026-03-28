// control.go — 控制类能力实现。
// 包含命中减速（onHitSlow）和流血（bleedDot）两种能力，通过 init() 自注册。
package abilities

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&OnHitSlowAbility{})
	tower.Register(&BleedDotAbility{})
}

// OnHitSlowAbility 命中减速能力：命中时使敌人减至 60% 速度，持续 1 秒。
type OnHitSlowAbility struct{}

func (a *OnHitSlowAbility) Name() string { return "onHitSlow" }
func (a *OnHitSlowAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Slow: &tower.SlowEffect{Factor: 0.6, Duration: 1.0},
	}
}

// BleedDotAbility 流血能力：命中后持续 3 秒，每秒 5 点伤害。
type BleedDotAbility struct{}

func (a *BleedDotAbility) Name() string { return "bleedDot" }
func (a *BleedDotAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Bleed: &tower.BleedEffect{DPS: 5, Duration: 3},
	}
}
