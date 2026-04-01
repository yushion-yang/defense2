package item

import (
	"testing"

	"defense2/internal/core/tower"
)

func TestNewInventory(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"zero", 0},
		{"three", 3},
		{"ten", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := NewInventory(tt.n)
			for _, k := range AllKinds {
				if got := inv.Count(k); got != tt.n {
					t.Errorf("Count(%d) = %d, want %d", k, got, tt.n)
				}
			}
		})
	}
}

func TestInventoryUse(t *testing.T) {
	inv := NewInventory(2)

	// First use succeeds
	if !inv.Use(KindBaseDamage) {
		t.Fatal("first Use should succeed")
	}
	if got := inv.Count(KindBaseDamage); got != 1 {
		t.Errorf("Count after first Use = %d, want 1", got)
	}

	// Second use succeeds
	if !inv.Use(KindBaseDamage) {
		t.Fatal("second Use should succeed")
	}
	if got := inv.Count(KindBaseDamage); got != 0 {
		t.Errorf("Count after second Use = %d, want 0", got)
	}

	// Third use fails
	if inv.Use(KindBaseDamage) {
		t.Fatal("third Use should fail when empty")
	}
	if got := inv.Count(KindBaseDamage); got != 0 {
		t.Errorf("Count after failed Use = %d, want 0", got)
	}

	// Other kinds unaffected
	if got := inv.Count(KindBaseSpeed); got != 2 {
		t.Errorf("KindBaseSpeed should still be 2, got %d", got)
	}
}

func TestTotalCount(t *testing.T) {
	inv := NewInventory(3)
	want := 3 * int(KindCount)
	if got := inv.TotalCount(); got != want {
		t.Errorf("TotalCount() = %d, want %d", got, want)
	}

	inv.Use(KindBaseDamage)
	inv.Use(KindBaseRange)
	want -= 2
	if got := inv.TotalCount(); got != want {
		t.Errorf("TotalCount() after 2 uses = %d, want %d", got, want)
	}
}

func TestDefsComplete(t *testing.T) {
	for _, k := range AllKinds {
		d := Defs[k]
		if d.Name == "" {
			t.Errorf("Defs[%d] has empty Name", k)
		}
		if d.BoostVal == 0 {
			t.Errorf("Defs[%d] (%s) has zero BoostVal", k, d.Name)
		}
		if d.Color.A == 0 {
			t.Errorf("Defs[%d] (%s) has zero alpha", k, d.Name)
		}
		if d.Kind != k {
			t.Errorf("Defs[%d].Kind = %d, want %d", k, d.Kind, k)
		}
	}
}

func TestApplyItem(t *testing.T) {
	tests := []struct {
		kind  Kind
		field string
		get   func(*tower.Tower) float64
	}{
		{KindBaseDamage, "BaseDamage", func(tw *tower.Tower) float64 { return tw.BaseDamage }},
		{KindPotentialDamage, "PotentialDamage", func(tw *tower.Tower) float64 { return tw.PotentialDamage }},
		{KindBaseSpeed, "BaseSpeed", func(tw *tower.Tower) float64 { return tw.BaseSpeed }},
		{KindPotentialSpeed, "PotentialSpeed", func(tw *tower.Tower) float64 { return tw.PotentialSpeed }},
		{KindBaseRange, "BaseRange", func(tw *tower.Tower) float64 { return tw.BaseRange }},
		{KindPotentialRange, "PotentialRange", func(tw *tower.Tower) float64 { return tw.PotentialRange }},
	}
	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			tw := &tower.Tower{}
			before := tt.get(tw)
			ApplyItem(tw, tt.kind)
			after := tt.get(tw)
			boost := Defs[tt.kind].BoostVal
			if diff := after - before; diff != boost {
				t.Errorf("ApplyItem(%s): delta = %f, want %f", tt.field, diff, boost)
			}
		})
	}
}

func TestApplyItemRecalcs(t *testing.T) {
	tw := &tower.Tower{
		BaseDamage:      10,
		PotentialDamage: 5,
	}
	// Without Strength, RecalcStats uses ratio=1.0: Damage = base + potential
	ApplyItem(tw, KindBaseDamage)
	// BaseDamage is now 12, PotentialDamage still 5, so Damage = 12 + 5 = 17
	want := 17.0
	if tw.Damage != want {
		t.Errorf("Damage after ApplyItem = %f, want %f", tw.Damage, want)
	}
}
