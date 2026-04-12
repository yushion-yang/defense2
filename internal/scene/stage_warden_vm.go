// stage_warden_vm.go — 战灵选择数据构建（从 config 加载并转为 hud.WardenOption）。
package scene

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/persistence"
	"defense2/internal/i18n"
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
	key := "game.warden.cat." + cat
	label := i18n.T(key)
	if label == key {
		return cat
	}
	return label
}

// BuildWardenOptions 从 wardens.json 配置构建选择列表。
// pm 可选；若提供则标记未解锁战灵为 Locked。
func BuildWardenOptions(pm ...*persistence.ProgressManager) []hud.WardenOption {
	cfgs, err := config.LoadWardenConfigs()
	if err != nil {
		return []hud.WardenOption{{Key: "none", Name: i18n.T("game.warden.none_name"), Category: "-", Description: i18n.T("game.warden.none_desc"), Color: color.RGBA{R: 120, G: 120, B: 130, A: 255}}}
	}

	var mgr *persistence.ProgressManager
	if len(pm) > 0 {
		mgr = pm[0]
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
			growthKill = fmt.Sprintf("+%.0f %s", c.GrowthOnKill, i18n.T("game.warden.str"))
		}
		growthWave := "-"
		if c.GrowthOnWaveClear > 0 {
			growthWave = fmt.Sprintf("+%.0f %s", c.GrowthOnWaveClear, i18n.T("game.warden.str"))
		}

		locked := false
		lockReason := ""
		if mgr != nil && !mgr.IsWardenUnlocked(key) {
			locked = true
			lockReason = persistence.UnlockRequirement("warden", key)
			if lockReason == "" {
				lockReason = i18n.T("game.warden.locked")
			}
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
			Locked:      locked,
			LockReason:  lockReason,
		})
	}

	// "不选" 选项
	opts = append(opts, hud.WardenOption{
		Key:         "none",
		Name:        i18n.T("game.warden.none_name"),
		Category:    "-",
		Description: i18n.T("game.warden.none_desc"),
		Color:       color.RGBA{R: 120, G: 120, B: 130, A: 255},
	})
	return opts
}

// buildStaticParams 从配置构建占位符参数（选择阶段用基础值，不含强度缩放）。
// 基础属性从 WardenConfig 顶层字段读取，类型特有参数从 Params map 读取。
// 派生值（如 fireballDmg = damage * fireballDmgRatio）在此计算。
func buildStaticParams(c config.WardenConfig) map[string]string {
	p := map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", c.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", c.Damage),
		"moveSpeed":      fmt.Sprintf("%.0f", c.MoveSpeed),
		"range":          fmt.Sprintf("%.0f", c.Range),
	}

	// 从 Params map 读取所有类型特有参数
	if c.Params != nil {
		for k, v := range c.Params {
			// 整数参数（无小数部分）用 %g 格式，否则 %.1f
			if v == float64(int(v)) {
				p[k] = fmt.Sprintf("%g", v)
			} else {
				p[k] = fmt.Sprintf("%.2f", v)
			}
		}
	}

	// 计算派生值（描述文本中需要的计算字段）
	paramOr := func(key string, fallback float64) float64 {
		if c.Params != nil {
			if v, ok := c.Params[key]; ok {
				return v
			}
		}
		return fallback
	}

	switch c.Key {
	case "prince":
		ratio := paramOr("fireballDmgRatio", 2.0)
		p["fireballDmg"] = fmt.Sprintf("%.0f", c.Damage*ratio)
		trailRatio := paramOr("trailDpsRatio", 0.5)
		p["trailDps"] = fmt.Sprintf("%.0f", c.Damage*trailRatio)
		p["fireballHpPct"] = fmt.Sprintf("%.0f", paramOr("fireballHpPct", 0.05)*100)
	case "core":
		p["execHpPct"] = fmt.Sprintf("%.0f", paramOr("execHpPct", 0.20)*100)
	case "skystrike":
		multiRatio := paramOr("multiDmgRatio", 2.0)
		p["multiDmg"] = fmt.Sprintf("%.0f", c.Damage*multiRatio)
		burstRatio := paramOr("burstDmgRatio", 1.0)
		p["burstDmg"] = fmt.Sprintf("%.0f", c.Damage*burstRatio)
		p["hpPct"] = fmt.Sprintf("%.0f", paramOr("hpPercent", 0.10)*100)
	case "envoy":
		p["buffBonus"] = "0" // 选择阶段无强度，默认为 0
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

// GetWardenOptions 返回战灵选项列表。
// pm 可选；若提供则包含解锁状态。不再缓存以确保解锁状态实时刷新。
func GetWardenOptions(pm ...*persistence.ProgressManager) []hud.WardenOption {
	return BuildWardenOptions(pm...)
}
