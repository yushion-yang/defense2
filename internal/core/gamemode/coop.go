// coop.go — 合作模式。
//
// 多人（含 AI 玩家）各守一个分区，敌人串联穿越所有分区。
// 继承战役模式的波次/胜负/经济逻辑，差异：
//   - 手动开波（给多人更多准备时间）
//   - 波间间隔 15s（比战役模式长）
//   - 经济奖励使用 coop 专用参数（fallback 到 casual）
package gamemode

import "defense2/internal/i18n"

// CoopMode 合作模式。
type CoopMode struct {
	CampaignMode
}

// NewCoopMode 创建合作模式。
func NewCoopMode() *CoopMode {
	return &CoopMode{CampaignMode: CampaignMode{baseMode: baseMode{id: "coop"}}}
}

// ShouldAutoStart 合作模式手动开波，给多人更多准备时间。
func (m *CoopMode) ShouldAutoStart() bool { return false }

// IntermissionSecs 波间间隔 15 秒（战役默认 10 秒）。
func (m *CoopMode) IntermissionSecs() float64 { return 15.0 }

// OnWaveCleared 发放波次奖励（使用 coop 经济参数，fallback 到 casual）。
func (m *CoopMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("coop")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      i18n.TF("mode.wave_clear", wave, bonus),
	}
}

// GetEndData 返回结算数据。
func (m *CoopMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: i18n.T("mode.coop.name"),
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"waves":    ctx.Wave,
			"maxWaves": ctx.MaxWaves,
			"kills":    ctx.Kills,
			"leaked":   ctx.Leaked,
		},
	}
}

// Ruleset 合作模式使用战役规则集（标准塔、概率掉落、启用战灵）。
func (m *CoopMode) Ruleset() TowerRuleset { return CampaignRuleset{} }
