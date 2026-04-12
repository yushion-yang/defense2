package regression_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
	"defense2/tests/regression/sim"
)

// ============================================================
// Enemy Ability Interactions in ApplyHit
// ============================================================

// makeEnemy creates a minimal enemy via the sim builder for direct ApplyHit testing.
func makeEnemy(hp, speed float64) *enemy.Enemy {
	s := sim.New().
		WithStraightPath(500).
		WithEnemy("normal", hp, speed).
		Build()
	return s.SpawnedEnemies[0]
}

// makeTower creates a minimal tower for ApplyHit input.
func makeTower(dmg float64) *tower.Tower {
	return &tower.Tower{
		X:           100,
		Y:           100,
		Damage:      dmg,
		Range:       200,
		AttackSpeed: 1.0,
		Key:         "test",
		InstanceKey: "test_0",
		Active:      true,
	}
}

// hitInput builds a HitInput with sensible defaults.
func hitInput(tw *tower.Tower, e *enemy.Enemy, dmg float64, style string) combat.HitInput {
	return combat.HitInput{
		Tower:      tw,
		Target:     e,
		BaseDamage: dmg,
		Style:      style,
	}
}

// ── Evasion ──

// BUG: Enemy with 100% evasion was still taking damage.
// Verify: EvasionChance=1.0 always dodges — HP unchanged, TotalDamage=0.
func TestRegression_Evasion_DodgesAll(t *testing.T) {
	e := makeEnemy(100, 100)
	e.EvasionChance = 1.0

	tw := makeTower(50)
	out := combat.ApplyHit(hitInput(tw, e, 50, "projectile"), nil)

	if out.TotalDamage != 0 {
		t.Fatalf("evasion should block all damage, got TotalDamage=%.1f", out.TotalDamage)
	}
	if e.HP != 100 {
		t.Fatalf("HP should be unchanged (100), got %.1f", e.HP)
	}
}

// BUG: AbilitySilenced did not disable evasion.
// Verify: EvasionChance=1.0 + AbilitySilenced=true => damage applies normally.
func TestRegression_Evasion_SilencedNoEvade(t *testing.T) {
	e := makeEnemy(100, 100)
	e.EvasionChance = 1.0
	e.AbilitySilenced = true

	tw := makeTower(50)
	out := combat.ApplyHit(hitInput(tw, e, 50, "projectile"), nil)

	if out.TotalDamage == 0 {
		t.Fatal("silenced enemy should not evade, but TotalDamage=0")
	}
	if e.HP >= 100 {
		t.Fatalf("silenced enemy should take damage, HP=%.1f", e.HP)
	}
}

// ── ArmorFlat ──

// BUG: ArmorFlat was not reducing damage.
// Verify: ArmorFlat=5, BaseDamage=10 => 5 damage dealt.
func TestRegression_ArmorFlat_ReducesDamage(t *testing.T) {
	e := makeEnemy(100, 100)
	e.ArmorFlat = 5

	tw := makeTower(10)
	out := combat.ApplyHit(hitInput(tw, e, 10, "projectile"), nil)

	if out.TotalDamage != 5 {
		t.Fatalf("ArmorFlat=5 with 10 dmg should deal 5, got %.1f", out.TotalDamage)
	}
	if e.HP != 95 {
		t.Fatalf("HP should be 95, got %.1f", e.HP)
	}
}

// BUG: ArmorFlat could reduce damage to 0 or negative.
// Verify: ArmorFlat=100, BaseDamage=10 => minimum 1 damage.
func TestRegression_ArmorFlat_MinimumOne(t *testing.T) {
	e := makeEnemy(100, 100)
	e.ArmorFlat = 100

	tw := makeTower(10)
	out := combat.ApplyHit(hitInput(tw, e, 10, "projectile"), nil)

	if out.TotalDamage != 1 {
		t.Fatalf("ArmorFlat minimum damage should be 1, got %.1f", out.TotalDamage)
	}
	if e.HP != 99 {
		t.Fatalf("HP should be 99 (100-1), got %.1f", e.HP)
	}
}

// ── ProjectileBlock ──

// BUG: ProjectileBlockChance was zeroing damage instead of just blocking propagation.
// Verify: Enemy with ProjectileBlockChance=1, style="bounce" => damage still applies,
// ProjectileBlocked=true (blocks propagation, not damage).
func TestRegression_ProjectileBlock_NormalDamage(t *testing.T) {
	e := makeEnemy(100, 100)
	e.ProjectileBlockChance = 1.0

	tw := makeTower(20)
	out := combat.ApplyHit(hitInput(tw, e, 20, "bounce"), nil)

	if out.TotalDamage == 0 {
		t.Fatal("projectile block should not zero damage, but TotalDamage=0")
	}
	if !out.ProjectileBlocked {
		t.Fatal("ProjectileBlocked should be true for bounce style")
	}
	if e.HP >= 100 {
		t.Fatalf("enemy should take damage, HP=%.1f", e.HP)
	}
}

