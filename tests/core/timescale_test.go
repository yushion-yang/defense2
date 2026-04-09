package core_test

import (
	"testing"

	"defense2/internal/core/timescale"
)

func TestIdleReturnsOne(t *testing.T) {
	c := timescale.New()
	if s := c.Update(0.016); s != 1.0 {
		t.Errorf("idle scale = %v, want 1.0", s)
	}
	if c.Active() {
		t.Error("should not be active when idle")
	}
}

func TestTriggerEaseInHoldEaseOut(t *testing.T) {
	c := timescale.New()
	c.Trigger(0.3, 0.1, 0.3, 0.3)

	if !c.Active() {
		t.Fatal("should be active after trigger")
	}

	// Ease in: at halfway (0.05s), scale should be between 1.0 and 0.3
	s := c.Update(0.05)
	if s >= 1.0 || s <= 0.3 {
		t.Errorf("ease-in halfway scale = %v, want between 0.3 and 1.0", s)
	}

	// Complete ease-in
	s = c.Update(0.06)
	// Should be at or near target (0.3)
	if s > 0.35 {
		t.Errorf("after ease-in scale = %v, want near 0.3", s)
	}

	// Hold phase: stay at target for 0.3s
	s = c.Update(0.15)
	if s < 0.25 || s > 0.35 {
		t.Errorf("hold scale = %v, want ~0.3", s)
	}
	s = c.Update(0.16)
	// Should start easing out now

	// Ease out: eventually return to 1.0
	s = c.Update(0.35)
	if s != 1.0 {
		t.Errorf("after ease-out scale = %v, want 1.0", s)
	}
	if c.Active() {
		t.Error("should be idle after full sequence")
	}
}

func TestStrongerOverridesWeaker(t *testing.T) {
	c := timescale.New()
	c.Trigger(0.5, 0.1, 0.2, 0.2) // weaker
	c.Update(0.05)                  // enter ease-in

	c.Trigger(0.2, 0.1, 0.3, 0.3) // stronger (lower scale)
	if s := c.Scale(); s > 0.5 {
		// After override, should reset to ease-in toward 0.2
	}
	if !c.Active() {
		t.Error("should be active after stronger trigger")
	}

	// Weaker trigger should be ignored
	c.Trigger(0.8, 0.1, 0.1, 0.1)
	// Scale target should still be 0.2
	// Run through to hold
	c.Update(0.2)
	s := c.Scale()
	if s > 0.3 {
		t.Errorf("scale = %v, should be near 0.2 (stronger trigger maintained)", s)
	}
}

func TestZeroDurations(t *testing.T) {
	c := timescale.New()
	c.Trigger(0.5, 0, 0, 0)
	s := c.Update(0.001)
	if s != 1.0 {
		t.Errorf("zero-duration trigger should resolve immediately, got %v", s)
	}
	if c.Active() {
		t.Error("should be idle after zero-duration trigger")
	}
}

func TestScaleWithoutUpdate(t *testing.T) {
	c := timescale.New()
	if s := c.Scale(); s != 1.0 {
		t.Errorf("idle Scale() = %v, want 1.0", s)
	}
}
