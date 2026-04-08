// scenario_config.go — 测试场景快照的数据结构与加载。
// 支持保存/恢复完整的塔布局和场景配置，用于测试模式的自定义场景。
package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ScenarioData describes a reusable test scenario (tower layout + scene config).
type ScenarioData struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	MapID       string          `json:"mapID"`
	Gold        int             `json:"gold"`
	Lives       int             `json:"lives"`
	Waves       int             `json:"waves"`
	EnemyFilter string          `json:"enemyFilter"`
	ManualWave  bool            `json:"manualWave"`
	Towers      []TowerSnapshot `json:"towers"`
}

// TowerSnapshot captures a placed tower's full state for scenario restore.
type TowerSnapshot struct {
	Row             int       `json:"row"`
	Col             int       `json:"col"`
	Key             string    `json:"key"`
	AbilitySlots    [6]string `json:"abilitySlots"`
	DamageTier      string    `json:"damageTier"`
	SpeedTier       string    `json:"speedTier"`
	RangeTier       string    `json:"rangeTier"`
	BaseDamage      float64   `json:"baseDamage"`
	PotentialDamage float64   `json:"potentialDamage"`
	BaseSpeed       float64   `json:"baseSpeed"`
	PotentialSpeed  float64   `json:"potentialSpeed"`
	BaseRange       float64   `json:"baseRange"`
	PotentialRange  float64   `json:"potentialRange"`
}

// ParseScenarioData parses a single scenario JSON.
func ParseScenarioData(data []byte) (*ScenarioData, error) {
	var sd ScenarioData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	return &sd, nil
}

// LoadScenarios loads all scenario JSON files from config/scenarios/.
func LoadScenarios() ([]*ScenarioData, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load scenarios: dataFS not initialized")
	}
	dir := "config/scenarios"
	entries, err := fs.ReadDir(dataFS, dir)
	if err != nil {
		return nil, nil // directory doesn't exist = no custom scenarios
	}
	var results []*ScenarioData
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := dataFS.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sd, err := ParseScenarioData(data)
		if err != nil {
			continue
		}
		results = append(results, sd)
	}
	return results, nil
}