// BUG: ProjectileBlockChance blocked non-projectile styles.
// Verify: style="splash" => ProjectileBlocked=false (splash is not blocked).
func TestRegression_ProjectileBlock_NoBlockForSplash(t *testing.T) {
	e := makeEnemy(100, 100)
	e.ProjectileBlockChance = 1.0

	tw := makeTower(20)
	out := combat.ApplyHit(hitInput(tw, e, 20, "splash"), nil)

	if out.ProjectileBlocked {
		t.Fatal("ProjectileBlocked should be false for splash style")
	}
}

// Verify: ShouldShieldBlock returns correct result for each attack style.
func TestRegression_ShouldShieldBlock_StyleSet(t *testing.T) {
	cases := []struct {
		name     string
		chance   float64
		silenced bool
		style    string
		want     bool
	}{
		{"bounce-blocked", 1.0, false, "bounce", true},
		{"scatter-blocked", 1.0, false, "scatter", true},
		{"radial-blocked", 1.0, false, "radial", true},
		{"fireball-blocked", 1.0, false, "fireball", true},
		{"splash-not-blocked", 1.0, false, "splash", false},
		{"projectile-not-blocked", 1.0, false, "projectile", false},
		{"empty-not-blocked", 1.0, false, "", false},
		{"silenced-no-block", 1.0, true, "bounce", false},
		{"no-chance-no-block", 0, false, "bounce", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEnemy(100, 100)
			e.ProjectileBlockChance = tc.chance
			e.AbilitySilenced = tc.silenced
			got := combat.ShouldShieldBlock(e, tc.style)
			if got != tc.want {
				t.Fatalf("ShouldShieldBlock(chance=%.1f, silenced=%v, style=%q) = %v, want %v",
					tc.chance, tc.silenced, tc.style, got, tc.want)
			}
		})
	}
}

// Verify: ProjectileBlocked=true for fireball style against shielded enemy.
func TestRegression_ProjectileBlock_FireballBlocked(t *testing.T) {
	e := makeEnemy(100, 100)
	e.ProjectileBlockChance = 1.0

	tw := makeTower(20)
	out := combat.ApplyHit(hitInput(tw, e, 20, "fireball"), nil)

	if !out.ProjectileBlocked {
		t.Fatal("ProjectileBlocked should be true for fireball style")
	}
	if out.TotalDamage == 0 {
		t.Fatal("projectile block should not zero damage for fireball")
	}
}

// ── DashOnHit ──

// BUG: DashOnHit trigger was not activating on hit.
// Verify: After hit, DashActiveT > 0.
func TestRegression_DashOnHit_Trigger(t *testing.T) {
	e := makeEnemy(100, 100)
	e.DashSpeedBoost = 0.5
	e.DashDuration = 2.0
	e.DashCooldown = 5.0
	e.DashCooldownT = 0 // not on cooldown

	tw := makeTower(10)
	combat.ApplyHit(hitInput(tw, e, 10, "projectile"), nil)

	if e.DashActiveT <= 0 {
		t.Fatalf("DashActiveT should be >0 after hit, got %.2f", e.DashActiveT)
	}
	if e.DashActiveT != 2.0 {
		t.Fatalf("DashActiveT should be 2.0 (DashDuration), got %.2f", e.DashActiveT)
	}
	if e.DashCooldownT != 5.0 {
		t.Fatalf("DashCooldownT should be 5.0 (DashCooldown), got %.2f", e.DashCooldownT)
	}
}

// ── PhaseShift (IsDamageImmune) ──

// BUG: IsDamageImmune was not blocking damage in ApplyDamage pipeline.
// Verify: IsDamageImmune=true => damage=0, blocked.
func TestRegression_PhaseShift_BlocksDamage(t *testing.T) {
	e := makeEnemy(100, 100)
	e.IsDamageImmune = true

	tw := makeTower(50)
	out := combat.ApplyHit(hitInput(tw, e, 50, "projectile"), nil)

	if out.TotalDamage != 0 {
		t.Fatalf("IsDamageImmune should block all damage, got TotalDamage=%.1f", out.TotalDamage)
	}
	if e.HP != 100 {
		t.Fatalf("HP should be unchanged (100), got %.1f", e.HP)
	}
}

// ── CritMultiplier ──

// BUG: CritBonus=1.0 (100% crit) did not apply CritMultiplier.
// Verify: damage = baseDmg * config.GlobalBalance().Combat.CritMultiplier.
func TestRegression_CritMultiplier(t *testing.T) {
	e := makeEnemy(1000, 100)

	tw := makeTower(100)
	tw.CritBonus = 1.0 // 100% crit rate (normally computed from BuffList by RecalcStats)

	out := combat.ApplyHit(hitInput(tw, e, 100, "projectile"), nil)

	critMult := config.GlobalBalance().Combat.CritMultiplier
	expectedDmg := 100.0 * critMult

	if !out.IsCrit {
		t.Fatal("CritBonus=1.0 should always crit, but IsCrit=false")
	}
	if out.TotalDamage != expectedDmg {
		t.Fatalf("crit damage should be %.1f (100 * %.1f), got %.1f",
			expectedDmg, critMult, out.TotalDamage)
	}
}
