// manager.go — 统一输入管理器。
// 将键盘、鼠标和触摸输入统一为每帧一个 Command 结构。
// 支持桌面和移动端，触摸优先于鼠标。
package input

import (
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Command 统一输入命令，每帧由 Manager 生成。
type Command struct {
	// 点击/触摸事件
	TapX, TapY int  // 本帧点击位置（无点击时为 -1）
	Tapped     bool // 本帧是否发生了点击/触摸

	// 持续状态
	CursorX, CursorY int    // 当前鼠标/触摸位置
	Source           string // 输入来源："mouse"、"touch"
}

// Manager 输入管理器。
type Manager struct{}

// NewManager 创建输入管理器。
func NewManager() *Manager {
	return &Manager{}
}

// Update 每帧调用，收集所有输入源并生成统一命令。
func (m *Manager) Update() Command {
	cmd := Command{
		TapX: -1, TapY: -1,
	}

	// 鼠标位置（逻辑坐标）
	lx, ly := draw.CursorPos()
	cmd.CursorX, cmd.CursorY = int(lx), int(ly)

	// 鼠标左键点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cmd.Tapped = true
		cmd.TapX, cmd.TapY = cmd.CursorX, cmd.CursorY
		cmd.Source = "mouse"
	}

	// 触摸（优先于鼠标）
	touchIDs := inpututil.JustPressedTouchIDs()
	if len(touchIDs) > 0 {
		ltx, lty := draw.TouchPos(touchIDs[0])
		tx, ty := int(ltx), int(lty)
		cmd.Tapped = true
		cmd.TapX, cmd.TapY = tx, ty
		cmd.CursorX, cmd.CursorY = tx, ty
		cmd.Source = "touch"
	}

	return cmd
}
