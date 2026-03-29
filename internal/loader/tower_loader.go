// tower_loader.go — 塔配置→运行时定义转换器。
// 从 config 包读取 JSON 塔数据，转换为 tower.TowerDef 运行时结构。
// 独立为 loader 包以避免 config↔tower 循环依赖。
package loader

import (
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// defaultTowerColor 默认塔颜色 RGB。
var defaultTowerColor = [3]uint8{80, 140, 220}

// TowerJSONToDef 将单个 JSON 塔配置转为运行时 TowerDef。
func TowerJSONToDef(key string, t *config.TowerJSON) tower.TowerDef {
	// 攻速直接从 JSON 读取（次/秒），base+potential 在强度100时叠加
	attackSpeed := t.BaseAttackSpeed + t.PotentialAttackSpeed
	if attackSpeed <= 0 {
		attackSpeed = 1.0
	}

	// 能力列表（直接从 JSON 字符串数组获取）
	abilities := make([]string, len(t.Abilities))
	copy(abilities, t.Abilities)
	label := t.ShortLabel
	if label == "" {
		label = t.Label
	}

	def := tower.TowerDef{
		Key:             key,
		Label:           label,
		Range:           t.BaseRange + t.PotentialRange,   // 强度100时的默认值
		Damage:          t.BaseDamage + t.PotentialDamage, // 强度100时的默认值
		AttackSpeed:     attackSpeed,
		Cost:            t.BuildCost,
		Abilities:       abilities,
		Color:           defaultTowerColor,
		AttackStyleID:   resolveAttackStyle(t),
		ProjectileSpeed: t.ProjectileSpeed,
	}

	// 战力基础值+潜力值
	def.CfgBaseDamage = t.BaseDamage
	def.CfgBaseSpeed = t.BaseAttackSpeed
	def.CfgBaseRange = t.BaseRange
	def.PotentialDamage = t.PotentialDamage
	def.PotentialSpeed = t.PotentialAttackSpeed
	def.PotentialRange = t.PotentialRange
	return def
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

// resolveAttackStyle 解析攻击方式。
func resolveAttackStyle(t *config.TowerJSON) tower.AttackStyle {
	if t.AttackStyle != "" {
		return tower.AttackStyle(t.AttackStyle)
	}
	return tower.StyleProjectile
}

// parseHexColor 解析 "#rrggbb" hex 颜色为 [3]uint8。
func parseHexColor(hex string) [3]uint8 {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return [3]uint8{200, 200, 200}
	}
	var r, g, b uint8
	for i, ptr := range []*uint8{&r, &g, &b} {
		hi := hexVal(hex[i*2])
		lo := hexVal(hex[i*2+1])
		*ptr = hi*16 + lo
	}
	return [3]uint8{r, g, b}
}

func hexVal(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}
