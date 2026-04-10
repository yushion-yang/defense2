// buff_combat_test.go — Integration test for the full BuffList combat loop.
// Verifies that real buff-stack.json rules, enemy BuffList, and TickStatusEffects
// work together correctly through slow → bleed → stun → purge → controlImmune flow.
package integration_test

import (
	"math"
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/enemy"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	config.LoadBalance()
	config.LoadBuffRules()
}

// makeEnemy creates a minimal enemy suitable for combat testing.
// Uses enemy pool (same path as production) with BuffList initialised from global rules.
func makeEnemy() *enemy.Enemy {
	pool := enemy.NewPool(4)
	e := pool.Spawn(100, 100, 500, 60, 1, "normal", nil)
	e.SpawnTimer = 0 // skip spawn animation
	return e
}

// TestFullCombatLoop exercises the complete buff lifecycle:
// slow → bleed → stun → purge → controlImmune → expiry.
func TestFullCombatLoop(t *testing.T) {
	e := makeEnemy()
	if e == nil {
		t.Fatal("failed to spawn enemy from pool")
	}
	if e.Buffs == nil {
		t.Fatal("enemy should have a non-nil BuffList after spawn")
	}

	// ────────────────────────────────────
	// Step 1: Apply slow via BuffList
	// ────────────────────────────────────
	t.Run("slow", func(t *testing.T) {
		e.Buffs.Add(buff.Buff{
			ID: "slow", Category: buff.CatCC, Source: "tower_1",
			Value: 0.5, Duration: 3.0, Remaining: 3.0,
		})
		if !e.IsSlowed() {
			t.Fatal("enemy should be slowed after adding slow buff")
		}
		factor := e.GetSlowFactor()
		if math.Abs(factor-0.5) > 1e-9 {
			t.Fatalf("slow factor should be 0.5, got %.4f", factor)
		}

		// Tick to let TickStatusEffects sync speed
		enemy.TickStatusEffects(e, 0.016)
		expectedSpeed := e.BaseSpeed * 0.5
		if math.Abs(e.Speed-expectedSpeed) > 1e-6 {
			t.Fatalf("speed should be %.1f (BaseSpeed*0.5), got %.1f", expectedSpeed, e.Speed)
		}
	})

	// ────────────────────────────────────
	// Step 2: Apply bleed via BuffList
	// ────────────────────────────────────
	t.Run("bleed", func(t *testing.T) {
		e.Buffs.Add(buff.Buff{
			ID: "bleed", Category: buff.CatDoT, Source: "tower_2",
			Value: 20.0, // 20 DPS
			Duration: 5.0, Remaining: 5.0,
		})
		if !e.IsBleeding() {
			t.Fatal("enemy should be bleeding after adding bleed buff")
		}
	})

	// ────────────────────────────────────
	// Step 3: Tick DoT for ~1 second (2 ticks at 0.5s interval)
	// ────────────────────────────────────
	t.Run("dot_damage", func(t *testing.T) {
		hpBefore := e.HP
		dotInterval := enemy.DotTickInterval()
		if dotInterval <= 0 {
			t.Fatal("DotTickInterval should be > 0 (from balance.json)")
		}

		// Tick for 1 second in small steps to accumulate 2 DoT ticks
		steps := 20
		stepDt := 1.0 / float64(steps)
		for i := 0; i < steps; i++ {
			enemy.TickStatusEffects(e, stepDt)
		}

		// DoT ticks are stored in LastDotDmg per tick (pipeline reads it).
		// After 1 second with dotInterval=0.5, we expect 2 ticks.
		// Total damage: 2 * (20 DPS * 0.5s) = 20 HP.
		// But LastDotDmg is only the *last* tick's damage (cleared by pipeline each frame).
		// For integration, we verify HP decreased.
		hpLoss := hpBefore - e.HP
		// Note: HP doesn't actually decrease from DoT damage in TickStatusEffects alone —
		// the damage is reported via LastDotDmg and applied by the pipeline.
		// So instead, verify LastDotDmg was computed.
		if e.LastDotDmg <= 0 {
			t.Fatalf("LastDotDmg should be > 0 after DoT ticks, got %.4f", e.LastDotDmg)
		}
		// The HP hasn't changed because TickStatusEffects only reports damage,
		// it doesn't apply it. Verify consistent with production flow.
		if hpLoss != 0 {
			t.Logf("hp loss=%.2f (expected 0, pipeline applies damage)", hpLoss)
		}
	})

	// ────────────────────────────────────
	// Step 4: Apply stun
	// ────────────────────────────────────
	t.Run("stun", func(t *testing.T) {
		e.Buffs.Add(buff.Buff{
			ID: "stun", Category: buff.CatCC, Source: "tower_3",
			Value: 1.0, Duration: 2.0, Remaining: 2.0,
		})
		if !e.IsStunned() {
			t.Fatal("enemy should be stunned after adding stun buff")
		}
	})

	// ────────────────────────────────────
	// Step 5: Purge (ClearByCategory CC+DoT+Debuff) → verify all cleared
	// ────────────────────────────────────
	t.Run("purge", func(t *testing.T) {
		// Verify pre-purge state: should have slow + bleed + stun
		if !e.IsSlowed() {
			t.Error("pre-purge: should be slowed")
		}
		if !e.IsBleeding() {
			t.Error("pre-purge: should be bleeding")
		}
		if !e.IsStunned() {
			t.Error("pre-purge: should be stunned")
		}

		e.Buffs.ClearByCategory(buff.CatCC, buff.CatDoT, buff.CatDebuff)

		if e.IsSlowed() {
			t.Error("post-purge: should not be slowed (CC cleared)")
		}
		if e.IsBleeding() {
			t.Error("post-purge: should not be bleeding (DoT cleared)")
		}
		if e.IsStunned() {
			t.Error("post-purge: should not be stunned (CC cleared)")
		}
		if e.Buffs.Count() != 0 {
			t.Errorf("post-purge: buff count should be 0, got %d", e.Buffs.Count())
		}
	})

	// ────────────────────────────────────
	// Step 6: Apply controlImmune → verify HasControlImmunity()
	// ────────────────────────────────────
	t.Run("controlImmune", func(t *testing.T) {
		e.Buffs.Add(buff.Buff{
			ID: "controlImmune", Category: buff.CatDefense, Source: "boss_ability",
			Value: 1.0, Duration: 2.0, Remaining: 2.0,
		})
		if !e.HasControlImmunity() {
			t.Fatal("enemy should have control immunity after adding controlImmune buff")
		}
	})

	// ────────────────────────────────────
	// Step 7: Tick until controlImmune expires → verify cleared
	// ────────────────────────────────────
	t.Run("controlImmune_expiry", func(t *testing.T) {
		// Tick 2.5 seconds to ensure the 2.0s controlImmune expires
		steps := 50
		stepDt := 2.5 / float64(steps)
		for i := 0; i < steps; i++ {
			enemy.TickStatusEffects(e, stepDt)
		}

		if e.HasControlImmunity() {
			t.Fatal("controlImmune should have expired after 2.5s")
		}
		if e.Buffs.Count() != 0 {
			t.Errorf("all buffs should have expired, got %d remaining", e.Buffs.Count())
		}
	})
}

