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
	Label        string          `json:"label"`        // 塔全名
	ShortLabel   string          `json:"shortLabel"`   // 塔简称（HUD 显示用）
	BuildCost    int             `json:"buildCost"`    // 建造费用（金币）
	BaseRange    float64         `json:"baseRange"`    // 基础攻击范围（像素）
	BaseDamage   float64         `json:"baseDamage"`   // 基础单发伤害
	BaseFireRate float64         `json:"baseFireRate"` // 基础射击间隔（秒/次，越小越快）
	Tags         []string        `json:"tags"`         // 标签列表（如 "energy"、"laser"）
	Abilities    []TowerAbilJSON `json:"abilities"`    // 该塔拥有的能力列表
}

// TowerAbilJSON 塔能力的 JSON 原始结构。
type TowerAbilJSON struct {
	Name string `json:"name"` // 能力注册名称（对应 ability.Registry 中的键）
}

// TowerFileData 塔配置文件的完整解析结果。
type TowerFileData struct {
	Meta   TowerFileMeta         // 文件级元数据（_meta 字段）
	Towers map[string]*TowerJSON // 塔定义映射（key → 塔配置）
}

// TowerFileMeta 塔配置文件的元数据（来自 JSON 的 _meta 字段）。
type TowerFileMeta struct {
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
