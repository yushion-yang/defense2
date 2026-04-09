// endless.go — 无尽模式。
// 波次永不停止，玩家坚持越久越好。基于战役模式扩展。
package gamemode

import "fmt"

// EndlessMode 无尽模式。
type EndlessMode struct {
	CampaignMode
}

// NewEndlessMode 创建无尽模式。
func NewEndlessMode() *EndlessMode {
	m := &EndlessMode{}
	m.id = "endless"
	return m
}

func (m *EndlessMode) ID() string { return "endless" }

func (m *EndlessMode) OnInit(ctx *Context) {
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(9999)
	}
}

func (m *EndlessMode) CheckVictory(_ *Context) bool { return false } // 无尽不可胜

func (m *EndlessMode) VictoryWaveTarget() int { return -1 }

func (m *EndlessMode) GetScore(ctx *Context) int {
	return ctx.Wave*100 + ctx.Kills*10
}

func (m *EndlessMode) OnWaveCleared(wave int, ctx *Context) WaveClearResult {
	econ := modeEcon("endless")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	msg := fmt.Sprintf("Wave %d clear! +$%d", wave, bonus)
	if wave%5 == 0 {
		msg = fmt.Sprintf("Wave %d clear! Boss 波! +$%d", wave, bonus)
	}
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      msg,
	}
}

func (m *EndlessMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "无尽模式",
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"wavesReached": ctx.Wave,
			"kills":        ctx.Kills,
			"leaked":       ctx.Leaked,
		},
	}
}
