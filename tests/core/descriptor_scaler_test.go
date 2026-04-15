package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// LinearScaler 测试
// ============================================================

func TestLinearScaler_Calc(t *testing.T) {
	tests := []struct {
		name      string
		base      float64
		potential float64
		strength  float64
		want      float64
	}{
		{"str=0 返回 base", 10, 5, 0, 10},
		{"str=100 返回 base+potential", 10, 5, 100, 15},
		{"str=200 返回 base+2*potential", 10, 5, 200, 20},
		{"负 potential 递减", 10, -3, 100, 7},
		{"base=0 纯缩放", 0, 20, 50, 10},
		{"全零", 0, 0, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := descriptor.LinearScaler{Base: tt.base, Potential: tt.potential}
			got := s.Calc(tt.strength)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Calc(%v) = %v, 期望 %v", tt.strength, got, tt.want)
			}
		})
	}
}

// ============================================================
// FixedScaler 测试
// ============================================================

func TestFixedScaler_IgnoresStrength(t *testing.T) {
	s := descriptor.FixedScaler{Value: 42}
	strengths := []float64{0, 50, 100, 200, -10}
	for _, str := range strengths {
		got := s.Calc(str)
		if got != 42 {
			t.Errorf("Calc(%v) = %v, 期望 42（固定值不随 Strength 变化）", str, got)
		}
	}
}

// ============================================================
// ParseScaler 测试
// ============================================================

func TestParseScaler_Linear(t *testing.T) {
	data := []byte(`{"scaler":"linear","base":10,"potential":5}`)
	s, err := descriptor.ParseScaler(data)
	if err != nil {
		t.Fatalf("ParseScaler 失败: %v", err)
	}
	ls, ok := s.(descriptor.LinearScaler)
	if !ok {
		t.Fatalf("期望 LinearScaler，实际 %T", s)
	}
	if ls.Base != 10 || ls.Potential != 5 {
		t.Errorf("解析结果 base=%v potential=%v, 期望 10/5", ls.Base, ls.Potential)
	}
	// 验证计算正确
	if math.Abs(s.Calc(100)-15) > 1e-9 {
		t.Errorf("解析后 Calc(100) = %v, 期望 15", s.Calc(100))
	}
}

func TestParseScaler_Fixed(t *testing.T) {
	data := []byte(`{"scaler":"fixed","value":42}`)
	s, err := descriptor.ParseScaler(data)
	if err != nil {
		t.Fatalf("ParseScaler 失败: %v", err)
	}
	fs, ok := s.(descriptor.FixedScaler)
	if !ok {
		t.Fatalf("期望 FixedScaler，实际 %T", s)
	}
	if fs.Value != 42 {
		t.Errorf("解析结果 value=%v, 期望 42", fs.Value)
	}
}

func TestParseScaler_UnknownType(t *testing.T) {
	data := []byte(`{"scaler":"exponential","base":1}`)
	_, err := descriptor.ParseScaler(data)
	if err == nil {
		t.Fatal("未知 scaler 类型应返回错误")
	}
}

func TestParseScaler_InvalidJSON(t *testing.T) {
	data := []byte(`{not json}`)
	_, err := descriptor.ParseScaler(data)
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

func TestParseScaler_MissingType(t *testing.T) {
	data := []byte(`{"base":10,"potential":5}`)
	_, err := descriptor.ParseScaler(data)
	if err == nil {
		t.Fatal("缺少 scaler 字段应返回错误")
	}
}
