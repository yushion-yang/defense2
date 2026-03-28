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

// OnHitSlowAbility slows enemies on hit.
type OnHitSlowAbility struct{}

func (a *OnHitSlowAbility) Name() string { return "onHitSlow" }
func (a *OnHitSlowAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Slow: &tower.SlowEffect{Factor: 0.6, Duration: 1.0},
	}
}

// BleedDotAbility applies damage over time.
type BleedDotAbility struct{}

func (a *BleedDotAbility) Name() string { return "bleedDot" }
func (a *BleedDotAbility) OnHit(_ *tower.Tower, _ *projectile.Projectile, _ *enemy.Enemy) *tower.HitResult {
	return &tower.HitResult{
		Bleed: &tower.BleedEffect{DPS: 5, Duration: 3},
	}
}
