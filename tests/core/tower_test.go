package core_test

import (
	"testing"

	"defense2/internal/core/tower"
)

// testTowerDef returns a minimal TowerDef for unit tests.
func testTowerDef() tower.TowerDef {
	return tower.TowerDef{
		Key: "test", Label: "Test",
		Range: 150, Damage: 10, AttackSpeed: 1.5, Cost: 50,
		Color: [3]uint8{80, 140, 220},
	}
}

func TestTowerPlaceAndRemove(t *testing.T) {
	p := tower.NewPool(4)
	def := testTowerDef()

	t1 := p.Place(2, 3, 210, 150, def)
	if t1 == nil || p.Count != 1 {
		t.Fatal("place should succeed")
	}
	if t1.Row != 2 || t1.Col != 3 {
		t.Fatalf("expected (2,3), got (%d,%d)", t1.Row, t1.Col)
	}
	if t1.Range != 150 || t1.Damage != 10 {
		t.Fatalf("stats mismatch: range=%.0f damage=%.0f", t1.Range, t1.Damage)
	}

	p.Remove(t1)
	if p.Count != 0 {
		t.Fatalf("count after remove should be 0, got %d", p.Count)
	}
}

func TestTowerAt(t *testing.T) {
	p := tower.NewPool(4)
	def := testTowerDef()
	p.Place(2, 3, 210, 150, def)

	found := p.At(2, 3)
	if found == nil {
		t.Fatal("At should find tower at (2,3)")
	}
	missing := p.At(0, 0)
	if missing != nil {
		t.Fatal("At should return nil for empty cell")
	}
}

func TestTowerPoolOverflow(t *testing.T) {
	p := tower.NewPool(1)
	def := testTowerDef()
	p.Place(0, 0, 30, 30, def)
	t2 := p.Place(1, 1, 90, 90, def)
	if t2 != nil {
		t.Fatal("overflow place should return nil")
	}
}
