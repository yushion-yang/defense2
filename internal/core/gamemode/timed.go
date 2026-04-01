// timed.go — 限时防守模式。
// 在指定时间内存活即胜利。倒计时到 0 时若生命 > 0 则胜利。
package gamemode

import (
	"fmt"
	"math"
)

const defaultTargetSeconds = 300.0 // 5 分钟

// TimedMode 限时防守模式。
type TimedMode struct {
	baseMode
	targetSeconds float64
	remainingTime float64
}

// NewTimedMode 创建限时防守模式（默认 5 分钟）。
func NewTimedMode() *TimedMode {
	return &TimedMode{
		baseMode:      baseMode{id: "timed"},
		targetSeconds: defaultTargetSeconds,
	}
}

func (m *TimedMode) OnInit(ctx *Context) {
	m.remainingTime = m.targetSeconds
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(9999) // 波次无上限
	}
}

func (m *TimedMode) OnTick(dt float64, _ *Context) {
	m.remainingTime -= dt
}

func (m *TimedMode) IntermissionSecs() float64 { return 3 }
func (m *TimedMode) ShouldAutoStart() bool     { return true }
func (m *TimedMode) VictoryWaveTarget() int     { return -1 }

func (m *TimedMode) CheckVictory(ctx *Context) bool {
	return m.remainingTime <= 0 && ctx.Lives > 0
}

func (m *TimedMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

func (m *TimedMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	bonus := 8 + wave*3
	return WaveClearResult{
		BonusGold: bonus,
		Message:   fmt.Sprintf("Wave %d clear! +$%d", wave, bonus),
	}
}

func (m *TimedMode) GetScore(ctx *Context) int {
	return ctx.Kills*5 + ctx.Lives*50
}

func (m *TimedMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{
		ShowTimer:    true,
		TimerSeconds: math.Max(0, m.remainingTime),
	}
}

func (m *TimedMode) GetEndData(ctx *Context) EndData {
	survived := m.targetSeconds - math.Max(0, m.remainingTime)
	return EndData{
		ModeName: "限时防守",
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"survivedSeconds": math.Round(survived),
			"targetSeconds":   m.targetSeconds,
			"kills":           ctx.Kills,
			"leaked":          ctx.Leaked,
			"livesRemaining":  ctx.Lives,
		},
	}
}
