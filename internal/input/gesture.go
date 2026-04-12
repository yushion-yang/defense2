// gesture.go — 统一手势识别器。
// 封装鼠标/触摸输入，区分 Tap（点击）和 Drag（拖拽）。
//
// 业界标准流程（同 iOS/Android/Web 触摸事件模型）：
//
//	Press:
//	  记录起点 + 判定区域（UI or 游戏区）
//	Hold + Move:
//	  游戏区：距离 > 阈值 → Drag（实时平移相机）
//	  UI 区：不启动拖拽（手指可以滑走取消）
//	Release:
//	  未拖拽 → Tap（在松开位置触发操作）
//	  已拖拽 → 结束拖拽，不触发 Tap
//
// 防误触机制：
//	- 所有操作在松开时触发，不在按下时触发
//	- UI 区域按下后滑开 → 松开时不触发（和 iOS 按钮行为一致）
//	- 游戏区小幅移动（< 阈值）仍判定为 Tap
package input

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Gesture 统一手势识别器。
type Gesture struct {
	// 配置（创建后设置）
	DragThreshold float64                         // 拖拽判定阈值（逻辑像素），默认 5
	DragEnabled   bool                            // 是否允许拖拽（由外部状态机控制）
	IsOnUI        func(x, y float64) bool         // 判断点是否在 UI 区域（UI 区不启动拖拽）
	ToLogical     func(x, y float64) (float64, float64) // 原生→逻辑坐标转换

	// ── 当前帧输出（每帧 Update 后读取） ──
	tapped     bool
	tapX, tapY float64
	dragging   bool
	dragDX     float64
	dragDY     float64
	scrollDX   float64
	scrollDY   float64
	cursorX    float64
	cursorY    float64

	// ── 内部状态 ──
	tracking     bool    // 是否在追踪一次按压
	onUI         bool    // 本次按压起点是否在 UI 上
	startX       float64 // 按下起点（逻辑坐标）
	startY       float64
	prevX, prevY float64
	movedBeyond  bool    // 是否移动超过阈值
}

// NewGesture 创建手势识别器。
func NewGesture() *Gesture {
	return &Gesture{DragThreshold: 5}
}

// Update 每帧调用。
func (g *Gesture) Update() {
	// 重置帧输出
	g.tapped = false
	g.dragDX = 0
	g.dragDY = 0
	g.scrollDX, g.scrollDY = ebiten.Wheel()

	px, py := g.logicalPos()
	g.cursorX, g.cursorY = px, py
	down := g.isDown()

	switch {
	case !g.tracking && g.isJustDown():
		// ── 按下 ──
		g.tracking = true
		g.startX, g.startY = px, py
		g.prevX, g.prevY = px, py
		g.movedBeyond = false
		g.dragging = false
		g.onUI = g.IsOnUI != nil && g.IsOnUI(px, py)

	case g.tracking && down:
		// ── 按住中 ──
		dx := px - g.startX
		dy := py - g.startY

		if !g.movedBeyond && math.Sqrt(dx*dx+dy*dy) > g.DragThreshold {
			g.movedBeyond = true
			// 允许拖拽 + 不在 UI 区域 → 启动拖拽
			if g.DragEnabled && !g.onUI {
				g.dragging = true
			}
		}

		if g.dragging {
			g.dragDX = px - g.prevX
			g.dragDY = py - g.prevY
		}
		g.prevX, g.prevY = px, py

	case g.tracking && !down:
		// ── 松开 ──
		if !g.movedBeyond {
			// 没移动过 → Tap
			g.tapped = true
			g.tapX, g.tapY = px, py
		} else if !g.dragging {
			// 移动过但没启动拖拽。区分两种情况：
			// (a) DragEnabled=false → 模式本身禁用拖拽，移动不应吞掉点击
			// (b) DragEnabled=true + onUI → UI 保护拦截了拖拽，
			//     按住 UI 按钮并滑开应取消操作（同 iOS/Android 行为）
			if !(g.DragEnabled && g.onUI) {
				g.tapped = true
				g.tapX, g.tapY = px, py
			}
		}
		// 拖拽中松开：结束拖拽，不触发 Tap
		g.tracking = false
		g.dragging = false
		g.movedBeyond = false
	}
}

// ── 查询接口 ──

func (g *Gesture) JustTapped() bool                { return g.tapped }
func (g *Gesture) TapPos() (float64, float64)      { return g.tapX, g.tapY }
func (g *Gesture) IsDragging() bool                { return g.dragging }
func (g *Gesture) DragDelta() (float64, float64)   { return g.dragDX, g.dragDY }
func (g *Gesture) ScrollDelta() (float64, float64) { return g.scrollDX, g.scrollDY }
func (g *Gesture) CursorPos() (float64, float64)   { return g.cursorX, g.cursorY }

// ── 底层指针 ──

func (g *Gesture) logicalPos() (float64, float64) {
	rx, ry := g.rawPos()
	if g.ToLogical != nil {
		return g.ToLogical(rx, ry)
	}
	return rx, ry
}

func (g *Gesture) rawPos() (float64, float64) {
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		tx, ty := ebiten.TouchPosition(ids[0])
		return float64(tx), float64(ty)
	}
	mx, my := ebiten.CursorPosition()
	return float64(mx), float64(my)
}

func (g *Gesture) isDown() bool {
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(nil)) > 0
}

func (g *Gesture) isJustDown() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}
