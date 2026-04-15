package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── DamageEffect ─────────────────────────────────────────────

func TestDamageEffect_FlatMode(t *testing.T) {
	eff := descriptor.DamageEffect{
		Mode:  descriptor.DmgFlat,
		Value: descriptor.FixedScaler{Value: 10},
	}
	ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 50, TargetMaxHp: 1000}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeDamage {
		t.Errorf("type = %v, want EffTypeDamage", r.Type)
	}
	if r.DamageMode != descriptor.DmgFlat {
		t.Errorf("damageMode = %v, want DmgFlat", r.DamageMode)
	}
	if r.Damage != 10 {
		t.Errorf("damage = %v, want 10", r.Damage)
	}
}

func TestDamageEffect_RatioMode(t *testing.T) {
	eff := descriptor.DamageEffect{
		Mode:  descriptor.DmgRatio,
		Value: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 50, TargetMaxHp: 1000}
	r := eff.Apply(ctx)

	if r.Damage != 25 {
		t.Errorf("damage = %v, want 25 (0.5 * 50)", r.Damage)
	}
	if r.DamageMode != descriptor.DmgRatio {
		t.Errorf("damageMode = %v, want DmgRatio", r.DamageMode)
	}
}

func TestDamageEffect_HpPercentMode(t *testing.T) {
	eff := descriptor.DamageEffect{
		Mode:  descriptor.DmgHpPercent,
		Value: descriptor.FixedScaler{Value: 0.1},
	}
	ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 50, TargetMaxHp: 2000}
	r := eff.Apply(ctx)

	if r.Damage != 200 {
		t.Errorf("damage = %v, want 200 (0.1 * 2000)", r.Damage)
	}
	if r.DamageMode != descriptor.DmgHpPercent {
		t.Errorf("damageMode = %v, want DmgHpPercent", r.DamageMode)
	}
}

func TestDamageEffect_LinearScaler(t *testing.T) {
	// base=5, potential=5, str=200 → value = 5 + 5*(200/100) = 15
	eff := descriptor.DamageEffect{
		Mode:  descriptor.DmgFlat,
		Value: descriptor.LinearScaler{Base: 5, Potential: 5},
	}
	ctx := descriptor.EffectCtx{Strength: 200}
	r := eff.Apply(ctx)

	if r.Damage != 15 {
		t.Errorf("damage = %v, want 15", r.Damage)
	}
}

// ── SlowEffect ───────────────────────────────────────────────

func TestSlowEffect_LinearScaler(t *testing.T) {
	// factor: base=0.3, potential=0.05, str=200 → 0.3 + 0.05*(200/100) = 0.4
	eff := descriptor.SlowEffect{
		Factor:   descriptor.LinearScaler{Base: 0.3, Potential: 0.05},
		Duration: descriptor.FixedScaler{Value: 2.0},
	}
	ctx := descriptor.EffectCtx{Strength: 200}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeSlow {
		t.Errorf("type = %v, want EffTypeSlow", r.Type)
	}
	if r.SlowFactor != 0.4 {
		t.Errorf("slowFactor = %v, want 0.4", r.SlowFactor)
	}
	if r.Duration != 2.0 {
		t.Errorf("duration = %v, want 2.0", r.Duration)
	}
}

// ── StunEffect ───────────────────────────────────────────────

