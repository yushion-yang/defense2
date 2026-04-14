// classic.go — 经典战役模式。
//
// 经典 TD 体验：11 种预设塔型（固定攻击方式+能力），强度有上限，
// 策略核心是阵容搭配与位置布局，而非单塔无限养成。
//
// 胜负条件、经济奖励与 campaign 相同，差异全部体现在 ClassicRuleset 中。
package gamemode

import "defense2/internal/i18n"

// ClassicMode 经典战役模式。
type ClassicMode struct {
	CampaignMode // 继承战役模式的胜负/经济/分数逻辑
}

// NewClassicMode 创建经典模式。
func NewClassicMode() *ClassicMode {
	m := &ClassicMode{}
	m.id = "classic"
	return m
}

// Ruleset 返回经典模式专属的塔建造规则。
func (m *ClassicMode) Ruleset() TowerRuleset { return ClassicRuleset{} }

// GetEndData 返回经典模式的结算数据。
func (m *ClassicMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: i18n.T("mode.classic.name"),
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"waves":      ctx.Wave,
			"maxWaves":   ctx.MaxWaves,
			"kills":      ctx.Kills,
			"leaked":     ctx.Leaked,
			"goldEarned": ctx.Gold,
		},
	}
}
