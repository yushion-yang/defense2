package core_test

import (
	"testing"

	"defense2/internal/core/game"
)

func TestQualityPresets(t *testing.T) {
	for i, s := range game.Presets {
		if s.MaxParticles <= 0 {
			t.Errorf("level %d: MaxParticles should be positive, got %d", i, s.MaxParticles)
		}
		if s.TargetTPS != 30 && s.TargetTPS != 60 {
			t.Errorf("level %d: unexpected TPS %d", i, s.TargetTPS)
		}
		if s.MaxLights < 0 {
			t.Errorf("level %d: MaxLights should be non-negative, got %d", i, s.MaxLights)
		}
		if s.TrailLen <= 0 {
			t.Errorf("level %d: TrailLen should be positive, got %d", i, s.TrailLen)
		}
	}
}

func TestQualityPresetsOrder(t *testing.T) {
	// High should have the most particles, Low the fewest.
	if game.Presets[game.QualityHigh].MaxParticles <= game.Presets[game.QualityLow].MaxParticles {
		t.Error("High preset should have more MaxParticles than Low")
	}
	if game.Presets[game.QualityHigh].TrailLen <= game.Presets[game.QualityLow].TrailLen {
		t.Error("High preset should have longer TrailLen than Low")
	}
}

func TestQualityString(t *testing.T) {
	tests := []struct {
		q    game.QualityLevel
		want string
	}{
		{game.QualityHigh, "High"},
		{game.QualityMedium, "Medium"},
		{game.QualityLow, "Low"},
		{game.QualityLevel(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.q.String(); got != tt.want {
			t.Errorf("QualityLevel(%d).String() = %q, want %q", int(tt.q), got, tt.want)
		}
	}
}

func TestSettingsReturnsCurrentPreset(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityMedium
	s := game.Settings()
	want := game.Presets[game.QualityMedium]
	if s != want {
		t.Errorf("Settings() = %+v, want %+v", s, want)
	}
}

func TestAdaptiveDowngrade(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityHigh
	qa := game.NewQualityAdaptive()

	// Feed 35 consecutive slow frames (> 14ms).
	for i := 0; i < 35; i++ {
		qa.Tick(15.0)
	}
	if game.CurrentQuality == game.QualityHigh {
		t.Error("should have downgraded from High after 35 slow frames")
	}
}

func TestAdaptiveDowngradeStopsAtLow(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityLow
	qa := game.NewQualityAdaptive()

	// Feed many slow frames — should not go below Low.
	for i := 0; i < 100; i++ {
		qa.Tick(20.0)
	}
	if game.CurrentQuality != game.QualityLow {
		t.Errorf("should stay at Low, got %v", game.CurrentQuality)
	}
}

func TestAdaptiveUpgrade(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityLow
	qa := game.NewQualityAdaptive()

	// Feed 125 consecutive fast frames (< 10ms).
	for i := 0; i < 125; i++ {
		qa.Tick(5.0)
	}
	if game.CurrentQuality == game.QualityLow {
		t.Error("should have upgraded from Low after 125 fast frames")
	}
}

func TestAdaptiveUpgradeStopsAtHigh(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityHigh
	qa := game.NewQualityAdaptive()

	// Feed many fast frames — should not go above High.
	for i := 0; i < 300; i++ {
		qa.Tick(3.0)
	}
	if game.CurrentQuality != game.QualityHigh {
		t.Errorf("should stay at High, got %v", game.CurrentQuality)
	}
}

func TestAdaptiveNoChangeInMiddle(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityMedium
	qa := game.NewQualityAdaptive()

	// Feed 50 frames in the "middle" band (10-14ms) — no change.
	for i := 0; i < 50; i++ {
		qa.Tick(12.0)
	}
	if game.CurrentQuality != game.QualityMedium {
		t.Errorf("should not change in middle range, got %v", game.CurrentQuality)
	}
}

func TestAdaptiveCounterResetOnMix(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityHigh
	qa := game.NewQualityAdaptive()

	// 25 slow frames (not enough to trigger).
	for i := 0; i < 25; i++ {
		qa.Tick(16.0)
	}
	// One middle-band frame resets the counter.
	qa.Tick(12.0)
	if qa.SlowFrames() != 0 {
		t.Error("slow counter should reset on middle-band frame")
	}

	// 25 more slow frames — still not enough total because counter was reset.
	for i := 0; i < 25; i++ {
		qa.Tick(16.0)
	}
	if game.CurrentQuality != game.QualityHigh {
		t.Error("should still be High — counter was reset mid-run")
	}
}

func TestAdaptiveMultiStepDowngrade(t *testing.T) {
	orig := game.CurrentQuality
	defer func() { game.CurrentQuality = orig }()

	game.CurrentQuality = game.QualityHigh
	qa := game.NewQualityAdaptive()

	// Feed enough slow frames to downgrade twice (High -> Medium -> Low).
	for i := 0; i < 70; i++ {
		qa.Tick(18.0)
	}
	if game.CurrentQuality != game.QualityLow {
		t.Errorf("expected Low after sustained slow frames, got %v", game.CurrentQuality)
	}
}
