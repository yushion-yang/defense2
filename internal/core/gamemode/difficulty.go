// difficulty.go — 难度配置加载。
package gamemode

import (
	"defense2/internal/config"
	"log"
)

// Difficulty 难度倍率配置。
type Difficulty struct {
	HPScale     float64 // 敌人 HP 倍率
	SpeedScale  float64 // 敌人速度倍率
	RewardScale float64 // 击杀/波次奖励倍率
	StartGold   int     // 初始金币
}

// DefaultDifficulty 返回 normal 难度。
func DefaultDifficulty() Difficulty {
	return Difficulty{
		HPScale:     1.0,
		SpeedScale:  1.0,
		RewardScale: 1.0,
		StartGold:   120,
	}
}

// LoadDifficulty 按 ID 加载难度配置，未找到返回 normal。
func LoadDifficulty(id string) Difficulty {
	if id == "" {
		id = "normal"
	}
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		log.Printf("难度配置加载失败，使用默认: %v", err)
		return DefaultDifficulty()
	}
	dm, ok := modes[id]
	if !ok {
		return DefaultDifficulty()
	}
	return Difficulty{
		HPScale:     dm.HPScale,
		SpeedScale:  dm.SpeedScale,
		RewardScale: dm.RewardScale,
		StartGold:   dm.StartGold,
	}
}
