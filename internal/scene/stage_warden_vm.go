// stage_warden_vm.go — 战灵选择数据构建（从 config 加载并转为 hud.WardenOption）。
package scene

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/config"
	"defense2/internal/render/hud"
)

// wardenColors 按 key 映射战灵主题色（不在 JSON 中的视觉属性）。
var wardenColors = map[string]color.RGBA{
	"prince":    {R: 255, G: 140, B: 30, A: 255},
	"core":      {R: 60, G: 140, B: 255, A: 255},
	"chain":     {R: 160, G: 80, B: 255, A: 255},
	"skystrike": {R: 80, G: 200, B: 255, A: 255},
	"envoy":     {R: 255, G: 200, B: 60, A: 255},
}

// wardenOrder 战灵在选择列表中的顺序。
var wardenOrder = []string{"prince", "core", "chain", "skystrike", "envoy"}

// categoryName 将 config category 转为显示名。
func categoryName(cat string) string {
	switch cat {
	case "mobile":
		return "移动型"
	case "indirect":
		return "间接型"
	default:
		return cat
	}
}

// BuildWardenOptions 从 wardens.json 配置构建选择列表。
func BuildWardenOptions() []hud.WardenOption {
	cfgs, err := config.LoadWardenConfigs()
	if err != nil {
		return []hud.WardenOption{{Key: "none", Name: "纯塔挑战", Category: "-", Description: "不选择战灵，纯靠塔防御。", Color: color.RGBA{R: 120, G: 120, B: 130, A: 255}}}
	}

	var opts []hud.WardenOption
	for _, key := range wardenOrder {
		c, ok := cfgs[key]
		if !ok {
			continue
		}
		clr := wardenColors[key]
		growthKill := "-"
		if c.GrowthOnKill > 0 {
			growthKill = fmt.Sprintf("+%.0f 强度", c.GrowthOnKill)
		}
		growthWave := "-"
		if c.GrowthOnWaveClear > 0 {
			growthWave = fmt.Sprintf("+%.0f 强度", c.GrowthOnWaveClear)
		}

		// 用配置基础值替换描述中的占位符
		params := buildStaticParams(c)
		opts = append(opts, hud.WardenOption{
			Key:         c.Key,
			Name:        c.Name,
			Category:    categoryName(c.Category),
			Description: c.Description,
			Color:       clr,
			AttackName:  c.AttackName,
			AttackDesc:  replaceParams(c.AttackDesc, params),
			SpecialName: c.SpecialName,
			SpecialDesc: replaceParams(c.SpecialDesc, params),
			Tips:        []string{c.StrengthDesc, c.CounterTip},
			Damage:      fmt.Sprintf("%.0f", c.Damage),
			Interval:    fmt.Sprintf("%.1fs", c.AttackInterval),
			Speed:       fmt.Sprintf("%.0f", c.MoveSpeed),
			AoE:         "-",
			Duration:    "-",
			DoT:         "-",
			GrowthKill:  growthKill,
			GrowthWave:  growthWave,
		})
	}

	// "不选" 选项
	opts = append(opts, hud.WardenOption{
		Key:         "none",
		Name:        "纯塔挑战",
		Category:    "-",
		Description: "不选择战灵，纯靠塔防御。",
		Color:       color.RGBA{R: 120, G: 120, B: 130, A: 255},
	})
	return opts
}

// buildStaticParams 从配置构建占位符参数（选择阶段用基础值，不含强度缩放）。
func buildStaticParams(c config.WardenConfig) map[string]string {
	p := map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", c.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", c.Damage),
		"moveSpeed":      fmt.Sprintf("%.0f", c.MoveSpeed),
		"range":          fmt.Sprintf("%.0f", c.Range),
	}
	// 按 key 补充类型特有参数的默认值（与 Go Init 中的硬编码一致）
	switch c.Key {
	case "prince":
		p["fireballInterval"] = "4"
		p["fireballDmg"] = fmt.Sprintf("%.0f", c.Damage*2) // 200%
		p["trailDuration"] = "2"
		p["trailDps"] = fmt.Sprintf("%.0f", c.Damage*0.5) // 50%
	case "core":
		p["aoeThreshold"] = "4"
		p["execHpPct"] = "20"
	case "chain":
		p["chainRange"] = "150"
		p["bonusPerTower"] = "10"
	case "skystrike":
		p["specialInterval"] = "1"
		p["multiTargets"] = "3"
		p["multiDmg"] = fmt.Sprintf("%.0f", c.Damage*2) // 200%
		p["burstHits"] = "5"
		p["burstDmg"] = fmt.Sprintf("%.0f", c.Damage*1) // 100%
		p["hpTargets"] = "3"
		p["hpPct"] = "10"
	case "envoy":
		p["buffInterval"] = "10"
		p["buffDuration"] = "6"
		p["buffThreshold"] = "100"
		p["buffBonus"] = "0"
		p["permGrant"] = "5"
	}
	return p
}

// replaceParams 替换字符串中的 {key} 占位符。
func replaceParams(s string, params map[string]string) string {
	for k, v := range params {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// wardenOptionsCache 延迟初始化的选项列表（首次访问时从配置构建）。
var wardenOptionsCache []hud.WardenOption

// GetWardenOptions 返回战灵选项列表（懒加载）。
func GetWardenOptions() []hud.WardenOption {
	if wardenOptionsCache == nil {
		wardenOptionsCache = BuildWardenOptions()
	}
	return wardenOptionsCache
}
