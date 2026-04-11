package timescale

import "defense2/internal/render/easing"

// Phase of the slow-mo sequence
type phase int

const (
	phaseIdle    phase = iota
	phaseEaseIn        // transitioning from 1.0 to target
	phaseHold          // holding at target scale
	phaseEaseOut       // transitioning from target back to 1.0
)

// Controller manages a slow-motion time-scale effect.
// Update should be called with wall-clock dt (not gameDT).
type Controller struct {
	scale      float64 // current scale value
	target     float64
	easeInDur  float64
	holdDur    float64
	easeOutDur float64
	elapsed    float64
	phase      phase
}

// New creates a controller at normal speed.
func New() *Controller {
	return &Controller{scale: 1.0, phase: phaseIdle}
}

// Trigger starts a slow-mo sequence.
// scale: target time scale (0.1-1.0, lower = slower).
// easeIn/hold/easeOut: durations in seconds.
// A stronger (lower scale) trigger overrides a weaker one.
func (c *Controller) Trigger(scale, easeIn, hold, easeOut float64) {
	if c.phase != phaseIdle && scale >= c.target {
		return // current effect is stronger or equal
	}
	c.target = scale
	c.easeInDur = easeIn
	c.holdDur = hold
	c.easeOutDur = easeOut
	c.elapsed = 0
	c.phase = phaseEaseIn
	c.scale = 1.0
}

// Tick advances the controller and returns the current time scale (1.0 = normal).
// dt should be wall-clock delta time, NOT gameDT.
func (c *Controller) Tick(dt float64) float64 {
	if c.phase == phaseIdle {
		return 1.0
	}

	c.elapsed += dt

	// Loop allows zero-duration phases to cascade in a single Tick call.
	for c.phase != phaseIdle {
		switch c.phase {
		case phaseEaseIn:
			if c.easeInDur <= 0 || c.elapsed >= c.easeInDur {
				c.scale = c.target
				c.elapsed -= c.easeInDur
				c.phase = phaseHold
				continue
			}
			t := c.elapsed / c.easeInDur
			c.scale = 1.0 + (c.target-1.0)*easing.EaseOutQuad(t)
			return c.scale
		case phaseHold:
			c.scale = c.target
			if c.elapsed >= c.holdDur {
				c.elapsed -= c.holdDur
				c.phase = phaseEaseOut
				continue
			}
			return c.scale
		case phaseEaseOut:
			if c.easeOutDur <= 0 || c.elapsed >= c.easeOutDur {
				c.scale = 1.0
				c.phase = phaseIdle
				return c.scale
			}
			t := c.elapsed / c.easeOutDur
			c.scale = c.target + (1.0-c.target)*easing.EaseOutQuad(t)
			return c.scale
		}
	}

	return c.scale
}

// Scale returns the current time scale without advancing.
func (c *Controller) Scale() float64 {
	if c.phase == phaseIdle {
		return 1.0
	}
	return c.scale
}

// Active returns true if a slow-mo sequence is in progress.
func (c *Controller) Active() bool {
	return c.phase != phaseIdle
}
