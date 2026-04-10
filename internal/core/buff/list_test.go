package buff

import "testing"

func testRules() map[string]StackRule {
	return map[string]StackRule{
		"stun":           {Mode: Override},
		"slow":           {Mode: Strongest, Cap: 0.8},
		"root":           {Mode: Override},
		"bleed":          {Mode: IndependentPerSource},
		"burn":           {Mode: IndependentPerSource},
		"poison":         {Mode: IndependentPerSource},
		"weaken":         {Mode: Strongest, Cap: 0.5},
		"controlImmune":  {Mode: Override, Priority: 80},
		"speedUp":        {Mode: Additive, Cap: 1.4},
		"damageDown":     {Mode: Multiplicative, Floor: 0.2},
	}
}

// ────────────────────────────────────────────
// Add / Has
// ────────────────────────────────────────────

func TestBuffList_AddHas(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	if !bl.Has("stun") {
		t.Error("expected stun to be present")
	}
	if bl.Has("slow") {
		t.Error("unexpected slow")
	}
}

// ────────────────────────────────────────────
// Override mode
// ────────────────────────────────────────────

func TestBuffList_Override(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 1, Remaining: 1})
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t2", Duration: 3, Remaining: 3})

	if bl.Count() != 1 {
		t.Fatalf("override: want 1 buff, got %d", bl.Count())
	}
	b, ok := bl.Get("stun")
	if !ok {
		t.Fatal("stun not found")
	}
	if b.Remaining != 3 {
		t.Errorf("want remaining 3, got %f", b.Remaining)
	}
	if b.Source != "t2" {
		t.Errorf("want source t2, got %s", b.Source)
	}
}

// ────────────────────────────────────────────
// Strongest mode
// ────────────────────────────────────────────

func TestBuffList_Strongest_HigherValueWins(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t1", Value: 0.2, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t2", Value: 0.4, Duration: 2, Remaining: 2})

	if bl.Count() != 1 {
		t.Fatalf("strongest: want 1 buff, got %d", bl.Count())
	}
	b, _ := bl.Get("weaken")
	if b.Value != 0.4 {
		t.Errorf("want Value 0.4, got %f", b.Value)
	}
}

func TestBuffList_Strongest_LongerRemainingWins(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t1", Value: 0.3, Duration: 5, Remaining: 5})
	// Same value but shorter remaining — should NOT replace
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t2", Value: 0.3, Duration: 2, Remaining: 2})

	b, _ := bl.Get("weaken")
	if b.Remaining != 5 {
		t.Errorf("same value: should keep longer remaining, got %f", b.Remaining)
	}
}

func TestBuffList_Strongest_Cap(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "slow", Category: CatCC, Source: "t1", Value: 0.9, Duration: 3, Remaining: 3})

	b, _ := bl.Get("slow")
	// Cap is 0.8
	if b.Value != 0.8 {
		t.Errorf("slow should be capped to 0.8, got %f", b.Value)
	}
}

// ────────────────────────────────────────────
// IndependentPerSource mode
// ────────────────────────────────────────────

func TestBuffList_IndependentPerSource(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t2", Value: 15, Duration: 2, Remaining: 2})

	if bl.Count() != 2 {
		t.Fatalf("want 2 bleed sources, got %d", bl.Count())
	}

	// Same source overwrites
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 20, Duration: 4, Remaining: 4})
	if bl.Count() != 2 {
		t.Fatalf("same source overwrite: want 2, got %d", bl.Count())
	}

	all := bl.GetAll("bleed")
	if len(all) != 2 {
		t.Fatalf("GetAll: want 2, got %d", len(all))
	}
	for _, b := range all {
		if b.Source == "t1" && b.Value != 20 {
			t.Errorf("t1 bleed: want Value 20, got %f", b.Value)
		}
		if b.Source == "t1" && b.Remaining != 4 {
			t.Errorf("t1 bleed: want Remaining 4, got %f", b.Remaining)
		}
	}
}

// ────────────────────────────────────────────
// Additive mode
// ────────────────────────────────────────────

