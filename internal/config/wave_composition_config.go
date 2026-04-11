// wave_composition_config.go — 波次出怪组合配置加载。
// 从 config/systems/wave-compositions.json 读取按波次阶段定义的原型权重分布。
package config

import (
	"encoding/json"
	"fmt"
	"sort"
)

// WaveComposition 波次阶段出怪配置。
type WaveComposition struct {
	MaxWave int            `json:"maxWave"` // 该阶段适用的最大波次号（0=无上限）
	Enemies map[string]int `json:"enemies"` // archetype → weight
}

// SortedEnemies 返回按原型名字母序排列的 (archetype, weight) 列表。
// 用于确定性迭代（map 遍历顺序不稳定）。
func (wc *WaveComposition) SortedEnemies() [][2]interface{} {
	keys := make([]string, 0, len(wc.Enemies))
	for k := range wc.Enemies {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([][2]interface{}, len(keys))
	for i, k := range keys {
		result[i] = [2]interface{}{k, wc.Enemies[k]}
	}
	return result
}

// waveCompositionFile 对应 JSON 文件顶层结构。
type waveCompositionFile struct {
	Compositions []WaveComposition `json:"compositions"`
}

// globalWaveCompositions 全局缓存。
var globalWaveCompositions []WaveComposition

// GlobalWaveCompositions 返回全局波次组合配置。LoadWaveCompositions 成功后可用。
func GlobalWaveCompositions() []WaveComposition {
	return globalWaveCompositions
}

// LoadWaveCompositions 从 config/systems/wave-compositions.json 加载波次组合配置。
// 必须在 SetDataFS() 之后调用。
func LoadWaveCompositions() error {
	if dataFS == nil {
		return fmt.Errorf("load wave compositions: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/systems/wave-compositions.json")
	if err != nil {
		return fmt.Errorf("load wave compositions: %w", err)
	}
	var f waveCompositionFile
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse wave compositions: %w", err)
	}
	if len(f.Compositions) == 0 {
		return fmt.Errorf("wave compositions: empty compositions array")
	}
	globalWaveCompositions = f.Compositions
	return nil
}

// WaveCompositionArchetypes 返回波次组合中引用的所有原型名（去重）。
// 用于契约测试验证原型存在于 enemies-core.json。
func WaveCompositionArchetypes() []string {
	seen := make(map[string]bool)
	var result []string
	for _, c := range globalWaveCompositions {
		for arch := range c.Enemies {
			if !seen[arch] {
				seen[arch] = true
				result = append(result, arch)
			}
		}
	}
	sort.Strings(result)
	return result
}
