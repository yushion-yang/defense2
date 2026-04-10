// base.go — Mode 接口的默认空实现。
// 各具体模式 embed baseMode 减少样板代码，只需覆写关注的方法。
package gamemode

import (
	"fmt"

	"defense2/internal/config"
)

// baseMode 提供 Mode 接口的零值默认实现。
type baseMode struct {
	id string
}

func (b *baseMode) ID() string                       { return b.id }
func (b *baseMode) OnInit(_ *Context)                {}
func (b *baseMode) OnGameStart(_ *Context)           {}
func (b *baseMode) OnTick(_ float64, _ *Context)     {}
func (b *baseMode) OnWaveStart(_ int, _ *Context)    {}
func (b *baseMode) OnEnemyKilled(_ bool, _ *Context) {}
func (b *baseMode) OnEnemyLeaked(_ *Context)         {}
func (b *baseMode) ShouldAutoStart() bool            { return true }
func (b *baseMode) IntermissionSecs() float64 {
	if wi := config.GlobalBalance().Spawner.WaveInterval; wi > 0 {
		return wi
	}
	return 10
}
func (b *baseMode) CheckVictory(ctx *Context) bool { return !ctx.Spawning && ctx.Wave >= ctx.MaxWaves }
func (b *baseMode) CheckDefeat(ctx *Context) bool  { return ctx.Lives <= 0 }
func (b *baseMode) GetScore(_ *Context) int        { return 0 }
func (b *baseMode) VictoryWaveTarget() int         { return -1 }
func (b *baseMode) EnableEvents() bool             { return false }

// OnWaveCleared returns wave-clear rewards. PerfectBonus is 0 by default;
// modes that support perfect-wave bonuses (e.g. campaign) should override.
func (b *baseMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon(b.id)
	bonus := econ.WaveBonus.Calc(wave)
	return WaveClearResult{
		BonusGold: bonus,
		Message:   fmt.Sprintf("Wave %d clear! +$%d", wave, bonus),
	}
}

// modeEcon 返回指定模式的经济配置，回退到 campaign。
func modeEcon(modeID string) config.ModeEconomy {
	spec := config.GlobalEconomySpec()
	if spec == nil || spec.Modes == nil {
		return config.ModeEconomy{
			WaveBonus:    config.BonusFormula{Base: 12, PerWave: 4},
			PerfectBonus: config.BonusFormula{Base: 8, PerWave: 2},
		}
	}
	if m, ok := spec.Modes[modeID]; ok {
		return m
	}
	if m, ok := spec.Modes["campaign"]; ok {
		return m
	}
	return config.ModeEconomy{}
}

func (b *baseMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{}
}

func (b *baseMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: b.id,
		Score:    b.GetScore(ctx),
	}
}
