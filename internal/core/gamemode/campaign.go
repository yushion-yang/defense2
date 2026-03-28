// campaign.go — 战役模式。
// 标准模式：通过全部波次即胜利。波次通关有金币奖励和完美波次奖励。
package gamemode

import "fmt"

// CampaignMode 战役模式。
type CampaignMode struct {
	baseMode
}

// NewCampaignMode 创建战役模式。
func NewCampaignMode() *CampaignMode {
	return &CampaignMode{baseMode: baseMode{id: "campaign"}}
}

func (m *CampaignMode) IntermissionSecs() float64 { return 10 }
func (m *CampaignMode) ShouldAutoStart() bool     { return true }
func (m *CampaignMode) EnableEvents() bool         { return true }

func (m *CampaignMode) VictoryWaveTarget() int { return -1 } // 由地图 Waves 决定

func (m *CampaignMode) CheckVictory(ctx *Context) bool {
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

func (m *CampaignMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

func (m *CampaignMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	bonus := 12 + wave*4
	perfect := 8 + wave*2
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      fmt.Sprintf("Wave %d clear! +$%d", wave, bonus),
	}
}

func (m *CampaignMode) GetScore(ctx *Context) int {
	wavesCleared := ctx.Wave
	if wavesCleared > ctx.MaxWaves {
		wavesCleared = ctx.MaxWaves
	}
	return wavesCleared*100 + ctx.Kills*10 - ctx.Leaked*50
}

func (m *CampaignMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "战役模式",
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
