// enemy_config.go — 敌人原型配置与能力系统的加载层。
//
// 数据来源：
//   - 原型定义：config/enemies/defs/{key}.json（目录模式）或 enemies-core.json（单文件模式）
//   - 能力定义：config/enemies/abilities.json（全局能力表，怪物原型通过引用装配）
//
// 双模式加载设计：
//   - 目录模式优先（每个原型一个文件，便于版本管理和 diff）
//   - 单文件模式作为回退（数组格式保留定义顺序，用于 UI 展示排序）
//   - 两种模式互斥，不会混合加载
//
// 能力装配机制：
//   - 原型 JSON 的 "abilities" 字段支持混合写法：字符串（用默认参数）或对象（覆盖参数）
//   - 例: ["stealth", {"type":"healAura","base":0.24,"param":105}]
//   - ResolveEnemyAbilityAtWave() 在运行时合并默认参数+覆盖参数+波次缩放
//   - 波次缩放公式：effectiveBase = base + potential * wave（让能力随波次增强）
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// EnemyArchetype 敌人原型模板（JSON 配置）。
// 每种原型定义一套基础属性倍率，运行时与波次缩放参数相乘得到实际属性。
// 18 种原型覆盖不同战术定位：normal(标准)、tank(肉盾)、fast(速攻)、healer(治疗)等。
type EnemyArchetype struct {
	ID          string  `json:"id"`          // 原型标识
	Label       string  `json:"label"`       // 显示名称
	Color       string  `json:"color"`       // 显示颜色（hex）
	Sprite      string  `json:"sprite"`      // 精灵目录名（assets/enemies/sprites/{sprite}/）
	HPScale     float64 `json:"hpScale"`     // 血量倍率（相对基准值）
	SpeedScale  float64 `json:"speedScale"`  // 速度倍率
	Radius      float64 `json:"radius"`      // 碰撞半径（像素绝对值）
	RewardScale float64 `json:"rewardScale"` // 击杀奖励倍率
	Boss        bool    `json:"boss"`        // 是否为 Boss

	// 能力装配（能力驱动行为，取代旧的硬编码字段）
	// 支持两种写法：字符串（用默认参数）或对象（覆盖参数）
	// 例: ["stealth"] 或 [{"type":"healAura","base":0.24,"param":105}]
	RawAbilities []json.RawMessage `json:"abilities"`
	Abilities    []EnemyAbilityRef `json:"-"` // 解析后的能力引用列表
}

// EnemyAbilityRef 怪物装配的能力引用（可覆盖默认参数）。
type EnemyAbilityRef struct {
	Type      string  // 能力类型标识
	Base      float64 // 覆盖 base（0=用默认）
	Potential float64 // 覆盖 potential（0=用默认）
	Param     float64 // 覆盖 param（0=用默认）
}

// EnemyAbilityDef 怪物能力定义（从 abilities.json 加载）。
type EnemyAbilityDef struct {
	Type        string  `json:"type"`
	Label       string  `json:"label"`
	Icon        string  `json:"icon"`
	Category    string  `json:"category"`
	ScaleDim    string  `json:"scaleDim"`
	Base        float64 `json:"base"`
	Potential   float64 `json:"potential"`
	Param       float64 `json:"param"`
	ParamDim    string  `json:"paramDim"`
	Param2      float64 `json:"param2"`
	Param2Dim   string  `json:"param2Dim"`
	SpawnArch   string  `json:"spawnArch"` // deathSpawn 子体原型（默认 "normal"）
	Description string  `json:"description"`
	Visual      string  `json:"visual"`      // 视觉效果描述
	Silenceable bool    `json:"silenceable"` // 是否可被沉默禁用
}

// LoadEnemyArchetypes 加载所有敌人原型。
// 优先从目录模式加载（config/enemies/defs/），不存在时回退到单文件模式。
//
// 加载流程：
//  1. 尝试读取 config/enemies/defs/ 目录 → 逐文件解析（跳过 _ 前缀）
//  2. 若目录不存在或为空 → 从 enemies-core.json 读取数组格式（保留定义顺序）
//  3. 每个原型调用 applyEnemyDefaults() 填充缺省值 + 解析能力引用
//  4. 结果写入 globalEnemyArchetypes 全局缓存
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

	// 回退到单文件模式（数组格式，天然保序）
	data, err := dataFS.ReadFile("config/enemies/enemies-core.json")
	if err != nil {
		return nil, fmt.Errorf("load enemies: %w", err)
	}

	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse enemies array: %w", err)
	}

	enemyArchetypeOrder = nil
	for _, raw := range list {
		var a EnemyArchetype
		if err := json.Unmarshal(raw, &a); err != nil {
			continue
		}
		if a.ID == "" || strings.HasPrefix(a.ID, "_") {
			continue
		}
		applyEnemyDefaults(&a)
		result[a.ID] = &a
		enemyArchetypeOrder = append(enemyArchetypeOrder, a.ID)
	}
	globalEnemyArchetypes = result
	return result, nil
}

