package core_test

import (
	"testing"

	"defense2/internal/render/postprocess"
)

// ============================================================
// Pipeline creation and defaults
// ============================================================

func TestPipelineCreation(t *testing.T) {
	p := postprocess.NewPipeline()

	if !p.BloomEnabled {
		t.Error("bloom should be enabled by default")
	}
	if p.BloomThreshold != 0.65 {
		t.Errorf("default threshold: got %v, want 0.65", p.BloomThreshold)
	}
	if p.BloomIntensity != 0.6 {
		t.Errorf("default intensity: got %v, want 0.6", p.BloomIntensity)
	}
	if p.BloomPasses != 2 {
		t.Errorf("default passes: got %v, want 2", p.BloomPasses)
	}
}

// ============================================================
// Bloom presets
// ============================================================

func TestBloomPresets(t *testing.T) {
	tests := []struct {
		name      string
		preset    postprocess.BloomPreset
		threshold float64
		intensity float64
		passes    int
	}{
		{"Default", postprocess.BloomDefault, 0.65, 0.6, 2},
		{"Boss", postprocess.BloomBoss, 0.5, 1.0, 3},
		{"Skill", postprocess.BloomSkill, 0.55, 0.8, 2},
		{"Subtle", postprocess.BloomSubtle, 0.75, 0.3, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify preset values.
			if tc.preset.Threshold != tc.threshold {
				t.Errorf("threshold: got %v, want %v", tc.preset.Threshold, tc.threshold)
			}
			if tc.preset.Intensity != tc.intensity {
				t.Errorf("intensity: got %v, want %v", tc.preset.Intensity, tc.intensity)
			}
			if tc.preset.Passes != tc.passes {
				t.Errorf("passes: got %v, want %v", tc.preset.Passes, tc.passes)
			}

			// Verify ApplyPreset updates pipeline.
			p := postprocess.NewPipeline()
			p.ApplyPreset(tc.preset)

			if p.BloomThreshold != tc.threshold {
				t.Errorf("applied threshold: got %v, want %v", p.BloomThreshold, tc.threshold)
			}
			if p.BloomIntensity != tc.intensity {
				t.Errorf("applied intensity: got %v, want %v", p.BloomIntensity, tc.intensity)
			}
			if p.BloomPasses != tc.passes {
				t.Errorf("applied passes: got %v, want %v", p.BloomPasses, tc.passes)
			}
		})
	}
}

// ============================================================
// Scene buffer resize
// ============================================================

func TestSceneBufferResize(t *testing.T) {
	p := postprocess.NewPipeline()

	// First call creates the buffer.
	buf1 := p.SceneBuffer(800, 600)
	if buf1 == nil {
		t.Fatal("SceneBuffer returned nil")
	}
	if buf1.Bounds().Dx() != 800 || buf1.Bounds().Dy() != 600 {
		t.Errorf("buffer size: got %dx%d, want 800x600",
			buf1.Bounds().Dx(), buf1.Bounds().Dy())
	}

	// Same dimensions: returns same buffer (reuse).
	buf2 := p.SceneBuffer(800, 600)
	if buf2 != buf1 {
		t.Error("same dimensions should reuse buffer")
	}

	// Different dimensions: creates new buffer.
	buf3 := p.SceneBuffer(1600, 900)
	if buf3 == nil {
		t.Fatal("SceneBuffer returned nil on resize")
	}
	if buf3.Bounds().Dx() != 1600 || buf3.Bounds().Dy() != 900 {
		t.Errorf("resized buffer: got %dx%d, want 1600x900",
			buf3.Bounds().Dx(), buf3.Bounds().Dy())
	}
}

func TestSceneBufferClampsMinimum(t *testing.T) {
	p := postprocess.NewPipeline()

	// Zero or negative dimensions should be clamped to 1.
	buf := p.SceneBuffer(0, 0)
	if buf == nil {
		t.Fatal("SceneBuffer returned nil for zero dimensions")
	}
	if buf.Bounds().Dx() < 1 || buf.Bounds().Dy() < 1 {
		t.Errorf("buffer should be at least 1x1, got %dx%d",
			buf.Bounds().Dx(), buf.Bounds().Dy())
	}
}

func TestShadersReadyDefault(t *testing.T) {
	// Without calling InitShaders, ShadersReady should be false.
	// Note: we cannot call InitShaders in unit tests because it needs GPU context.
	if postprocess.ShadersReady() {
		t.Error("ShadersReady should be false before InitShaders")
	}
}

func TestApplyWithoutShaders(t *testing.T) {
	// When shaders are not compiled, Apply should blit scene directly (no panic).
	p := postprocess.NewPipeline()
	buf := p.SceneBuffer(100, 100)
	if buf == nil {
		t.Fatal("SceneBuffer returned nil")
	}

	// Apply requires GPU context (ebiten.Image.DrawImage panics without it).
	// We verify the pipeline is correctly configured for fallback:
	// shaders not ready + bloom enabled = will use direct blit path.
	if postprocess.ShadersReady() {
		t.Error("shaders should not be ready in test env")
	}
	if !p.BloomEnabled {
		t.Error("bloom should be enabled by default")
	}
	// Actual rendering tested via `make run`.
}
