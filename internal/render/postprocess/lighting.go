// lighting.go — simplified point-light system state manager.
// Supports up to 4 point lights per frame, rendered via a Kage shader pass.
package postprocess

import "image/color"

// MaxLights is the maximum number of simultaneous point lights (shader limit).
const MaxLights = 4

// PointLight describes a single point light source.
type PointLight struct {
	X, Y      float64    // position in logical coordinates
	Color     color.RGBA // light color
	Radius    float64    // falloff radius in logical pixels
	Intensity float64    // brightness multiplier (0-2 typical)
}

// LightingState manages per-frame point light collection.
type LightingState struct {
	Enabled bool               // master toggle
	Ambient float64            // base ambient light level (0-1, default 0.85)
	Lights  [MaxLights]PointLight
	Count   int // number of active lights this frame (0-MaxLights)
}

// NewLightingState creates a LightingState with sensible defaults.
func NewLightingState() *LightingState {
	return &LightingState{
		Enabled: true,
		Ambient: 0.85,
	}
}

// Clear resets the light count to 0 for the next frame.
// Does not modify Enabled or Ambient.
func (ls *LightingState) Clear() {
	ls.Count = 0
}

// AddLight adds a point light for this frame. Ignored if already at MaxLights.
func (ls *LightingState) AddLight(light PointLight) {
	if ls.Count >= MaxLights {
		return
	}
	ls.Lights[ls.Count] = light
	ls.Count++
}
