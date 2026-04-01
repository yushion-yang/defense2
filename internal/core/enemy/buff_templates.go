// buff_templates.go — Buff 模板系统。
// 预定义 14 种敌人 buff 模板，支持通过 ID 快速施加。
// 同时提供 flags 标记和旧版类型名映射。
package enemy

import tel "defense2/internal/core/telemetry"

// BuffTemplate buff 模板定义。
type BuffTemplate struct {
	ID          string  // 模板唯一标识
	Description string  // 模板描述
	Category    string  // 分类（offense/defense/utility/death）

	// ── 狂暴参数 ──
	BerserkThreshold  float64 // 狂暴触发血量比例
	BerserkSpeedScale float64 // 狂暴速度倍率

	// ── 回血参数 ──
	RegenPerSec float64 // 每秒回血量倍率（占 MaxHP 比例）

	// ── 治疗光环参数 ──
	HealPower    float64 // 治疗量
	HealRadius   float64 // 治疗范围（像素）
	HealInterval float64 // 治疗间隔（秒）

	// ── 速度光环参数 ──
	SpeedAuraFactor float64 // 速度光环加成倍率

	// ── 减伤参数 ──
	DamageReduce float64 // 伤害减免比例（0~1）

	// ── 死亡效果参数 ──
	DeathSplitCount int     // 死亡分裂数量
	DeathSlowFactor float64 // 死亡减速倍率
	DeathSlowRadius float64 // 死亡减速范围
	DeathSlowDura   float64 // 死亡减速持续时间

	// ── 反伤参数 ──
	ReflectPercent float64 // 反伤比例

	// ── 复活参数 ──
	ReviveHPPercent float64 // 复活血量百分比

	// ── 召唤参数 ──
	SpawnCount int    // 召唤数量
	SpawnType  string // 召唤敌人类型

	// ── 通用标记 ──
	Flags []string // 附加标记（如 "elite"、"boss"）
}

// templates buff 模板注册表。
var templates map[string]BuffTemplate

// InitBuffTemplates 初始化 14 种预定义 buff 模板。
func InitBuffTemplates() {
	templates = map[string]BuffTemplate{
		"berserk": {
			ID:                "berserk",
			Description:       "低血量时进入狂暴，提升移动速度",
			Category:          "offense",
			BerserkThreshold:  0.5,
			BerserkSpeedScale: 1.5,
		},
		"regen": {
			ID:          "regen",
			Description: "持续回复生命值",
			Category:    "defense",
			RegenPerSec: 0.02, // 每秒回复 2% 最大血量
		},
		"healAura": {
			ID:           "healAura",
			Description:  "治疗周围友军",
			Category:     "utility",
			HealPower:    10,
			HealRadius:   80,
			HealInterval: 2.0,
		},
		"speedAura": {
			ID:              "speedAura",
			Description:     "提升周围友军移动速度",
			Category:        "utility",
			SpeedAuraFactor: 0.2, // +20% 速度
		},
		"damageReduce": {
			ID:           "damageReduce",
			Description:  "减少受到的伤害",
			Category:     "defense",
			DamageReduce: 0.3, // 减伤 30%
		},
		"empBurst": {
			ID:          "empBurst",
			Description: "电磁脉冲爆发，干扰塔攻击",
			Category:    "offense",
		},
		"blink": {
			ID:          "blink",
			Description: "短距离闪现前进",
			Category:    "utility",
		},
		"deathSplit": {
			ID:              "deathSplit",
			Description:     "死亡时分裂为多个小单位",
			Category:        "death",
			DeathSplitCount: 2,
		},
		"deathSlow": {
			ID:              "deathSlow",
			Description:     "死亡时对周围敌人施加减速",
			Category:        "death",
			DeathSlowFactor: 0.5,
			DeathSlowRadius: 60,
			DeathSlowDura:   3.0,
		},
		"reflect": {
			ID:             "reflect",
			Description:    "反弹部分受到的伤害",
			Category:       "defense",
			ReflectPercent: 0.15, // 反伤 15%
		},
		"timewarp": {
			ID:          "timewarp",
			Description: "时间扭曲，局部加速",
			Category:    "utility",
		},
		"revive": {
			ID:              "revive",
			Description:     "死亡后复活一次",
			Category:        "death",
			ReviveHPPercent: 0.5, // 50% 血量复活
		},
		"spawnMinions": {
			ID:          "spawnMinions",
			Description: "周期性召唤小兵",
			Category:    "offense",
			SpawnCount:  3,
			SpawnType:   "normal",
		},
	}
}

