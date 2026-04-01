// warden_config.go — 战灵配置加载。
// 支持目录模式（config/wardens/defs/{key}.json）和单文件模式（wardens.json）。
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// WardenConfig 单个战灵类型的完整配置（扁平结构，表格化数组格式）。
type WardenConfig struct {
	Key         string `json:"key"` // 类型标识（prince/core/chain/skystrike/envoy）
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Category    string `json:"category"` // "mobile" or "indirect"

	// 基础属性
	Damage         float64 `json:"damage"`
	AttackInterval float64 `json:"attackInterval"`
	Range          float64 `json:"range"`
	MoveSpeed      float64 `json:"moveSpeed"`

	// 行为描述（UI 展示用）
	AttackName   string `json:"attackName"`
	AttackDesc   string `json:"attackDesc"`
	SpecialName  string `json:"specialName"`
	SpecialDesc  string `json:"specialDesc"`
	StrengthDesc string `json:"strengthDesc"`
	CounterTip   string `json:"counterTip"`

	// 成长
	GrowthOnKill      float64 `json:"growthOnKill"`
	GrowthOnWaveClear float64 `json:"growthOnWaveClear"`
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

	// 回退到单文件模式（表格化数组格式）
	data, err := dataFS.ReadFile("config/wardens/wardens.json")
	if err != nil {
		return nil, fmt.Errorf("read wardens config: %w", err)
	}

	// 尝试数组格式（表格化）
	var arr []WardenConfig
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		for _, cfg := range arr {
			if cfg.Key != "" {
				configs[cfg.Key] = cfg
			}
		}
		return configs, nil
	}

	// 兼容旧 map 格式
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
		cfg.Key = key
		configs[key] = cfg
	}
	return configs, nil
}
