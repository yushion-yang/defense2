// manager.go — 简化版输入管理器（旧接口，部分场景仍在使用）。
//
// 将鼠标和触摸输入统一为每帧一个 Command 结构体。
// 与 gesture.go 的区别：
//   - Manager 只检测 JustPressed（按下瞬间），不区分 Tap/Drag
//   - Gesture 实现完整的按→拖→放状态机，有阈值判定和防误触
//   - 新场景应优先使用 Gesture；Manager 保留是为了兼容已有的非 Stage 场景
//
// 坐标约定：所有输出均为逻辑坐标（1200x540），通过 draw.CursorPos/TouchPos 转换。
// 触摸优先于鼠标：同一帧内若有触摸事件，会覆盖鼠标检测结果。
package input

import (
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Command 统一输入命令，每帧由 Manager 生成。
// 值类型语义：拷贝即安全，无需担心被后续帧覆盖。
type Command struct {
	// 点击/触摸事件（一次性，只在触发帧有值）
	TapX, TapY int  // 本帧点击位置（无点击时为 -1，用 -1 而非 0 避免与左上角混淆）
	Tapped     bool // 本帧是否发生了点击/触摸

	// 持续状态（每帧都有值）
	CursorX, CursorY int    // 当前鼠标/触摸位置（逻辑坐标）
	Source           string // 输入来源："mouse" 或 "touch"（空字符串表示无事件）
}

// Manager 输入管理器。
type Manager struct{}

// NewManager 创建输入管理器。
func NewManager() *Manager {
	return &Manager{}
}

// Update 每帧调用，收集所有输入源并生成统一命令。
// 检测顺序：先鼠标后触摸。若同帧有触摸事件，触摸结果会覆盖鼠标结果。
// 注意：此方法用 JustPressed 检测，意味着按下瞬间就触发——
// 与 Gesture.JustTapped()（松开时触发）行为不同。
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
