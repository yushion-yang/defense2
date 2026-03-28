// ability_config.go — 能力配置数据结构与加载。
// 从 config/abilities/abilities.json 加载能力定义表。
// 每个能力有 base 和 potential 两组参数，运行时通过强度计算实际值:
//
//	effective[key] = base[key] + potential[key] * (strength / 100)
package config

import (
	"encoding/json"
	"fmt"
)

// AbilityDef 单个能力的完整定义。
type AbilityDef struct {
	Type        string             `json:"type"`        // 能力类型标识（与 map key 一致）
	Label       string             `json:"label"`       // 显示名称（如"减速"、"弹射"）
	Description string             `json:"description"` // 描述模板（支持 {key} 占位符）
	Icon        string             `json:"icon"`        // 图标名称（对应 assets/icons/）
	Category    string             `json:"category"`    // 分类（combat/control/aura/zone/economy）
	Base        map[string]float64 `json:"base"`        // 基础参数（强度0时的底线值）
	Potential   map[string]float64 `json:"potential"`   // 潜力参数（强度100时 = base + potential）
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

// CalcParam 根据强度计算某个参数的实际值。
// 公式: base[key] + potential[key] * (strength / 100)
// 参数不存在时返回 defaultVal。
func (d *AbilityDef) CalcParam(key string, strength float64, defaultVal float64) float64 {
	if d == nil {
		return defaultVal
	}
	base, hasBase := d.Base[key]
	pot, hasPot := d.Potential[key]
	if !hasBase && !hasPot {
		return defaultVal
	}
	return base + pot*(strength/100.0)
}

// CalcAllParams 根据强度计算所有参数的实际值。
// 返回新 map，不修改原始 Base/Potential。
func (d *AbilityDef) CalcAllParams(strength float64) map[string]float64 {
	if d == nil {
		return nil
	}
	// 收集所有参数 key
	keys := make(map[string]bool)
	for k := range d.Base {
		keys[k] = true
	}
	for k := range d.Potential {
		keys[k] = true
	}

	result := make(map[string]float64, len(keys))
	for k := range keys {
		base := d.Base[k]
		pot := d.Potential[k]
		result[k] = base + pot*(strength/100.0)
	}
	return result
}

// GetBase 获取基础参数值（不含战力缩放）。
func (d *AbilityDef) GetBase(key string, defaultVal float64) float64 {
	if d == nil || d.Base == nil {
		return defaultVal
	}
	if v, ok := d.Base[key]; ok {
		return v
	}
	return defaultVal
}

// GetPotential 获取潜力参数值。
func (d *AbilityDef) GetPotential(key string, defaultVal float64) float64 {
	if d == nil || d.Potential == nil {
		return defaultVal
	}
	if v, ok := d.Potential[key]; ok {
		return v
	}
	return defaultVal
}