func TestBuffList_Additive(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "speedUp", Category: CatAura, Source: "t1", Value: 0.2, Duration: -1, Remaining: -1})
	bl.Add(Buff{ID: "speedUp", Category: CatAura, Source: "t2", Value: 0.3, Duration: -1, Remaining: -1})

	if bl.Count() != 2 {
		t.Fatalf("additive: want 2, got %d", bl.Count())
	}
	b, ok := bl.Get("speedUp")
	if !ok {
		t.Fatal("speedUp not found")
	}
	// Sum = 0.5
	if b.Value < 0.49 || b.Value > 0.51 {
		t.Errorf("want Value ~0.5, got %f", b.Value)
	}
}

func TestBuffList_Additive_Cap(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "speedUp", Category: CatAura, Source: "t1", Value: 0.8, Duration: -1, Remaining: -1})
	bl.Add(Buff{ID: "speedUp", Category: CatAura, Source: "t2", Value: 0.8, Duration: -1, Remaining: -1})

	b, _ := bl.Get("speedUp")
	// Cap = 1.4, sum = 1.6 → capped to 1.4
	if b.Value != 1.4 {
		t.Errorf("want Value capped to 1.4, got %f", b.Value)
	}
}

// ────────────────────────────────────────────
// Multiplicative mode
// ────────────────────────────────────────────

func TestBuffList_Multiplicative(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "damageDown", Category: CatDefense, Source: "t1", Value: 0.5, Duration: -1, Remaining: -1})
	bl.Add(Buff{ID: "damageDown", Category: CatDefense, Source: "t2", Value: 0.5, Duration: -1, Remaining: -1})

	b, ok := bl.Get("damageDown")
	if !ok {
		t.Fatal("damageDown not found")
	}
	// Product = 0.25, but floor = 0.2, so 0.25 stays
	if b.Value < 0.24 || b.Value > 0.26 {
		t.Errorf("want Value ~0.25, got %f", b.Value)
	}
}

func TestBuffList_Multiplicative_Floor(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "damageDown", Category: CatDefense, Source: "t1", Value: 0.1, Duration: -1, Remaining: -1})
	bl.Add(Buff{ID: "damageDown", Category: CatDefense, Source: "t2", Value: 0.1, Duration: -1, Remaining: -1})

	b, _ := bl.Get("damageDown")
	// Product = 0.01, floor = 0.2 → clamped to 0.2
	if b.Value != 0.2 {
		t.Errorf("want Value floored to 0.2, got %f", b.Value)
	}
}

// ────────────────────────────────────────────
// Default mode (unknown buff ID → Override)
// ────────────────────────────────────────────

func TestBuffList_DefaultOverride(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "unknown", Source: "t1", Value: 1, Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "unknown", Source: "t2", Value: 2, Duration: 3, Remaining: 3})

	if bl.Count() != 1 {
		t.Fatalf("default override: want 1, got %d", bl.Count())
	}
	b, _ := bl.Get("unknown")
	if b.Value != 2 {
		t.Errorf("want Value 2, got %f", b.Value)
	}
}

// ────────────────────────────────────────────
// Remove
// ────────────────────────────────────────────

func TestBuffList_Remove(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})

	bl.Remove("stun", "t1")
	if bl.Has("stun") {
		t.Error("stun should be removed")
	}
	if !bl.Has("bleed") {
		t.Error("bleed should remain")
	}
}

func TestBuffList_RemoveByID(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t2", Value: 15, Duration: 2, Remaining: 2})

	bl.RemoveByID("bleed")
	if bl.Has("bleed") {
		t.Error("all bleed should be removed")
	}
	if bl.Count() != 0 {
		t.Errorf("want 0, got %d", bl.Count())
	}
}

// ────────────────────────────────────────────
// ClearByCategory
// ────────────────────────────────────────────

