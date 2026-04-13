// campaign.go — 战役模式（当前唯一可正常游玩的模式）。
//
// 胜利条件：击退地图定义的全部波次（MaxWaves 由地图 JSON 决定）。
// 失败条件：生命值归零。
// 经济特色：每波通关有基础金币奖励 + 完美波次额外奖励（零泄漏）。
// 分数公式：波次×100 + 击杀×10 - 泄漏×50（鼓励零泄漏打法）。
package gamemode

import "defense2/internal/i18n"

// CampaignMode 战役模式。
type CampaignMode struct {
	baseMode
}

// NewCampaignMode 创建战役模式。
func NewCampaignMode() *CampaignMode {
	return &CampaignMode{baseMode: baseMode{id: "campaign"}}
}

// ShouldAutoStart 战役模式自动开始下一波，玩家无需手动触发。
func (m *CampaignMode) ShouldAutoStart() bool { return true }

// VictoryWaveTarget 返回 -1，表示目标波数由地图 JSON 的 Waves 字段决定，非模式固定。
func (m *CampaignMode) VictoryWaveTarget() int { return -1 }

// CheckVictory 当前波次 >= 最大波次且场上无存活敌人时判定胜利。
// 必须同时满足两个条件：防止最后一波出完怪但还没打完就提前胜利。
func (m *CampaignMode) CheckVictory(ctx *Context) bool {
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

// CheckDefeat 生命值归零即失败。
func (m *CampaignMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

// OnWaveCleared 发放波次通关奖励。
// 战役模式同时发放完美波次奖励（PerfectBonus），这是与 baseMode 的关键差异。
// 实际是否发放完美奖励由 Stage 层判断（本波是否有泄漏），此处只提供金额。
func (m *CampaignMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("campaign")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      i18n.TF("mode.wave_clear", wave, bonus),
	}
}

// GetScore 计算战役模式分数。
// 公式：波次×100 + 击杀×10 - 泄漏×50。
// wavesCleared 上限钳制防止异常波数（如回归测试注入大值）导致分数膨胀。
func (m *CampaignMode) GetScore(ctx *Context) int {
	wavesCleared := ctx.Wave
	if wavesCleared > ctx.MaxWaves {
		wavesCleared = ctx.MaxWaves
	}
	return wavesCleared*100 + ctx.Kills*10 - ctx.Leaked*50
}

// GetEndData 返回结算屏幕所需的完整数据（用于 Result 场景渲染）。
func (m *CampaignMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: i18n.T("mode.campaign.name"),
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
