// manager.go — 统一输入管理器。
// 将键盘、鼠标和触摸输入统一为每帧一个 Command 结构。
// 支持桌面和移动端，触摸优先于鼠标。
package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Command 统一输入命令，每帧由 Manager 生成。
type Command struct {
	// 点击/触摸事件
	TapX, TapY   int    // 本帧点击位置（无点击时为 -1）
	Tapped       bool   // 本帧是否发生了点击/触摸
	RightTapped  bool   // 本帧是否右键点击
	RightTapX    int    // 右键点击 X
	RightTapY    int    // 右键点击 Y

	// 持续状态
	CursorX, CursorY int  // 当前鼠标/触摸位置
	Source           string // 输入来源："keyboard"、"mouse"、"touch"

	// 快捷键
	NumberKey    int  // 按下的数字键（1-9），0 表示无
	Escape       bool // ESC 键
	Enter        bool // Enter 键
	SpeedToggle  bool // 加速键（Space）
	Pause        bool // 暂停键（P）
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
		RightTapX: -1, RightTapY: -1,
	}

	// 鼠标位置
	cmd.CursorX, cmd.CursorY = ebiten.CursorPosition()

	// 鼠标左键点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cmd.Tapped = true
		cmd.TapX, cmd.TapY = cmd.CursorX, cmd.CursorY
		cmd.Source = "mouse"
	}

	// 鼠标右键点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		cmd.RightTapped = true
		cmd.RightTapX, cmd.RightTapY = cmd.CursorX, cmd.CursorY
	}

	// 触摸（优先于鼠标）
	touchIDs := inpututil.JustPressedTouchIDs()
	if len(touchIDs) > 0 {
		tx, ty := ebiten.TouchPosition(touchIDs[0])
		cmd.Tapped = true
		cmd.TapX, cmd.TapY = tx, ty
		cmd.CursorX, cmd.CursorY = tx, ty
		cmd.Source = "touch"
	}

	// 数字键
	for i := 0; i < 9; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			cmd.NumberKey = i + 1
			cmd.Source = "keyboard"
		}
	}

	// 功能键
	cmd.Escape = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	cmd.Enter = inpututil.IsKeyJustPressed(ebiten.KeyEnter)
	cmd.SpeedToggle = inpututil.IsKeyJustPressed(ebiten.KeySpace)
	cmd.Pause = inpututil.IsKeyJustPressed(ebiten.KeyP)

	return cmd
}
