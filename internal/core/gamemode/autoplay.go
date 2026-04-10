// autoplay.go — 自动对局游戏模式。
// 不可失败（immortal）、自动开波、短间歇、事件路径禁用。
package gamemode

// AutoPlayMode 自动对局模式。
type AutoPlayMode struct {
	baseMode
}

// NewAutoPlayMode 创建自动对局模式。
func NewAutoPlayMode() *AutoPlayMode {
	return &AutoPlayMode{baseMode: baseMode{id: "autoplay"}}
}

// CheckDefeat 自动对局永不失败。
func (m *AutoPlayMode) CheckDefeat(_ *Context) bool { return false }

// ShouldAutoStart 自动开波。
func (m *AutoPlayMode) ShouldAutoStart() bool { return true }

// IntermissionSecs 波间歇 2 秒（加快测试速度）。
func (m *AutoPlayMode) IntermissionSecs() float64 { return 2 }

// VictoryWaveTarget 由地图波次数决定。
func (m *AutoPlayMode) VictoryWaveTarget() int { return -1 }

// CheckVictory 所有波次清完即胜利。
func (m *AutoPlayMode) CheckVictory(ctx *Context) bool {
	if ctx.MaxWaves <= 0 {
		return false
	}
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

// OnWaveCleared 波次奖金（与 campaign 相同）。
func (m *AutoPlayMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("autoplay")
	bonus := econ.WaveBonus.Calc(wave)
	return WaveClearResult{BonusGold: bonus}
}

// GetScore 自动对局不计分。
func (m *AutoPlayMode) GetScore(_ *Context) int { return 0 }

// GetHUDConfig 默认 HUD 配置。
func (m *AutoPlayMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{}
}

// GetEndData 返回自动对局结束数据。
func (m *AutoPlayMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "AutoPlay",
		Score:    0,
		Extra: map[string]any{
			"waves": ctx.Wave,
			"kills": ctx.Kills,
		},
	}
}
