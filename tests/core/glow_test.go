package core_test

import (
	"testing"

	"defense2/internal/render/draw"
)

func TestGlowPassActiveDefault(t *testing.T) {
	if draw.GlowPassActive() {
		t.Error("glow pass should not be active by default")
	}
}

func TestGlowTargetNilWhenInactive(t *testing.T) {
	if draw.GlowTarget() != nil {
		t.Error("glow target should be nil when inactive")
	}
}

func TestEndGlowPassSafeWhenInactive(t *testing.T) {
	// EndGlowPass should be a no-op when no pass is active.
	// This must not panic.
	draw.EndGlowPass(nil)
}
