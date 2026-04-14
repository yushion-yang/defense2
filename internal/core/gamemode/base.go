// base.go — Mode 接口的默认空实现（Null Object 模式）。
//
// baseMode 提供所有接口方法的合理默认行为：
//   - 胜利条件：波次打完且无存活敌人
//   - 失败条件：生命值归零
//   - 波间休息：从 balance.json spawner.waveInterval 读取（默认 10 秒）
//   - 经济奖励：从 economy.json 读取对应模式配置，无则回退 campaign
//
// 具体模式（如 CampaignMode）通过 embed baseMode 继承默认行为，
// 只需覆写差异化方法，符合"组合优于继承"原则。
package gamemode

import (
	"defense2/internal/config"
	"defense2/internal/i18n"
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
	if wi := config.GlobalSpawnerConfig().Timing.WaveInterval; wi > 0 {
		return wi
	}
	return 10
}
func (b *baseMode) CheckVictory(ctx *Context) bool { return !ctx.Spawning && ctx.Wave >= ctx.MaxWaves }
func (b *baseMode) CheckDefeat(ctx *Context) bool  { return ctx.Lives <= 0 }
func (b *baseMode) GetScore(_ *Context) int        { return 0 }
func (b *baseMode) VictoryWaveTarget() int         { return -1 }
func (b *baseMode) EnableEvents() bool             { return false }
func (b *baseMode) Ruleset() TowerRuleset          { return CampaignRuleset{} }

// OnWaveCleared 返回波次通关奖励。
// 默认不发放完美波次奖励（PerfectBonus=0），需要此功能的模式应覆写。
// 奖励公式来自 economy.json 的 waveBonus 配置。
func (b *baseMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon(b.id)
	bonus := econ.WaveBonus.Calc(wave)
	return WaveClearResult{
		BonusGold: bonus,
		Message:   i18n.TF("mode.wave_clear", wave, bonus),
	}
}

// modeEcon 返回指定模式的经济配置。
// 查找优先级：指定模式 → campaign → 硬编码默认值。
// 这样新增模式时即使忘记配置也有合理的奖励。
func modeEcon(modeID string) config.ModeEconomy {
	spec := config.GlobalEconomySpec()
	if spec == nil || spec.Modes == nil {
		return config.ModeEconomy{
			WaveBonus:    config.BonusFormula{Base: 12, PerWave: 4},
			PerfectBonus: config.BonusFormula{Base: 2, PerWave: 2},
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
