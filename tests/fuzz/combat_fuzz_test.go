// combat_fuzz_test.go — Fuzz tests for the combat damage pipeline and CC system.
// Verifies invariants: no panics, no negative HP, no negative final damage,
// killed-but-positive-HP contradiction.
package fuzz

import (
	"math"
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	_, _ = config.LoadBalance()
	_ = config.LoadBuffRules()
}

// FuzzProcessDamage exercises the 8-step damage pipeline with arbitrary inputs.
func FuzzProcessDamage(f *testing.F) {
	// Seed corpus — representative edge cases.
	f.Add(100.0, 1000.0, 1000.0, false, false, false)
	f.Add(0.0, 100.0, 100.0, false, false, false)
	f.Add(999999.0, 1.0, 1.0, true, false, false)
	f.Add(1.0, 1.0, 1.0, false, true, false)
	f.Add(50.0, 50.0, 100.0, false, false, true)
	f.Add(0.001, 0.001, 0.001, false, false, false)

	f.Fuzz(func(t *testing.T, rawDmg, hp, maxHP float64, isInvincible, isDamageImmune, isBoss bool) {
		// Skip non-finite inputs (fuzzer generates NaN/Inf).
		if math.IsNaN(rawDmg) || math.IsInf(rawDmg, 0) {
			return
		}
		if math.IsNaN(hp) || math.IsInf(hp, 0) {
			return
		}
		if math.IsNaN(maxHP) || math.IsInf(maxHP, 0) {
			return
		}

		// Clamp to reasonable game ranges.
		if rawDmg < 0 {
			rawDmg = 0
		}
		if hp < 0 {
			hp = 0
		}
		if maxHP < 1 {
			maxHP = 1
		}
		if hp > maxHP {
			hp = maxHP
		}

		pool := enemy.NewPool(4)
		e := pool.Spawn(100, 100, maxHP, 60, 1, "normal", nil)
		if e == nil {
			return
		}
		e.SpawnTimer = 0 // skip spawn animation
		e.HP = hp
		e.IsInvincible = isInvincible
		e.IsDamageImmune = isDamageImmune
		e.Boss = isBoss

		result := combat.ProcessDamage(combat.DamageInput{
			Target:     e,
			RawDamage:  rawDmg,
			DamageType: "physical",
		})

		// ── Invariants that must ALWAYS hold ──

		if e.HP < 0 {
			t.Errorf("HP went negative: %f (rawDmg=%f, hp=%f, maxHP=%f)", e.HP, rawDmg, hp, maxHP)
		}
		if result.FinalDamage < 0 {
			t.Errorf("FinalDamage negative: %f (rawDmg=%f)", result.FinalDamage, rawDmg)
		}
		if result.Killed && e.HP > 0 {
			t.Errorf("Killed but HP > 0: HP=%f (rawDmg=%f)", e.HP, rawDmg)
		}
		if result.Blocked && result.FinalDamage > 0 {
			t.Errorf("Blocked but FinalDamage > 0: %f", result.FinalDamage)
		}
		// Invincible/immune enemies should block (unless pure damage).
		if isInvincible && !result.Blocked {
			t.Errorf("Invincible enemy not blocked (rawDmg=%f)", rawDmg)
		}
		if isDamageImmune && !isInvincible && !result.Blocked {
			t.Errorf("DamageImmune enemy not blocked (rawDmg=%f)", rawDmg)
		}
	})
}

// FuzzApplyStun exercises the stun CC path with arbitrary duration/tenacity.
func FuzzApplyStun(f *testing.F) {
	f.Add(1.0, 0.0, false)  // normal stun
	f.Add(0.0, 0.0, false)  // zero duration
	f.Add(5.0, 1.0, false)  // full tenacity (should not stun)
	f.Add(2.0, 0.5, true)   // immune
	f.Add(0.1, 0.99, false) // near-full tenacity

	f.Fuzz(func(t *testing.T, duration, tenacity float64, immune bool) {
		if math.IsNaN(duration) || math.IsInf(duration, 0) {
			return
		}
		if math.IsNaN(tenacity) || math.IsInf(tenacity, 0) {
			return
		}
		if duration < 0 {
			duration = 0
		}
		if tenacity < 0 {
			tenacity = 0
		}
		if tenacity > 1 {
			tenacity = 1
		}

		pool := enemy.NewPool(4)
		e := pool.Spawn(100, 100, 100, 60, 1, "normal", nil)
		if e == nil {
			return
		}
		e.SpawnTimer = 0
		e.Tenacity = tenacity
		e.IsControlImmune = immune

		applied := combat.ApplyStun(e, duration, "fuzz")

		// Immune enemies must not be stunned.
		if immune && applied {
			t.Errorf("Stun applied to immune enemy (duration=%f, tenacity=%f)", duration, tenacity)
		}
		// Full tenacity (1.0) should result in zero effective duration → not applied.
		if tenacity >= 1.0 && applied {
			t.Errorf("Stun applied with tenacity >= 1.0 (duration=%f, tenacity=%f)", duration, tenacity)
		}
	})
}

// FuzzApplySlow exercises the slow CC path with arbitrary factor/duration/tenacity.
func FuzzApplySlow(f *testing.F) {
	f.Add(0.5, 2.0, 0.0, false)
	f.Add(0.0, 0.0, 0.0, false)
	f.Add(1.0, 5.0, 1.0, true)
	f.Add(0.3, 1.0, 0.5, false)
	f.Add(0.1, 0.1, 0.0, false)

	f.Fuzz(func(t *testing.T, factor, duration, tenacity float64, immune bool) {
		if math.IsNaN(factor) || math.IsInf(factor, 0) {
			return
		}
		if math.IsNaN(duration) || math.IsInf(duration, 0) {
			return
		}
		if math.IsNaN(tenacity) || math.IsInf(tenacity, 0) {
			return
		}
		if factor < 0 {
			factor = 0
		}
		if factor > 1 {
			factor = 1
		}
		if duration < 0 {
			duration = 0
		}
		if tenacity < 0 {
			tenacity = 0
		}
		if tenacity > 1 {
			tenacity = 1
		}

		pool := enemy.NewPool(4)
		e := pool.Spawn(100, 100, 100, 60, 1, "normal", nil)
		if e == nil {
			return
		}
		e.SpawnTimer = 0
		e.Tenacity = tenacity
		e.IsControlImmune = immune

		applied := combat.ApplySlow(e, factor, duration, "fuzz")

		// Immune enemies must not be slowed.
		if immune && applied {
			t.Errorf("Slow applied to immune enemy (factor=%f, duration=%f)", factor, duration)
		}
		// Full tenacity → zero effective duration → not applied.
		if tenacity >= 1.0 && applied {
			t.Errorf("Slow applied with tenacity >= 1.0 (factor=%f, duration=%f)", factor, duration)
		}
		// Speed must never go negative after slow.
		if e.Speed < 0 {
			t.Errorf("Speed negative after slow: %f", e.Speed)
		}
	})
}
