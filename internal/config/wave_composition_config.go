// wave_composition_config.go — 波次出怪组合类型定义和辅助函数。
// 波次组合数据现由 config/systems/spawner.json 提供（通过 spawner_config.go 加载）。
// 本文件保留 WaveComposition 类型和辅助工具，以及向后兼容的访问函数。
package config

import "sort"

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

// globalWaveCompositions 全局缓存（由 LoadSpawnerConfig 写入）。
var globalWaveCompositions []WaveComposition

// GlobalWaveCompositions 返回全局波次组合配置。
// 数据来自 spawner.json 的 compositions 区段，由 LoadSpawnerConfig 加载。
func GlobalWaveCompositions() []WaveComposition {
	return globalWaveCompositions
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
