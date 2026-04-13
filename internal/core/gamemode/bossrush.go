// bossrush.go — Boss 竞速模式（当前锁定，显示"敬请期待"）。
//
// 核心玩法：每波只出 Boss，击败全部 Boss 即胜利。
// 分数机制：与速度挂钩——越快通关分数越高（10000 - 用时×10）。
// 波间休息 15 秒（比战役的 10 秒长），给玩家更多准备 Boss 的时间。
// HUD 专属：显示 Boss 击杀进度（如 "3/5"）。
package gamemode

import (
	"math"

	"defense2/internal/i18n"
)

// defaultTotalBosses 默认需要击败的 Boss 数量。
const defaultTotalBosses = 5

// BossRushMode Boss 竞速模式。
// 维护自己的 bossesKilled 计数器，不依赖 Context.Kills（那是总击杀）。
type BossRushMode struct {
	baseMode
	totalBosses  int // 需要击败的 Boss 总数
	bossesKilled int // 已击败的 Boss 数（运行时状态）
}

// NewBossRushMode 创建 Boss 竞速模式。
func NewBossRushMode() *BossRushMode {
	return &BossRushMode{
		baseMode:    baseMode{id: "bossRush"},
		totalBosses: defaultTotalBosses,
	}
}

// OnInit 重置 Boss 击杀计数，并将最大波次设为 Boss 总数。
func (m *BossRushMode) OnInit(ctx *Context) {
	m.bossesKilled = 0
	if ctx.SetMaxWaves != nil {
		ctx.SetMaxWaves(m.totalBosses)
	}
}

// IntermissionSecs Boss 战间隔 15 秒，比标准模式更长（给玩家布阵时间）。
func (m *BossRushMode) IntermissionSecs() float64 { return 15 }
func (m *BossRushMode) ShouldAutoStart() bool     { return true }

// VictoryWaveTarget 返回 Boss 总数，用于 HUD 显示进度。
func (m *BossRushMode) VictoryWaveTarget() int { return m.totalBosses }

// OnEnemyKilled 只统计 Boss 击杀，普通敌人不计入。
// boss 标记由 spawner 在生成时通过 ApplyFlags 设置。
func (m *BossRushMode) OnEnemyKilled(boss bool, _ *Context) {
	if boss {
		m.bossesKilled++
	}
}

// CheckVictory 击败全部 Boss 即胜利（不依赖波次数）。
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
		Message:      i18n.TF("mode.bossrush.boss_defeated", wave, bonus),
	}
}

// GetScore Boss 竞速分数 = 时间分(10000-用时×10) + 击杀×20。
// 设计意图：极速通关可得满分 10000，每多花 1 秒扣 10 分。
// 击杀权重是战役的 2 倍（20 vs 10），因为 Boss 难度高。
func (m *BossRushMode) GetScore(ctx *Context) int {
	elapsed := math.Round(ctx.ElapsedTime)
	timeScore := int(math.Max(0, 10000-elapsed*10))
	return timeScore + ctx.Kills*20
}

// GetHUDConfig 显示 Boss 击杀进度（如 "3/5"）。
func (m *BossRushMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{
		ShowBossCount: true,
		BossesKilled:  m.bossesKilled,
		TotalBosses:   m.totalBosses,
	}
}

func (m *BossRushMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: i18n.T("mode.bossrush.name"),
		Score:    m.GetScore(ctx),
		Extra: map[string]any{
			"bossesKilled": m.bossesKilled,
			"totalBosses":  m.totalBosses,
			"totalTime":    math.Round(ctx.ElapsedTime),
			"kills":        ctx.Kills,
		},
	}
}
