// easing.go — centralized easing/interpolation utilities.
package easing

import "math"

// Clamp01 clamps v to [0, 1].
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Lerp linearly interpolates between a and b by t (unclamped).
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// EaseOutQuad decelerating ease: t*(2-t).
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInQuad accelerating ease: t*t.
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutBack overshoot ease (snappy spring feel).
func EaseOutBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

// EaseOutElastic elastic ease with bounce.
func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	const c4 = (2 * math.Pi) / 3
	return math.Pow(2, -10*t) * math.Sin((t*10-0.75)*c4) + 1
}

// EaseInOutCubic smooth start and end: cubic in-out.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}
