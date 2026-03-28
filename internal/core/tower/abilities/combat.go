// combat.go — 战斗类能力实现。
// 包含溅射（splash）和暴击（crit）两种能力，通过 init() 自注册到全局注册表。
package abilities

import (
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func init() {
	tower.Register(&SplashAbility{})
	tower.Register(&CritAbility{})
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
