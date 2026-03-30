package core_test

import (
	"math"
	"testing"
	"time"

	"defense2/internal/core/debug"
)

func TestPerfTrackerCreation(t *testing.T) {
	pt := debug.NewPerfTracker()
	if pt == nil {
		t.Fatal("NewPerfTracker returned nil")
	}
	if pt.FPS != 0 {
		t.Errorf("initial FPS = %f, want 0", pt.FPS)
	}
	if pt.AvgUpdateMs != 0 {
		t.Errorf("initial AvgUpdateMs = %f, want 0", pt.AvgUpdateMs)
	}
	if pt.HeapMB != 0 {
		t.Errorf("initial HeapMB = %f, want 0", pt.HeapMB)
	}
}

func TestPerfTrackerRecording(t *testing.T) {
	pt := debug.NewPerfTracker()
	// Simulate 10 frames without panic
	for i := 0; i < 10; i++ {
		pt.BeginUpdate()
		time.Sleep(100 * time.Microsecond) // tiny sleep to get non-zero times
		pt.EndUpdate()

		pt.BeginDraw()
		time.Sleep(100 * time.Microsecond)
		pt.EndDraw()
	}

	// After 10 frames (< 1 second), stats may not have been computed yet.
	// But the tracker should not panic and should have recorded frames.
}

func TestPerfTrackerStatsAfterForce(t *testing.T) {
	pt := debug.NewPerfTracker()
	// Record some frames
	for i := 0; i < 5; i++ {
		pt.BeginUpdate()
		time.Sleep(time.Millisecond)
		pt.EndUpdate()
		pt.BeginDraw()
		time.Sleep(time.Millisecond)
		pt.EndDraw()
	}
	// The stats are computed internally; we just verify no panic
	// and that the tracker is usable.
}

func TestAvgHelper(t *testing.T) {
	tests := []struct {
		name string
		vals []float64
		want float64
	}{
		{"three values", []float64{1, 2, 3}, 2.0},
		{"single value", []float64{5}, 5.0},
		{"empty", []float64{}, 0.0},
		{"negative", []float64{-2, 2}, 0.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := debug.Avg(tc.vals)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("Avg(%v) = %f, want %f", tc.vals, got, tc.want)
			}
		})
	}
}

func TestPercentileHelper(t *testing.T) {
	// Build [1..100]
	vals := make([]float64, 100)
	for i := range vals {
		vals[i] = float64(i + 1)
	}

	tests := []struct {
		name string
		pct  int
		want float64
	}{
		{"p99 of 1..100", 99, 100.0},
		{"p50 of 1..100", 50, 51.0},
		{"p0 of 1..100", 0, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := debug.Percentile(vals, tc.pct)
			if got != tc.want {
				t.Errorf("Percentile(%d) = %f, want %f", tc.pct, got, tc.want)
			}
		})
	}

	// Empty slice
	if got := debug.Percentile(nil, 99); got != 0 {
		t.Errorf("Percentile(nil, 99) = %f, want 0", got)
	}
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	vals := []float64{5, 3, 1, 4, 2}
	original := make([]float64, len(vals))
	copy(original, vals)

	debug.Percentile(vals, 50)

	for i := range vals {
		if vals[i] != original[i] {
			t.Errorf("Percentile mutated input: vals[%d] = %f, want %f", i, vals[i], original[i])
		}
	}
}
