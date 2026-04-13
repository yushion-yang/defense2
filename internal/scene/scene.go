// scene.go — 场景系统的核心接口定义。
//
// 本文件定义了两个核心抽象：
//   - Scene:    所有游戏场景的通用接口（Update + Draw），与 Ebitengine 的帧驱动模型对齐
//   - Switcher: 场景切换器 + 全局资源访问器，由 Game struct 实现并注入到各场景
//
// 场景流转: Title → Select → {CampaignSelect|TestSelect|Settings} → Stage → Result
// 每个场景通过 Switcher.SwitchScene() 请求跳转，Game 负责执行淡入淡出过渡（见 game.go）。
//
// 另有两个可选接口用于吉祥物向导系统：
//   - MascotSnapshotProvider: 场景可提供游戏快照供吉祥物条件判断（仅 StageScene 实现）
//   - MascotActionExecutor:   场景可执行吉祥物发起的动作（如自动建塔、提示等）
package scene

import (
	gameAudio "defense2/internal/audio"
	"defense2/internal/core/event"
	"defense2/internal/core/mascot"

	"github.com/hajimehoshi/ebiten/v2"
)

// Scene 游戏场景接口，所有场景必须实现。
// Ebitengine 每帧先调用 Update()（固定 60 TPS），再调用 Draw()（按显示器刷新率），
// 两者可能不同频率。每个场景自行管理自己的状态和渲染逻辑。
type Scene interface {
	Update() error             // 每帧逻辑更新（60 TPS），返回非 nil error 会导致游戏退出
	Draw(screen *ebiten.Image) // 每帧渲染，screen 是当前帧的渲染目标
}

// MascotSnapshotProvider 由能提供游戏状态快照的场景实现（目前仅 StageScene）。
// 吉祥物向导系统通过此接口获取波次、金币、塔数等信息，用于触发条件对话。
type MascotSnapshotProvider interface {
	MascotSnapshot() mascot.StageSnapshot
}

// MascotActionExecutor 由能执行吉祥物动作的场景实现。
// 当吉祥物向导产生动作请求（如建议建塔、提示升级）时，Game 会将动作路由到当前场景执行。
type MascotActionExecutor interface {
	ExecuteMascotAction(action *mascot.MascotAction) bool
}

// Switcher 场景切换器接口，由 Game struct 实现。
// 各场景通过构造函数接收 Switcher，从而能够：
//  1. 请求场景跳转（SwitchScene 会触发淡入淡出过渡）
//  2. 访问全局音效管理器（跨场景共享的单例，避免重复加载 WAV）
//  3. 访问全局事件总线（跨场景的发布-订阅通道，场景切换时自动清理订阅）
type Switcher interface {
	SwitchScene(next Scene)           // 请求跳转到指定场景（带淡入淡出过渡）
	AudioManager() *gameAudio.Manager // 返回全局音效管理器（跨场景复用）
	EventBus() *event.Bus             // 返回全局事件总线（跨场景复用，切换时 Clear）
}
