// tier_presets.go — 塔属性档位预设加载。
// 从 tier-presets.json 读取 S/B/D 三档属性值 + 共享潜力基数。
package config

import (
	"encoding/json"
	"fmt"
)

// TierValue 单个档位的基础值和潜力。
type TierValue struct {
	Base      float64 `json:"base"`
	Potential float64 `json:"potential"`
}

// AttrTiers 单个属性的档位预设。
type AttrTiers struct {
	BasePotential float64              `json:"basePotential"` // 共享潜力基数
	Tiers         map[string]TierValue `json:"tiers"`         // "S"/"B"/"D"
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

// TierNames 三档名称。
var TierNames = []string{"S", "B", "D"}

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
