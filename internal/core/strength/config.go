// config.go — 战力配置解析，将塔属性与战力值绑定。
package strength

import "strings"

// StrengthBinding 单个属性的战力绑定配置。
type StrengthBinding struct {
	Base      float64 // 基础值
	Potential float64 // 潜力值（随战力线性缩放）
}

// StrengthConfig 塔的战力配置。
type StrengthConfig struct {
	Bindings map[string]*StrengthBinding // 属性路径 → 绑定（如 "attackDamage", "effects.slowFactor"）
}

// ParseStrengthConfig 从原始 JSON map 解析战力配置。
// 期望格式: { "attackDamage": { "base": 10, "potential": 5 }, "effects.slowFactor": { "base": 0.3, "potential": 0.1 } }
func ParseStrengthConfig(raw map[string]interface{}) *StrengthConfig {
	cfg := &StrengthConfig{
		Bindings: make(map[string]*StrengthBinding),
	}
	if raw == nil {
		return cfg
	}

	for path, v := range raw {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		b := &StrengthBinding{}
		if base, ok := obj["base"]; ok {
			b.Base = toFloat64(base)
		}
		if pot, ok := obj["potential"]; ok {
			b.Potential = toFloat64(pot)
		}
		cfg.Bindings[path] = b
	}
	return cfg
}

// CalcAttribute 计算属性在指定有效战力下的值。
// 公式: base + potential * (effectiveStrength / 100)
// 未绑定的属性返回 fallback。
func (c *StrengthConfig) CalcAttribute(path string, effectiveStrength, fallback float64) float64 {
	b := c.ResolveBinding(path)
	if b == nil {
		return fallback
	}
	return b.Base + b.Potential*(effectiveStrength/100.0)
}


// ResolveBinding 查找属性路径的绑定配置，支持点号分隔路径（如 "effects.slowFactor"）。
// 精确匹配优先；无匹配返回 nil。
func (c *StrengthConfig) ResolveBinding(path string) *StrengthBinding {
	// 精确匹配
	if b, ok := c.Bindings[path]; ok {
		return b
	}

	// 尝试点号路径的各段匹配（从完整路径到最后一段）
	parts := strings.Split(path, ".")
	for i := 1; i < len(parts); i++ {
		sub := strings.Join(parts[i:], ".")
		if b, ok := c.Bindings[sub]; ok {
			return b
		}
	}

	return nil
}

// IsLinked 检查属性路径是否有战力绑定。
func (c *StrengthConfig) IsLinked(path string) bool {
	return c.ResolveBinding(path) != nil
}

// toFloat64 将 interface{} 转换为 float64（支持 JSON 数值类型）。
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
