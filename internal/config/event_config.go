// event_config.go — 事件配置加载。
// 从 events-ally-v2.json 和 events-enemy-v2.json 加载事件定义。
package config

import (
	"encoding/json"
	"fmt"
)

// EventJSON 事件的 JSON 原始结构。
type EventJSON struct {
	ID          string  `json:"id"`          // 事件唯一标识
	Label       string  `json:"label"`       // 显示名称
	Description string  `json:"description"` // 效果描述
	Kind        string  `json:"kind"`        // 处理器类型标识
	Tier        int     `json:"tier"`        // 等级（1=经济, 2=全局）
	Value       float64 `json:"value"`       // 效果数值
	Effect      string  `json:"effect"`      // 子效果类型
	Weight      float64 `json:"weight"`      // 抽取权重
	MinWave     int     `json:"minWave"`     // 最早出现波次
	MaxWave     int     `json:"maxWave"`     // 最晚出现波次
}

// LoadAllyEvents 加载增益事件列表。
func LoadAllyEvents() ([]EventJSON, error) {
	return loadEventFile("config/events/events-ally-v2.json")
}

// LoadEnemyEvents 加载减益事件列表。
func LoadEnemyEvents() ([]EventJSON, error) {
	return loadEventFile("config/events/events-enemy-v2.json")
}

func loadEventFile(path string) ([]EventJSON, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load events %s: dataFS not initialized", path)
	}
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load events %s: %w", path, err)
	}
	var events []EventJSON
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parse events %s: %w", path, err)
	}
	return events, nil
}
