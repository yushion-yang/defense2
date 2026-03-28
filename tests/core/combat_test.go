package core_test

import (
	"testing"

	_ "defense2/internal/core/tower/abilities" // register abilities

	"defense2/internal/core/enemy"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func TestFindNearestEnemy(t *testing.T) {
	ep := enemy.NewPool(8)
	ep.Spawn(200, 100, 10, 60, 8, 1)
	ep.Spawn(300, 100, 10, 60, 8, 1)
	ep.Spawn(500, 100, 10, 60, 8, 1)

	tw := &tower.Tower{X: 100, Y: 100, Range: 250, Active: true}
	target := tower.FindNearestEnemy(tw, ep)
	if target == nil {
		t.Fatal("should find a target")
	}
	if target.X != 200 {
		t.Fatalf("expected nearest at X=200, got X=%.0f", target.X)
	}
}

func TestFindNearestEnemyNoneInRange(t *testing.T) {
	ep := enemy.NewPool(4)
	ep.Spawn(500, 500, 10, 60, 8, 1)

	tw := &tower.Tower{X: 100, Y: 100, Range: 100, Active: true}
	target := tower.FindNearestEnemy(tw, ep)
	if target != nil {
		t.Fatal("should not find target out of range")
	}
}

func TestProjectileFireAndMove(t *testing.T) {
	pp := projectile.NewPool(16)
	pp.Fire(0, 0, 100, 0, 10, 200, 4)

	if pp.Count != 1 {
		t.Fatalf("expected 1 projectile, got %d", pp.Count)
	}

	pp.Update(0.25)
	var px float64
	pp.Each(func(p *projectile.Projectile) { px = p.X })
	if px < 49 || px > 51 {
		t.Fatalf("expected X~50 after 0.25s, got %.1f", px)
	}
}

func TestTickProjectileHits(t *testing.T) {
	ep := enemy.NewPool(4)
	ep.Spawn(100, 0, 10, 60, 8, 1)

	tp := tower.NewPool(1) // empty tower pool (no abilities to resolve)

	pp := projectile.NewPool(16)
	pp.Fire(95, 0, 100, 0, 15, 200, 4)
	pp.Update(0.01)

	kills := pipeline.TickProjectileHits(pp, ep, tp)
	if kills != 1 {
		t.Fatalf("expected 1 kill, got %d", kills)
	}
	if pp.Count != 0 {
		t.Fatalf("projectile should be released, count=%d", pp.Count)
	}
}

func TestTickTowerCombatFires(t *testing.T) {
	tp := tower.NewPool(4)
	def := tower.TowerDef{Key: "test", Label: "Test",
		Range: 200, Damage: 10, AttackSpeed: 2, Cost: 50}
	tp.Place(0, 0, 100, 100, def)

	ep := enemy.NewPool(4)
	ep.Spawn(150, 100, 20, 60, 8, 1)

	pp := projectile.NewPool(16)

	pipeline.TickTowerCombat(tp, ep, pp, 1.0/60)
	if pp.Count != 1 {
		t.Fatalf("tower should fire, projectile count=%d", pp.Count)
	}

	pipeline.TickTowerCombat(tp, ep, pp, 1.0/60)
	if pp.Count != 1 {
		t.Fatalf("tower should be on cooldown, projectile count=%d", pp.Count)
	}
}

func TestAbilityRegistry(t *testing.T) {
	// Abilities registered via init() in abilities package
	expected := []string{"splash", "crit", "onHitSlow", "bleedDot"}
	for _, name := range expected {
		if _, ok := tower.Registry[name]; !ok {
			t.Fatalf("ability %q not registered", name)
		}
	}
}

func TestSlowEffect(t *testing.T) {
	e := &enemy.Enemy{
		HP: 100, MaxHP: 100, Speed: 100, BaseSpeed: 100, Active: true,
	}
	e.SlowTimer = 1.0
	e.SlowFactor = 0.5

	enemy.TickStatusEffects(e, 0.5)
	if e.Speed != 50 {
		t.Fatalf("expected speed 50 when slowed, got %.0f", e.Speed)
	}

	enemy.TickStatusEffects(e, 0.6) // timer expires
	if e.Speed != 100 {
		t.Fatalf("expected speed restored to 100, got %.0f", e.Speed)
	}
}

func TestBleedEffect(t *testing.T) {
	e := &enemy.Enemy{
		HP: 100, MaxHP: 100, Speed: 100, BaseSpeed: 100, Active: true,
		BleedTimer: 2.0, BleedDPS: 10,
	}

	enemy.TickStatusEffects(e, 1.0) // 10 DPS * 1s = 10 damage
	if e.HP != 90 {
		t.Fatalf("expected HP 90 after 1s bleed, got %.0f", e.HP)
	}
}
