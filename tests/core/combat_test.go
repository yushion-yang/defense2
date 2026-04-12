package core_test

import (
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/tower/abilities"

	"defense2/internal/core/enemy"
	"defense2/internal/core/pipeline"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func TestFindNearestEnemy(t *testing.T) {
	ep := enemy.NewPool(8)
	e1 := ep.Spawn(200, 100, 10, 60, 1, "normal", nil)
	e2 := ep.Spawn(300, 100, 10, 60, 1, "normal", nil)
	e3 := ep.Spawn(500, 100, 10, 60, 1, "normal", nil)
	// Clear spawn animation so enemies are targetable
	e1.SpawnTimer = 0
	e2.SpawnTimer = 0
	e3.SpawnTimer = 0

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
	ep.Spawn(500, 500, 10, 60, 1, "normal", nil)

	tw := &tower.Tower{X: 100, Y: 100, Range: 100, Active: true}
	target := tower.FindNearestEnemy(tw, ep)
	if target != nil {
		t.Fatal("should not find target out of range")
	}
}

func TestProjectileFireAndMove(t *testing.T) {
	pp := projectile.NewPool(16)
	pp.Fire(0, 0, 100, 0, 10, 200, 4, nil, "")

	if pp.Count != 1 {
		t.Fatalf("expected 1 projectile, got %d", pp.Count)
	}

	pp.Tick(0.25)
	var px float64
	pp.Each(func(p *projectile.Projectile) { px = p.X })
	if px < 49 || px > 51 {
		t.Fatalf("expected X~50 after 0.25s, got %.1f", px)
	}
}

func TestTickProjectileHits(t *testing.T) {
	ep := enemy.NewPool(4)
	e := ep.Spawn(100, 0, 10, 60, 1, "normal", nil)
	e.SpawnTimer = 0 // Clear spawn animation so enemy is hittable

	tp := tower.NewPool(1) // empty tower pool (no abilities to resolve)

	pp := projectile.NewPool(16)
	pp.Fire(95, 0, 100, 0, 15, 200, 4, nil, "")
	pp.Tick(0.01)

	kills := pipeline.TickProjectileHits(pp, ep, tp, nil, nil, nil, nil)
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
	en := ep.Spawn(150, 100, 20, 60, 1, "normal", nil)
	en.SpawnTimer = 0 // Clear spawn animation so enemy is targetable

	pp := projectile.NewPool(16)

	pipeline.TickTowerCombat(tp, ep, pp, nil, 1.0/60, nil, nil, nil, nil)
	if pp.Count != 1 {
		t.Fatalf("tower should fire, projectile count=%d", pp.Count)
	}

	pipeline.TickTowerCombat(tp, ep, pp, nil, 1.0/60, nil, nil, nil, nil)
	if pp.Count != 1 {
		t.Fatalf("tower should be on cooldown, projectile count=%d", pp.Count)
	}
}

func TestAbilityRegistry(t *testing.T) {
	config.SetDataFS(&defense2.DataFS)
	abilities.InitConfigAbilities()
	expected := []string{"splash", "crit", "slowPower", "bleedDot"}
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
	e.Buffs = buff.NewDefaultBuffList()
	e.Buffs.Add(buff.Buff{
		ID: "slow", Category: buff.CatCC, Source: "test",
		Value: 0.5, Duration: 1.0, Remaining: 1.0,
	})

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
	}
	e.Buffs = buff.NewDefaultBuffList()
	e.Buffs.Add(buff.Buff{
		ID: "bleed", Category: buff.CatDoT, Source: "test",
		Value: 10, Duration: 2.0, Remaining: 2.0,
	})

	// DoT damage is now deferred to LastDotDmg (applied by pipeline via ApplyDamage).
	// DotTickInterval=0.5: after 0.5s, one tick fires with damage = 10 DPS * 0.5s = 5.
	enemy.TickStatusEffects(e, 0.5)
	if e.LastDotDmg != 5 {
		t.Fatalf("expected LastDotDmg 5 after 0.5s bleed tick, got %.0f", e.LastDotDmg)
	}
}
