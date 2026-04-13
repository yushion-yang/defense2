// endless.go — 无尽模式（当前锁定，显示"敬请期待"）。
//
// 继承战役模式的经济和奖励逻辑，关键差异：
//   - 胜利条件：永不胜利（CheckVictory 恒返回 false）
//   - 波数上限：设为 9999，实际无限
//   - 分数公式：纯正向（无泄漏惩罚），鼓励尽可能打到高波次
//   - Boss 波提示：每 N 波出 Boss 时显示特殊通关消息
//
// 设计决策：embed CampaignMode 而非 baseMode，复用完美波次奖励等逻辑。
package gamemode

import (
	"defense2/internal/config"
	"defense2/internal/i18n"
)

// EndlessMode 无尽模式。
// 组合 CampaignMode 以复用波次奖励和结算逻辑。
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

// OnInit 将最大波次设为 9999，使 spawner 永不停止生成。
func (m *EndlessMode) OnInit(ctx *Context) {
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(9999)
	}
}

// CheckVictory 无尽模式永不胜利，只能失败结束。
func (m *EndlessMode) CheckVictory(_ *Context) bool { return false }

// VictoryWaveTarget 返回 -1，HUD 不显示目标波数进度条。
func (m *EndlessMode) VictoryWaveTarget() int { return -1 }

// GetScore 无尽模式分数：波次×100 + 击杀×10。
// 与战役不同，不扣除泄漏分——无尽模式下泄漏是不可避免的。
func (m *EndlessMode) GetScore(ctx *Context) int {
	return ctx.Wave*100 + ctx.Kills*10
}

// OnWaveCleared 发放波次奖励，Boss 波显示特殊提示文本。
// Boss 出现周期从 spawner.json boss.everyNWaves 读取。
func (m *EndlessMode) OnWaveCleared(wave int, ctx *Context) WaveClearResult {
	econ := modeEcon("endless")
	bonus := econ.WaveBonus.Calc(wave)
	perfect := econ.PerfectBonus.Calc(wave)
	msg := i18n.TF("mode.wave_clear", wave, bonus)
	bossN := config.GlobalSpawnerConfig().Boss.EveryNWaves
	if bossN > 0 && wave%bossN == 0 {
		msg = i18n.TF("mode.wave_clear_boss", wave, bonus)
	}
	return WaveClearResult{
		BonusGold:    bonus,
		PerfectBonus: perfect,
		Message:      msg,
	}
}

// GetEndData 无尽模式结算，核心指标是"达到的最高波次"。
func (m *EndlessMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: i18n.T("mode.endless.name"),
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"wavesReached": ctx.Wave,
			"kills":        ctx.Kills,
			"leaked":       ctx.Leaked,
		},
	}
}
