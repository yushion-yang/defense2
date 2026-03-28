package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func TestFindNearestEnemy(t *testing.T) {
	ep := enemy.NewPool(8)
	ep.Spawn(200, 100, 10, 60, 8, 1) // 100px away
	ep.Spawn(300, 100, 10, 60, 8, 1) // 200px away
	ep.Spawn(500, 100, 10, 60, 8, 1) // 400px away (out of range)

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
	ep.Spawn(500, 500, 10, 60, 8, 1) // far away

	tw := &tower.Tower{X: 100, Y: 100, Range: 100, Active: true}
	target := tower.FindNearestEnemy(tw, ep)
	if target != nil {
		t.Fatal("should not find target out of range")
	}
}

func TestProjectileFireAndMove(t *testing.T) {
	pp := projectile.NewPool(16)
	pp.Fire(0, 0, 100, 0, 10, 200, 4) // fire right at 200px/s

	if pp.Count != 1 {
		t.Fatalf("expected 1 projectile, got %d", pp.Count)
	}

	pp.Update(0.25) // move 50px
	var px float64
	pp.Each(func(p *projectile.Projectile) { px = p.X })
	if px < 49 || px > 51 {
		t.Fatalf("expected X~50 after 0.25s, got %.1f", px)
	}
}

func TestTickProjectileHits(t *testing.T) {
	ep := enemy.NewPool(4)
	ep.Spawn(100, 0, 10, 60, 8, 1) // at (100, 0), HP=10

	pp := projectile.NewPool(16)
	pp.Fire(95, 0, 100, 0, 15, 200, 4) // starts very close, damage=15

	// Move just a bit so projectile overlaps enemy
	pp.Update(0.01)

	kills := pipeline.TickProjectileHits(pp, ep)
	if kills != 1 {
		t.Fatalf("expected 1 kill, got %d", kills)
	}
	if pp.Count != 0 {
		t.Fatalf("projectile should be released after hit, count=%d", pp.Count)
	}
	if ep.Count != 0 {
		t.Fatalf("enemy should be dead, count=%d", ep.Count)
	}
}

func TestTickTowerCombatFires(t *testing.T) {
	tp := tower.NewPool(4)
	def := tower.TowerDef{Key: "test", Faction: "base", Range: 200, Damage: 10, AttackSpeed: 2, Cost: 50}
	tp.Place(0, 0, 100, 100, def)

	ep := enemy.NewPool(4)
	ep.Spawn(150, 100, 20, 60, 8, 1) // 50px away, in range

	pp := projectile.NewPool(16)

	pipeline.TickTowerCombat(tp, ep, pp, 1.0/60)
	if pp.Count != 1 {
		t.Fatalf("tower should fire, projectile count=%d", pp.Count)
	}

	// Second tick should NOT fire (cooldown)
	pipeline.TickTowerCombat(tp, ep, pp, 1.0/60)
	if pp.Count != 1 {
		t.Fatalf("tower should be on cooldown, projectile count=%d", pp.Count)
	}
}
