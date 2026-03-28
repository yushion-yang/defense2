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

// SplashAbility deals area damage around the hit target.
type SplashAbility struct{}

func (a *SplashAbility) Name() string { return "splash" }
func (a *SplashAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Splash: &tower.SplashEffect{Radius: 50, Ratio: 0.4},
	}
}

// CritAbility has a chance to deal bonus damage.
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
