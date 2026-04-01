// base.go — Mode 接口的默认空实现。
// 各具体模式 embed baseMode 减少样板代码，只需覆写关注的方法。
package gamemode

import "fmt"

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
func (b *baseMode) IntermissionSecs() float64        { return 10 }
func (b *baseMode) CheckVictory(ctx *Context) bool   { return !ctx.Spawning && ctx.Wave >= ctx.MaxWaves }
func (b *baseMode) CheckDefeat(ctx *Context) bool    { return ctx.Lives <= 0 }
func (b *baseMode) GetScore(_ *Context) int          { return 0 }
func (b *baseMode) VictoryWaveTarget() int           { return -1 }
func (b *baseMode) EnableEvents() bool               { return false }

// OnWaveCleared returns wave-clear rewards. PerfectBonus is 0 by default;
// modes that support perfect-wave bonuses (e.g. campaign) should override.
func (b *baseMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	bonus := 12 + wave*4
	return WaveClearResult{
		BonusGold: bonus,
		Message:   fmt.Sprintf("Wave %d clear! +$%d", wave, bonus),
	}
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
