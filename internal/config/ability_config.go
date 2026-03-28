// ability_config.go — 能力配置数据结构与加载。
// 从 config/abilities/abilities.json 加载能力定义表。
// 塔配置通过 abilities 数组中的 type 字段关联到此表。
package config

import (
	"encoding/json"
	"fmt"
)

// AbilityDef 单个能力的完整定义。
type AbilityDef struct {
	Label       string             `json:"label"`       // 显示名称（如"减速"、"弹射"）
	Description string             `json:"description"` // 简短描述（如"命中减速32%持续1.4s"）
	Icon        string             `json:"icon"`        // 图标名称（对应 assets/icons/ 下的文件）
	Category    string             `json:"category"`    // 分类（combat/control/aura/zone/economy）
	Params      map[string]float64 `json:"params"`      // 能力参数（键值对，具体含义由各能力实现解读）
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

// GetParam 从能力定义中安全获取参数，不存在时返回 defaultVal。
func (d *AbilityDef) GetParam(key string, defaultVal float64) float64 {
	if d == nil || d.Params == nil {
		return defaultVal
	}
	if v, ok := d.Params[key]; ok {
		return v
	}
	return defaultVal
}
