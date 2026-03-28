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
	"path/filepath"
	"strings"
)

// TowerJSON 塔的 JSON 配置原始结构（与 JS 版 JSON 字段一致）。
type TowerJSON struct {
	Label          string              `json:"label"`          // 塔全名
	ShortLabel     string              `json:"shortLabel"`     // 塔简称（HUD 显示用）
	BuildCost      int                 `json:"buildCost"`      // 建造费用（金币）
	BaseRange      float64             `json:"baseRange"`      // 基础攻击范围（像素）
	BaseDamage     float64             `json:"baseDamage"`     // 基础单发伤害
	BaseFireRate   float64             `json:"baseFireRate"`   // 基础射击间隔（秒/次，越小越快）
	Tags           []string            `json:"tags"`           // 标签列表（如 "energy"、"laser"）
	Abilities      []TowerAbilJSON     `json:"abilities"`      // 该塔拥有的能力列表
	BounceConfig   *BounceConfigJSON   `json:"bounceConfig"`   // 弹射配置（electric 塔）
	AbilityUnlocks []AbilityUnlockJSON `json:"abilityUnlocks"` // 等级解锁能力

	// 攻击方式配置
	AttackStyle      string             `json:"attackStyle"`      // "projectile"/"laser"/"wideBeam"/"scatter"/"charge"/"spin_aoe"/"pierce"/"aura_dot"
	ProjectileSpeed  float64            `json:"projectileSpeed"`  // 弹射物速度（px/s）
	Beam             *BeamConfigJSON    `json:"beam"`             // laser/wideBeam 光束配置
	ScatterConfig    *ScatterConfigJSON `json:"scatterConfig"`    // scatter 散射配置
	ChargeConfig     *ChargeConfigJSON  `json:"chargeConfig"`     // charge 蓄力配置
	InnerDamageBonus float64            `json:"innerDamageBonus"` // spin_aoe 内圈加伤倍率
	InnerRadiusRatio float64            `json:"innerRadiusRatio"` // spin_aoe 内圈比例
	PierceConfig     *PierceConfigJSON  `json:"pierceConfig"`     // pierce 穿刺配置
	PoisonConfig     *PoisonConfigJSON  `json:"poisonConfig"`     // aura_dot 持续毒伤配置
}

// BeamConfigJSON 光束配置。
type BeamConfigJSON struct {
	Duration float64 `json:"duration"` // 显示时长（秒）
	Width    float64 `json:"width"`    // 宽度（像素）
	Color    string  `json:"color"`    // 颜色 hex（如 "#93c5fd"）
}

// ScatterConfigJSON 散射配置。
type ScatterConfigJSON struct {
	Pellets     int     `json:"pellets"`     // 弹丸数（默认 3）
	SpreadAngle float64 `json:"spreadAngle"` // 散射角度（度，默认 60）
}

// ChargeConfigJSON 蓄力配置。
type ChargeConfigJSON struct {
	DamageMultiplier float64 `json:"damageMultiplier"` // 蓄力伤害倍率（默认 3）
}

// PierceConfigJSON 穿刺配置。
type PierceConfigJSON struct {
	Targets int     `json:"targets"` // 最大穿透目标数（默认 2）
	Decay   float64 `json:"decay"`   // 每次穿透伤害衰减（默认 0.8）
}

// PoisonConfigJSON 持续毒伤配置。
type PoisonConfigJSON struct {
	DPS      float64 `json:"dps"`      // 每秒伤害
	Interval float64 `json:"interval"` // 伤害间隔（秒）
}

// TowerAbilJSON 塔能力的 JSON 原始结构。
type TowerAbilJSON struct {
	Name string `json:"type"` // 能力注册名称（JSON 中为 "type" 字段）
}

// BounceConfigJSON 弹射配置。
type BounceConfigJSON struct {
	BaseBounces int     `json:"baseBounces"`
	Range       float64 `json:"range"`
	DamageDecay float64 `json:"damageDecay"`
}

// AbilityUnlockJSON 等级解锁的能力。
type AbilityUnlockJSON struct {
	Level int    `json:"level"`
	Type  string `json:"type"`
	Name  string `json:"name"` // 显示名称（可选，回退到 Type）
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

// LoadTowerDir 从目录加载塔配置（每个塔一个 JSON 文件）。
// 文件名（不含后缀）作为塔的 key，_meta.json 和下划线开头的文件被跳过。
func LoadTowerDir(dirPath string) (*TowerFileData, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load tower dir %s: dataFS not initialized", dirPath)
	}
	entries, err := dataFS.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read tower dir %s: %w", dirPath, err)
	}

	result := &TowerFileData{
		Towers: make(map[string]*TowerJSON),
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		if strings.HasPrefix(name, "_") {
			continue // 跳过 _meta.json 等元数据文件
		}
		data, err := dataFS.ReadFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read tower %s/%s: %w", dirPath, entry.Name(), err)
		}
		var t TowerJSON
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parse tower %s/%s: %w", dirPath, entry.Name(), err)
		}
		result.Towers[name] = &t
	}
	return result, nil
}

// LoadAllTowers 加载所有塔配置并合并。
// 优先从目录模式加载（config/towers/core/ + config/towers/defs/），
// 目录不存在时回退到单文件模式。
func LoadAllTowers() (map[string]*TowerJSON, error) {
	all := make(map[string]*TowerJSON)

	// 目录模式
	dirs := []string{
		"config/towers/core",
		"config/towers/defs",
	}
	// 单文件回退
	files := []string{
		"config/towers/towers-core.json",
		"config/towers/towers.json",
	}

	for i, dir := range dirs {
		fd, err := LoadTowerDir(dir)
		if err == nil && len(fd.Towers) > 0 {
			for k, v := range fd.Towers {
				all[k] = v
			}
			continue
		}
		// 目录不存在或为空，回退单文件
		fd2, err2 := LoadTowerFile(files[i])
		if err2 != nil {
			return nil, err2
		}
		for k, v := range fd2.Towers {
			all[k] = v
		}
	}
	return all, nil
}
