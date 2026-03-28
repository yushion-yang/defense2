package tower

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
)

// HitResult describes the effects of an ability when a projectile hits.
type HitResult struct {
	BonusDamage float64
	Splash      *SplashEffect
	Slow        *SlowEffect
	Stun        *StunEffect
	Bleed       *BleedEffect
	Bounce      *BounceEffect
	IsCrit      bool
}

// SplashEffect deals damage to nearby enemies.
type SplashEffect struct {
	Radius float64
	Ratio  float64 // fraction of projectile damage
}

// SlowEffect reduces enemy speed.
type SlowEffect struct {
	Factor   float64 // speed multiplier (0.5 = 50% speed)
	Duration float64
}

// StunEffect freezes enemy movement.
type StunEffect struct {
	Duration float64
}

// BleedEffect deals damage over time.
type BleedEffect struct {
	DPS      float64 // damage per second
	Duration float64
}

// BounceEffect causes projectile to chain to nearby enemies.
type BounceEffect struct {
	MaxBounces  int
	Range       float64
	DamageDecay float64
}

// Ability is the interface for tower abilities.
type Ability interface {
	// Name returns the unique ability identifier.
	Name() string
	// OnHit is called when a projectile from this tower hits an enemy.
	OnHit(t *Tower, p *projectile.Projectile, e *enemy.Enemy) *HitResult
}

// Registry holds all registered abilities.
var Registry = map[string]Ability{}

// Register adds an ability to the global registry.
func Register(a Ability) {
	Registry[a.Name()] = a
}

// TowerAbility links an ability name to tower-specific parameters.
type TowerAbility struct {
	Name   string             `json:"name"`
	Params map[string]float64 `json:"params,omitempty"`
}
