package core_test

import (
	"testing"

	"defense2/internal/render/postprocess"
)

func TestDesatDefaultZero(t *testing.T) {
	fx := postprocess.NewEffects()
	if fx.DesatStrength != 0 {
		t.Errorf("DesatStrength = %v, want 0", fx.DesatStrength)
	}
}

func TestDesatLerpsToTarget(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.SetDesaturation(0.7, 3.0, 0.5, 0.5, 0.6)

	// After some updates, strength should approach target
	for i := 0; i < 30; i++ {
		fx.Update(1.0 / 60.0)
	}
	// 30 frames at 1/60 = 0.5s, speed=3.0 -> moved 1.5 units, clamped to 0.7
	if fx.DesatStrength < 0.65 || fx.DesatStrength > 0.75 {
		t.Errorf("DesatStrength = %v, want ~0.7 after 0.5s at speed 3.0", fx.DesatStrength)
	}
}

func TestDesatLerpsBack(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.DesatStrength = 0.7
	fx.SetDesaturation(0.0, 4.0, 0.5, 0.5, 0.6)

	for i := 0; i < 20; i++ {
		fx.Update(1.0 / 60.0)
	}
	// Should be decreasing
	if fx.DesatStrength >= 0.7 {
		t.Errorf("DesatStrength = %v, should be decreasing", fx.DesatStrength)
	}
}

func TestDesatClampsToRange(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.SetDesaturation(1.5, 10.0, 0, 0, 0) // overshoot target
	fx.Update(1.0)
	if fx.DesatStrength > 1.5 {
		t.Errorf("DesatStrength = %v, should clamp to target 1.5", fx.DesatStrength)
	}
}
