// bossrush.go — Boss 竞速模式。
// 每波只出 Boss，击败全部 Boss 即胜利。分数与速度挂钩。
package gamemode

import (
	"fmt"
	"math"
)

const defaultTotalBosses = 5

// BossRushMode Boss 竞速模式。
type BossRushMode struct {
	baseMode
	totalBosses  int
	bossesKilled int
}

// NewBossRushMode 创建 Boss 竞速模式。
func NewBossRushMode() *BossRushMode {
	return &BossRushMode{
		baseMode:    baseMode{id: "bossRush"},
		totalBosses: defaultTotalBosses,
	}
}

func (m *BossRushMode) OnInit(ctx *Context) {
	m.bossesKilled = 0
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(m.totalBosses)
	}
}

func (m *BossRushMode) IntermissionSecs() float64 { return 15 }
func (m *BossRushMode) ShouldAutoStart() bool     { return true }
func (m *BossRushMode) VictoryWaveTarget() int     { return m.totalBosses }

func (m *BossRushMode) OnEnemyKilled(boss bool, _ *Context) {
	if boss {
		m.bossesKilled++
	}
}

func (m *BossRushMode) CheckVictory(_ *Context) bool {
	return m.bossesKilled >= m.totalBosses
}

func (m *BossRushMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

func (m *BossRushMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("bossRush")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      fmt.Sprintf("首领%d击败! +$%d", wave, bonus),
	}
}

func (m *BossRushMode) GetScore(ctx *Context) int {
	elapsed := math.Round(ctx.ElapsedTime)
	timeScore := int(math.Max(0, 10000-elapsed*10))
	return timeScore + ctx.Kills*20
}

func (m *BossRushMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{
		ShowBossCount: true,
		BossesKilled:  m.bossesKilled,
		TotalBosses:   m.totalBosses,
	}
}

func (m *BossRushMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "Boss 竞速",
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"bossesKilled": m.bossesKilled,
			"totalBosses":  m.totalBosses,
			"totalTime":    math.Round(ctx.ElapsedTime),
			"kills":        ctx.Kills,
		},
	}
}
