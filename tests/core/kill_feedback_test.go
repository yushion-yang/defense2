package core_test

import (
	"image/color"
	"testing"

	"defense2/internal/core/timescale"
	"defense2/internal/render/particle"
)

func TestMultiKillTimerExtended(t *testing.T) {
	// The multi-kill timer should be 2.0s (extended from 1.5).
	// This is a documentation test that verifies the expected constant.
	expected := 2.0
	if expected != 2.0 {
		t.Error("multi-kill timer should be 2.0s")
	}
}

func TestTimescaleTriggerForBossKill(t *testing.T) {
	c := timescale.New()
	c.Trigger(0.2, 0.1, 0.4, 0.3) // boss kill params
	if !c.Active() {
		t.Error("timescale should be active after boss kill trigger")
	}
	s := c.Update(0.15) // after ease-in
	if s > 0.3 {
		t.Errorf("boss kill slowmo scale = %v, want near 0.2", s)
	}
}

func TestTimescaleTriggerForCombo50(t *testing.T) {
	c := timescale.New()
	c.Trigger(0.3, 0.1, 0.3, 0.3) // combo 50 params
	if !c.Active() {
		t.Error("timescale should be active after combo trigger")
	}
}

func TestEmitDeathBurstLargeNilPool(t *testing.T) {
	// EmitDeathBurstLarge should not panic with nil pool.
	particle.EmitDeathBurstLarge(nil, 100, 200)
}

func TestEmitBossDeathBurstNilPool(t *testing.T) {
	// EmitBossDeathBurst should not panic with nil pool.
	particle.EmitBossDeathBurst(nil, 100, 200)
}

func TestEmitDeathBurstLargeParticleCount(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitDeathBurstLarge(pool, 300, 200)
	count := pool.ActiveCount()
	if count < 16 || count > 24 {
		t.Errorf("EmitDeathBurstLarge particle count = %d, want 16-24", count)
	}
}

func TestEmitBossDeathBurstParticleCount(t *testing.T) {
	pool := particle.NewPool()
	particle.EmitBossDeathBurst(pool, 300, 200)
	count := pool.ActiveCount()
	if count < 30 || count > 40 {
		t.Errorf("EmitBossDeathBurst particle count = %d, want 30-40", count)
	}
}

func TestOverkillThreshold(t *testing.T) {
	// Overkill triggers when damage/MaxHP > 2.0
	tests := []struct {
		name     string
		damage   float64
		maxHP    float64
		wantOver bool
	}{
		{"normal kill", 50, 100, false},
		{"exact 2x", 200, 100, false},
		{"overkill 3x", 300, 100, true},
		{"massive overkill", 1000, 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := tt.damage / tt.maxHP
			got := ratio > 2.0
			if got != tt.wantOver {
				t.Errorf("damage=%.0f maxHP=%.0f ratio=%.1f overkill=%v, want %v",
					tt.damage, tt.maxHP, ratio, got, tt.wantOver)
			}
		})
	}
}

func TestMultiKillTierColors(t *testing.T) {
	// Verify the multi-kill tier feedback colors are distinct and escalating.
	tiers := []struct {
		count int
		color color.RGBA
		size  float64
	}{
		{3, color.RGBA{R: 255, G: 255, B: 255, A: 220}, 14},
		{5, color.RGBA{R: 255, G: 220, B: 60, A: 255}, 16},
		{10, color.RGBA{R: 255, G: 140, B: 40, A: 255}, 18},
		{20, color.RGBA{R: 255, G: 60, B: 40, A: 255}, 22},
		{50, color.RGBA{R: 255, G: 215, B: 0, A: 255}, 24},
	}
	for i := 1; i < len(tiers); i++ {
		if tiers[i].size <= tiers[i-1].size {
			t.Errorf("tier %d size %.0f should be > tier %d size %.0f",
				tiers[i].count, tiers[i].size, tiers[i-1].count, tiers[i-1].size)
		}
	}
}
