package core_test

import (
	"testing"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/tower"
)

func makeTower(style tower.AttackStyle, damage, rng, speed float64) *tower.Tower {
	return &tower.Tower{
		X: 100, Y: 100,
		Range: rng, Damage: damage, AttackSpeed: speed,
		Active: true, Key: "test",
		AttackStyleID:   style,
		ProjectileSpeed: 400,
		ScatterPellets:  3, ScatterSpread: 0.52, // ~30 deg
		ChargeMult:      3,
		InnerDmgBonus:   1.5, InnerRatioR: 0.5,
		PierceTargets:   2, PierceDecay: 0.8,
	}
}

func makeEnemy(x, y, hp float64) *enemy.Enemy {
	return &enemy.Enemy{X: x, Y: y, HP: hp, MaxHP: hp, Active: true, Radius: 8}
}

func makeCtx(enemies *enemy.Pool, projs *projectile.Pool, beams *combat.BeamPool) *combat.AttackContext {
	return &combat.AttackContext{
		Enemies:     enemies,
		Projectiles: projs,
		Beams:       beams,
		DT:          1.0 / 60,
	}
}

// ── Handler Registry ──

func TestAllHandlersRegistered(t *testing.T) {
	styles := []tower.AttackStyle{
		tower.StyleProjectile, tower.StyleLaser, tower.StyleWideBeam,
		tower.StyleScatter, tower.StyleCharge, tower.StyleSpinAoE,
		tower.StylePierce, tower.StyleAuraDot,
	}
	for _, s := range styles {
		if combat.Get(s) == nil {
			t.Errorf("handler not registered: %s", s)
		}
	}
}

func TestSelfManagedStyles(t *testing.T) {
	selfManaged := []tower.AttackStyle{tower.StyleCharge, tower.StyleSpinAoE, tower.StyleAuraDot}
	for _, s := range selfManaged {
		if !combat.IsSelfManaged(s) {
			t.Errorf("%s should be self-managed", s)
		}
	}
	notSelfManaged := []tower.AttackStyle{tower.StyleProjectile, tower.StyleLaser, tower.StyleWideBeam, tower.StyleScatter, tower.StylePierce}
	for _, s := range notSelfManaged {
		if combat.IsSelfManaged(s) {
			t.Errorf("%s should NOT be self-managed", s)
		}
	}
}

// ── Projectile Handler ──

func TestProjectileHandlerFires(t *testing.T) {
	tw := makeTower(tower.StyleProjectile, 10, 200, 1)
	e := makeEnemy(150, 100, 50)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ePool := enemy.NewPool(4)
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleProjectile)
	h.Fire(tw, e, ctx)

	if pool.Count != 1 {
		t.Errorf("expected 1 projectile, got %d", pool.Count)
	}
}

// ── Laser Handler ──

func TestLaserInstantDamage(t *testing.T) {
	tw := makeTower(tower.StyleLaser, 25, 200, 1)
	e := makeEnemy(150, 100, 100)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ePool := enemy.NewPool(4)
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleLaser)
	h.Fire(tw, e, ctx)

	// Laser does instant damage
	if e.HP != 75 {
		t.Errorf("expected HP=75, got %f", e.HP)
	}
	// Should create a beam visual
	if beams.Count() != 1 {
		t.Errorf("expected 1 beam, got %d", beams.Count())
	}
	// No projectile created
	if pool.Count != 0 {
		t.Errorf("laser should not create projectiles, got %d", pool.Count)
	}
}

// ── WideBeam Handler ──

func TestWideBeamHitsMultiple(t *testing.T) {
	tw := makeTower(tower.StyleWideBeam, 20, 200, 1)
	ePool := enemy.NewPool(8)
	// Place 3 enemies in a line from tower
	e1 := ePool.Spawn(150, 100, 100, 50, 1, "normal", nil)
	e2 := ePool.Spawn(200, 100, 100, 50, 1, "normal", nil)
	e3 := ePool.Spawn(100, 200, 100, 50, 1, "normal", nil) // off-axis

	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleWideBeam)
	h.Fire(tw, e1, ctx)

	// e1 and e2 should be hit (in the beam line), e3 should not
	if e1.HP >= 100 {
		t.Error("e1 should be damaged")
	}
	if e2.HP >= 100 {
		t.Error("e2 should be damaged (in beam path)")
	}
	if e3.HP < 100 {
		t.Error("e3 should NOT be damaged (off axis)")
	}
}

// ── Scatter Handler ──

func TestScatterCreatesVisualProjectiles(t *testing.T) {
	tw := makeTower(tower.StyleScatter, 20, 150, 1)
	tw.ScatterPellets = 5
	e := makeEnemy(200, 100, 100)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ePool := enemy.NewPool(4)
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleScatter)
	h.Fire(tw, e, ctx)

	// Should create visual projectiles
	if pool.Count != 5 {
		t.Errorf("expected 5 scatter pellets, got %d", pool.Count)
	}
}

// ── Charge Handler ──

func TestChargeAccumulation(t *testing.T) {
	tw := makeTower(tower.StyleCharge, 30, 200, 1)
	tw.AttackSpeed = 0.4 // 2.5 sec to charge
	ePool := enemy.NewPool(4)
	ePool.Spawn(150, 100, 500, 50, 1, "normal", nil)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleCharge).(combat.TickHandler)

	// Tick for 2 seconds (should not fire yet with attackSpeed=0.4 → chargeTime=2.5s)
	for i := 0; i < 120; i++ {
		ctx.DT = 1.0 / 60
		h.Tick(tw, ctx)
	}
	if tw.ChargeReady {
		t.Error("should not be ready after 2s (needs 2.5s)")
	}

	// Tick another 1 second (total 3s > 2.5s)
	for i := 0; i < 60; i++ {
		ctx.DT = 1.0 / 60
		h.Tick(tw, ctx)
	}

	// Should have fired at some point (pool has a projectile)
	if pool.Count == 0 {
		t.Error("should have fired a charge projectile")
	}
}

