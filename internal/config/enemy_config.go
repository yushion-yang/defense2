// enemy_config.go — 敌人配置数据结构与加载。
// 支持目录模式（config/enemies/defs/{key}.json）和单文件模式（enemies-core.json）。
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// EnemyArchetype 敌人原型模板（JSON 配置）。
type EnemyArchetype struct {
	Label            string  `json:"label"`            // 显示名称
	Color            string  `json:"color"`            // 显示颜色（hex）
	Description      string  `json:"description"`      // 描述文本
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
	AuraArmor        float64 `json:"auraArmor"`        // 光环护甲值
}

// LoadEnemyArchetypes 加载所有敌人原型。
// 优先从目录模式加载（config/enemies/defs/），不存在时回退到单文件模式。
func LoadEnemyArchetypes() (map[string]*EnemyArchetype, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load enemies: dataFS not initialized")
	}

	result := make(map[string]*EnemyArchetype)

	// 尝试目录模式
	if entries, err := dataFS.ReadDir("config/enemies/defs"); err == nil && len(entries) > 0 {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".json")
			if strings.HasPrefix(name, "_") {
				continue
			}
			data, err := dataFS.ReadFile(filepath.Join("config/enemies/defs", entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read enemy %s: %w", entry.Name(), err)
			}
			var a EnemyArchetype
			if err := json.Unmarshal(data, &a); err != nil {
				return nil, fmt.Errorf("parse enemy %s: %w", entry.Name(), err)
			}
			applyEnemyDefaults(&a)
			result[name] = &a
		}
		return result, nil
	}

	// 回退到单文件模式
	data, err := dataFS.ReadFile("config/enemies/enemies-core.json")
	if err != nil {
		return nil, fmt.Errorf("load enemies: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse enemies: %w", err)
	}

	for key, val := range raw {
		if strings.HasPrefix(key, "_") {
			continue
		}
		var a EnemyArchetype
		if err := json.Unmarshal(val, &a); err != nil {
			continue
		}
		applyEnemyDefaults(&a)
		result[key] = &a
	}
	return result, nil
}

// applyEnemyDefaults 为缺省字段设置默认值。
// 注意：SpeedScale=0 是合法值（dummy 原型不移动），不做默认覆盖。
func applyEnemyDefaults(a *EnemyArchetype) {
	// HPScale 和 SpeedScale 由 JSON 显式指定，不设默认（0=合法值）
	if a.Radius <= 0 {
		a.Radius = 8
	}
	if a.RewardScale == 0 {
		a.RewardScale = 1
	}
}
