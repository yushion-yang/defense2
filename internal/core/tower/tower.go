package tower

// Tower represents a placed tower entity.
type Tower struct {
	X, Y        float64 // pixel center position
	Row, Col    int     // grid cell
	Range       float64
	Damage      float64
	AttackSpeed float64 // attacks per second
	FireTimer   float64 // countdown to next shot
	Cost        int
	Faction     string
	Key         string
	Active      bool
}
