// buff_fuzz_test.go — Fuzz tests for the BuffList add/tick/query cycle.
// Verifies invariants: no panics on arbitrary buff IDs/values/durations,
// correct count semantics, and TickDoT non-negative return.
package fuzz

import (
	"math"
	"testing"

	"defense2/internal/core/buff"
)

// FuzzBuffListAddTick exercises the full BuffList lifecycle with arbitrary inputs.
func FuzzBuffListAddTick(f *testing.F) {
	f.Add("stun", 1.0, 0.5, 2.0)
	f.Add("slow", 0.5, 0.3, 3.0)
	f.Add("bleed", 10.0, 0.0, 5.0)
	f.Add("burn", 20.0, 0.0, 4.0)
	f.Add("", 0.0, 0.0, 0.0)
	f.Add("weaken", 0.3, 0.0, -1.0) // permanent buff

	f.Fuzz(func(t *testing.T, id string, value, value2, duration float64) {
		// Skip non-finite inputs.
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return
		}
		if math.IsNaN(value2) || math.IsInf(value2, 0) {
			return
		}
		if math.IsNaN(duration) || math.IsInf(duration, 0) {
			return
		}
		// Clamp duration: -1 = permanent, otherwise non-negative.
		if duration < -1 {
			duration = -1
		}

		bl := buff.NewDefaultBuffList()

		// Add should not panic.
		bl.Add(buff.Buff{
			ID:        id,
			Category:  buff.CatCC,
			Source:    "fuzz",
			Value:     value,
			Value2:    value2,
			Duration:  duration,
			Remaining: duration,
		})

		// All query methods should not panic.
		bl.Has(id)
		bl.Get(id)
		bl.GetAll(id)

		// Tick one frame — should not panic.
		bl.Tick(0.016)

		// TickDoT — should not panic and return >= 0.
		dotDmg := bl.TickDoT(0.016, 0.5)
		if dotDmg < 0 {
			t.Errorf("TickDoT returned negative: %f (id=%q, value=%f)", dotDmg, id, value)
		}

		// Active/Count should not panic.
		active := bl.Active()
		count := bl.Count()
		if count < 0 {
			t.Errorf("Count negative: %d", count)
		}
		if len(active) != count {
			t.Errorf("Active() length %d != Count() %d", len(active), count)
		}

		// ClearByCategory should not panic.
		bl.ClearByCategory(buff.CatCC)
	})
}

// FuzzBuffListMultiOp exercises multiple operations in sequence.
func FuzzBuffListMultiOp(f *testing.F) {
	f.Add("bleed", 10.0, 3.0, "burn", 5.0, 2.0)
	f.Add("stun", 0.0, 1.0, "slow", 0.5, 2.0)
	f.Add("poison", 8.0, 4.0, "weaken", 0.3, 5.0)

	f.Fuzz(func(t *testing.T, id1 string, val1, dur1 float64, id2 string, val2, dur2 float64) {
		if math.IsNaN(val1) || math.IsInf(val1, 0) {
			return
		}
		if math.IsNaN(val2) || math.IsInf(val2, 0) {
			return
		}
		if math.IsNaN(dur1) || math.IsInf(dur1, 0) {
			return
		}
		if math.IsNaN(dur2) || math.IsInf(dur2, 0) {
			return
		}
		if dur1 < -1 {
			dur1 = -1
		}
		if dur2 < -1 {
			dur2 = -1
		}

		bl := buff.NewDefaultBuffList()

		// Add two buffs.
		bl.Add(buff.Buff{
			ID: id1, Category: buff.CatDoT, Source: "fuzz1",
			Value: val1, Duration: dur1, Remaining: dur1,
		})
		bl.Add(buff.Buff{
			ID: id2, Category: buff.CatCC, Source: "fuzz2",
			Value: val2, Duration: dur2, Remaining: dur2,
		})

		countBefore := bl.Count()
		if countBefore < 0 {
			t.Fatalf("Count negative after adds: %d", countBefore)
		}

		// Simulate 10 ticks.
		for i := 0; i < 10; i++ {
			dotDmg := bl.TickDoT(0.016, 0.5)
			if dotDmg < 0 {
				t.Errorf("TickDoT negative at tick %d: %f", i, dotDmg)
			}
			bl.Tick(0.016)
		}

		countAfter := bl.Count()
		if countAfter < 0 {
			t.Errorf("Count negative after ticks: %d", countAfter)
		}

		// Remove by ID — should not panic.
		bl.RemoveByID(id1)

		// Clear — should not panic and result in zero buffs.
		bl.Clear()
		if bl.Count() != 0 {
			t.Errorf("Count after Clear: %d, want 0", bl.Count())
		}
	})
}
