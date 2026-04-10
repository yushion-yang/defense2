// ability_config.go — 能力配置数据结构与加载。
// 每个能力结构完全统一，可直接转化为表格:
//
//	type | label | icon | category | scaleDim | base | potential | param | paramDim
//
// 运行时公式: scaledValue = base + potential * (strength / 100)
// param 为固定常量，paramDim 标识其含义，各能力按 type 分发处理。
package config

import (
	"encoding/json"
	"fmt"
)

// AbilityDef 单个能力的完整定义（结构完全统一）。
type AbilityDef struct {
	Type      string  `json:"type"`      // 能力类型标识
	Label     string  `json:"label"`     // 显示名称
	Icon      string  `json:"icon"`      // 图标名称
	Category  string  `json:"category"`  // 分类（attack/cc/damage/buff/dot/zone）
	ScaleDim  string  `json:"scaleDim"`  // 可提升维度名（空=无缩放，运行时按 type 解读）
	Base      float64 `json:"base"`      // 缩放维度的基础值
	Potential float64 `json:"potential"` // 缩放维度的潜力值
	Param     float64 `json:"param"`     // 固定常量参数值（0=无）
	ParamDim  string  `json:"paramDim"`  // 固定参数的含义标识（空=无，运行时按 type 解读）
	Display   string  `json:"display"`   // HUD 展示模板（{s}=缩放值 {s%}=缩放百分比 {si}=缩放取整 {sh%}=缩放减半百分比 {p}=参数 {p%}=参数百分比）
}

// AbilityTable 能力定义表（abilityType → AbilityDef）。
type AbilityTable map[string]*AbilityDef

// globalAbilityTable 全局缓存（首次加载后复用）。
var globalAbilityTable AbilityTable

// GlobalAbilityTable 返回全局能力定义表。LoadAbilityTable 成功后可用。
func GlobalAbilityTable() AbilityTable {
	return globalAbilityTable
}

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
	globalAbilityTable = table
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

// 六大能力类别常量。
const (
	AbilityCatAttack = 0 // 攻击模式
	AbilityCatCC     = 1 // 控制效果
	AbilityCatDamage = 2 // 命中加伤
	AbilityCatBuff   = 3 // 增益光环
	AbilityCatDoT    = 4 // 持续伤害
	AbilityCatZone   = 5 // 范围效果
	AbilityCatCount  = 6 // 类别总数
)

// 类别名 → 索引映射。
var categoryIndex = map[string]int{
	"attack": AbilityCatAttack,
	"cc":     AbilityCatCC,
	"damage": AbilityCatDamage,
	"buff":   AbilityCatBuff,
	"dot":    AbilityCatDoT,
	"zone":   AbilityCatZone,
	// 旧分类名兼容
	"combat":  AbilityCatDamage,
	"control": AbilityCatCC,
	"aura":    AbilityCatBuff,
	"economy": AbilityCatBuff,
}

// CategoryIndex 返回能力的类别索引 (0-5)，未知类别返回 -1。
func (d *AbilityDef) CategoryIndex() int {
	if d == nil {
		return -1
	}
	if idx, ok := categoryIndex[d.Category]; ok {
		return idx
	}
	return -1
}

// HasScale 是否有可缩放维度。
func (d *AbilityDef) HasScale() bool {
	return d != nil && d.ScaleDim != ""
}

// HasParam 是否有固定参数。
func (d *AbilityDef) HasParam() bool {
	return d != nil && d.ParamDim != ""
}

// FormatScale 格式化缩放维度为 HUD 展示字符串。
// base < 1 时用百分比格式，否则用绝对值格式。无缩放返回空。
func (d *AbilityDef) FormatScale(strength float64) string {
	if !d.HasScale() {
		return ""
	}
	scaled := d.Potential * (strength / 100.0)
	total := d.Base + scaled
	if d.Base < 1 {
		return fmt.Sprintf("%s %.0f%%+(%.0f%%)=%.0f%%", d.ScaleDim, d.Base*100, scaled*100, total*100)
	}
	return fmt.Sprintf("%s %.0f+(%.0f)=%.0f", d.ScaleDim, d.Base, scaled, total)
}
