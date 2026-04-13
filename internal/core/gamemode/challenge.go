// challenge.go — 挑战模式（当前锁定，显示"敬请期待"）。
//
// 在特殊规则约束下完成战役目标。约束越严格，分数乘数越高：
//   - 金币限制（GoldMultiplier < 1）：+50% 分数
//   - 塔种限制（AllowedTowers <= 3）：+30% 分数
//
// 继承 CampaignMode 的胜负条件和波次逻辑，只修改分数计算和 HUD。
package gamemode

import (
	"math"

	"defense2/internal/i18n"
)

// ChallengeMode 挑战模式，组合 CampaignMode 并叠加额外约束。
// 约束字段由关卡配置注入（当前为默认值，未来从 challenge.json 加载）。
type ChallengeMode struct {
	CampaignMode
	GoldMultiplier float64  // 金币倍率（<1 = 更少金币，如 0.5 表示半价经济）
	AllowedTowers  []string // 限制可用塔种（nil = 全部可用）
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

// GetScore 在战役分数基础上乘以约束奖励系数。
// 金币限制（经济减半）+50%，塔种限制（<=3 种可用）+30%。
// 两者可叠加，最高 1.8x。用 Round 避免浮点截断丢分。
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

// GetHUDConfig 在 HUD 上显示当前激活的挑战规则。
// 规则文本由管道符拼接，如 "挑战规则 | 金币限制 | 塔种限制"。
func (m *ChallengeMode) GetHUDConfig(ctx *Context) HUDConfig {
	rules := i18n.T("mode.challenge.rules")
	if m.GoldMultiplier < 1 {
		rules += " | " + i18n.T("mode.challenge.gold_limit")
	}
	if m.AllowedTowers != nil {
		rules += " | " + i18n.T("mode.challenge.tower_limit")
	}
	_ = ctx // 保留参数签名一致性，未来可能用到
	return HUDConfig{
		ShowRules: true,
		RulesText: rules,
	}
}

// GetEndData 复用战役结算数据，追加挑战专属的金币倍率信息。
func (m *ChallengeMode) GetEndData(ctx *Context) EndData {
	data := m.CampaignMode.GetEndData(ctx)
	data.ModeName = i18n.T("mode.challenge.name")
	data.Score = m.GetScore(ctx)
	data.Extra["goldMultiplier"] = m.GoldMultiplier
	return data
}