// GetBuffTemplate 获取指定 ID 的 buff 模板（未找到返回 nil）。
func GetBuffTemplate(id string) *BuffTemplate {
	if templates == nil {
		InitBuffTemplates()
	}
	t, ok := templates[id]
	if !ok {
		return nil
	}
	return &t
}

// ApplyBuffTemplate 将指定 buff 模板的效果应用到敌人上。
// 返回 true 表示成功施加。
func ApplyBuffTemplate(e *Enemy, templateID string) bool {
	tmpl := GetBuffTemplate(templateID)
	if tmpl == nil {
		return false
	}
	// 遥测：记录敌人 buff 模板使用
	tel.T.Record("enemy_template", templateID)

	// 狂暴参数
	if tmpl.BerserkThreshold > 0 {
		e.BerserkThreshold = tmpl.BerserkThreshold
		e.BerserkSpeedScale = tmpl.BerserkSpeedScale
	}

	// 回血参数（按最大血量比例）
	if tmpl.RegenPerSec > 0 {
		e.RegenPerSec = e.MaxHP * tmpl.RegenPerSec
	}

	// 治疗光环
	if tmpl.HealPower > 0 {
		e.HealPower = tmpl.HealPower
		e.HealRadius = tmpl.HealRadius
		e.HealInterval = tmpl.HealInterval
		e.HealCooldown = 0
	}

	// 速度光环（buff 模板设置光环参数）
	if tmpl.SpeedAuraFactor > 0 {
		e.AuraSpeedUp = tmpl.SpeedAuraFactor
		if e.AuraRange <= 0 {
			e.AuraRange = 80 // 默认光环范围
		}
	}

	// 减伤
	if tmpl.DamageReduce > 0 {
		e.DamageReduceRatio = tmpl.DamageReduce
	}

	// 反伤
	if tmpl.ReflectPercent > 0 {
		e.ReflectPercent = tmpl.ReflectPercent
	}

	// 复活
	if tmpl.ReviveHPPercent > 0 {
		e.ReviveHPPercent = tmpl.ReviveHPPercent
	}

	// 死亡分裂
	if tmpl.DeathSplitCount > 0 {
		e.SplitCount = tmpl.DeathSplitCount
		if e.SplitHPRatio <= 0 {
			e.SplitHPRatio = 0.3
		}
		if e.SplitSpeedScale <= 0 {
			e.SplitSpeedScale = 1.4
		}
	}

	// 应用附加标记
	if len(tmpl.Flags) > 0 {
		ApplyFlags(e, tmpl.Flags)
	}

	return true
}

// ApplyFlags 对敌人应用标记效果。
// 支持的标记：
//   - "elite": HP*3, Speed*1.1, Reward*2
//   - "boss": HP*30, Reward*5
func ApplyFlags(e *Enemy, flags []string) {
	for _, flag := range flags {
		switch flag {
		case "elite":
			e.MaxHP *= 3
			e.HP *= 3
			e.BaseSpeed *= 1.1
			if e.SlowTimer <= 0 {
				e.Speed = e.BaseSpeed
			} else {
				e.Speed = e.BaseSpeed * e.SlowFactor
			}
			e.Reward *= 2
			e.Elite = true

		case "boss":
			e.MaxHP *= 30
			e.HP *= 30
			e.Reward *= 5
			e.Boss = true
		}
	}
}

// MapLegacyType 将旧版敌人类型名映射到新系统的原型、标记和 buff 列表。
// 用于向后兼容旧版配置数据。
func MapLegacyType(typeName string) (archetype string, flags []string, buffIDs []string) {
	switch typeName {
	case "normal":
		return "normal", nil, nil
	case "fast", "runner":
		return "runner", nil, nil
	case "tank", "armored":
		return "tank", nil, nil
	case "flying":
		return "flying", nil, nil
	case "healer":
		return "healer", nil, []string{"healAura"}
	case "berserker":
		return "berserker", nil, []string{"berserk"}
	case "regenerator":
		return "regenerator", nil, []string{"regen"}
	case "splitter":
		return "splitter", nil, []string{"deathSplit"}
	case "summoner":
		return "summoner", nil, []string{"spawnMinions"}
	case "reflector":
		return "reflector", nil, []string{"reflect"}
	case "elite":
		return "normal", []string{"elite"}, nil
	case "boss":
		return "normal", []string{"boss"}, nil
	default:
		return "normal", nil, nil
	}
}
