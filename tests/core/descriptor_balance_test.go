// descriptor_balance_test.go — 自定义能力平衡约束测试。
//
// 覆盖：
//   - ValidateCustomAbility: onTick+CC 必须有 cooldown
//   - CalcAbilityCost: 单管线无加成、多管线有加成
//   - CC 效果费用提升验证
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── ValidateCustomAbility 测试 ──────────────────────────────

func TestValidateCustomAbility_OnTickStunRequiresCooldown(t *testing.T) {
	// onTick + allInRange + stun 无 cooldown → 应报错
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_op",
		Label: "OP Stun",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnTick,
				Conditions: nil, // 无 cooldown
				Selector:   descriptor.AllInRangeSelector{},
				Effects:    []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.5}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Fatal("expected validation error for onTick+stun without cooldown, got none")
	}
	if errs[0].Field != "pipeline[0]" {
		t.Errorf("error field: got %q, want %q", errs[0].Field, "pipeline[0]")
	}
}

func TestValidateCustomAbility_OnTickStunWithCooldown(t *testing.T) {
	// onTick + cooldown(0.5) + allInRange + stun → 应通过
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_ok",
		Label: "Balanced Stun",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnTick,
				Conditions: []descriptor.Condition{descriptor.NewCooldownCondition(0.5)},
				Selector:   descriptor.AllInRangeSelector{},
				Effects:    []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.5}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidateCustomAbility_OnHitStunNoCooldownOK(t *testing.T) {
	// onHit + stun 无 cooldown → 应通过（非持续触发）
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_onhit",
		Label: "OnHit Stun",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnHit,
				Conditions: nil,
				Selector:   descriptor.CurrentTargetSelector{},
				Effects:    []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 1.0}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for onHit+stun, got %v", errs)
	}
}

func TestValidateCustomAbility_OnTickRootRequiresCooldown(t *testing.T) {
	// onTick + root 无 cooldown → 应报错
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_root",
		Label: "OP Root",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger: descriptor.TriggerOnTick,
				Selector: descriptor.AllInRangeSelector{},
				Effects: []descriptor.Effect{descriptor.RootEffect{Duration: descriptor.FixedScaler{Value: 1.0}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Fatal("expected validation error for onTick+root without cooldown")
	}
}

func TestValidateCustomAbility_OnTickSlowRequiresCooldown(t *testing.T) {
	// onTick + slow 无 cooldown → 应报错
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_slow",
		Label: "OP Slow",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.CurrentTargetSelector{},
				Effects: []descriptor.Effect{
					descriptor.SlowEffect{
						Factor:   descriptor.FixedScaler{Value: 0.5},
						Duration: descriptor.FixedScaler{Value: 1.0},
					},
				},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Fatal("expected validation error for onTick+slow without cooldown")
	}
}

func TestValidateCustomAbility_OnTickSilenceRequiresCooldown(t *testing.T) {
	// onTick + silence 无 cooldown → 应报错
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_silence",
		Label: "OP Silence",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.AllInRangeSelector{},
				Effects:  []descriptor.Effect{descriptor.SilenceEffect{}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) == 0 {
		t.Fatal("expected validation error for onTick+silence without cooldown")
	}
}

func TestValidateCustomAbility_OnTickDamageNoCooldownOK(t *testing.T) {
	// onTick + damage（非 CC）无 cooldown → 应通过
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_dmg",
		Label: "Tick Damage",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.DamageEffect{Value: descriptor.FixedScaler{Value: 10}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for onTick+damage, got %v", errs)
	}
}

func TestValidateCustomAbility_MultiplePipelines(t *testing.T) {
	// 2 条管线：第一条合法，第二条 onTick+stun 无 cooldown → 应报 1 个错误
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_multi",
		Label: "Multi Pipeline",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnHit,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.DamageEffect{Value: descriptor.FixedScaler{Value: 5}}},
			},
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.AllInRangeSelector{},
				Effects:  []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.3}}},
			},
		},
	}

	errs := descriptor.ValidateCustomAbility(desc)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "pipeline[1]" {
		t.Errorf("error field: got %q, want %q", errs[0].Field, "pipeline[1]")
	}
}

func TestValidateCustomAbility_NilDescriptor(t *testing.T) {
	errs := descriptor.ValidateCustomAbility(nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for nil descriptor, got %v", errs)
	}
}

