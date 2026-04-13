// timed.go — 限时防守模式（当前锁定，显示"敬请期待"）。
//
// 核心玩法：在指定时间内（默认 5 分钟）存活即胜利。
// 关键差异：
//   - 波次无上限（9999），波间仅 3 秒（高压节奏）
//   - 胜利条件：倒计时归零且生命 > 0
//   - 分数公式：击杀×5 + 剩余生命×50（鼓励防守而非进攻）
//   - HUD 专属：显示倒计时器
package gamemode

import (
	"math"

	"defense2/internal/i18n"
)

// defaultTargetSeconds 默认目标存活时间（5 分钟 = 300 秒）。
const defaultTargetSeconds = 300.0

// TimedMode 限时防守模式。
// 自行维护 remainingTime 倒计时，每帧在 OnTick 中递减。
type TimedMode struct {
	baseMode
	targetSeconds float64 // 目标存活时间（秒）
	remainingTime float64 // 剩余时间（运行时状态，每帧递减）
}

// NewTimedMode 创建限时防守模式（默认 5 分钟）。
func NewTimedMode() *TimedMode {
	return &TimedMode{
		baseMode:      baseMode{id: "timed"},
		targetSeconds: defaultTargetSeconds,
	}
}

// OnInit 初始化倒计时并设置波次无上限。
func (m *TimedMode) OnInit(ctx *Context) {
	m.remainingTime = m.targetSeconds
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(9999) // 波次无上限，靠时间结束而非波次
	}
}

// OnTick 每帧递减倒计时。dt 是本帧时间增量（秒）。
// 可以递减为负数——CheckVictory 只检查 <= 0。
func (m *TimedMode) OnTick(dt float64, _ *Context) {
	m.remainingTime -= dt
}

// IntermissionSecs 限时模式波间仅 3 秒（标准 10 秒），营造高压节奏。
func (m *TimedMode) IntermissionSecs() float64 { return 3 }
func (m *TimedMode) ShouldAutoStart() bool     { return true }
func (m *TimedMode) VictoryWaveTarget() int    { return -1 }

// CheckVictory 倒计时归零且玩家仍存活 = 胜利。
// 注意必须同时检查 Lives > 0：最后一刻被击杀不算胜利。
func (m *TimedMode) CheckVictory(ctx *Context) bool {
	return m.remainingTime <= 0 && ctx.Lives > 0
}

// CheckDefeat 生命归零即失败（即使倒计时未结束）。
func (m *TimedMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

func (m *TimedMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("timed")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      i18n.TF("mode.wave_clear", wave, bonus),
	}
}

// GetScore 限时模式分数：击杀×5 + 剩余生命×50。
// 击杀权重低（5 vs 战役的 10），生命权重极高（50），
// 设计意图是强调"存活防守"而非"激进击杀"。
func (m *TimedMode) GetScore(ctx *Context) int {
	return ctx.Kills*5 + ctx.Lives*50
}

// GetHUDConfig 显示倒计时器，钳制最小值为 0 避免负数显示。
func (m *TimedMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{
		ShowTimer:    true,
		TimerSeconds: math.Max(0, m.remainingTime),
	}
}

// GetEndData 结算数据包含实际存活时长和目标时长（用于显示完成百分比）。
func (m *TimedMode) GetEndData(ctx *Context) EndData {
	survived := m.targetSeconds - math.Max(0, m.remainingTime)
	return EndData{
		ModeName: i18n.T("mode.timed.name"),
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
