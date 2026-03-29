// effects.go — screen-level effect state manager.
// Manages vignette, hit flash, radial blur, and hit-stop.
package postprocess

// ScreenEffect tracks a timed screen-level effect.
type ScreenEffect struct {
	Active   bool
	Timer    float64
	Duration float64
}

// Progress returns a 0..1 value representing how far through the effect we are
// (0 = just started, 1 = about to expire).
func (e *ScreenEffect) Progress() float64 {
	if e.Duration <= 0 {
		return 1
	}
	return 1 - e.Timer/e.Duration
}

// Effects holds the state for all screen-level post-processing effects.
type Effects struct {
	// Vignette: always-on edge darkening.
	VignetteStrength float64 // default 0.3

	// HitFlash: red screen tint when player takes damage.
	HitFlash ScreenEffect
	HitTintR float64 // default 1.0
	HitTintG float64 // default 0.1
	HitTintB float64 // default 0.1

	// RadialBlur: zoom blur from a center point.
	RadialBlur   ScreenEffect
	BlurCenterX  float64
	BlurCenterY  float64
	BlurStrength float64

	// HitStop: freeze frames (game logic pauses, rendering continues).
	HitStopFrames int
}

// NewEffects creates an Effects instance with sensible defaults.
func NewEffects() *Effects {
	return &Effects{
		VignetteStrength: 0.3,
		HitTintR:         1.0,
		HitTintG:         0.1,
		HitTintB:         0.1,
	}
}

// TriggerHitFlash starts a screen-flash effect that decays over duration seconds.
func (fx *Effects) TriggerHitFlash(duration float64) {
	fx.HitFlash = ScreenEffect{
		Active:   true,
		Timer:    duration,
		Duration: duration,
	}
}

// TriggerRadialBlur starts a radial blur effect from (cx, cy) with the given
// strength that decays over duration seconds.
func (fx *Effects) TriggerRadialBlur(cx, cy, strength, duration float64) {
	fx.RadialBlur = ScreenEffect{
		Active:   true,
		Timer:    duration,
		Duration: duration,
	}
	fx.BlurCenterX = cx
	fx.BlurCenterY = cy
	fx.BlurStrength = strength
}

// TriggerHitStop freezes game logic for the given number of frames.
func (fx *Effects) TriggerHitStop(frames int) {
	if frames > fx.HitStopFrames {
		fx.HitStopFrames = frames
	}
}

// Update advances all effect timers by dt seconds.
// Returns true if the game should freeze this frame (hit-stop active).
func (fx *Effects) Update(dt float64) bool {
	// Hit-stop: consume one frame.
	if fx.HitStopFrames > 0 {
		fx.HitStopFrames--
		return true
	}

	// Decay timed effects.
	if fx.HitFlash.Active {
		fx.HitFlash.Timer -= dt
		if fx.HitFlash.Timer <= 0 {
			fx.HitFlash.Active = false
			fx.HitFlash.Timer = 0
		}
	}
	if fx.RadialBlur.Active {
		fx.RadialBlur.Timer -= dt
		if fx.RadialBlur.Timer <= 0 {
			fx.RadialBlur.Active = false
			fx.RadialBlur.Timer = 0
		}
	}

	return false
}
