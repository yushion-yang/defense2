// tower_config.go — 塔配置数据结构与加载。
// 支持两种加载方式：
//  1. 单文件模式：一个 JSON 包含多个塔 { "_meta": {}, "key1": {}, "key2": {} }
//  2. 目录模式：每个塔一个 JSON 文件 config/towers/defs/{key}.json
//
// 优先使用目录模式，不存在时回退到单文件模式。
package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TowerJSON 塔的 JSON 配置原始结构（与 JS 版 JSON 字段一致）。
type TowerJSON struct {
	Label       string `json:"label"`       // 塔全名
	ShortLabel  string `json:"shortLabel"`  // 塔简称（HUD 显示用）
	Description string `json:"description"` // 塔描述文本
	BuildCost   int    `json:"buildCost"`   // 建造费用（金币）

	// 基础属性 + 潜力属性（战力缩放）
	BaseDamage         float64 `json:"baseDamage"`         // 基础伤害（强度0时的底线）
	PotentialDamage    float64 `json:"potentialDamage"`    // 潜力伤害（强度100时 = base+potential）
	BaseAttackSpeed    float64 `json:"baseAttackSpeed"`    // 基础攻速（次/秒）
	PotentialAttackSpeed float64 `json:"potentialAttackSpeed"` // 潜力攻速
	BaseRange          float64 `json:"baseRange"`          // 基础射程（像素）
	PotentialRange     float64 `json:"potentialRange"`     // 潜力射程

	Abilities      []string            `json:"abilities"`      // 能力 key 列表（引用 abilities.json）
	// 攻击方式配置
	AttackStyle     string  `json:"attackStyle"`     // "projectile"/"wideBeam"/"scatter"/"spin_aoe"
	ProjectileSpeed float64 `json:"projectileSpeed"` // 弹射物速度（px/s）

	// 升级系统
	UpgradeCosts []int `json:"upgradeCosts"` // 每次升级费用（长度=最大升级次数）
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

// globalTowerTable 全局塔配置缓存。
var globalTowerTable map[string]*TowerJSON

// GlobalTowerTable 返回全局缓存的塔配置表。
func GlobalTowerTable() map[string]*TowerJSON {
	return globalTowerTable
}

// LoadAllTowers 加载所有塔配置（从 towers.json）并缓存。
func LoadAllTowers() (map[string]*TowerJSON, error) {
	fd, err := LoadTowerFile("config/towers/towers.json")
	if err != nil {
		return nil, err
	}
	globalTowerTable = fd.Towers
	return fd.Towers, nil
}
