// descriptor_scaler_v3_test.go — Phase 3 SteppedScaler 测试。
//
// 覆盖 SteppedScaler 计算逻辑、边界情况及 ParseScaler 集成。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// SteppedScaler 测试
// ============================================================

func TestSteppedScaler_SingleStep(t *testing.T) {
	// 只有一个断点时，任何 strength 都返回该值
	s := descriptor.SteppedScaler{
		Steps: []descriptor.StepThreshold{
			{Strength: 50, Value: 3},
		},
	}
	tests := []struct {
		str  float64
		want float64
	}{
		{0, 3},
		{50, 3},
		{100, 3},
		{999, 3},
	}
	for _, tt := range tests {
		got := s.Calc(tt.str)
		if got != tt.want {
			t.Errorf("Calc(%v) = %v, 期望 %v", tt.str, got, tt.want)
		}
	}
}

func TestSteppedScaler_BelowFirstStep(t *testing.T) {
	// strength 低于第一个断点 → 返回第一个断点的值
	s := descriptor.SteppedScaler{
		Steps: []descriptor.StepThreshold{
			{Strength: 50, Value: 2},
			{Strength: 200, Value: 8},
		},
	}
	got := s.Calc(0)
	if got != 2 {
		t.Errorf("Calc(0) = %v, 期望 2 (第一个断点值)", got)
	}
	got = s.Calc(25)
	if got != 2 {
		t.Errorf("Calc(25) = %v, 期望 2 (低于第一个断点)", got)
	}
}

func TestSteppedScaler_AboveLastStep(t *testing.T) {
	// strength 高于最后一个断点 → 返回最后一个断点的值
	s := descriptor.SteppedScaler{
		Steps: []descriptor.StepThreshold{
			{Strength: 0, Value: 1},
			{Strength: 100, Value: 5},
		},
	}
	got := s.Calc(200)
	if got != 5 {
		t.Errorf("Calc(200) = %v, 期望 5 (最后一个断点值)", got)
	}
	got = s.Calc(9999)
	if got != 5 {
		t.Errorf("Calc(9999) = %v, 期望 5", got)
	}
}

func TestSteppedScaler_LinearInterpolation(t *testing.T) {
	// 三个断点，strength 在中间时线性插值
	s := descriptor.SteppedScaler{
		Steps: []descriptor.StepThreshold{
			{Strength: 0, Value: 1},
			{Strength: 100, Value: 3},
			{Strength: 300, Value: 5},
		},
	}

	tests := []struct {
		name string
		str  float64
		want float64
	}{
		// str=50 在 [0,100] 之间，插值 1 + (3-1) * (50-0)/(100-0) = 2
		{"区间 1 中点", 50, 2.0},
		// str=200 在 [100,300] 之间，插值 3 + (5-3) * (200-100)/(300-100) = 4
		{"区间 2 中点", 200, 4.0},
		// str=25 在 [0,100] 之间，插值 1 + 2 * 25/100 = 1.5
		{"区间 1 四分之一", 25, 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Calc(tt.str)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Calc(%v) = %v, 期望 %v", tt.str, got, tt.want)
			}
		})
	}
}

func TestSteppedScaler_ExactStepMatch(t *testing.T) {
	// strength 恰好等于断点时返回精确值
	s := descriptor.SteppedScaler{
		Steps: []descriptor.StepThreshold{
			{Strength: 0, Value: 1},
			{Strength: 100, Value: 3},
			{Strength: 300, Value: 5},
		},
	}

	tests := []struct {
		str  float64
		want float64
	}{
		{0, 1},
		{100, 3},
		{300, 5},
	}
	for _, tt := range tests {
		got := s.Calc(tt.str)
		if got != tt.want {
			t.Errorf("Calc(%v) = %v, 期望 %v", tt.str, got, tt.want)
		}
	}
}

func TestParseScaler_Stepped(t *testing.T) {
	data := []byte(`{
		"scaler": "stepped",
		"steps": [
			{"strength": 0, "value": 1},
			{"strength": 100, "value": 3},
			{"strength": 300, "value": 5}
		]
	}`)
	s, err := descriptor.ParseScaler(data)
	if err != nil {
		t.Fatalf("ParseScaler 失败: %v", err)
	}
	ss, ok := s.(descriptor.SteppedScaler)
	if !ok {
		t.Fatalf("期望 SteppedScaler，实际 %T", s)
	}
	if len(ss.Steps) != 3 {
		t.Fatalf("期望 3 个断点，实际 %d", len(ss.Steps))
	}

	// 验证计算正确：str=50 → 2.0
	got := s.Calc(50)
	if math.Abs(got-2.0) > 1e-9 {
		t.Errorf("解析后 Calc(50) = %v, 期望 2.0", got)
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Scaler = descriptor.SteppedScaler{}
