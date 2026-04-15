// descriptor_condition_v2_test.go — Phase 2 新增 5 种条件测试。
//
// 覆盖 IsBoss / NotBoss / Every / BuffActive / BuffAbsent 条件类型。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// IsBossCondition 测试
// ============================================================

func TestIsBossCondition_TrueWhenBoss(t *testing.T) {
	cond := descriptor.IsBossCondition{}
	ctx := descriptor.ConditionCtx{IsBoss: true}
	if !cond.Eval(ctx) {
		t.Error("IsBoss=true 时应通过")
	}
}

func TestIsBossCondition_FalseWhenNotBoss(t *testing.T) {
	cond := descriptor.IsBossCondition{}
	ctx := descriptor.ConditionCtx{IsBoss: false}
	if cond.Eval(ctx) {
		t.Error("IsBoss=false 时不应通过")
	}
}

// ============================================================
// NotBossCondition 测试
// ============================================================

func TestNotBossCondition_TrueWhenNotBoss(t *testing.T) {
	cond := descriptor.NotBossCondition{}
	ctx := descriptor.ConditionCtx{IsBoss: false}
	if !cond.Eval(ctx) {
		t.Error("IsBoss=false 时应通过")
	}
}

func TestNotBossCondition_FalseWhenBoss(t *testing.T) {
	cond := descriptor.NotBossCondition{}
	ctx := descriptor.ConditionCtx{IsBoss: true}
	if cond.Eval(ctx) {
		t.Error("IsBoss=true 时不应通过")
	}
}

// ============================================================
// EveryCondition 测试
// ============================================================

func TestEveryCondition_PassesEveryN(t *testing.T) {
	cond := descriptor.NewEveryCondition(3)
	ctx := descriptor.ConditionCtx{}

	// 第 1 次：不通过（计数 1）
	if cond.Eval(ctx) {
		t.Error("第 1 次调用不应通过（每 3 次才触发）")
	}
	// 第 2 次：不通过（计数 2）
	if cond.Eval(ctx) {
		t.Error("第 2 次调用不应通过")
	}
	// 第 3 次：通过（计数 3，重置）
	if !cond.Eval(ctx) {
		t.Error("第 3 次调用应通过")
	}
	// 第 4 次：不通过（重新计数 1）
	if cond.Eval(ctx) {
		t.Error("第 4 次调用不应通过（重置后重新计数）")
	}
}

func TestEveryCondition_N1AlwaysPasses(t *testing.T) {
	cond := descriptor.NewEveryCondition(1)
	ctx := descriptor.ConditionCtx{}

	for i := 0; i < 5; i++ {
		if !cond.Eval(ctx) {
			t.Errorf("N=1 时第 %d 次调用应通过", i+1)
		}
	}
}

func TestEveryCondition_N0NeverPasses(t *testing.T) {
	// N<=0 视为无效，永远不触发
	cond := descriptor.NewEveryCondition(0)
	ctx := descriptor.ConditionCtx{}

	for i := 0; i < 5; i++ {
		if cond.Eval(ctx) {
			t.Errorf("N=0 时第 %d 次调用不应通过", i+1)
		}
	}
}

// ============================================================
// BuffActiveCondition 测试
// ============================================================

func TestBuffActiveCondition_PresentInList(t *testing.T) {
	cond := descriptor.BuffActiveCondition{BuffID: "burn"}
	ctx := descriptor.ConditionCtx{
		ActiveBuffIDs: []string{"slow", "burn", "stun"},
	}
	if !cond.Eval(ctx) {
		t.Error("burn 在 ActiveBuffIDs 中应通过")
	}
}

func TestBuffActiveCondition_AbsentFromList(t *testing.T) {
	cond := descriptor.BuffActiveCondition{BuffID: "poison"}
	ctx := descriptor.ConditionCtx{
		ActiveBuffIDs: []string{"slow", "burn"},
	}
	if cond.Eval(ctx) {
		t.Error("poison 不在 ActiveBuffIDs 中不应通过")
	}
}

func TestBuffActiveCondition_EmptyList(t *testing.T) {
	cond := descriptor.BuffActiveCondition{BuffID: "burn"}
	ctx := descriptor.ConditionCtx{ActiveBuffIDs: nil}
	if cond.Eval(ctx) {
		t.Error("空列表时不应通过")
	}
}

// ============================================================
// BuffAbsentCondition 测试
// ============================================================

func TestBuffAbsentCondition_AbsentFromList(t *testing.T) {
	cond := descriptor.BuffAbsentCondition{BuffID: "poison"}
	ctx := descriptor.ConditionCtx{
		ActiveBuffIDs: []string{"slow", "burn"},
	}
	if !cond.Eval(ctx) {
		t.Error("poison 不在列表中应通过（buff 缺失）")
	}
}

func TestBuffAbsentCondition_PresentInList(t *testing.T) {
	cond := descriptor.BuffAbsentCondition{BuffID: "burn"}
	ctx := descriptor.ConditionCtx{
		ActiveBuffIDs: []string{"slow", "burn"},
	}
	if cond.Eval(ctx) {
		t.Error("burn 在列表中不应通过（buff 存在）")
	}
}

func TestBuffAbsentCondition_EmptyList(t *testing.T) {
	cond := descriptor.BuffAbsentCondition{BuffID: "burn"}
	ctx := descriptor.ConditionCtx{ActiveBuffIDs: nil}
	if !cond.Eval(ctx) {
		t.Error("空列表时应通过（buff 缺失）")
	}
}

// ============================================================
// EvalAll 与新条件组合
// ============================================================

func TestEvalAll_WithNewConditions(t *testing.T) {
	conds := []descriptor.Condition{
		descriptor.IsBossCondition{},
		descriptor.BuffActiveCondition{BuffID: "burn"},
	}
	ctx := descriptor.ConditionCtx{
		IsBoss:        true,
		ActiveBuffIDs: []string{"burn"},
	}
	if !descriptor.EvalAll(conds, ctx) {
		t.Error("IsBoss + BuffActive(burn) 都满足应返回 true")
	}

	// 不满足 IsBoss
	ctx.IsBoss = false
	if descriptor.EvalAll(conds, ctx) {
		t.Error("IsBoss 不满足应返回 false")
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Condition = descriptor.IsBossCondition{}
var _ descriptor.Condition = descriptor.NotBossCondition{}
var _ descriptor.Condition = (*descriptor.EveryCondition)(nil)
var _ descriptor.Condition = descriptor.BuffActiveCondition{}
var _ descriptor.Condition = descriptor.BuffAbsentCondition{}
