// descriptor_scaler_v2_test.go — Phase 2 新增 2 种缩放器测试。
//
// 覆盖 DiminishingScaler / CappedScaler 及 ParseScaler 集成。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// DiminishingScaler 测试
// ============================================================

func TestDiminishingScaler_Calc(t *testing.T) {
	tests := []struct {
		name     string
		base     float64
		pot      float64
		k        float64
		strength float64
		want     float64
	}{
		{"str=0 返回 base", 10, 50, 100, 0, 10},
		{"大 strength 逼近 base+potential", 10, 50, 100, 10000, 60},
		{"中等 strength", 10, 50, 100, 100, 10 + 50*(1-math.Exp(-1))},
		{"k 很大时增长很慢", 10, 50, 10000, 100, 10 + 50*(1-math.Exp(-0.01))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := descriptor.DiminishingScaler{Base: tt.base, Potential: tt.pot, K: tt.k}
			got := s.Calc(tt.strength)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("Calc(%v) = %v, 期望 %v", tt.strength, got, tt.want)
			}
		})
	}
}

func TestDiminishingScaler_MonotonicallyIncreasing(t *testing.T) {
	s := descriptor.DiminishingScaler{Base: 0, Potential: 100, K: 100}
	prev := s.Calc(0)
	for str := 10.0; str <= 500; str += 10 {
		cur := s.Calc(str)
		if cur < prev {
			t.Errorf("Calc(%v)=%v 小于 Calc(%v)=%v，应单调递增", str, cur, str-10, prev)
		}
		prev = cur
	}
}

func TestDiminishingScaler_NeverExceedBasePlusPotential(t *testing.T) {
	s := descriptor.DiminishingScaler{Base: 10, Potential: 50, K: 100}
	cap := 10.0 + 50.0
	for str := 0.0; str <= 10000; str += 100 {
		got := s.Calc(str)
		if got > cap+0.001 {
			t.Errorf("Calc(%v)=%v 超过上限 %v", str, got, cap)
		}
	}
}

// ============================================================
// CappedScaler 测试
// ============================================================

func TestCappedScaler_Calc(t *testing.T) {
	tests := []struct {
		name     string
		base     float64
		pot      float64
		cap      float64
		strength float64
		want     float64
	}{
		{"str=0 返回 base", 10, 5, 100, 0, 10},
		{"str=100 未达上限", 10, 5, 100, 100, 15},
		{"str=200 达上限", 10, 5, 20, 200, 20},
		{"str=1000 限制在 cap", 10, 5, 20, 1000, 20},
		{"cap 低于 base", 5, 10, 3, 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := descriptor.CappedScaler{Base: tt.base, Potential: tt.pot, Cap: tt.cap}
			got := s.Calc(tt.strength)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Calc(%v) = %v, 期望 %v", tt.strength, got, tt.want)
			}
		})
	}
}

func TestCappedScaler_NeverExceedsCap(t *testing.T) {
	s := descriptor.CappedScaler{Base: 10, Potential: 5, Cap: 25}
	for str := 0.0; str <= 10000; str += 100 {
		got := s.Calc(str)
		if got > 25+1e-9 {
			t.Errorf("Calc(%v)=%v 超过 cap 25", str, got)
		}
	}
}

// ============================================================
// ParseScaler 集成测试
// ============================================================

func TestParseScaler_Diminishing(t *testing.T) {
	data := []byte(`{"scaler":"diminishing","base":10,"potential":50,"k":100}`)
	s, err := descriptor.ParseScaler(data)
	if err != nil {
		t.Fatalf("ParseScaler 失败: %v", err)
	}
	ds, ok := s.(descriptor.DiminishingScaler)
	if !ok {
		t.Fatalf("期望 DiminishingScaler，实际 %T", s)
	}
	if ds.Base != 10 || ds.Potential != 50 || ds.K != 100 {
		t.Errorf("解析结果 base=%v potential=%v k=%v, 期望 10/50/100",
			ds.Base, ds.Potential, ds.K)
	}

	// 验证计算正确
	got := s.Calc(100)
	want := 10 + 50*(1-math.Exp(-1))
	if math.Abs(got-want) > 0.001 {
		t.Errorf("解析后 Calc(100) = %v, 期望 %v", got, want)
	}
}

func TestParseScaler_Capped(t *testing.T) {
	data := []byte(`{"scaler":"capped","base":10,"potential":5,"cap":20}`)
	s, err := descriptor.ParseScaler(data)
	if err != nil {
		t.Fatalf("ParseScaler 失败: %v", err)
	}
	cs, ok := s.(descriptor.CappedScaler)
	if !ok {
		t.Fatalf("期望 CappedScaler，实际 %T", s)
	}
	if cs.Base != 10 || cs.Potential != 5 || cs.Cap != 20 {
		t.Errorf("解析结果 base=%v potential=%v cap=%v, 期望 10/5/20",
			cs.Base, cs.Potential, cs.Cap)
	}

	// str=1000 应限制在 cap
	got := s.Calc(1000)
	if got != 20 {
		t.Errorf("Calc(1000) = %v, 期望 20 (cap)", got)
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Scaler = descriptor.DiminishingScaler{}
var _ descriptor.Scaler = descriptor.CappedScaler{}
