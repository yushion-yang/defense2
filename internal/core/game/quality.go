// quality.go — quality level system with adaptive frame-rate downgrade.
// Defines High/Medium/Low presets controlling post-processing, particles,
// lights, trail length and target TPS. An adaptive ticker monitors frame
// times and automatically shifts between quality levels.
package game

// QualityLevel represents a visual quality tier.
type QualityLevel int

const (
	QualityHigh   QualityLevel = iota // desktop default
	QualityMedium                     // mid-range mobile
	QualityLow                        // low-end mobile
)

// QualitySettings holds per-level visual fidelity caps.
type QualitySettings struct {
	PostProcessing bool // enable vignette/lighting shader passes
	MaxParticles   int  // particle pool active cap
	MaxLights      int  // dynamic light count cap
	TrailLen       int  // projectile trail frames
	TargetTPS      int  // target ticks per second (60 or 30)
}

// Presets defines the three quality levels.
var Presets = [3]QualitySettings{
	{true, 2048, 4, 6, 60},  // High
	{true, 1024, 2, 3, 60},  // Medium
	{false, 512, 0, 1, 30},  // Low
}

// CurrentQuality is the active quality level (global, mutable by adaptive ticker).
var CurrentQuality QualityLevel = QualityHigh

// Settings returns the QualitySettings for the current quality level.
func Settings() QualitySettings {
	return Presets[CurrentQuality]
}

// String returns a human-readable name for the quality level.
func (q QualityLevel) String() string {
	switch q {
	case QualityHigh:
		return "High"
	case QualityMedium:
		return "Medium"
	case QualityLow:
		return "Low"
	default:
		return "Unknown"
	}
}

// --- Adaptive quality ticker ---

const (
	slowThresholdMs = 14.0 // frames slower than this are "slow"
	fastThresholdMs = 10.0 // frames faster than this are "fast"
	downgradeAfter  = 30   // consecutive slow frames before downgrade (~0.5s at 60fps)
	upgradeAfter    = 120  // consecutive fast frames before upgrade (~2s at 60fps)
)

// QualityAdaptive monitors frame performance and adjusts CurrentQuality.
type QualityAdaptive struct {
	slowFrames int // consecutive frames where total > slowThresholdMs
	fastFrames int // consecutive frames where total < fastThresholdMs
}

// NewQualityAdaptive creates a new adaptive quality ticker.
func NewQualityAdaptive() *QualityAdaptive {
	return &QualityAdaptive{}
}

// Tick checks frame performance and adjusts quality level.
// frameMs is the total Update+Draw time in milliseconds.
func (q *QualityAdaptive) Tick(frameMs float64) {
	if frameMs > slowThresholdMs {
		q.slowFrames++
		q.fastFrames = 0
	} else if frameMs < fastThresholdMs {
		q.fastFrames++
		q.slowFrames = 0
	} else {
		// In the middle band — reset both counters.
		q.slowFrames = 0
		q.fastFrames = 0
	}

	// Downgrade after sustained slow frames.
	if q.slowFrames > downgradeAfter && CurrentQuality < QualityLow {
		CurrentQuality++
		q.slowFrames = 0
	}

	// Upgrade after sustained fast frames.
	if q.fastFrames > upgradeAfter && CurrentQuality > QualityHigh {
		CurrentQuality--
		q.fastFrames = 0
	}
}

// SlowFrames returns the current consecutive slow frame count (for testing).
func (q *QualityAdaptive) SlowFrames() int { return q.slowFrames }

// FastFrames returns the current consecutive fast frame count (for testing).
func (q *QualityAdaptive) FastFrames() int { return q.fastFrames }
