package enemy

// Enemy represents a single enemy entity.
type Enemy struct {
	X, Y       float64
	HP         float64
	MaxHP      float64
	Speed      float64
	BaseSpeed  float64
	Radius     float64
	PathIndex  int
	ReachedEnd bool
	Active     bool
	StunTimer  float64
	SlowTimer  float64
	SlowFactor float64 // speed multiplier when slowed (0.5 = half speed)
	BleedTimer float64
	BleedDPS   float64
}

// TickStatusEffects updates enemy status effects.
func TickStatusEffects(e *Enemy, dt float64) {
	// Slow
	if e.SlowTimer > 0 {
		e.SlowTimer -= dt
		e.Speed = e.BaseSpeed * e.SlowFactor
		if e.SlowTimer <= 0 {
			e.Speed = e.BaseSpeed
		}
	}

	// Bleed
	if e.BleedTimer > 0 {
		e.BleedTimer -= dt
		e.HP -= e.BleedDPS * dt
	}

	// Stun
	// (handled in movement.go)
}
