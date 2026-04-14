// settings_config.go — 全局设置配置加载。
// 从 settings.json 读取难度等配置。
package config

import "encoding/json"

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
// 返回 modes map 和默认难度 ID。加载失败时回退到硬编码默认值。
func LoadDifficultyModes() (map[string]DifficultyMode, string, error) {
	if dataFS == nil {
		modes, def := defaultDifficultyModes()
		return modes, def, nil
	}
	data, err := dataFS.ReadFile("config/settings.json")
	if err != nil {
		modes, def := defaultDifficultyModes()
		return modes, def, nil
	}
	var s settingsJSON
	if err := json.Unmarshal(data, &s); err != nil {
		modes, def := defaultDifficultyModes()
		return modes, def, nil
	}
	return s.Difficulty.Modes, s.Difficulty.Default, nil
}

// defaultDifficultyModes 返回硬编码的难度配置（与 settings.json 一致）。
// 用于 JSON 加载失败时的 fallback，避免返回 error 导致游戏无法启动。
func defaultDifficultyModes() (map[string]DifficultyMode, string) {
	return map[string]DifficultyMode{
		"easy": {
			Label: "简单", HPScale: 0.7, SpeedScale: 0.85,
			RewardScale: 1.3, StartGold: 180, StartingLives: 25,
		},
		"normal": {
			Label: "普通", HPScale: 1, SpeedScale: 1,
			RewardScale: 1, StartGold: 120, StartingLives: 20,
		},
		"hard": {
			Label: "困难", HPScale: 1.4, SpeedScale: 1.15,
			RewardScale: 0.8, StartGold: 100, StartingLives: 15,
		},
		"extreme": {
			Label: "极限", HPScale: 2, SpeedScale: 1.3,
			RewardScale: 0.6, StartGold: 80, StartingLives: 10,
		},
	}, "normal"
}
