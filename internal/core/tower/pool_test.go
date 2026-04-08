package tower

import (
	"testing"

	"defense2/internal/config"
)

func TestPlaceFromSnapshot(t *testing.T) {
	p := NewPool(4)
	def := TowerDef{
		Key: "prism", Label: "Prism", Cost: 70,
		AttackStyleID: StyleWideBeam, ProjectileSpeed: 300,
		CfgBaseDamage: 10, PotentialDamage: 5,
		CfgBaseSpeed: 1.0, PotentialSpeed: 0.5,
		CfgBaseRange: 100, PotentialRange: 20,
	}
	snap := config.TowerSnapshot{
		Row: 3, Col: 5, Key: "prism",
		AbilitySlots:    [6]string{"scatter", "", "auraDamage", "", "", ""},
		DamageTier:      "A",
		SpeedTier:       "B",
		RangeTier:       "S",
		BaseDamage:      12.0,
		PotentialDamage: 8.0,
		BaseSpeed:       1.5,
		PotentialSpeed:  0.5,
		BaseRange:       120.0,
		PotentialRange:  30.0,
	}
	tw := p.PlaceFromSnapshot(3, 5, 200.0, 300.0, def, snap)
	if tw == nil {
		t.Fatal("PlaceFromSnapshot returned nil")
	}
	if tw.Row != 3 || tw.Col != 5 {
		t.Errorf("pos = (%d,%d), want (3,5)", tw.Row, tw.Col)
	}
	if tw.Key != "prism" {
		t.Errorf("key = %q, want %q", tw.Key, "prism")
	}
	if tw.DamageTier != "A" {
		t.Errorf("DamageTier = %q, want %q", tw.DamageTier, "A")
	}
	if tw.SpeedTier != "B" {
		t.Errorf("SpeedTier = %q, want %q", tw.SpeedTier, "B")
	}
	if tw.RangeTier != "S" {
		t.Errorf("RangeTier = %q, want %q", tw.RangeTier, "S")
	}
	// Verify snapshot values used, not def values
	if diff := tw.BaseDamage - 12.0; diff > 0.01 || diff < -0.01 {
		t.Errorf("BaseDamage = %f, want 12.0", tw.BaseDamage)
	}
	if diff := tw.PotentialDamage - 8.0; diff > 0.01 || diff < -0.01 {
		t.Errorf("PotentialDamage = %f, want 8.0", tw.PotentialDamage)
	}
	if diff := tw.BaseRange - 120.0; diff > 0.01 || diff < -0.01 {
		t.Errorf("BaseRange = %f, want 120.0", tw.BaseRange)
	}
	if p.Count != 1 {
		t.Errorf("Count = %d, want 1", p.Count)
	}
	if !tw.Active {
		t.Error("tower should be Active")
	}
}

func TestPlaceFromSnapshotPoolFull(t *testing.T) {
	p := NewPool(1)
	def := TowerDef{Key: "sentinel", Cost: 50, AttackStyleID: StyleProjectile}
	snap := config.TowerSnapshot{Row: 0, Col: 0, Key: "sentinel", BaseDamage: 10, BaseSpeed: 1, BaseRange: 100}
	p.PlaceFromSnapshot(0, 0, 100, 100, def, snap) // fill the only slot
	tw := p.PlaceFromSnapshot(1, 1, 200, 200, def, snap)
	if tw != nil {
		t.Error("expected nil when pool is full")
	}
}
