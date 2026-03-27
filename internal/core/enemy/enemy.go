package enemy

// Enemy represents a single enemy entity.
type Enemy struct {
	X, Y       float64
	HP         float64
	MaxHP      float64
	Speed      float64
	Radius     float64
	PathIndex  int
	ReachedEnd bool
	Active     bool
	StunTimer  float64
}
