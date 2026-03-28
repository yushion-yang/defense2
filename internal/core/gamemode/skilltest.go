// skilltest.go — 技能测试模式。
// 用于快速验证塔技能效果：无限金币/生命、固定波次组合、自动出怪。
package gamemode

// SkillTestMode 技能测试模式（无限资源 + 固定波次组合）。
type SkillTestMode struct {
	baseMode
}

// NewSkillTestMode 创建技能测试模式。
func NewSkillTestMode() *SkillTestMode {
	return &SkillTestMode{baseMode: baseMode{id: "skillTest"}}
}

// OnInit 初始化：注入极大金币和生命。
func (m *SkillTestMode) OnInit(ctx *Context) {
	if ctx.SetGold != nil {
		ctx.SetGold(99999)
	}
	if ctx.SetLives != nil {
		ctx.SetLives(999)
	}
}

// IntermissionSecs 波间休息 3 秒（快速迭代）。
func (m *SkillTestMode) IntermissionSecs() float64 { return 3 }

// ShouldAutoStart 自动开始下一波。
func (m *SkillTestMode) ShouldAutoStart() bool { return true }

// EnableEvents 不触发事件系统。
func (m *SkillTestMode) EnableEvents() bool { return false }

// VictoryWaveTarget 无胜利目标（无限波次）。
func (m *SkillTestMode) VictoryWaveTarget() int { return -1 }

// CheckVictory 永远不胜利。
func (m *SkillTestMode) CheckVictory(_ *Context) bool { return false }

// CheckDefeat 永远不失败。
func (m *SkillTestMode) CheckDefeat(_ *Context) bool { return false }

// OnWaveCleared 波次通关奖励（保持金币充足）。
func (m *SkillTestMode) OnWaveCleared(wave int, _ *Context) WaveClearResult {
	return WaveClearResult{
		BonusGold: 500,
		Message:   "skill test wave clear",
	}
}

// GetScore 技能测试无分数。
func (m *SkillTestMode) GetScore(_ *Context) int { return 0 }

// GetHUDConfig 无特殊 HUD。
func (m *SkillTestMode) GetHUDConfig(_ *Context) HUDConfig {
	return HUDConfig{}
}

// GetEndData 结算数据。
func (m *SkillTestMode) GetEndData(ctx *Context) EndData {
	return EndData{
		ModeName: "技能测试",
		Score:    0,
		Extra: map[string]any{
			"waves": ctx.Wave,
			"kills": ctx.Kills,
		},
	}
}

// ── 固定波次计划 ───────────────────────────────────

// SkillTestWavePlan 返回技能测试模式的固定波次组合。
// 每波：3 normal + 1 tank + 1 runner。
func SkillTestWavePlan() []WaveEntry {
	return []WaveEntry{
		{Archetype: "normal", Count: 3},
		{Archetype: "tank", Count: 1},
		{Archetype: "runner", Count: 1},
	}
}

// WaveEntry 波次中的一组敌人。
type WaveEntry struct {
	Archetype string // 敌人原型名称
	Count     int    // 数量
}
