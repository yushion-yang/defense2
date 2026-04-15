// descriptor_effect_v2_test.go — Phase 2 新增 2 种效果测试。
//
// 覆盖 CritEffect 和 PurgeEffect。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// CritEffect 测试
// ============================================================

func TestCritEffect_BasicMultiplier(t *testing.T) {
	eff := descriptor.CritEffect{Multiplier: 2.5}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeCrit {
		t.Errorf("type = %v, want EffTypeCrit", r.Type)
	}
	if !r.IsCrit {
		t.Error("CritEffect 应设置 IsCrit=true")
	}
	if r.CritMult != 2.5 {
		t.Errorf("CritMult = %v, want 2.5", r.CritMult)
	}
}

func TestCritEffect_DefaultMultiplierOne(t *testing.T) {
	eff := descriptor.CritEffect{Multiplier: 1.0}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if !r.IsCrit {
		t.Error("IsCrit 应为 true")
	}
	if r.CritMult != 1.0 {
		t.Errorf("CritMult = %v, want 1.0", r.CritMult)
	}
}

// ============================================================
// PurgeEffect 测试
// ============================================================

func TestPurgeEffect_BasicCount(t *testing.T) {
	eff := descriptor.PurgeEffect{Count: 3}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypePurge {
		t.Errorf("type = %v, want EffTypePurge", r.Type)
	}
	if r.PurgeCount != 3 {
		t.Errorf("PurgeCount = %v, want 3", r.PurgeCount)
	}
}

func TestPurgeEffect_CountZero(t *testing.T) {
	eff := descriptor.PurgeEffect{Count: 0}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypePurge {
		t.Errorf("type = %v, want EffTypePurge", r.Type)
	}
	if r.PurgeCount != 0 {
		t.Errorf("PurgeCount = %v, want 0", r.PurgeCount)
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Effect = descriptor.CritEffect{}
var _ descriptor.Effect = descriptor.PurgeEffect{}