// TestSlowCapFromRealConfig verifies that the real buff-stack.json slow cap (0.8)
// is applied when using NewDefaultBuffList.
func TestSlowCapFromRealConfig(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("GlobalRules() is nil — LoadBuffRules() not called")
	}
	slowRule, ok := rules["slow"]
	if !ok {
		t.Fatal("buff-stack.json missing 'slow' rule")
	}
	if slowRule.Cap != 0.8 {
		t.Errorf("slow cap should be 0.8, got %.2f", slowRule.Cap)
	}

	// Apply a slow exceeding cap via DefaultBuffList
	bl := buff.NewDefaultBuffList()
	bl.Add(buff.Buff{ID: "slow", Source: "test", Value: 0.95, Duration: 5, Remaining: 5})
	b, ok := bl.Get("slow")
	if !ok {
		t.Fatal("slow buff should exist")
	}
	if math.Abs(b.Value-0.8) > 1e-9 {
		t.Errorf("slow value should be capped at 0.8, got %.4f", b.Value)
	}
}

// TestDotIndependentPerSource verifies that DoT buffs from different sources
// coexist (independentPerSource mode from real config).
func TestDotIndependentPerSource(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("GlobalRules() is nil")
	}
	dotRule, ok := rules["dot"]
	if !ok {
		t.Fatal("buff-stack.json missing 'dot' rule")
	}
	if dotRule.Mode != buff.IndependentPerSource {
		t.Errorf("dot mode should be IndependentPerSource, got %d", dotRule.Mode)
	}
}

// TestAllPhase1RulesPresent verifies that all Phase 1 buff IDs have
// rules defined in buff-stack.json.
func TestAllPhase1RulesPresent(t *testing.T) {
	rules := buff.GlobalRules()
	if rules == nil {
		t.Fatal("GlobalRules() is nil")
	}

	required := []string{
		"slow", "stun", "root",
		"bleed", "burn", "poison",
		"weaken", "controlImmune", "dot",
	}
	for _, id := range required {
		if _, ok := rules[id]; !ok {
			t.Errorf("buff-stack.json missing rule for Phase 1 buff: %q", id)
		}
	}
}
