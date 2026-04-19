package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── Rule 1: onTick + CC 必须有 cooldown ──────────────────────

func TestValidate_OnTickCC_NoCooldown_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_perma_freeze",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnTick,
			Conditions: nil, // 无 cooldown
			Selector:   descriptor.AllInRangeSelector{},
			Effects:    []descriptor.Effect{descriptor.StunEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onTick + stun without cooldown")
	}
}

func TestValidate_OnTickCC_WithCooldown_Passes(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_freeze_zone",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnTick,
			Conditions: []descriptor.Condition{descriptor.NewCooldownCondition(1.0)},
			Selector:   descriptor.AllInRangeSelector{},
			Effects:    []descriptor.Effect{descriptor.StunEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// ── Rule 2: onHit + 空间查询选择器（除 aoeRadius/chain）──────

func TestValidate_OnHit_AllInRange_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_allinrange",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.AllInRangeSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onHit + allInRange selector")
	}
}

func TestValidate_OnHit_Cone_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_cone",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.ConeSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onHit + cone selector")
	}
}

func TestValidate_OnHit_CurrentTarget_Passes(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_current",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.CurrentTargetSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors for onHit + currentTarget, got %v", errs)
	}
}

func TestValidate_OnHit_AoeRadius_Passes(t *testing.T) {
	// aoeRadius 在 onHit 时被特殊处理（合成 Splash），所以允许
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_splash",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.AoeRadiusSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors for onHit + aoeRadius (synthesized to splash), got %v", errs)
	}
}

func TestValidate_OnHit_Chain_Passes(t *testing.T) {
	// chain 在 onHit 时被特殊处理（合成 Bounce），所以允许
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_bounce",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.ChainSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors for onHit + chain (synthesized to bounce), got %v", errs)
	}
}

// ── Rule 3: onHit + tick 专属效果 ────────────────────────────

func TestValidate_OnHit_BuffEffect_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_buff",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.CurrentTargetSelector{},
			Effects:    []descriptor.Effect{descriptor.BuffEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onHit + buff effect")
	}
}

func TestValidate_OnHit_GoldEffect_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onhit_gold",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnHit,
			Conditions: nil,
			Selector:   descriptor.CurrentTargetSelector{},
			Effects:    []descriptor.Effect{descriptor.GoldEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onHit + gold effect")
	}
}

// ── Rule 4: onTick + hit 专属效果 ────────────────────────────

func TestValidate_OnTick_CritEffect_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_ontick_crit",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnTick,
			Conditions: nil,
			Selector:   descriptor.AllInRangeSelector{},
			Effects:    []descriptor.Effect{descriptor.CritEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onTick + crit effect")
	}
}

// ── Rule 5: onKill + currentTarget ──────────────────────────

func TestValidate_OnKill_CurrentTarget_Fails(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onkill_current",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnKill,
			Conditions: nil,
			Selector:   descriptor.CurrentTargetSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Error("expected validation error for onKill + currentTarget selector")
	}
}

func TestValidate_OnKill_AoeRadius_Passes(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID: "test_onkill_aoe",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnKill,
			Conditions: nil,
			Selector:   descriptor.AoeRadiusSelector{},
			Effects:    []descriptor.Effect{descriptor.DamageEffect{}, descriptor.StunEffect{}},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors for onKill + aoeRadius, got %v", errs)
	}
}

// ── 组合场景 ────────────────────────────────────────────────

func TestValidate_ValidAbility_AllPasses(t *testing.T) {
	// 一个完整有效的能力：击杀核爆
	desc := &descriptor.AbilityDescriptor{
		ID: "test_kill_bomb",
		Pipelines: []descriptor.Pipeline{{
			Trigger:    descriptor.TriggerOnKill,
			Conditions: nil,
			Selector:   descriptor.AoeRadiusSelector{Radius: descriptor.FixedScaler{Value: 80}},
			Effects: []descriptor.Effect{
				descriptor.DamageEffect{Mode: descriptor.DmgRatio, Value: descriptor.FixedScaler{Value: 1.5}},
				descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.8}},
			},
		}},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid kill bomb ability, got %v", errs)
	}
}

func TestValidate_MultiPipeline_PartialErrors(t *testing.T) {
	// 多管线能力：第一个有效，第二个无效
	desc := &descriptor.AbilityDescriptor{
		ID: "test_multi_pipeline",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnHit,
				Conditions: nil,
				Selector:   descriptor.CurrentTargetSelector{},
				Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
			},
			{
				Trigger:    descriptor.TriggerOnHit,
				Conditions: nil,
				Selector:   descriptor.AllInRangeSelector{}, // 无效
				Effects:    []descriptor.Effect{descriptor.DamageEffect{}},
			},
		},
	}
	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 1 {
		t.Errorf("expected exactly 1 error for partially invalid multi-pipeline, got %d: %v", len(errs), errs)
	}
}