// ── CalcAbilityCost 测试 ────────────────────────────────────

func TestCalcAbilityCost_SinglePipeline(t *testing.T) {
	// 单管线：onHit(0) + cooldown(1) + currentTarget(0) + stun(6) = 7，无加成
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_single",
		Label: "Single",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnHit,
				Conditions: []descriptor.Condition{descriptor.NewCooldownCondition(2.0)},
				Selector:   descriptor.CurrentTargetSelector{},
				Effects:    []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.5}}},
			},
		},
	}

	cost := descriptor.CalcAbilityCost(desc)

	// onHit=0, cooldown=1, currentTarget=0, stun=6 → total=7
	if cost != 7 {
		t.Errorf("CalcAbilityCost single pipeline: got %d, want 7", cost)
	}
}

func TestCalcAbilityCost_MultiPipeline_Two(t *testing.T) {
	// 2 管线：基础费用 = (0+0+3) + (0+0+3) = 6，乘数 = 1.2 → ceil(7.2) = 8
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_two",
		Label: "Two Pipes",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnHit,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.DamageEffect{Value: descriptor.FixedScaler{Value: 10}}},
			},
			{
				Trigger:  descriptor.TriggerOnKill,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.GoldEffect{Amount: descriptor.FixedScaler{Value: 2}}},
			},
		},
	}

	cost := descriptor.CalcAbilityCost(desc)

	// pipe1: onHit(0) + currentTarget(0) + damage(3) = 3
	// pipe2: onKill(0) + currentTarget(0) + gold(2) = 2
	// base = 5, multiplier = 1.2 → ceil(6.0) = 6
	if cost != 6 {
		t.Errorf("CalcAbilityCost 2 pipelines: got %d, want 6", cost)
	}
}

func TestCalcAbilityCost_MultiPipeline_Three(t *testing.T) {
	// 3 管线：乘数 = 1.0 + 0.2*2 = 1.4
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_three",
		Label: "Three Pipes",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnHit,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.DamageEffect{Value: descriptor.FixedScaler{Value: 10}}},
			},
			{
				Trigger:  descriptor.TriggerOnKill,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.GoldEffect{Amount: descriptor.FixedScaler{Value: 2}}},
			},
			{
				Trigger:    descriptor.TriggerOnTick,
				Conditions: []descriptor.Condition{descriptor.NewCooldownCondition(3.0)},
				Selector:   descriptor.AllInRangeSelector{},
				Effects:    []descriptor.Effect{descriptor.SlowEffect{Factor: descriptor.FixedScaler{Value: 0.3}, Duration: descriptor.FixedScaler{Value: 1}}},
			},
		},
	}

	cost := descriptor.CalcAbilityCost(desc)

	// pipe1: onHit(0) + currentTarget(0) + damage(3) = 3
	// pipe2: onKill(0) + currentTarget(0) + gold(2) = 2
	// pipe3: onTick(0) + cooldown(1) + allInRange(2) + slow(3) = 6
	// base = 11, multiplier = 1.4 → ceil(15.4) = 16
	if cost != 16 {
		t.Errorf("CalcAbilityCost 3 pipelines: got %d, want 16", cost)
	}
}

func TestCalcAbilityCost_NilDescriptor(t *testing.T) {
	cost := descriptor.CalcAbilityCost(nil)
	if cost != 0 {
		t.Errorf("CalcAbilityCost nil: got %d, want 0", cost)
	}
}

func TestCalcAbilityCost_EmptyPipelines(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "test_empty",
		Label: "Empty",
	}
	cost := descriptor.CalcAbilityCost(desc)
	if cost != 0 {
		t.Errorf("CalcAbilityCost empty: got %d, want 0", cost)
	}
}

// ── CC 效果费用提升验证 ────────────────────────────────────

func TestCCEffectCostsIncreased(t *testing.T) {
	effects := descriptor.AllEffectMeta()
	costMap := map[string]int{}
	for _, m := range effects {
		costMap[m.ID] = m.Cost
	}

	tests := []struct {
		id   string
		want int
	}{
		{"stun", 6},
		{"silence", 6},
		{"root", 5},
		// 非 CC 效果应保持不变
		{"slow", 3},
		{"damage", 3},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := costMap[tt.id]
			if got != tt.want {
				t.Errorf("effect %q cost: got %d, want %d", tt.id, got, tt.want)
			}
		})
	}
}