// applyEnemyDefaults 为缺省字段设置默认值。
// 注意：SpeedScale=0 是合法值（dummy 原型不移动），不做默认覆盖。
func applyEnemyDefaults(a *EnemyArchetype) {
	if a.Radius <= 0 {
		a.Radius = 8
	}
	if a.RewardScale == 0 {
		a.RewardScale = 1
	}
	// 解析能力引用
	for _, raw := range a.RawAbilities {
		ref := parseAbilityRef(raw)
		if ref.Type != "" {
			a.Abilities = append(a.Abilities, ref)
		}
	}
}

// parseAbilityRef 解析能力引用：字符串 "stealth" 或对象 {"type":"healAura","base":0.24}。
func parseAbilityRef(raw json.RawMessage) EnemyAbilityRef {
	// 先尝试字符串
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return EnemyAbilityRef{Type: s}
	}
	// 再尝试对象
	var obj struct {
		Type      string  `json:"type"`
		Base      float64 `json:"base"`
		Potential float64 `json:"potential"`
		Param     float64 `json:"param"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Type != "" {
		return EnemyAbilityRef{Type: obj.Type, Base: obj.Base, Potential: obj.Potential, Param: obj.Param}
	}
	return EnemyAbilityRef{}
}

// globalEnemyArchetypes 全局敌人原型缓存（由 spawner 首次加载时填充）。
var globalEnemyArchetypes map[string]*EnemyArchetype

// GlobalEnemyArchetypes 返回全局缓存的敌人原型表。
func GlobalEnemyArchetypes() map[string]*EnemyArchetype {
	return globalEnemyArchetypes
}

// enemyArchetypeOrder JSON 中的原型 key 顺序。
var enemyArchetypeOrder []string

// EnemyArchetypeOrder 返回原型的 JSON 定义顺序。
func EnemyArchetypeOrder() []string { return enemyArchetypeOrder }

// enemyAbilityTable 全局怪物能力配置表。
var enemyAbilityTable map[string]*EnemyAbilityDef

// LoadEnemyAbilities 加载怪物能力配置表（config/enemies/abilities.json）。
// 能力表是全局共享的——多个原型可以引用同一种能力，通过 EnemyAbilityRef 覆盖参数。
// 文件不存在时返回 nil（无能力），不视为错误（兼容旧版配置）。
// _ 前缀的 key 被视为注释/禁用条目，跳过不加载。
func LoadEnemyAbilities() (map[string]*EnemyAbilityDef, error) {
	if dataFS == nil {
		return nil, fmt.Errorf("load enemy abilities: dataFS not initialized")
	}
	data, err := dataFS.ReadFile("config/enemies/abilities.json")
	if err != nil {
		return nil, nil // 文件不存在 = 无能力
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse enemy abilities: %w", err)
	}
	result := make(map[string]*EnemyAbilityDef)
	for key, val := range raw {
		if strings.HasPrefix(key, "_") {
			continue
		}
		var d EnemyAbilityDef
		if err := json.Unmarshal(val, &d); err != nil {
			continue
		}
		result[key] = &d
	}
	enemyAbilityTable = result
	return result, nil
}

// GlobalEnemyAbilityTable 返回全局怪物能力表。
func GlobalEnemyAbilityTable() map[string]*EnemyAbilityDef {
	return enemyAbilityTable
}

// GlobalEnemyAbilityDef 返回指定类型的怪物能力定义，不存在返回 nil。
func GlobalEnemyAbilityDef(abilType string) *EnemyAbilityDef {
	if enemyAbilityTable == nil {
		return nil
	}
	return enemyAbilityTable[abilType]
}

// ResolveEnemyAbility 解析能力引用，合并默认参数和覆盖参数（wave=0，不应用 potential）。
func ResolveEnemyAbility(ref EnemyAbilityRef) *EnemyAbilityDef {
	return ResolveEnemyAbilityAtWave(ref, 0)
}

// ResolveEnemyAbilityAtWave 解析能力引用并应用波次缩放。
//
// 合并优先级：ref 覆盖值 > abilities.json 默认值。
// 缩放公式：effectiveBase = base + potential * wave（wave=0 时不缩放）。
// 返回值是 def 的浅拷贝，修改不影响全局能力表。
func ResolveEnemyAbilityAtWave(ref EnemyAbilityRef, wave int) *EnemyAbilityDef {
	table := GlobalEnemyAbilityTable()
	if table == nil {
		return nil
	}
	def, ok := table[ref.Type]
	if !ok {
		return nil
	}
	// 覆盖参数
	resolved := *def
	if ref.Base != 0 {
		resolved.Base = ref.Base
	}
	if ref.Potential != 0 {
		resolved.Potential = ref.Potential
	}
	if ref.Param != 0 {
		resolved.Param = ref.Param
	}
	// 应用波次缩放：effectiveBase = base + potential * wave
	if wave > 0 && resolved.Potential != 0 {
		resolved.Base += resolved.Potential * float64(wave)
	}
	return &resolved
}

// ResolveEffectivePotential 返回能力引用的有效 potential 值（ref 覆盖 > def 默认）。
func ResolveEffectivePotential(ref EnemyAbilityRef) float64 {
	if ref.Potential != 0 {
		return ref.Potential
	}
	table := GlobalEnemyAbilityTable()
	if table == nil {
		return 0
	}
	def, ok := table[ref.Type]
	if !ok {
		return 0
	}
	return def.Potential
}
