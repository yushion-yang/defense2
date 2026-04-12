package regression_test

import (
	"math"
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/tests/regression/sim"
)

// ============================================================
// Barrage Ability Tests
// ============================================================

// TestBarrage_AbilityInAttackCategory verifies barrage appears in the attack
// category candidate pool (category index 0).
func TestBarrage_AbilityInAttackCategory(t *testing.T) {
	if config.GlobalAbilityTable() == nil {
		t.Skip("ability table not loaded")
	}
	abilities := tower.AbilitiesForCategory(config.AbilityCatAttack)
	found := false
	for _, a := range abilities {
		if a.Type == tower.AbilityBarrage {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("barrage should appear in attack category abilities")
	}
}

// TestBarrage_CalcScale_Bullets verifies bullet count scaling:
// str=100 → 2, str=200 → 3, str=300 → 4.
func TestBarrage_CalcScale_Bullets(t *testing.T) {
	table := config.GlobalAbilityTable()
	if table == nil {
		t.Skip("ability table not loaded")
	}
	def := table[tower.AbilityBarrage]
	if def == nil {
		t.Fatal("barrage ability not found in table")
	}

	tests := []struct {
		str     float64
		bullets int
	}{
		{100, 2},
		{200, 3},
		{300, 4},
		{50, 1},
	}
	for _, tt := range tests {
		got := int(math.Floor(def.CalcScale(tt.str)))
		if got != tt.bullets {
			t.Errorf("str=%.0f: want %d bullets, got %d", tt.str, tt.bullets, got)
		}
	}
}

// TestBarrage_DamageRatio verifies each bullet uses 50% damage ratio.
func TestBarrage_DamageRatio(t *testing.T) {
	table := config.GlobalAbilityTable()
	if table == nil {
		t.Skip("ability table not loaded")
	}
	def := table[tower.AbilityBarrage]
	if def == nil {
		t.Fatal("barrage ability not found")
	}
	if def.Param != 0.5 {
		t.Errorf("damageRatio (param) should be 0.5, got %f", def.Param)
	}
}

// TestBarrage_SelfManaged verifies the barrage handler is registered and self-managed.
func TestBarrage_SelfManaged(t *testing.T) {
	h := combat.Get(tower.StyleBarrage)
	if h == nil {
		t.Fatal("barrage handler not registered")
	}
	if !combat.IsSelfManaged(tower.StyleBarrage) {
		t.Fatal("barrage should be self-managed")
	}
}

// TestBarrage_ResolveAttackStyle verifies tower resolves barrage to StyleBarrage.
func TestBarrage_ResolveAttackStyle(t *testing.T) {
	tw := &tower.Tower{}
	tw.AbilitySlots[0] = tower.AbilityBarrage
	style := tw.ResolveAttackStyle()
	if style != tower.StyleBarrage {
		t.Errorf("expected StyleBarrage, got %q", style)
	}
}

// TestBarrage_SpriteKey verifies barrage maps to gatling sprite.
func TestBarrage_SpriteKey(t *testing.T) {
	key := tower.AbilitySpriteKey(tower.AbilityBarrage)
	if key != "gatling" {
		t.Errorf("expected gatling sprite key, got %q", key)
	}
	label := tower.SpriteLabelFor(key)
	if label != "加特林" {
		t.Errorf("expected 加特林 label, got %q", label)
	}
}

// TestBarrage_BurstFiresMultipleProjectiles verifies the barrage handler creates
// multiple projectiles during burst firing over successive ticks.
func TestBarrage_BurstFiresMultipleProjectiles(t *testing.T) {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", 1000, 50).
		Build()

	e := s.SpawnedEnemies[0]
	projPool := projectile.NewPool(64)

	tw := &tower.Tower{
		X:             100,
		Y:             100,
		Damage:        20,
		Range:         300,
		AttackSpeed:   1.0,
		Active:        true,
		Key:           "test",
		InstanceKey:   "test_barrage_0",
		AttackStyleID: tower.StyleBarrage,
		Strength:      strength.NewStrengthData(),
	}
	tw.AbilitySlots[0] = tower.AbilityBarrage
	tw.Abilities = tw.AllAbilities()
	tw.Target = e

	ctx := &combat.AttackContext{
		Enemies:     s.Enemies,
		Projectiles: projPool,
		DT:          1.0 / 60.0,
		Style:       string(tower.StyleBarrage),
	}

	handler := combat.Get(tower.StyleBarrage)
	if handler == nil {
		t.Fatal("barrage handler not found")
	}
	th, ok := handler.(combat.TickHandler)
	if !ok {
		t.Fatal("barrage handler should implement TickHandler")
	}

	// First tick: should start burst (BarrageBurst set, timer=0)
	tw.FireTimer = 0 // ready to fire
	th.Tick(tw, ctx)

	if tw.BarrageBurst <= 0 {
		t.Fatal("after first tick, BarrageBurst should be > 0 (burst initiated)")
	}

	// Tick enough times to fire all bullets
	for i := 0; i < 200; i++ {
		th.Tick(tw, ctx)
		if tw.BarrageBurst <= 0 {
			break
		}
	}

	// At str=100, expect 2 bullets → 2 projectiles
	if projPool.Count < 2 {
		t.Errorf("expected at least 2 projectiles from barrage burst, got %d", projPool.Count)
	}
	t.Logf("barrage burst created %d projectiles", projPool.Count)
}

// TestBarrage_CountersDamageCap verifies that barrage deals more total damage
// than a single hit against an enemy with damageCap.
func TestBarrage_CountersDamageCap(t *testing.T) {
	// Enemy with damageCap=10, HP=100
	e1 := makeEnemy(100, 50)
	e1.DamageCap = 10

	// Single hit of 50 damage → capped to 10
	tw := makeTower(50)
	out := combat.ApplyHit(hitInput(tw, e1, 50, "projectile"), nil)
	singleDmg := out.TotalDamage

	if singleDmg > 10 {
		t.Fatalf("single hit should be capped at 10, got %.1f", singleDmg)
	}

	// Barrage: 2 separate hits of 25 (50*0.5) each → each capped to 10 → total 20
	e2 := makeEnemy(100, 50)
	e2.DamageCap = 10

	tw2 := makeTower(50)
	var totalBarrageDmg float64
	for i := 0; i < 2; i++ {
		out := combat.ApplyHit(hitInput(tw2, e2, 25, "barrage"), nil)
		totalBarrageDmg += out.TotalDamage
	}

	if totalBarrageDmg <= singleDmg {
		t.Errorf("barrage (2 hits) should deal more than single hit vs damageCap: barrage=%.1f single=%.1f", totalBarrageDmg, singleDmg)
	}
	t.Logf("damageCap=10: single_hit=%.1f, barrage_2hits=%.1f", singleDmg, totalBarrageDmg)
}
