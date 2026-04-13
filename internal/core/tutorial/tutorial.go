// tutorial.go — 新手教程系统。
// 8 步引导式教程，支持事件触发推进和自动倒计时推进。
// 完成后通过持久化标记不再重复显示。
package tutorial

import "defense2/internal/i18n"

// Step 单个教程步骤。
type Step struct {
	Message     string  // 显示给玩家的提示文本
	Event       string  // 触发推进的事件名称（空 = 无事件要求，点击推进）
	AutoAdvance float64 // 超时自动推进秒数（0 = 仅靠事件/点击推进）
}

// Tutorial 教程状态。
type Tutorial struct {
	Steps      []Step  // 步骤列表
	CurrentIdx int     // 当前步骤索引（完成全部时 == len(Steps)）
	Active     bool    // 是否正在运行
	Done       bool    // 是否已全部完成
	autoTimer  float64 // 当前步骤的自动推进计时器
}

// DefaultTutorial 创建默认 8 步教程。
func DefaultTutorial() *Tutorial {
	return &Tutorial{
		Steps: []Step{
			{Message: i18n.T("tutorial.step_1")},
			{Message: i18n.T("tutorial.step_2"), Event: "build"},
			{Message: i18n.T("tutorial.step_3"), Event: "tower_placed"},
			{Message: i18n.T("tutorial.step_4"), Event: "wave_start"},
			{Message: i18n.T("tutorial.step_5"), Event: "tower_select", AutoAdvance: 8},
			{Message: i18n.T("tutorial.step_6"), Event: "upgrade", AutoAdvance: 8},
			{Message: i18n.T("tutorial.step_7"), Event: "item_use", AutoAdvance: 10},
			{Message: i18n.T("tutorial.step_8"), AutoAdvance: 3},
		},
		Active: true,
	}
}

// CurrentStep 返回当前步骤数据（教程结束时返回 nil）。
func (t *Tutorial) CurrentStep() *Step {
	if !t.Active || t.Done || t.CurrentIdx >= len(t.Steps) {
		return nil
	}
	return &t.Steps[t.CurrentIdx]
}

// CurrentMessage 返回当前步骤的提示文本（教程结束时返回空字符串）。
// 兼容旧调用。
func (t *Tutorial) CurrentMessage() string {
	s := t.CurrentStep()
	if s == nil {
		return ""
	}
	return s.Message
}

// StepIndex 返回当前步骤索引（0-based）。
func (t *Tutorial) StepIndex() int {
	return t.CurrentIdx
}

// StepCount 返回总步骤数。
func (t *Tutorial) StepCount() int {
	return len(t.Steps)
}

// Tick 每帧更新自动推进计时器。dt 为秒。
func (t *Tutorial) Tick(dt float64) {
	step := t.CurrentStep()
	if step == nil {
		return
	}
	if step.AutoAdvance <= 0 {
		return
	}
	t.autoTimer += dt
	if t.autoTimer >= step.AutoAdvance {
		t.advance()
	}
}

// Trigger 响应游戏事件，匹配当前步骤则推进。
// 返回 true 表示教程状态发生了变化。
func (t *Tutorial) Trigger(eventName string) bool {
	step := t.CurrentStep()
	if step == nil {
		return false
	}
	if step.Event != "" && step.Event == eventName {
		t.advance()
		return true
	}
	return false
}

// ClickAdvance 点击推进：仅当当前步骤无事件要求时推进。
// 用于欢迎/完成等步骤的任意点击推进。
// 返回 true 表示教程状态发生了变化。
func (t *Tutorial) ClickAdvance() bool {
	step := t.CurrentStep()
	if step == nil {
		return false
	}
	if step.Event == "" {
		t.advance()
		return true
	}
	return false
}

// OnEvent 兼容旧 API，映射旧事件名到新事件名。
func (t *Tutorial) OnEvent(eventName string) bool {
	// 映射旧事件名
	switch eventName {
	case "gameStart":
		// 旧版 gameStart 触发欢迎步骤，新版欢迎步骤靠点击推进，忽略
		return false
	case "towerBuilt":
		return t.Trigger("tower_placed")
	case "waveStarted":
		return t.Trigger("wave_start")
	case "waveCleared":
		return t.Trigger("wave_clear")
	case "enemyKilled":
		return t.Trigger("enemy_killed")
	default:
		return t.Trigger(eventName)
	}
}

// Skip 跳过教程。
func (t *Tutorial) Skip() {
	t.Active = false
	t.Done = true
}

// IsComplete 教程是否已完成。
func (t *Tutorial) IsComplete() bool {
	return t.Done
}

// advance 推进到下一步。
func (t *Tutorial) advance() {
	t.CurrentIdx++
	t.autoTimer = 0
	if t.CurrentIdx >= len(t.Steps) {
		t.Done = true
	}
}
