// bloom.go — bloom preset configurations for different game contexts.
package postprocess

// BloomPreset defines a bloom configuration for a specific visual context.
type BloomPreset struct {
	Threshold float64 // luminance cutoff (0..1)
	Intensity float64 // bloom strength multiplier
	Passes    int     // number of blur iterations (more = softer/wider)
}

// Predefined bloom presets for different game scenarios.
var (
	BloomDefault = BloomPreset{Threshold: 0.65, Intensity: 0.6, Passes: 2}
	BloomBoss    = BloomPreset{Threshold: 0.5, Intensity: 1.0, Passes: 3}
	BloomSubtle  = BloomPreset{Threshold: 0.75, Intensity: 0.3, Passes: 1}
)

// ApplyPreset sets the pipeline's bloom parameters from a preset.
func (p *Pipeline) ApplyPreset(preset BloomPreset) {
	p.BloomThreshold = preset.Threshold
	p.BloomIntensity = preset.Intensity
	p.BloomPasses = preset.Passes
}
