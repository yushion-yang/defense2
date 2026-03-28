// tower_config.go — 塔配置数据结构与加载。
// 从 JSON 文件（towers.json、towers-core.json）反序列化塔定义，
// 转换为运行时可用的 TowerDef 列表。
package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TowerJSON 塔的 JSON 配置原始结构（与 JS 版 JSON 字段一致）。
type TowerJSON struct {
	Label         string          `json:"label"`
	ShortLabel    string          `json:"shortLabel"`
	Faction       string          `json:"faction"`
	BuildCost     int             `json:"buildCost"`
	BaseRange     float64         `json:"baseRange"`
	BaseDamage    float64         `json:"baseDamage"`
	BaseFireRate  float64         `json:"baseFireRate"` // 秒/次（越小越快）
	Tags          []string        `json:"tags"`
	Abilities     []TowerAbilJSON `json:"abilities"`
}

// TowerAbilJSON 塔能力的 JSON 原始结构。
type TowerAbilJSON struct {
	Name string `json:"name"`
}

// TowerFileData 塔配置文件内容（key→塔定义 + _meta 元数据）。
type TowerFileData struct {
	Meta   TowerFileMeta          // _meta 字段
	Towers map[string]*TowerJSON  // key → 塔定义
}

// TowerFileMeta 塔配置文件的元数据。
type TowerFileMeta struct {
	Faction     string `json:"faction"`
	FactionName string `json:"factionName"`
}

// LoadTowerFile 加载一个塔配置 JSON 文件。
func LoadTowerFile(path string) (*TowerFileData, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load tower %s: dataFS not initialized", path)
	}
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load tower %s: %w", path, err)
	}

	// 先解析为 map[string]json.RawMessage 以分离 _meta
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse tower %s: %w", path, err)
	}

	result := &TowerFileData{
		Towers: make(map[string]*TowerJSON),
	}

	// 解析 _meta
	if metaRaw, ok := raw["_meta"]; ok {
		if err := json.Unmarshal(metaRaw, &result.Meta); err != nil {
			return nil, fmt.Errorf("parse tower _meta %s: %w", path, err)
		}
		delete(raw, "_meta")
	}

	// 解析各塔定义
	for key, val := range raw {
		if strings.HasPrefix(key, "_") || key == "?" {
			continue // 跳过元数据和占位键
		}
		var t TowerJSON
		if err := json.Unmarshal(val, &t); err != nil {
			return nil, fmt.Errorf("parse tower %s/%s: %w", path, key, err)
		}
		// 若塔自身未指定 faction，继承文件级 faction
		if t.Faction == "" {
			t.Faction = result.Meta.Faction
		}
		result.Towers[key] = &t
	}

	return result, nil
}

// LoadAllTowers 加载所有塔配置文件并合并。
func LoadAllTowers() (map[string]*TowerJSON, error) {
	all := make(map[string]*TowerJSON)

	files := []string{
		"config/towers/towers-core.json",
		"config/towers/towers.json",
	}
	for _, f := range files {
		fd, err := LoadTowerFile(f)
		if err != nil {
			return nil, err
		}
		for k, v := range fd.Towers {
			all[k] = v
		}
	}
	return all, nil
}
