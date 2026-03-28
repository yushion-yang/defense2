// warden_config.go — 战灵配置加载。
// 从 config/wardens/wardens.json 读取所有战灵定义。
package config

import (
	"encoding/json"
	"fmt"
)

// WardenConfig 单个战灵类型的完整配置。
type WardenConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"` // "mobile" or "indirect"
	EntityCount int           `json:"entityCount"`
	Levels      []WardenLevel `json:"levels"`
	Growth      WardenGrowth  `json:"growth"`
}

// WardenLevel 战灵某一等级的属性。
type WardenLevel struct {
	Level                  int     `json:"level"`
	Damage                 float64 `json:"damage"`
	AttackInterval         float64 `json:"attackInterval"`
	Range                  float64 `json:"range"`
	MoveSpeed              float64 `json:"moveSpeed"`
	AoERadius              float64 `json:"aoeRadius"`
	EffectDPS              float64 `json:"effectDps"`
	EffectDuration         float64 `json:"effectDuration"`
	PossessDuration        float64 `json:"possessDuration"`
	Cooldown               float64 `json:"cooldown"`
	PermanentStrengthGrant float64 `json:"permanentStrengthGrant"`
}

// WardenGrowth 战灵成长配置。
type WardenGrowth struct {
	OnKill      float64 `json:"onKill"`
	OnWaveClear float64 `json:"onWaveClear"`
}

// LoadWardenConfigs 加载所有战灵配置。
// 返回 map[wardenType]WardenConfig，跳过 _meta 键。
func LoadWardenConfigs() (map[string]WardenConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/wardens/wardens.json")
	if err != nil {
		return nil, fmt.Errorf("read wardens config: %w", err)
	}

	// 先解析为 map[string]json.RawMessage 以跳过 _meta
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse wardens config: %w", err)
	}

	configs := make(map[string]WardenConfig)
	for key, val := range raw {
		if key == "_meta" {
			continue
		}
		var cfg WardenConfig
		if err := json.Unmarshal(val, &cfg); err != nil {
			return nil, fmt.Errorf("parse warden %q: %w", key, err)
		}
		configs[key] = cfg
	}
	return configs, nil
}
