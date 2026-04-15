// scaler.go — 数值缩放器接口及基础实现。
//
// Scaler 是 descriptor 引擎的核心抽象之一，用于将塔的强度（Strength）
// 映射为具体的效果数值（半径、次数、持续时间等）。
// 所有 Selector/Effect 中的可变参数都通过 Scaler 接口获取，
// 使得同一个描述符在不同强度下产生不同效果。
//
// 内置实现：
//   - FixedScaler: 固定值，不随强度变化
//   - LinearScaler: 线性缩放 base + potential × (strength / 100)
//   - DiminishingScaler: 收益递减 base + potential × (1 - exp(-str/k))（Phase 2）
//   - CappedScaler: 带上限线性 min(base + potential × str/100, cap)（Phase 2）
//
// JSON 解析：ParseScaler 根据 "scaler" 字段分派到对应实现。
package descriptor

import (
	"encoding/json"
	"fmt"
	"math"
)

// Scaler 缩放器接口 — 将 Strength 映射为最终参数值。
type Scaler interface {
	Calc(strength float64) float64
}

// FixedScaler 固定值，不随 Strength 变化。
type FixedScaler struct {
	Value float64 `json:"value"`
}

func (f FixedScaler) Calc(_ float64) float64 { return f.Value }

// LinearScaler 线性缩放：base + potential * (strength / 100)。
// 与 config.AbilityDef.CalcScale 使用相同公式。
type LinearScaler struct {
	Base      float64 `json:"base"`
	Potential float64 `json:"potential"`
}

func (l LinearScaler) Calc(strength float64) float64 {
	return l.Base + l.Potential*(strength/100.0)
}

// DiminishingScaler 收益递减缩放：base + potential * (1 - exp(-strength/k))。
// 强度越高增长越慢，最终趋近 base + potential。
// K 控制曲线形状：K 越大增长越线性。
type DiminishingScaler struct {
	Base      float64 `json:"base"`
	Potential float64 `json:"potential"`
	K         float64 `json:"k"`
}

func (d DiminishingScaler) Calc(strength float64) float64 {
	return d.Base + d.Potential*(1-math.Exp(-strength/d.K))
}

// CappedScaler 带上限的线性缩放：min(base + potential * (strength/100), cap)。
// 与 LinearScaler 相同公式，但结果不超过 Cap。
type CappedScaler struct {
	Base      float64 `json:"base"`
	Potential float64 `json:"potential"`
	Cap       float64 `json:"cap"`
}

func (c CappedScaler) Calc(strength float64) float64 {
	v := c.Base + c.Potential*(strength/100.0)
	if v > c.Cap {
		return c.Cap
	}
	return v
}

// scalerEnvelope 是 JSON 解析的中间结构，先提取 scaler 类型字段。
type scalerEnvelope struct {
	Scaler string `json:"scaler"`
}

// ParseScaler 从 JSON 字节解析 Scaler。
//
// 支持的格式：
//
//	{"scaler":"linear","base":10,"potential":5}
//	{"scaler":"fixed","value":42}
//
// 未知类型或缺少 scaler 字段时返回错误。
func ParseScaler(data []byte) (Scaler, error) {
	var env scalerEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse scaler envelope: %w", err)
	}

	switch env.Scaler {
	case "linear":
		var s LinearScaler
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse linear scaler: %w", err)
		}
		return s, nil
	case "fixed":
		var s FixedScaler
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse fixed scaler: %w", err)
		}
		return s, nil
	case "diminishing":
		var s DiminishingScaler
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse diminishing scaler: %w", err)
		}
		return s, nil
	case "capped":
		var s CappedScaler
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse capped scaler: %w", err)
		}
		return s, nil
	case "":
		return nil, fmt.Errorf("missing scaler type field")
	default:
		return nil, fmt.Errorf("unknown scaler type: %q", env.Scaler)
	}
}
