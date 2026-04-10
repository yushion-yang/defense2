// settings_config.go — 全局设置配置加载。
// 从 settings.json 读取难度等配置。
package config

import (
	"encoding/json"
	"fmt"
)

// DifficultyMode 单个难度模式的配置。
type DifficultyMode struct {
	Label         string  `json:"label"`
	HPScale       float64 `json:"hpScale"`
	SpeedScale    float64 `json:"speedScale"`
	RewardScale   float64 `json:"rewardScale"`
	StartGold     int     `json:"startGold"`
	StartingLives int     `json:"startingLives"`
}

// settingsJSON settings.json 的顶层结构（只解析需要的字段）。
type settingsJSON struct {
	Difficulty struct {
		Modes   map[string]DifficultyMode `json:"modes"`
		Default string                    `json:"default"`
	} `json:"difficulty"`
}

// LoadDifficultyModes 从 settings.json 加载难度模式配置。
// 返回 modes map 和默认难度 ID。
func LoadDifficultyModes() (map[string]DifficultyMode, string, error) {
	if dataFS == nil {
		return nil, "", fmt.Errorf("load settings: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/settings.json")
	if err != nil {
		return nil, "", fmt.Errorf("load settings: %w", err)
	}
	var s settingsJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, "", fmt.Errorf("parse settings: %w", err)
	}
	return s.Difficulty.Modes, s.Difficulty.Default, nil
}
