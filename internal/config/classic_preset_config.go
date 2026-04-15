// classic_preset_config.go — 经典模式预设炮塔配置加载。
//
// 从 config/towers/classic-presets.json 读取 11 种预设塔的完整定义。
// 每种塔包含固定的攻击方式、2 个预设能力、按角色分配的属性档位。
//
// 与 towers.json 的区别：towers.json 定义"基础塔模板"（属性由 RollTowerStats 随机），
// classic-presets.json 定义"成品塔"（属性由 tiers 字段固定，能力出厂内置）。
package config

import (
	"encoding/json"
	"fmt"
	"log"
)

// ClassicPreset 经典模式单个预设塔的配置。
// StrengthConfig 强度升级配置。
type StrengthConfig struct {
	Cost         int     `json:"cost"`         // 单次购买费用
	Amount       float64 `json:"amount"`       // 单次增加的强度值
	MaxPurchases int     `json:"maxPurchases"` // 最大购买次数（-1=无限）
}

type ClassicPreset struct {
	Key         string            `json:"key"`         // 唯一标识（cl_sentinel 等）
	Name        string            `json:"name"`        // 显示名称
	Category    string            `json:"category"`    // 角色分类: dps / aoe / support
	Abilities   []string          `json:"abilities"`   // 预设能力列表（第 1 个=攻击能力）
	AbilityMode string            `json:"abilityMode"` // 能力获取方式: preset/paid/byWave/allUnlocked
	Specialty   string            `json:"specialty"`   // 专精属性: damage/atkSpeed/range
	Tiers       map[string]string `json:"tiers"`       // 属性档位
	Strength    StrengthConfig    `json:"strength"`    // 强度升级规则
	SpriteKey   string            `json:"spriteKey"`   // 精灵键名
	BuildCost   int               `json:"buildCost"`   // 建造费用
	Description string            `json:"description"` // 简短描述
}

// classicPresetsDefaults JSON 中 defaults 区段的映射。
// 塔未指定的字段会从此处继承。
type classicPresetsDefaults struct {
	BuildCost   int            `json:"buildCost"`
	AbilityMode string         `json:"abilityMode"`
	Strength    StrengthConfig `json:"strength"`
}

// classicPresetsFile JSON 顶层结构（含 defaults + towers）。
type classicPresetsFile struct {
	Defaults classicPresetsDefaults `json:"defaults"`
	Towers   []ClassicPreset        `json:"towers"`
}

// ClassicPresetsConfig 经典模式预设表。
type ClassicPresetsConfig struct {
	Towers []ClassicPreset `json:"towers"`
}

var globalClassicPresets *ClassicPresetsConfig

// GlobalClassicPresets 返回全局经典预设表。LoadClassicPresets 成功后可用。
func GlobalClassicPresets() *ClassicPresetsConfig {
	return globalClassicPresets
}

// LoadClassicPresets 从 config/towers/classic-presets.json 加载经典模式预设。
// JSON 中 defaults 区段的公共字段会合并到每个塔（塔自身显式指定的值优先）。
func LoadClassicPresets() error {
	if dataFS == nil {
		return fmt.Errorf("load classic-presets: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/towers/classic-presets.json")
	if err != nil {
		return fmt.Errorf("load classic-presets: %w", err)
	}
	var file classicPresetsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse classic-presets: %w", err)
	}

	// 将 defaults 合并到每个塔（塔自身值为零值时取 defaults）
	for i := range file.Towers {
		t := &file.Towers[i]
		if t.BuildCost == 0 {
			t.BuildCost = file.Defaults.BuildCost
		}
		if t.AbilityMode == "" {
			t.AbilityMode = file.Defaults.AbilityMode
		}
		if t.Strength == (StrengthConfig{}) {
			t.Strength = file.Defaults.Strength
		}
		// spriteKey 默认与 key 相同
		if t.SpriteKey == "" {
			t.SpriteKey = t.Key
		}
	}

	globalClassicPresets = &ClassicPresetsConfig{Towers: file.Towers}
	log.Printf("[config] loaded %d classic tower presets", len(file.Towers))
	return nil
}

// ClassicPresetByKey 按 key 查找预设塔。未找到返回 nil。
func ClassicPresetByKey(key string) *ClassicPreset {
	if globalClassicPresets == nil {
		return nil
	}
	for i := range globalClassicPresets.Towers {
		if globalClassicPresets.Towers[i].Key == key {
			return &globalClassicPresets.Towers[i]
		}
	}
	return nil
}

// ClassicPresetsByCategory 返回指定分类的所有预设塔。
func ClassicPresetsByCategory(category string) []ClassicPreset {
	if globalClassicPresets == nil {
		return nil
	}
	var result []ClassicPreset
	for _, t := range globalClassicPresets.Towers {
		if t.Category == category {
			result = append(result, t)
		}
	}
	return result
}
