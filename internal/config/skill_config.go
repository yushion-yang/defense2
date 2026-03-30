// skill_config.go — 技能配置加载。
package config

import (
	"encoding/json"
	"fmt"
)

// SkillConfig 单个技能的配置。
type SkillConfig struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Cooldown    float64 `json:"cooldown"`
	DamageRatio float64 `json:"damageRatio"`
	Duration    float64 `json:"duration"`
	Width       float64 `json:"width"`
	RangeBonus  float64 `json:"rangeBonus"`
	HitInterval float64 `json:"hitInterval"`
}

// LoadSkillConfigs 加载所有技能配置。
func LoadSkillConfigs() (map[string]SkillConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/skills/skills.json")
	if err != nil {
		return nil, fmt.Errorf("read skills config: %w", err)
	}

	var arr []SkillConfig
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("parse skills config: %w", err)
	}

	configs := make(map[string]SkillConfig, len(arr))
	for _, c := range arr {
		if c.Key != "" {
			configs[c.Key] = c
		}
	}
	return configs, nil
}
