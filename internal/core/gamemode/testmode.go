// testmode.go — 测试模式。
// 用于开发调试：无失败、自定义金币/生命/波次、不触发事件。
package gamemode

// TestMode 测试模式。
type TestMode struct {
	baseMode
}

// NewTestMode 创建测试模式。
func NewTestMode() *TestMode {
	return &TestMode{baseMode: baseMode{id: "test"}}
}

func (m *TestMode) IntermissionSecs() float64 { return 5 }
func (m *TestMode) ShouldAutoStart() bool     { return true }
func (m *TestMode) VictoryWaveTarget() int     { return -1 }

// CheckDefeat 测试模式永远不失败。
func (m *TestMode) CheckDefeat(_ *Context) bool { return false }

// CheckVictory 有波次上限时才可胜利。
func (m *TestMode) CheckVictory(ctx *Context) bool {
	if ctx.MaxWaves <= 0 {
		return false
	}
	return ctx.Wave >= ctx.MaxWaves && !ctx.Spawning
}

func (m *TestMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	return WaveClearResult{
		Message: "test wave clear",
	}
}

func (m *TestMode) GetScore(_ *Context) int { return 0 }

func (m *TestMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{}
}

func (m *TestMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "测试模式",
		Score:    0,
		Extra: map[string]any{
			"waves": ctx.Wave,
			"kills": ctx.Kills,
		},
	}
}
