// tutorial.go — 新手教程系统。
// 基于步骤驱动的教程，每步由特定游戏事件触发推进。
// 完成后通过持久化标记不再重复显示。
package tutorial

// Step 单个教程步骤。
type Step struct {
	ID      string // 步骤唯一标识
	Message string // 显示给玩家的提示文本
	Trigger string // 触发推进的事件名称
}

// Tutorial 教程状态。
type Tutorial struct {
	Steps       []Step // 步骤列表
	CurrentStep int    // 当前步骤索引（完成全部时 == len(Steps)）
	Active      bool   // 是否正在运行
	Done        bool   // 是否已全部完成
}

// DefaultTutorial 创建默认 5 步教程。
func DefaultTutorial() *Tutorial {
	return &Tutorial{
		Steps: []Step{
			{ID: "welcome", Message: "Welcome! Click a green circle to build a tower.", Trigger: "gameStart"},
			{ID: "build", Message: "Tower built! Towers auto-attack enemies in range.", Trigger: "towerBuilt"},
			{ID: "wave", Message: "Enemies incoming! They follow the path to your base.", Trigger: "waveStarted"},
			{ID: "kill", Message: "Enemy killed! You earn gold for each kill.", Trigger: "enemyKilled"},
			{ID: "clear", Message: "Wave cleared! Bonus gold + interest awarded.", Trigger: "waveCleared"},
		},
		Active: true,
	}
}

// CurrentMessage 返回当前步骤的提示文本（教程结束时返回空字符串）。
func (t *Tutorial) CurrentMessage() string {
	if !t.Active || t.Done || t.CurrentStep >= len(t.Steps) {
		return ""
	}
	return t.Steps[t.CurrentStep].Message
}

// CurrentTrigger 返回当前步骤等待的事件名称。
func (t *Tutorial) CurrentTrigger() string {
	if !t.Active || t.Done || t.CurrentStep >= len(t.Steps) {
		return ""
	}
	return t.Steps[t.CurrentStep].Trigger
}

// OnEvent 响应游戏事件，匹配则推进到下一步。
// 返回 true 表示教程状态发生了变化。
func (t *Tutorial) OnEvent(eventName string) bool {
	if !t.Active || t.Done {
		return false
	}
	if t.CurrentStep >= len(t.Steps) {
		t.Done = true
		return true
	}
	if t.Steps[t.CurrentStep].Trigger == eventName {
		t.CurrentStep++
		if t.CurrentStep >= len(t.Steps) {
			t.Done = true
		}
		return true
	}
	return false
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
