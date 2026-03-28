// ability_config.go — 能力配置数据结构与加载。
// 每个能力只有一个可提升维度（scaleDim），其余参数为固定常量。
// 运行时公式: scaledValue = base + potential * (strength / 100)
package config

import (
	"encoding/json"
	"fmt"
)

// AbilityDef 单个能力的完整定义。
type AbilityDef struct {
	Type      string             `json:"type"`      // 能力类型标识
	Label     string             `json:"label"`     // 显示名称
	Icon      string             `json:"icon"`      // 图标名称
	Category  string             `json:"category"`  // 分类（combat/control/aura/zone/economy）
	ScaleDim  string             `json:"scaleDim"`  // 可提升维度名（如"factor"/"chance"/"dps"，空=无缩放）
	Base      float64            `json:"base"`      // 缩放维度的基础值
	Potential float64            `json:"potential"` // 缩放维度的潜力值
	Params    map[string]float64 `json:"params"`    // 固定常量参数
}

// AbilityTable 能力定义表（abilityType → AbilityDef）。
type AbilityTable map[string]*AbilityDef

// LoadAbilityTable 从 abilities.json 加载能力定义表。
func LoadAbilityTable() (AbilityTable, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load abilities: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/abilities/abilities.json")
	if err != nil {
		return nil, fmt.Errorf("load abilities: %w", err)
	}

	var table AbilityTable
	if err := json.Unmarshal(data, &table); err != nil {
		return nil, fmt.Errorf("parse abilities: %w", err)
	}
	return table, nil
}

// CalcScale 根据强度计算缩放维度的实际值。
// 公式: base + potential * (strength / 100)
// 无缩放维度时返回 0。
func (d *AbilityDef) CalcScale(strength float64) float64 {
	if d == nil || d.ScaleDim == "" {
		return 0
	}
	return d.Base + d.Potential*(strength/100.0)
}

// GetParam 获取固定常量参数，不存在时返回 defaultVal。
func (d *AbilityDef) GetParam(key string, defaultVal float64) float64 {
	if d == nil || d.Params == nil {
		return defaultVal
	}
	if v, ok := d.Params[key]; ok {
		return v
	}
	return defaultVal
}

// HasScale 是否有可缩放维度。
func (d *AbilityDef) HasScale() bool {
	return d != nil && d.ScaleDim != ""
}

// FormatScale 格式化缩放维度为 "base+(scaled)=total" 字符串。
// 用于 HUD 展示。无缩放返回空。
func (d *AbilityDef) FormatScale(strength float64) string {
	if !d.HasScale() {
		return ""
	}
	scaled := d.Potential * (strength / 100.0)
	total := d.Base + scaled
	// 根据数值大小选择格式
	if d.Base >= 1 {
		return fmt.Sprintf("%s %.0f+(%.0f)=%.0f", d.ScaleDim, d.Base, scaled, total)
	}
	return fmt.Sprintf("%s %.0f%%+(%.0f%%)=%.0f%%", d.ScaleDim, d.Base*100, scaled*100, total*100)
}
