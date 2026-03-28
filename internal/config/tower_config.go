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
	Label     string `json:"label"`     // 塔全名
	ShortLabel string `json:"shortLabel"` // 塔简称（HUD 显示用）
	BuildCost int    `json:"buildCost"` // 建造费用（金币）

	// 基础属性 + 潜力属性（战力缩放）
	BaseDamage         float64 `json:"baseDamage"`         // 基础伤害（强度0时的底线）
	PotentialDamage    float64 `json:"potentialDamage"`    // 潜力伤害（强度100时 = base+potential）
	BaseAttackSpeed    float64 `json:"baseAttackSpeed"`    // 基础攻速（次/秒）
	PotentialAttackSpeed float64 `json:"potentialAttackSpeed"` // 潜力攻速
	BaseRange          float64 `json:"baseRange"`          // 基础射程（像素）
	PotentialRange     float64 `json:"potentialRange"`     // 潜力射程

	// 战力效果绑定（能力参数的强度缩放，如减速倍率）
	StrengthEffects *StrengthEffectsJSON `json:"strengthEffects"`

	Tags           []string            `json:"tags"`           // 标签列表（如 "energy"、"laser"）
	Abilities      []string            `json:"abilities"`      // 能力 key 列表（引用 abilities.json）
	BounceConfig   *BounceConfigJSON   `json:"bounceConfig"`   // 弹射配置（electric 塔）
	// 攻击方式配置
	AttackStyle      string             `json:"attackStyle"`      // "projectile"/"laser"/"wideBeam"/"scatter"/"charge"/"spin_aoe"/"pierce"/"aura_dot"
	ProjectileSpeed  float64            `json:"projectileSpeed"`  // 弹射物速度（px/s）
	Beam             *BeamConfigJSON    `json:"beam"`             // laser/wideBeam 光束配置
	ScatterConfig    *ScatterConfigJSON `json:"scatterConfig"`    // scatter 散射配置
	ChargeConfig     *ChargeConfigJSON  `json:"chargeConfig"`     // charge 蓄力配置
	InnerDamageBonus float64            `json:"innerDamageBonus"` // spin_aoe 内圈加伤倍率
	InnerRadiusRatio float64            `json:"innerRadiusRatio"` // spin_aoe 内圈比例
	PierceConfig     *PierceConfigJSON  `json:"pierceConfig"`     // pierce 穿刺配置

	PoisonConfig *PoisonConfigJSON `json:"poisonConfig"` // aura_dot 持续毒伤配置
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

// BindingJSON 基础值+潜力值（用于 effects 内的嵌套绑定）。
type BindingJSON struct {
	Base      float64 `json:"base"`      // 基础值
	Potential float64 `json:"potential"` // 潜力值
}

// StrengthEffectsJSON 能力效果的战力绑定。
type StrengthEffectsJSON struct {
	SlowFactor         *BindingJSON `json:"slowFactor"`         // 减速倍率
	PercentHp          *BindingJSON `json:"percentHp"`          // 百分比HP伤害
	ExecutionThreshold *BindingJSON `json:"executionThreshold"` // 斩杀阈值
	SplashRadius       *BindingJSON `json:"splashRadius"`       // 溅射半径
	BurnDps            *BindingJSON `json:"burnDps"`            // 灼烧DPS
	BleedDps           *BindingJSON `json:"bleedDps"`           // 流血DPS
	StunDuration       *BindingJSON `json:"stunDuration"`       // 眩晕时长
	BounceRange        *BindingJSON `json:"bounceRange"`        // 弹射范围
}

// BounceConfigJSON 弹射配置。
type BounceConfigJSON struct {
	BaseBounces int     `json:"baseBounces"`
	Range       float64 `json:"range"`
	DamageDecay float64 `json:"damageDecay"`
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

// LoadAllTowers 加载所有塔配置（从 towers.json）。
func LoadAllTowers() (map[string]*TowerJSON, error) {
	fd, err := LoadTowerFile("config/towers/towers.json")
	if err != nil {
		return nil, err
	}
	return fd.Towers, nil
}
