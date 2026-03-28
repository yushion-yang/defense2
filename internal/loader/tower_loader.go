// tower_loader.go — 塔配置→运行时定义转换器。
// 从 config 包读取 JSON 塔数据，转换为 tower.TowerDef 运行时结构。
// 独立为 loader 包以避免 config↔tower 循环依赖。
package loader

import (
	"math"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/tower"
)

// defaultTowerColor 默认塔颜色 RGB。
var defaultTowerColor = [3]uint8{80, 140, 220}

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
	// bounceConfig → 自动追加 bounce 能力
	if t.BounceConfig != nil {
		abilities = append(abilities, "bounce")
	}


	label := t.ShortLabel
	if label == "" {
		label = t.Label
	}

	def := tower.TowerDef{
		Key:             key,
		Label:           label,
		Range:           t.BaseRange,
		Damage:          t.BaseDamage,
		AttackSpeed:     attackSpeed,
		Cost:            t.BuildCost,
		Abilities:       abilities,
		Color:           defaultTowerColor,
		AttackStyleID:   resolveAttackStyle(t),
		ProjectileSpeed: t.ProjectileSpeed,
	}

	// Beam 配置
	if t.Beam != nil {
		def.BeamDuration = t.Beam.Duration
		def.BeamWidth = t.Beam.Width
		def.BeamColor = parseHexColor(t.Beam.Color)
	}
	// Scatter 配置
	if t.ScatterConfig != nil {
		def.ScatterPellets = t.ScatterConfig.Pellets
		if def.ScatterPellets == 0 {
			def.ScatterPellets = 3
		}
		def.ScatterSpread = t.ScatterConfig.SpreadAngle * math.Pi / 180 / 2 // 半角
	}
	// Charge 配置
	if t.ChargeConfig != nil {
		def.ChargeMult = t.ChargeConfig.DamageMultiplier
		if def.ChargeMult == 0 {
			def.ChargeMult = 3.0
		}
	}
	// SpinAoE 配置
	def.InnerDmgBonus = t.InnerDamageBonus
	def.InnerRatioR = t.InnerRadiusRatio
	// Pierce 配置
	if t.PierceConfig != nil {
		def.PierceTargets = t.PierceConfig.Targets
		if def.PierceTargets == 0 {
			def.PierceTargets = 2
		}
		def.PierceDecay = t.PierceConfig.Decay
		if def.PierceDecay == 0 {
			def.PierceDecay = 0.8
		}
	}

	// 战力绑定配置
	def.StrengthJSON = t.Strength

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

// resolveAttackStyle 解析攻击方式（含自动推断）。
func resolveAttackStyle(t *config.TowerJSON) tower.AttackStyle {
	if t.AttackStyle != "" {
		return tower.AttackStyle(t.AttackStyle)
	}
	// 自动推断（与 JS 版 tickTowerCombat.js 一致）
	if t.ChargeConfig != nil {
		return tower.StyleCharge
	}
	if t.ScatterConfig != nil {
		return tower.StyleScatter
	}
	if t.PierceConfig != nil {
		return tower.StylePierce
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