func TestBuffList_ClearByCategory(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "slow", Category: CatCC, Source: "t1", Value: 0.5, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t1", Value: 0.2, Duration: 4, Remaining: 4})

	bl.ClearByCategory(CatCC)
	if bl.Has("stun") || bl.Has("slow") {
		t.Error("CC buffs should be cleared")
	}
	if !bl.Has("bleed") {
		t.Error("DoT should remain")
	}
	if !bl.Has("weaken") {
		t.Error("Debuff should remain")
	}
}

func TestBuffList_ClearByCategory_Multiple(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})
	bl.Add(Buff{ID: "weaken", Category: CatDebuff, Source: "t1", Value: 0.2, Duration: 4, Remaining: 4})

	bl.ClearByCategory(CatCC, CatDoT, CatDebuff)
	if bl.Count() != 0 {
		t.Errorf("all should be cleared, got %d", bl.Count())
	}
}

// ────────────────────────────────────────────
// Clear
// ────────────────────────────────────────────

func TestBuffList_Clear(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})

	bl.Clear()
	if bl.Count() != 0 {
		t.Errorf("want 0, got %d", bl.Count())
	}
}

// ────────────────────────────────────────────
// Active snapshot
// ────────────────────────────────────────────

func TestBuffList_Active(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 2, Remaining: 2})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3, Remaining: 3})

	snap := bl.Active()
	if len(snap) != 2 {
		t.Fatalf("want 2 in snapshot, got %d", len(snap))
	}
	// Verify it's a copy
	snap[0].Remaining = 999
	b, _ := bl.Get("stun")
	if b.Remaining == 999 {
		t.Error("Active() should return a copy, not a reference")
	}
}

// ────────────────────────────────────────────
// Get returns not found for missing buff
// ────────────────────────────────────────────

func TestBuffList_Get_NotFound(t *testing.T) {
	bl := NewBuffList(testRules())
	_, ok := bl.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent buff")
	}
}

func TestBuffList_GetAll_Empty(t *testing.T) {
	bl := NewBuffList(testRules())
	all := bl.GetAll("nonexistent")
	if len(all) != 0 {
		t.Errorf("want 0, got %d", len(all))
	}
}

// ────────────────────────────────────────────
// Tick — timer countdown + expiry
// ────────────────────────────────────────────

func TestBuffList_Tick(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 1.0, Remaining: 1.0})
	bl.Add(Buff{ID: "bleed", Category: CatDoT, Source: "t1", Value: 10, Duration: 3.0, Remaining: 3.0})

	bl.Tick(0.5)
	b, _ := bl.Get("stun")
	if b.Remaining < 0.49 || b.Remaining > 0.51 {
		t.Errorf("stun remaining: want ~0.5, got %f", b.Remaining)
	}

	bl.Tick(0.6) // stun expires (0.5 - 0.6 < 0)
	if bl.Has("stun") {
		t.Error("stun should have expired")
	}
	if !bl.Has("bleed") {
		t.Error("bleed should still be active")
	}
	b2, _ := bl.Get("bleed")
	// 3.0 - 0.5 - 0.6 = 1.9
	if b2.Remaining < 1.89 || b2.Remaining > 1.91 {
		t.Errorf("bleed remaining: want ~1.9, got %f", b2.Remaining)
	}
}

func TestBuffList_Tick_Permanent(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "controlImmune", Category: CatDefense, Source: "boss", Duration: -1, Remaining: -1})
	bl.Tick(100)
	if !bl.Has("controlImmune") {
		t.Error("permanent buff should not expire")
	}
}

func TestBuffList_Tick_AllExpire(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 1, Remaining: 1})
	bl.Add(Buff{ID: "root", Category: CatCC, Source: "t2", Duration: 0.5, Remaining: 0.5})

	bl.Tick(2)
	if bl.Count() != 0 {
		t.Errorf("all should expire, got %d", bl.Count())
	}
}

func TestBuffList_Tick_ExactExpiry(t *testing.T) {
	bl := NewBuffList(testRules())
	bl.Add(Buff{ID: "stun", Category: CatCC, Source: "t1", Duration: 1, Remaining: 1})

	bl.Tick(1.0) // remaining = 0 → should expire
	if bl.Has("stun") {
		t.Error("buff with remaining=0 should be expired")
	}
}
