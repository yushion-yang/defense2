// enemy_config.go — 敌人配置数据结构与加载。
// 从 enemies-core.json 加载 13 种敌人原型模板。
package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EnemyArchetype 敌人原型模板（JSON 配置）。
type EnemyArchetype struct {
	Label            string  `json:"label"`            // 显示名称
	HPScale          float64 `json:"hpScale"`          // 血量倍率（相对基准值）
	SpeedScale       float64 `json:"speedScale"`       // 速度倍率
	Radius           float64 `json:"radius"`           // 碰撞半径（像素绝对值）
	RewardScale      float64 `json:"rewardScale"`      // 击杀奖励倍率
	Boss             bool    `json:"boss"`             // 是否为 Boss
	ShieldScale      float64 `json:"shieldScale"`      // 护盾倍率（0 = 无护盾）
	SplitCount       int     `json:"splitCount"`       // 分裂数量（0 = 不分裂）
	MovementType     string  `json:"movementType"`     // 移动类型："ground" 或 "flying"
	StealthDuration  float64 `json:"stealthDuration"`  // 隐身持续时间（秒）
	TeleportInterval float64 `json:"teleportInterval"` // 瞬移间隔（秒）
	TeleportSkip     int     `json:"teleportSkipSegments"` // 瞬移跳过路径段数
	HealScale        float64 `json:"healScale"`        // 治疗量倍率
	HealRadius       float64 `json:"healRadius"`       // 治疗范围（像素）
	HealInterval     float64 `json:"healInterval"`     // 治疗间隔（秒）
	AuraRange        float64 `json:"auraRange"`        // 光环范围
	AuraSpeedUp      float64 `json:"auraSpeedUp"`      // 光环加速比例
}

// LoadEnemyArchetypes 从 enemies-core.json 加载所有敌人原型。
func LoadEnemyArchetypes() (map[string]*EnemyArchetype, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load enemies: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/enemies/enemies-core.json")
	if err != nil {
		return nil, fmt.Errorf("load enemies: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse enemies: %w", err)
	}

	result := make(map[string]*EnemyArchetype)
	for key, val := range raw {
		if strings.HasPrefix(key, "_") {
			continue
		}
		var a EnemyArchetype
		if err := json.Unmarshal(val, &a); err != nil {
			continue
		}
		// 默认值
		if a.HPScale == 0 {
			a.HPScale = 1
		}
		if a.SpeedScale == 0 {
			a.SpeedScale = 1
		}
		if a.Radius == 0 {
			a.Radius = 8
		}
		if a.RewardScale == 0 {
			a.RewardScale = 1
		}
		result[key] = &a
	}
	return result, nil
}