// ── SpinAoE Handler ──

func TestSpinAoEDamagesAllInRange(t *testing.T) {
	tw := makeTower(tower.StyleSpinAoE, 10, 100, 2)
	ePool := enemy.NewPool(8)
	e1 := ePool.Spawn(150, 100, 100, 50, 1, "normal", nil) // 50px away (in range)
	e2 := ePool.Spawn(170, 100, 100, 50, 1, "normal", nil) // 70px away (in range)
	e3 := ePool.Spawn(300, 100, 100, 50, 1, "normal", nil) // 200px away (out of range)

	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ctx := makeCtx(ePool, pool, beams)
	ctx.DT = 1.0

	h := combat.Get(tower.StyleSpinAoE).(combat.TickHandler)
	h.Tick(tw, ctx)

	if e1.HP >= 100 {
		t.Error("e1 should be damaged (in range)")
	}
	if e2.HP >= 100 {
		t.Error("e2 should be damaged (in range)")
	}
	if e3.HP < 100 {
		t.Error("e3 should NOT be damaged (out of range)")
	}
}

// ── Pierce Projectile ──

func TestPierceProjectileCreated(t *testing.T) {
	tw := makeTower(tower.StylePierce, 20, 200, 1)
	e := makeEnemy(150, 100, 100)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ePool := enemy.NewPool(4)
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StylePierce)
	h.Fire(tw, e, ctx)

	if pool.Count != 1 {
		t.Errorf("expected 1 pierce projectile, got %d", pool.Count)
	}
	// Verify pierce flag
	pool.Each(func(p *projectile.Projectile) {
		if !p.Pierce {
			t.Error("projectile should have Pierce=true")
		}
		if p.PierceMax != 2 {
			t.Errorf("PierceMax=%d, want 2", p.PierceMax)
		}
	})
}

// ── AuraDot Handler ──

func TestAuraDotDamagesInRange(t *testing.T) {
	tw := makeTower(tower.StyleAuraDot, 5, 80, 1)
	ePool := enemy.NewPool(4)
	e1 := ePool.Spawn(150, 100, 100, 50, 1, "normal", nil) // 50px
	e2 := ePool.Spawn(300, 100, 100, 50, 1, "normal", nil) // 200px (out)

	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ctx := makeCtx(ePool, pool, beams)
	ctx.DT = 2.0 // enough to clear cooldown

	h := combat.Get(tower.StyleAuraDot).(combat.TickHandler)
	h.Tick(tw, ctx)

	if e1.HP >= 100 {
		t.Error("e1 should be damaged (in range)")
	}
	if e2.HP < 100 {
		t.Error("e2 should NOT be damaged (out of range)")
	}
}

// ── Beam Pool ──

func TestBeamPoolLifecycle(t *testing.T) {
	bp := combat.NewBeamPool()
	bp.Add(combat.Beam{X1: 0, Y1: 0, X2: 100, Y2: 0, Life: 0.1, MaxLife: 0.1})
	bp.Add(combat.Beam{X1: 0, Y1: 0, X2: 100, Y2: 0, Life: 0.5, MaxLife: 0.5})

	if bp.Count() != 2 {
		t.Errorf("count=%d, want 2", bp.Count())
	}

	bp.Update(0.2) // first beam expires

	if bp.Count() != 1 {
		t.Errorf("count=%d after update, want 1", bp.Count())
	}
}

// ── Projectile Extensions ──

func TestScatterProjectileNoDamage(t *testing.T) {
	pool := projectile.NewPool(4)
	pool.FireScatter(100, 100, 0, 200, 400, "test")

	pool.Each(func(p *projectile.Projectile) {
		if !p.ScatterVisual {
			t.Error("scatter projectile should be visual-only")
		}
		if p.Damage != 0 {
			t.Errorf("scatter visual damage=%f, want 0", p.Damage)
		}
	})
}

func TestChargeProjectileFlag(t *testing.T) {
	pool := projectile.NewPool(4)
	e := &enemy.Enemy{X: 200, Y: 100, Active: true}
	pool.FireCharge(100, 100, 200, 100, 90, 600, e, "test")

	pool.Each(func(p *projectile.Projectile) {
		if !p.ChargeShot {
			t.Error("should be charge shot")
		}
		if p.Damage != 90 {
			t.Errorf("damage=%f, want 90", p.Damage)
		}
		if p.Radius != 8 {
			t.Errorf("radius=%f, want 8 (larger charge projectile)", p.Radius)
		}
	})
}

func TestPierceProjectileFlag(t *testing.T) {
	pool := projectile.NewPool(4)
	e := &enemy.Enemy{X: 200, Y: 100, Active: true}
	pool.FirePierce(100, 100, 200, 100, 30, 350, e, "test", 3, 0.7)

	pool.Each(func(p *projectile.Projectile) {
		if !p.Pierce {
			t.Error("should be pierce")
		}
		if p.PierceMax != 3 {
			t.Errorf("pierceMax=%d, want 3", p.PierceMax)
		}
		if p.PierceDecay != 0.7 {
			t.Errorf("pierceDecay=%f, want 0.7", p.PierceDecay)
		}
	})
}