func TestStunEffect_Fixed(t *testing.T) {
	eff := descriptor.StunEffect{
		Duration: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeStun {
		t.Errorf("type = %v, want EffTypeStun", r.Type)
	}
	if r.StunDur != 0.5 {
		t.Errorf("stunDur = %v, want 0.5", r.StunDur)
	}
}

// ── RootEffect ───────────────────────────────────────────────

func TestRootEffect(t *testing.T) {
	eff := descriptor.RootEffect{
		Duration: descriptor.LinearScaler{Base: 1.0, Potential: 0.5},
	}
	ctx := descriptor.EffectCtx{Strength: 200}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeRoot {
		t.Errorf("type = %v, want EffTypeRoot", r.Type)
	}
	// 1.0 + 0.5*(200/100) = 2.0
	if r.RootDur != 2.0 {
		t.Errorf("rootDur = %v, want 2.0", r.RootDur)
	}
}

// ── DotEffect ────────────────────────────────────────────────

func TestDotEffect_BurnRatio(t *testing.T) {
	eff := descriptor.DotEffect{
		Subtype:  "burn",
		Mode:     descriptor.DmgRatio,
		Value:    descriptor.FixedScaler{Value: 0.2},
		Duration: descriptor.FixedScaler{Value: 3.0},
	}
	ctx := descriptor.EffectCtx{Strength: 100, TowerDamage: 80}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeDot {
		t.Errorf("type = %v, want EffTypeDot", r.Type)
	}
	if r.DotSubtype != "burn" {
		t.Errorf("dotSubtype = %q, want \"burn\"", r.DotSubtype)
	}
	if r.DotMode != descriptor.DmgRatio {
		t.Errorf("dotMode = %v, want DmgRatio", r.DotMode)
	}
	// 0.2 * 80 = 16
	if r.DotValue != 16 {
		t.Errorf("dotValue = %v, want 16", r.DotValue)
	}
	if r.DotDuration != 3.0 {
		t.Errorf("dotDuration = %v, want 3.0", r.DotDuration)
	}
}

func TestDotEffect_BleedFlat(t *testing.T) {
	eff := descriptor.DotEffect{
		Subtype:  "bleed",
		Mode:     descriptor.DmgFlat,
		Value:    descriptor.LinearScaler{Base: 5, Potential: 2},
		Duration: descriptor.LinearScaler{Base: 2, Potential: 1},
	}
	// str=150 → value = 5 + 2*(150/100) = 8, duration = 2 + 1*(150/100) = 3.5
	ctx := descriptor.EffectCtx{Strength: 150}
	r := eff.Apply(ctx)

	if r.DotSubtype != "bleed" {
		t.Errorf("dotSubtype = %q, want \"bleed\"", r.DotSubtype)
	}
	if r.DotValue != 8 {
		t.Errorf("dotValue = %v, want 8", r.DotValue)
	}
	if r.DotDuration != 3.5 {
		t.Errorf("dotDuration = %v, want 3.5", r.DotDuration)
	}
}

// ── WeakenEffect ─────────────────────────────────────────────

func TestWeakenEffect(t *testing.T) {
	eff := descriptor.WeakenEffect{
		Amplify:  descriptor.FixedScaler{Value: 0.25},
		Duration: descriptor.FixedScaler{Value: 4.0},
	}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeWeaken {
		t.Errorf("type = %v, want EffTypeWeaken", r.Type)
	}
	if r.WeakenAmp != 0.25 {
		t.Errorf("weakenAmp = %v, want 0.25", r.WeakenAmp)
	}
	if r.WeakenDur != 4.0 {
		t.Errorf("weakenDur = %v, want 4.0", r.WeakenDur)
	}
}

// ── SilenceEffect ────────────────────────────────────────────

func TestSilenceEffect(t *testing.T) {
	eff := descriptor.SilenceEffect{}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeSilence {
		t.Errorf("type = %v, want EffTypeSilence", r.Type)
	}
}

// ── BuffEffect ───────────────────────────────────────────────

func TestBuffEffect(t *testing.T) {
	eff := descriptor.BuffEffect{
		Stat:  "damage",
		Bonus: descriptor.LinearScaler{Base: 10, Potential: 5},
	}
	// str=300 → bonus = 10 + 5*(300/100) = 25
	ctx := descriptor.EffectCtx{Strength: 300}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeBuff {
		t.Errorf("type = %v, want EffTypeBuff", r.Type)
	}
	if r.BuffStat != "damage" {
		t.Errorf("buffStat = %q, want \"damage\"", r.BuffStat)
	}
	if r.BuffBonus != 25 {
		t.Errorf("buffBonus = %v, want 25", r.BuffBonus)
	}
}

// ── SelfBuffEffect ───────────────────────────────────────────

func TestSelfBuffEffect(t *testing.T) {
	eff := descriptor.SelfBuffEffect{
		Stat:  "speed",
		Bonus: descriptor.FixedScaler{Value: 0.15},
	}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeSelfBuff {
		t.Errorf("type = %v, want EffTypeSelfBuff", r.Type)
	}
	if r.BuffStat != "speed" {
		t.Errorf("buffStat = %q, want \"speed\"", r.BuffStat)
	}
	if r.BuffBonus != 0.15 {
		t.Errorf("buffBonus = %v, want 0.15", r.BuffBonus)
	}
}

// ── GoldEffect ───────────────────────────────────────────────

func TestGoldEffect_LinearScaler(t *testing.T) {
	eff := descriptor.GoldEffect{
		Amount: descriptor.LinearScaler{Base: 2, Potential: 1},
	}
	// str=300 → amount = 2 + 1*(300/100) = 5
	ctx := descriptor.EffectCtx{Strength: 300}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeGold {
		t.Errorf("type = %v, want EffTypeGold", r.Type)
	}
	if r.GoldAmount != 5 {
		t.Errorf("goldAmount = %v, want 5", r.GoldAmount)
	}
}

// ── ModifyStatEffect ─────────────────────────────────────────

func TestModifyStatEffect(t *testing.T) {
	eff := descriptor.ModifyStatEffect{
		Stat:       "range",
		Multiplier: 1.5,
	}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeModifyStat {
		t.Errorf("type = %v, want EffTypeModifyStat", r.Type)
	}
	if r.BuffStat != "range" {
		t.Errorf("buffStat = %q, want \"range\"", r.BuffStat)
	}
	if r.StatMult != 1.5 {
		t.Errorf("statMult = %v, want 1.5", r.StatMult)
	}
}

// ── Effect 接口契约 ──────────────────────────────────────────

// 编译期验证所有类型实现 Effect 接口。
var _ descriptor.Effect = descriptor.DamageEffect{}
var _ descriptor.Effect = descriptor.SlowEffect{}
var _ descriptor.Effect = descriptor.StunEffect{}
var _ descriptor.Effect = descriptor.RootEffect{}
var _ descriptor.Effect = descriptor.DotEffect{}
var _ descriptor.Effect = descriptor.WeakenEffect{}
var _ descriptor.Effect = descriptor.SilenceEffect{}
var _ descriptor.Effect = descriptor.BuffEffect{}
var _ descriptor.Effect = descriptor.SelfBuffEffect{}
var _ descriptor.Effect = descriptor.GoldEffect{}
var _ descriptor.Effect = descriptor.ModifyStatEffect{}
