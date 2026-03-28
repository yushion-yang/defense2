package projectile

// Projectile represents a flying projectile.
type Projectile struct {
	X, Y    float64
	VX, VY  float64
	Damage  float64
	Radius  float64
	Speed   float64
	Active  bool
	Life    float64 // remaining lifetime in seconds
	MaxLife float64
}
