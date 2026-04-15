// descriptor_condition_test.go — Condition 条件门测试。
//
// 验证 6 种条件类型的求值逻辑，以及 EvalAll AND 组合器。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// ChanceCondition 测试
// ============================================================

func TestChanceCondition_AlwaysPassesAtRate1(t *testing.T) {
	cond := descriptor.ChanceCondition{Rate: descriptor.FixedScaler{Value: 1.0}}
	ctx := descriptor.ConditionCtx{Strength: 100}
	// rate=1.0 应总是通过，连续测试多次确认
	for i := 0; i < 100; i++ {
		if !cond.Eval(ctx) {
			t.Fatal("rate=1.0 不应失败")
		}
	}
}

func TestChanceCondition_NeverPassesAtRate0(t *testing.T) {
	cond := descriptor.ChanceCondition{Rate: descriptor.FixedScaler{Value: 0.0}}
	ctx := descriptor.ConditionCtx{Strength: 100}
	for i := 0; i < 100; i++ {
		if cond.Eval(ctx) {
			t.Fatal("rate=0.0 不应通过")
		}
	}
}

func TestChanceCondition_ScalesWithStrength(t *testing.T) {
	// base=0.0, potential=100 → str=50 时 rate=0.5
	cond := descriptor.ChanceCondition{
		Rate: descriptor.LinearScaler{Base: 0, Potential: 100},
	}
	// str=0 → rate=0, 不应通过
	ctx := descriptor.ConditionCtx{Strength: 0}
	for i := 0; i < 50; i++ {
		if cond.Eval(ctx) {
			t.Fatal("rate=0.0(str=0) 不应通过")
		}
	}
	// str=100 → rate=1.0, 应总是通过
	ctx.Strength = 100
	for i := 0; i < 50; i++ {
		if !cond.Eval(ctx) {
			t.Fatal("rate=1.0(str=100) 不应失败")
		}
	}
}

// ============================================================
// CooldownCondition 测试
// ============================================================

func TestCooldownCondition_FirstCallPasses(t *testing.T) {
	cond := descriptor.NewCooldownCondition(2.0) // 2 秒冷却
	ctx := descriptor.ConditionCtx{Elapsed: 0}
	if !cond.Eval(ctx) {
		t.Fatal("首次调用应通过（从未触发过）")
	}
}

func TestCooldownCondition_ImmediateSecondFails(t *testing.T) {
	cond := descriptor.NewCooldownCondition(2.0)
	ctx := descriptor.ConditionCtx{Elapsed: 0}
	cond.Eval(ctx) // 首次触发
	if cond.Eval(ctx) {
		t.Fatal("立即再次调用应失败（冷却中）")
	}
}

func TestCooldownCondition_PassesAfterCooldown(t *testing.T) {
	cond := descriptor.NewCooldownCondition(2.0)
	ctx := descriptor.ConditionCtx{Elapsed: 0}
	cond.Eval(ctx) // 首次触发，记录时间

	// 冷却未到
	ctx.Elapsed = 1.5
	if cond.Eval(ctx) {
		t.Fatal("冷却未到(1.5s < 2.0s)应失败")
	}

	// 冷却刚好到
	ctx.Elapsed = 2.0
	if !cond.Eval(ctx) {
		t.Fatal("冷却已到(2.0s >= 2.0s)应通过")
	}
}

func TestCooldownCondition_AccumulatesElapsed(t *testing.T) {
	cond := descriptor.NewCooldownCondition(1.0)
	ctx := descriptor.ConditionCtx{Elapsed: 0}
	cond.Eval(ctx) // 首次触发

	// 每次 Eval 都累计 elapsed
	ctx.Elapsed = 0.4
	cond.Eval(ctx) // 累计 0.4，不通过

	ctx.Elapsed = 0.4
	cond.Eval(ctx) // 累计 0.8，不通过

	ctx.Elapsed = 0.3
	if !cond.Eval(ctx) {
		t.Fatal("累计 1.1s >= 1.0s 应通过")
	}
}

// ============================================================
// HpBelowCondition 测试
// ============================================================

func TestHpBelowCondition_BelowThreshold(t *testing.T) {
	cond := descriptor.HpBelowCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.3}
	if !cond.Eval(ctx) {
		t.Error("0.3 < 0.5 应通过")
	}
}

func TestHpBelowCondition_AboveThreshold(t *testing.T) {
	cond := descriptor.HpBelowCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.7}
	if cond.Eval(ctx) {
		t.Error("0.7 > 0.5 不应通过")
	}
}

func TestHpBelowCondition_EqualThreshold(t *testing.T) {
	cond := descriptor.HpBelowCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.5}
	if cond.Eval(ctx) {
		t.Error("0.5 == 0.5 不应通过（严格小于）")
	}
}

// ============================================================
// HpAboveCondition 测试
// ============================================================

