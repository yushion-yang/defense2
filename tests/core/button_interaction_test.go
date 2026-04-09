package core_test

import (
	"math"
	"testing"

	"defense2/internal/render/ui"
)

func TestButtonStateDefault(t *testing.T) {
	bs := ui.ButtonState{}
	if bs.Scale() != 1.0 {
		t.Errorf("default scale = %v, want 1.0", bs.Scale())
	}
	if bs.ShakeOffsetX() != 0 {
		t.Errorf("default shakeX = %v, want 0", bs.ShakeOffsetX())
	}
}

func TestButtonStatePress(t *testing.T) {
	bs := ui.ButtonState{}
	bs.Update(0.016, true, true) // hovered + pressed
	if !bs.Pressed {
		t.Error("should be pressed")
	}
	if bs.ScaleT <= 0 {
		t.Error("ScaleT should be > 0 after press")
	}
	s := bs.Scale()
	if s >= 1.0 {
		t.Errorf("pressed scale = %v, should be < 1.0", s)
	}
}

func TestButtonStateReleaseBounce(t *testing.T) {
	bs := ui.ButtonState{}
	bs.Update(0.016, true, true)  // press
	bs.Update(0.016, true, false) // release
	s := bs.Scale()
	if s <= 1.0 {
		t.Errorf("release bounce scale = %v, should be > 1.0", s)
	}
}

func TestButtonStateDecay(t *testing.T) {
	bs := ui.ButtonState{}
	bs.Update(0.016, true, true) // press
	initial := bs.ScaleT
	for i := 0; i < 20; i++ {
		bs.Update(0.016, false, false)
	}
	if bs.ScaleT >= initial {
		t.Errorf("ScaleT should decay, got %v (was %v)", bs.ScaleT, initial)
	}
	if bs.ScaleT != 0 {
		t.Errorf("ScaleT should reach 0 after enough frames, got %v", bs.ScaleT)
	}
}

func TestButtonStateDisabledShake(t *testing.T) {
	bs := ui.ButtonState{}
	bs.TriggerDisabledShake()
	if bs.ShakeT <= 0 {
		t.Error("ShakeT should be > 0 after trigger")
	}
	// Advance one frame so ShakeT moves off the sin zero crossing at 1.0
	bs.Update(0.016, false, false)
	offset := bs.ShakeOffsetX()
	if math.Abs(offset) < 0.01 {
		t.Error("should have non-zero shake offset after one frame")
	}
	// Decay fully
	for i := 0; i < 30; i++ {
		bs.Update(0.016, false, false)
	}
	if bs.ShakeT != 0 {
		t.Errorf("ShakeT should reach 0 after decay, got %v", bs.ShakeT)
	}
}
