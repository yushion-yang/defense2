// tower_loader.go — 塔配置→运行时定义转换器。
// 从 config 包读取 JSON 塔数据，转换为 tower.TowerDef 运行时结构。
// 独立为 loader 包以避免 config↔tower 循环依赖。
package loader

import (
	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// 阵营颜色映射表（faction → RGB）。
var factionColors = map[string][3]uint8{
	"base":   {80, 140, 220},
	"nature": {60, 180, 80},
	"order":  {200, 180, 60},
	"mech":   {140, 140, 160},
	"shadow": {120, 60, 160},
	"fire":   {220, 80, 40},
	"wind":   {100, 200, 200},
	"wood":   {80, 140, 60},
	"elf":    {180, 220, 120},
	"holy":   {240, 220, 140},
	"gold":   {220, 180, 40},
	"water":  {60, 120, 220},
}

// TowerJSONToDef 将单个 JSON 塔配置转为运行时 TowerDef。
func TowerJSONToDef(key string, t *config.TowerJSON) tower.TowerDef {
	// 攻击速度 = 1 / fireRate（fireRate 是秒/次，转为次/秒）
	attackSpeed := 1.0
	if t.BaseFireRate > 0 {
		attackSpeed = 1.0 / t.BaseFireRate
	}

	// 收集能力名称
	var abilities []string
	for _, a := range t.Abilities {
		if a.Name != "" {
			abilities = append(abilities, a.Name)
		}
	}

	// 查阵营颜色，未找到用默认蓝色
	clr, ok := factionColors[t.Faction]
	if !ok {
		clr = factionColors["base"]
	}

	label := t.ShortLabel
	if label == "" {
		label = t.Label
	}

	return tower.TowerDef{
		Key:         key,
		Label:       label,
		Faction:     t.Faction,
		Range:       t.BaseRange,
		Damage:      t.BaseDamage,
		AttackSpeed: attackSpeed,
		Cost:        t.BuildCost,
		Abilities:   abilities,
		Color:       clr,
	}
}

// LoadTowerDefs 加载所有塔 JSON 并转为 TowerDef 切片（按费用升序）。
func LoadTowerDefs() ([]tower.TowerDef, error) {
	all, err := config.LoadAllTowers()
	if err != nil {
		return nil, err
	}

	defs := make([]tower.TowerDef, 0, len(all))
	for key, t := range all {
		defs = append(defs, TowerJSONToDef(key, t))
	}

	// 冒泡排序按费用升序
	for i := 0; i < len(defs); i++ {
		for j := i + 1; j < len(defs); j++ {
			if defs[j].Cost < defs[i].Cost {
				defs[i], defs[j] = defs[j], defs[i]
			}
		}
	}
	return defs, nil
}