func TestHpAboveCondition_AboveThreshold(t *testing.T) {
	cond := descriptor.HpAboveCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.7}
	if !cond.Eval(ctx) {
		t.Error("0.7 > 0.5 应通过")
	}
}

func TestHpAboveCondition_BelowThreshold(t *testing.T) {
	cond := descriptor.HpAboveCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.3}
	if cond.Eval(ctx) {
		t.Error("0.3 < 0.5 不应通过")
	}
}

func TestHpAboveCondition_EqualThreshold(t *testing.T) {
	cond := descriptor.HpAboveCondition{
		Threshold: descriptor.FixedScaler{Value: 0.5},
	}
	ctx := descriptor.ConditionCtx{Strength: 100, TargetHpRatio: 0.5}
	if cond.Eval(ctx) {
		t.Error("0.5 == 0.5 不应通过（严格大于）")
	}
}

// ============================================================
// DistanceMinCondition 测试
// ============================================================

func TestDistanceMinCondition_AboveDistance(t *testing.T) {
	cond := descriptor.DistanceMinCondition{Distance: 150}
	ctx := descriptor.ConditionCtx{TargetDistance: 200}
	if !cond.Eval(ctx) {
		t.Error("200 >= 150 应通过")
	}
}

func TestDistanceMinCondition_BelowDistance(t *testing.T) {
	cond := descriptor.DistanceMinCondition{Distance: 150}
	ctx := descriptor.ConditionCtx{TargetDistance: 100}
	if cond.Eval(ctx) {
		t.Error("100 < 150 不应通过")
	}
}

func TestDistanceMinCondition_EqualDistance(t *testing.T) {
	cond := descriptor.DistanceMinCondition{Distance: 150}
	ctx := descriptor.ConditionCtx{TargetDistance: 150}
	if !cond.Eval(ctx) {
		t.Error("150 >= 150 应通过（大于等于）")
	}
}

// ============================================================
// NoNearbyTowerCondition 测试
// ============================================================

func TestNoNearbyTowerCondition_FarAway(t *testing.T) {
	cond := descriptor.NoNearbyTowerCondition{Radius: 120}
	ctx := descriptor.ConditionCtx{NearestAllyDist: 200}
	if !cond.Eval(ctx) {
		t.Error("200 > 120 应通过（附近无塔）")
	}
}

func TestNoNearbyTowerCondition_TooClose(t *testing.T) {
	cond := descriptor.NoNearbyTowerCondition{Radius: 120}
	ctx := descriptor.ConditionCtx{NearestAllyDist: 80}
	if cond.Eval(ctx) {
		t.Error("80 < 120 不应通过（附近有塔）")
	}
}

func TestNoNearbyTowerCondition_ExactlyAtRadius(t *testing.T) {
	cond := descriptor.NoNearbyTowerCondition{Radius: 120}
	ctx := descriptor.ConditionCtx{NearestAllyDist: 120}
	if cond.Eval(ctx) {
		t.Error("120 == 120 不应通过（严格大于）")
	}
}

func TestNoNearbyTowerCondition_NoAllies(t *testing.T) {
	cond := descriptor.NoNearbyTowerCondition{Radius: 120}
	ctx := descriptor.ConditionCtx{NearestAllyDist: math.MaxFloat64}
	if !cond.Eval(ctx) {
		t.Error("MaxFloat64 > 120 应通过（无盟友塔）")
	}
}

// ============================================================
// EvalAll 测试
// ============================================================

func TestEvalAll_AllPass(t *testing.T) {
	conds := []descriptor.Condition{
		descriptor.HpBelowCondition{Threshold: descriptor.FixedScaler{Value: 0.8}},
		descriptor.DistanceMinCondition{Distance: 50},
	}
	ctx := descriptor.ConditionCtx{
		TargetHpRatio: 0.3,
		TargetDistance: 200,
	}
	if !descriptor.EvalAll(conds, ctx) {
		t.Error("全部通过时应返回 true")
	}
}

func TestEvalAll_OneFails(t *testing.T) {
	conds := []descriptor.Condition{
		descriptor.HpBelowCondition{Threshold: descriptor.FixedScaler{Value: 0.5}},
		descriptor.DistanceMinCondition{Distance: 300}, // 200 < 300，不通过
	}
	ctx := descriptor.ConditionCtx{
		TargetHpRatio: 0.3,
		TargetDistance: 200,
	}
	if descriptor.EvalAll(conds, ctx) {
		t.Error("有一个失败时应返回 false")
	}
}

func TestEvalAll_Empty(t *testing.T) {
	ctx := descriptor.ConditionCtx{}
	if !descriptor.EvalAll(nil, ctx) {
		t.Error("空列表应返回 true")
	}
	if !descriptor.EvalAll([]descriptor.Condition{}, ctx) {
		t.Error("空切片应返回 true")
	}
}
