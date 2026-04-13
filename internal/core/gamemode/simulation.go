// simulation.go — 仿真测试模式。
// 与 campaign 相同的胜负判定（可以真正输），但无 PerfectBonus，
// 波间歇 3 秒，不自动开波（由策略控制开波时机）。
package gamemode

// SimulationMode 仿真测试模式。
type SimulationMode struct {
	baseMode
}

// NewSimulationMode 创建仿真测试模式。
func NewSimulationMode() *SimulationMode {
	return &SimulationMode{baseMode: baseMode{id: "simulation"}}
}

// CheckDefeat 生命归零即失败（继承 baseMode，显式覆写以明确语义）。
func (m *SimulationMode) CheckDefeat(ctx *Context) bool {
	return ctx.Lives <= 0
}

// ShouldAutoStart 不自动开波，由策略自行控制。
func (m *SimulationMode) ShouldAutoStart() bool { return false }

// IntermissionSecs 波间歇 3 秒（比 campaign 快，但给策略留建造时间）。
func (m *SimulationMode) IntermissionSecs() float64 { return 3 }

// VictoryWaveTarget 由地图波次数决定。
func (m *SimulationMode) VictoryWaveTarget() int { return -1 }

// CheckVictory 所有波次清完即胜利。
func (m *SimulationMode) CheckVictory(ctx *Context) bool {
	if ctx.MaxWaves <= 0 {
		return false
	}
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

// OnWaveCleared 波次奖金（无 PerfectBonus，避免干扰经济数据）。
func (m *SimulationMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	econ := modeEcon("simulation")
	bonus := econ.WaveBonus.Calc(wave)
	return WaveClearResult{BonusGold: bonus}
}

// GetScore 仿真不计分。
func (m *SimulationMode) GetScore(_ *Context) int { return 0 }

// GetEndData 返回仿真结束数据。
func (m *SimulationMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "Simulation",
		Score:    0,
		Extra: map[string]any{
			"waves":  ctx.Wave,
			"kills":  ctx.Kills,
			"leaked": ctx.Leaked,
			"lives":  ctx.Lives,
		},
	}
}
