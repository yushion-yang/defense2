package buff

import "testing"

func TestDotAccumulator_SingleSource(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 20, Duration: 3, Remaining: 3}) // 20 DPS

	dotInterval := 0.5
	var totalDmg float64

	// Simulate 0.5s in 5 frames of 0.1s
	for i := 0; i < 5; i++ {
		totalDmg += bl.TickDoT(0.1, dotInterval)
	}

	// After 0.5s: bleed = 20 DPS * 0.5s = 10
	if totalDmg < 9.9 || totalDmg > 10.1 {
		t.Errorf("want ~10 damage after 0.5s, got %f", totalDmg)
	}
}

func TestDotAccumulator_MultipleSources(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 20, Duration: 3, Remaining: 3}) // 20 DPS
	bl.Add(Buff{ID: "burn", Category: CatDoT, Source: "t2", Value: 10, Duration: 2, Remaining: 2})   // 10 DPS

	dotInterval := 0.5
	var totalDmg float64

	// Simulate 0.5s in 5 frames of 0.1s
	for i := 0; i < 5; i++ {
		totalDmg += bl.TickDoT(0.1, dotInterval)
	}

	// After 0.5s: bleed=20*0.5=10, burn=10*0.5=5 → total=15
	if totalDmg < 14.9 || totalDmg > 15.1 {
		t.Errorf("want ~15 damage after 0.5s, got %f", totalDmg)
	}
}

func TestDotAccumulator_NoDot(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})

	dmg := bl.TickDoT(0.1, 0.5)
	if dmg != 0 {
		t.Errorf("want 0 damage for non-DoT buff, got %f", dmg)
	}
}

func TestDotAccumulator_Empty(t *testing.T) {
	bl := NewBuffList(testRules())

	dmg := bl.TickDoT(0.1, 0.5)
	if dmg != 0 {
		t.Errorf("want 0 damage for empty list, got %f", dmg)
	}
}

func TestDotAccumulator_TwoTicks(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "poison", Category: CatDoT, Source: "t1", Value: 30, Duration: 5, Remaining: 5}) // 30 DPS

	dotInterval := 0.5
	var totalDmg float64

	// Simulate 1.0s in 10 frames of 0.1s → should fire twice
	for i := 0; i < 10; i++ {
		totalDmg += bl.TickDoT(0.1, dotInterval)
	}

	// 2 ticks * 30 DPS * 0.5s = 30
	if totalDmg < 29.9 || totalDmg > 30.1 {
		t.Errorf("want ~30 damage after 1.0s (2 ticks), got %f", totalDmg)
	}
}

func TestDotAccumulator_ResetWhenNoDoTs(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 20, Duration: 3, Remaining: 3})

	// Tick a few frames to build up dotTimer
	bl.TickDoT(0.1, 0.5)
	bl.TickDoT(0.1, 0.5)

	// Remove DoT
	bl.RemoveByID("bleed")

	// Should reset timer and return 0
	dmg := bl.TickDoT(0.1, 0.5)
	if dmg != 0 {
		t.Errorf("want 0 after DoT removed, got %f", dmg)
	}
}

func TestDotAccumulator_ZeroDPSIgnored(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 0, Duration: 3, Remaining: 3}) // 0 DPS

	var totalDmg float64
	for i := 0; i < 5; i++ {
		totalDmg += bl.TickDoT(0.1, 0.5)
	}

	if totalDmg != 0 {
		t.Errorf("want 0 damage for 0 DPS DoT, got %f", totalDmg)
	}
}
