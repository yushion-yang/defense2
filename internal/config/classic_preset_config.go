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
type ClassicPreset struct {
	Key         string            `json:"key"`         // 唯一标识（cl_sentinel 等）
	Name        string            `json:"name"`        // 显示名称
	Category    string            `json:"category"`    // 角色分类: dps / aoe / support
	Abilities   []string          `json:"abilities"`   // 预设能力列表（第 1 个=攻击能力，决定攻击方式）
	Tiers       map[string]string `json:"tiers"`       // 属性档位 {"damage":"S","range":"B","atkSpeed":"B"}
	SpriteKey   string            `json:"spriteKey"`   // 精灵键名
	BuildCost   int               `json:"buildCost"`   // 建造费用
	Description string            `json:"description"` // 简短描述
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
func LoadClassicPresets() error {
	if dataFS == nil {
		return fmt.Errorf("load classic-presets: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/towers/classic-presets.json")
	if err != nil {
		return fmt.Errorf("load classic-presets: %w", err)
	}
	var cfg ClassicPresetsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse classic-presets: %w", err)
	}
	globalClassicPresets = &cfg
	log.Printf("[config] loaded %d classic tower presets", len(cfg.Towers))
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
