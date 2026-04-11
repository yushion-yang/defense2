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
		tower.StyleProjectile, tower.StyleWideBeam,
		tower.StyleScatter, tower.StyleSpinAoE, tower.StyleRadial,
	}
	for _, s := range styles {
		if combat.Get(s) == nil {
			t.Errorf("handler not registered: %s", s)
		}
	}
}

func TestSelfManagedStyles(t *testing.T) {
	selfManaged := []tower.AttackStyle{tower.StyleSpinAoE}
	for _, s := range selfManaged {
		if !combat.IsSelfManaged(s) {
			t.Errorf("%s should be self-managed", s)
		}
	}
	notSelfManaged := []tower.AttackStyle{tower.StyleProjectile, tower.StyleWideBeam, tower.StyleScatter}
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

// ── WideBeam Handler ──

func TestWideBeamHitsMultiple(t *testing.T) {
	tw := makeTower(tower.StyleWideBeam, 20, 200, 1)
	ePool := enemy.NewPool(8)
	// Place 3 enemies in a line from tower
	e1 := ePool.Spawn(150, 100, 100, 50, 1, "normal", nil)
	e2 := ePool.Spawn(200, 100, 100, 50, 1, "normal", nil)
	e3 := ePool.Spawn(100, 200, 100, 50, 1, "normal", nil) // off-axis
	// Clear spawn animation so enemies are targetable
	e1.SpawnTimer = 0
	e2.SpawnTimer = 0
	e3.SpawnTimer = 0

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
	e := makeEnemy(200, 100, 100)
	pool := projectile.NewPool(16)
	beams := combat.NewBeamPool()
	ePool := enemy.NewPool(4)
	ctx := makeCtx(ePool, pool, beams)

	h := combat.Get(tower.StyleScatter)
	h.Fire(tw, e, ctx)

	// scatter 默认 3 颗弹丸
	if pool.Count != 3 {
		t.Errorf("expected 3 scatter pellets, got %d", pool.Count)
	}
}

// ── SpinAoE Handler ──

func TestSpinAoEDamagesAllInRange(t *testing.T) {
	tw := makeTower(tower.StyleSpinAoE, 10, 100, 2)
	ePool := enemy.NewPool(8)
	e1 := ePool.Spawn(150, 100, 100, 50, 1, "normal", nil) // 50px away (in range)
	e2 := ePool.Spawn(170, 100, 100, 50, 1, "normal", nil) // 70px away (in range)
	e3 := ePool.Spawn(300, 100, 100, 50, 1, "normal", nil) // 200px away (out of range)
	// Clear spawn animation so enemies are targetable
	e1.SpawnTimer = 0
	e2.SpawnTimer = 0
	e3.SpawnTimer = 0

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

// ── Beam Pool ──

func TestBeamPoolLifecycle(t *testing.T) {
	bp := combat.NewBeamPool()
	bp.Add(combat.Beam{X1: 0, Y1: 0, X2: 100, Y2: 0, Life: 0.1, MaxLife: 0.1})
	bp.Add(combat.Beam{X1: 0, Y1: 0, X2: 100, Y2: 0, Life: 0.5, MaxLife: 0.5})

	if bp.Count() != 2 {
		t.Errorf("count=%d, want 2", bp.Count())
	}

	bp.Tick(0.2) // first beam expires

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

func TestPenetrateProjectileFlag(t *testing.T) {
	pool := projectile.NewPool(4)
	pool.FirePenetrate(100, 100, 200, 100, 30, 350, "test")

	pool.Each(func(p *projectile.Projectile) {
		if !p.Penetrate {
			t.Error("should be penetrate")
		}
		if p.Target != nil {
			t.Error("penetrate should have nil target (straight line)")
		}
	})
}
