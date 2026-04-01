package core_test

import (
	"testing"

	"defense2/internal/render/postprocess"
)

func TestNewEffectsDefaults(t *testing.T) {
	fx := postprocess.NewEffects()

	if fx.VignetteStrength != 0.3 {
		t.Errorf("VignetteStrength = %v, want 0.3", fx.VignetteStrength)
	}
	if fx.HitTintR != 1.0 {
		t.Errorf("HitTintR = %v, want 1.0", fx.HitTintR)
	}
	if fx.HitTintG != 0.1 {
		t.Errorf("HitTintG = %v, want 0.1", fx.HitTintG)
	}
	if fx.HitTintB != 0.1 {
		t.Errorf("HitTintB = %v, want 0.1", fx.HitTintB)
	}
	if fx.HitFlash.Active {
		t.Error("HitFlash should not be active by default")
	}
	if fx.RadialBlur.Active {
		t.Error("RadialBlur should not be active by default")
	}
	if fx.HitStopFrames != 0 {
		t.Errorf("HitStopFrames = %v, want 0", fx.HitStopFrames)
	}
}

func TestHitFlashLifecycle(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.TriggerHitFlash(0.15)

	if !fx.HitFlash.Active {
		t.Fatal("HitFlash should be active after trigger")
	}
	if fx.HitFlash.Duration != 0.15 {
		t.Errorf("HitFlash.Duration = %v, want 0.15", fx.HitFlash.Duration)
	}

	// Update half the duration — should still be active.
	freeze := fx.Update(0.07)
	if freeze {
		t.Error("Update should not freeze without hit-stop")
	}
	if !fx.HitFlash.Active {
		t.Fatal("HitFlash should still be active after half duration")
	}
	if fx.HitFlash.Timer <= 0 {
		t.Errorf("HitFlash.Timer should be positive, got %v", fx.HitFlash.Timer)
	}

	// Update past the remaining duration — should expire.
	fx.Update(0.10)
	if fx.HitFlash.Active {
		t.Error("HitFlash should be expired after full duration")
	}
}

func TestHitStop(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.TriggerHitStop(3)

	if fx.HitStopFrames != 3 {
		t.Fatalf("HitStopFrames = %v, want 3", fx.HitStopFrames)
	}

	// 3 frames should all return true (freeze).
	for i := 0; i < 3; i++ {
		if !fx.Update(1.0 / 60.0) {
			t.Errorf("frame %d: Update should return true (freeze)", i)
		}
	}

	// 4th frame should return false (no freeze).
	if fx.Update(1.0 / 60.0) {
		t.Error("frame 3: Update should return false (no freeze)")
	}
}

func TestHitStopTakesMax(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.TriggerHitStop(2)
	fx.TriggerHitStop(5)

	if fx.HitStopFrames != 5 {
		t.Errorf("HitStopFrames = %v, want 5 (max)", fx.HitStopFrames)
	}

	// Lower value should not override.
	fx.TriggerHitStop(1)
	if fx.HitStopFrames != 5 {
		t.Errorf("HitStopFrames = %v, want 5 (unchanged)", fx.HitStopFrames)
	}
}

func TestRadialBlurLifecycle(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.TriggerRadialBlur(600, 270, 0.03, 0.5)

	if !fx.RadialBlur.Active {
		t.Fatal("RadialBlur should be active after trigger")
	}
	if fx.BlurCenterX != 600 || fx.BlurCenterY != 270 {
		t.Errorf("BlurCenter = (%v, %v), want (600, 270)", fx.BlurCenterX, fx.BlurCenterY)
	}
	if fx.BlurStrength != 0.03 {
		t.Errorf("BlurStrength = %v, want 0.03", fx.BlurStrength)
	}

	// Advance partially.
	fx.Update(0.2)
	if !fx.RadialBlur.Active {
		t.Fatal("RadialBlur should still be active")
	}

	// Advance past duration.
	fx.Update(0.4)
	if fx.RadialBlur.Active {
		t.Error("RadialBlur should be expired after full duration")
	}
}

func TestMultipleEffectsSimultaneous(t *testing.T) {
	fx := postprocess.NewEffects()

	fx.TriggerHitFlash(0.2)
	fx.TriggerRadialBlur(100, 200, 0.05, 0.3)
	fx.TriggerHitStop(2)

	// All should be active/set.
	if !fx.HitFlash.Active {
		t.Error("HitFlash should be active")
	}
	if !fx.RadialBlur.Active {
		t.Error("RadialBlur should be active")
	}
	if fx.HitStopFrames != 2 {
		t.Errorf("HitStopFrames = %v, want 2", fx.HitStopFrames)
	}

	// During hit-stop, timed effects should NOT decay (Update returns true early).
	freeze := fx.Update(1.0 / 60.0)
	if !freeze {
		t.Error("should freeze during hit-stop")
	}
	// HitFlash timer should be unchanged since hit-stop returns before decay.
	if fx.HitFlash.Timer != 0.2 {
		t.Errorf("HitFlash.Timer = %v, want 0.2 (unchanged during hit-stop)", fx.HitFlash.Timer)
	}

	// Consume remaining hit-stop frame.
	fx.Update(1.0 / 60.0)

	// Now normal update: both effects should start decaying.
	freeze = fx.Update(0.1)
	if freeze {
		t.Error("should not freeze after hit-stop consumed")
	}
	if !fx.HitFlash.Active {
		t.Error("HitFlash should still be active (0.1 < 0.2)")
	}
	if !fx.RadialBlur.Active {
		t.Error("RadialBlur should still be active (0.1 < 0.3)")
	}

	// Finish both effects.
	fx.Update(0.3)
	if fx.HitFlash.Active {
		t.Error("HitFlash should have expired")
	}
	if fx.RadialBlur.Active {
		t.Error("RadialBlur should have expired")
	}
}

func TestScreenEffectProgress(t *testing.T) {
	fx := postprocess.NewEffects()
	fx.TriggerHitFlash(1.0)

	// At start: progress should be 0 (just triggered).
	p := fx.HitFlash.Progress()
	if p < 0 || p > 0.01 {
		t.Errorf("Progress at start = %v, want ~0", p)
	}

	// After half: progress ~0.5.
	fx.Update(0.5)
	p = fx.HitFlash.Progress()
	if p < 0.45 || p > 0.55 {
		t.Errorf("Progress at half = %v, want ~0.5", p)
	}
}
