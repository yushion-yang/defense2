package buff

import (
	"encoding/json"
	"fmt"
)

// StackRule defines stacking behavior for a buff ID, loaded from buff-stack.json.
type StackRule struct {
	Mode     StackMode
	Cap      float64 // 0 = no cap (Additive/Strongest)
	Floor    float64 // 0 = no floor (Multiplicative)
	Priority int     // Override mode: higher priority wins
}

// rawRule mirrors the JSON structure in buff-stack.json.
type rawRule struct {
	Mode     string  `json:"mode"`
	Cap      float64 `json:"cap"`
	Floor    float64 `json:"floor"`
	Priority int     `json:"priority"`
}

type rawConfig struct {
	Rules map[string]rawRule `json:"rules"`
}

var modeMap = map[string]StackMode{
	"strongest":            Strongest,
	"additive":             Additive,
	"multiplicative":       Multiplicative,
	"override":             Override,
	"independent":          Independent,
	"independentPerSource": IndependentPerSource,
}

// LoadRules parses buff-stack.json bytes and returns a map of buff ID to StackRule.
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
