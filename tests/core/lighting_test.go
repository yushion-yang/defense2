package core_test

import (
	"image/color"
	"testing"

	"defense2/internal/render/postprocess"
)

func TestLightingStateBasic(t *testing.T) {
	ls := postprocess.NewLightingState()
	if !ls.Enabled {
		t.Error("lighting should be enabled by default")
	}
	if ls.Ambient != 0.85 {
		t.Errorf("ambient = %v, want 0.85", ls.Ambient)
	}
	if ls.Count != 0 {
		t.Error("should start with 0 lights")
	}
}

func TestLightingAddAndClear(t *testing.T) {
	ls := postprocess.NewLightingState()
	ls.AddLight(postprocess.PointLight{
		X: 100, Y: 200,
		Color:     color.RGBA{R: 255, G: 100, B: 50, A: 255},
		Radius:    80,
		Intensity: 1.0,
	})
	if ls.Count != 1 {
		t.Errorf("count = %d, want 1", ls.Count)
	}
	if ls.Lights[0].X != 100 || ls.Lights[0].Y != 200 {
		t.Error("light position mismatch")
	}

	ls.Clear()
	if ls.Count != 0 {
		t.Errorf("count after clear = %d, want 0", ls.Count)
	}
}

func TestLightingMaxCap(t *testing.T) {
	ls := postprocess.NewLightingState()
	for i := 0; i < postprocess.MaxLights+5; i++ {
		ls.AddLight(postprocess.PointLight{X: float64(i), Radius: 50, Intensity: 1})
	}
	if ls.Count != postprocess.MaxLights {
		t.Errorf("count = %d, want %d (capped)", ls.Count, postprocess.MaxLights)
	}
}
