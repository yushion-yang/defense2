// warden_config.go — 战灵配置加载。
// 支持目录模式（config/wardens/defs/{key}.json）和单文件模式（wardens.json）。
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
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
// 优先从目录模式加载（config/wardens/defs/），不存在时回退到单文件模式。
func LoadWardenConfigs() (map[string]WardenConfig, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("dataFS not initialized")
	}

	configs := make(map[string]WardenConfig)

	// 尝试目录模式
	if entries, err := dataFS.ReadDir("config/wardens/defs"); err == nil && len(entries) > 0 {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".json")
			if strings.HasPrefix(name, "_") {
				continue
			}
			data, err := dataFS.ReadFile(filepath.Join("config/wardens/defs", entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read warden %s: %w", entry.Name(), err)
			}
			var cfg WardenConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parse warden %s: %w", entry.Name(), err)
			}
			configs[name] = cfg
		}
		return configs, nil
	}

	// 回退到单文件模式
	data, err := dataFS.ReadFile("config/wardens/wardens.json")
	if err != nil {
		return nil, fmt.Errorf("read wardens config: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse wardens config: %w", err)
	}

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
