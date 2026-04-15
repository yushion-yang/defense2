// descriptor_effect_v3_test.go — Phase 3 TeleportEffect 测试。
//
// 覆盖 TeleportEffect 的 Apply 输出、不同 Scaler 组合、效果类型检查。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// TeleportEffect 测试
// ============================================================

func TestTeleportEffect_FixedDistance(t *testing.T) {
	// 固定距离：distance=50，不随 strength 变化
	eff := descriptor.TeleportEffect{
		Distance: descriptor.FixedScaler{Value: 50},
	}
	ctx := descriptor.EffectCtx{Strength: 100}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeTeleport {
		t.Errorf("type = %v, 期望 EffTypeTeleport", r.Type)
	}
	if r.TeleportDist != 50 {
		t.Errorf("TeleportDist = %v, 期望 50", r.TeleportDist)
	}
}

func TestTeleportEffect_LinearScaler(t *testing.T) {
	// 线性距离：base=30 + potential=20 × (str/100)
	// str=200 → 30 + 20*2 = 70
	eff := descriptor.TeleportEffect{
		Distance: descriptor.LinearScaler{Base: 30, Potential: 20},
	}
	ctx := descriptor.EffectCtx{Strength: 200}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeTeleport {
		t.Errorf("type = %v, 期望 EffTypeTeleport", r.Type)
	}
	want := 70.0
	if math.Abs(r.TeleportDist-want) > 1e-9 {
		t.Errorf("TeleportDist = %v, 期望 %v", r.TeleportDist, want)
	}
}

func TestTeleportEffect_ResultType(t *testing.T) {
	// 确认 EffectResult 的 Type 字段正确
	eff := descriptor.TeleportEffect{
		Distance: descriptor.FixedScaler{Value: 10},
	}
	ctx := descriptor.EffectCtx{Strength: 0}
	r := eff.Apply(ctx)

	if r.Type != descriptor.EffTypeTeleport {
		t.Errorf("type = %v, 期望 EffTypeTeleport(%d)", r.Type, descriptor.EffTypeTeleport)
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Effect = descriptor.TeleportEffect{}
