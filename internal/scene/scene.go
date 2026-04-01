// scene.go — 场景接口定义。
// 定义所有游戏场景的通用接口和场景切换器接口。
package scene

import (
	gameAudio "defense2/internal/audio"
	"defense2/internal/core/event"

	"github.com/hajimehoshi/ebiten/v2"
)

// Scene 游戏场景接口，所有场景（标题、游戏、结算等）必须实现。
type Scene interface {
	Update() error             // 每帧逻辑更新
	Draw(screen *ebiten.Image) // 每帧渲染
}

// Switcher 场景切换器接口，允许场景请求跳转到另一个场景。
type Switcher interface {
	SwitchScene(next Scene)           // 将下一帧切换到指定场景
	AudioManager() *gameAudio.Manager // 返回全局音效管理器
	EventBus() *event.Bus             // 返回全局事件总线
}
