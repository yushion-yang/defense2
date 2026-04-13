// rules.go — 堆叠规则定义与 JSON 配置加载。
//
// 堆叠规则决定了同 ID 的多个 buff 实例如何合并。
// 配置源: config/systems/buff-stack.json，在游戏启动时一次性加载为全局单例。
// 若某个 buff ID 在 JSON 中未配置，BuffList.getRule() 默认返回 Override 模式。
package buff

import (
	"encoding/json"
	"fmt"
)

// StackRule 定义单个 buff ID 的堆叠行为。
type StackRule struct {
	Mode     StackMode // 堆叠模式（6 种之一）
	Cap      float64   // 聚合后的上限钳制值；0 表示不限制。如 slow 的 Cap=0.8 防止减速超过 80%
	Floor    float64   // 聚合后的下限钳制值；0 表示不限制。如 damageDown 的 Floor=0.2 保证最少受到 20% 伤害
	Priority int       // 仅 Override 模式使用：高优先级覆盖低优先级（如 invincible=99 > controlImmune=80）
}

// rawRule 镜像 buff-stack.json 中每条规则的 JSON 结构。
type rawRule struct {
	Mode     string  `json:"mode"`
	Cap      float64 `json:"cap"`
	Floor    float64 `json:"floor"`
	Priority int     `json:"priority"`
}

// rawConfig 镜像 buff-stack.json 的顶层结构。
// JSON 中的 "modes" 和 "_description" 字段仅作文档用途，加载时忽略。
type rawConfig struct {
	Rules map[string]rawRule `json:"rules"`
}

// modeMap 将 JSON 中的字符串模式名映射到 StackMode 枚举。
// 键名必须与 buff-stack.json 的 "mode" 字段值一致。
var modeMap = map[string]StackMode{
	"strongest":            Strongest,
	"additive":             Additive,
	"multiplicative":       Multiplicative,
	"override":             Override,
	"independent":          Independent,
	"independentPerSource": IndependentPerSource,
}

// LoadRules 解析 buff-stack.json 字节内容，返回 buff ID → StackRule 的映射。
// 遇到未知的 mode 字符串时返回错误，防止配置笔误导致运行时 buff 行为异常。
func LoadRules(data []byte) (map[string]StackRule, error) {
	var raw rawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal buff rules: %w", err)
	}
	rules := make(map[string]StackRule, len(raw.Rules))
	for id, r := range raw.Rules {
		mode, ok := modeMap[r.Mode]
		if !ok {
			return nil, fmt.Errorf("unknown stack mode %q for buff %q", r.Mode, id)
		}
		rules[id] = StackRule{
			Mode:     mode,
			Cap:      r.Cap,
			Floor:    r.Floor,
			Priority: r.Priority,
		}
	}
	return rules, nil
}
