// challenge.go — 挑战模式。
// 在特殊规则约束下完成战役目标。分数有额外乘数。
package gamemode

import (
	"math"

	"defense2/internal/i18n"
)

// ChallengeMode 挑战模式，基于战役扩展。
type ChallengeMode struct {
	CampaignMode
	GoldMultiplier float64  // 金币倍率（<1 = 更少金币）
	AllowedTowers  []string // 限制可用塔（nil = 全部可用）
}

// NewChallengeMode 创建挑战模式。
func NewChallengeMode() *ChallengeMode {
	m := &ChallengeMode{
		GoldMultiplier: 1.0,
	}
	m.id = "challenge"
	return m
}

func (m *ChallengeMode) ID() string { return "challenge" }

func (m *ChallengeMode) GetScore(ctx *Context) int {
	base := m.CampaignMode.GetScore(ctx)
	multiplier := 1.0
	if m.GoldMultiplier > 0 && m.GoldMultiplier < 1 {
		multiplier += 0.5 // 金币限制加分
	}
	if m.AllowedTowers != nil && len(m.AllowedTowers) <= 3 {
		multiplier += 0.3 // 塔限制加分
	}
	return int(math.Round(float64(base) * multiplier))
}

func (m *ChallengeMode) GetHUDConfig(ctx *Context) HUDConfig {
	rules := i18n.T("mode.challenge.rules")
	if m.GoldMultiplier < 1 {
		rules += " | " + i18n.T("mode.challenge.gold_limit")
	}
	if m.AllowedTowers != nil {
		rules += " | " + i18n.T("mode.challenge.tower_limit")
	}
	_ = ctx
	return HUDConfig{
		ShowRules: true,
		RulesText: rules,
	}
}

func (m *ChallengeMode) GetEndData(ctx *Context) EndData {
	data := m.CampaignMode.GetEndData(ctx)
	data.ModeName = i18n.T("mode.challenge.name")
	data.Score = m.GetScore(ctx)
	data.Extra["goldMultiplier"] = m.GoldMultiplier
	return data
}
