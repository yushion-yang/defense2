// tier_presets.go — 塔属性档位预设加载。
// 从 tier-presets.json 读取 S/A/B/C/D 五档属性范围。
package config

import (
	"encoding/json"
	"fmt"
)

// TierRange 单个档位的数值范围。
type TierRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Ref float64 `json:"ref"`
}

// AttrTiers 单个属性的五档预设。
type AttrTiers struct {
	Tiers map[string]TierRange `json:"tiers"` // "S"/"A"/"B"/"C"/"D"
}

// TierPresets 完整预设表。
type TierPresets struct {
	Damage      AttrTiers `json:"damage"`
	AttackSpeed AttrTiers `json:"attackSpeed"`
	Range       AttrTiers `json:"range"`
}

var globalTierPresets *TierPresets

// GlobalTierPresets 返回全局预设表。
func GlobalTierPresets() *TierPresets { return globalTierPresets }

// TierNames 五档名称（从高到低）。
var TierNames = []string{"S", "A", "B", "C", "D"}

// LoadTierPresets 加载档位预设。
func LoadTierPresets() (*TierPresets, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load tier-presets: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/towers/tier-presets.json")
	if err != nil {
		return nil, fmt.Errorf("load tier-presets: %w", err)
	}
	var raw struct {
		Damage      AttrTiers `json:"damage"`
		AttackSpeed AttrTiers `json:"attackSpeed"`
		Range       AttrTiers `json:"range"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse tier-presets: %w", err)
	}
	tp := &TierPresets{
		Damage:      raw.Damage,
		AttackSpeed: raw.AttackSpeed,
		Range:       raw.Range,
	}
	globalTierPresets = tp
	return tp, nil
}
