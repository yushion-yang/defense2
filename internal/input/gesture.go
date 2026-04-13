// gesture.go — 统一手势识别器（按→拖→放 三态状态机）。
//
// 在塔防游戏中，同一块屏幕区域需要同时处理「点击选塔」和「拖拽平移地图」，
// 此文件封装鼠标/触摸输入，通过 5px 位移阈值区分 Tap（点击）和 Drag（拖拽）。
//
// 状态机流程（同 iOS/Android/Web 触摸事件模型）：
//
//	Press（按下）:
//	  记录起点坐标 + 判定起点区域（UI 还是游戏区）
//	Hold + Move（按住移动）:
//	  游戏区：累计位移 > 5px → 切换为 Drag 模式（实时平移相机）
//	  UI 区：永不启动拖拽（手指可以滑走，松开时取消操作）
//	Release（松开）:
//	  未拖拽 → 触发 Tap（在松开位置执行点击操作）
//	  已拖拽 → 结束拖拽，不触发 Tap（避免平移结束时误选塔）
//
// 防误触机制：
//   - 所有操作在松开时触发，不在按下时触发（与 JustTapped 行为一致）
//   - UI 区域按下后滑开 → 松开时不触发（和 iOS 按钮行为一致）
//   - 游戏区小幅移动（< 5px 阈值）仍判定为 Tap，容忍手指抖动
//   - 长按悬浮（draw.LongPressConsumed）会消费释放事件，不再触发 Tap
//
// 外部控制点：
//   - DragEnabled: 由场景状态机控制，某些 mode 下禁用拖拽（如建塔放置模式）
//   - IsOnUI: 回调函数，判断坐标是否在 HUD/按钮区域，UI 区内不启动拖拽
package input

import (
	"math"

	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Gesture 统一手势识别器。
type Gesture struct {
	// 配置（创建后设置）
	DragThreshold float64                               // 拖拽判定阈值（逻辑像素），默认 5
	DragEnabled   bool                                  // 是否允许拖拽（由外部状态机控制）
	IsOnUI        func(x, y float64) bool               // 判断点是否在 UI 区域（UI 区不启动拖拽）
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
	movedBeyond  bool // 是否移动超过阈值
}

// NewGesture 创建手势识别器。
func NewGesture() *Gesture {
	return &Gesture{DragThreshold: 5}
}

// Update 每帧调用，驱动手势状态机。
// 三态转换：非追踪+按下→追踪中 / 追踪中+按住→判定拖拽 / 追踪中+松开→输出Tap或结束Drag。
// 每帧开头清零所有输出字段，确保只反映当前帧的事件。
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
		// 长按悬浮已消费此次释放 → 不触发 Tap
		if draw.LongPressConsumed() {
			g.tracking = false
			g.dragging = false
			g.movedBeyond = false
			break
		}
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

// ── 查询接口（每帧 Update 后读取，只反映当前帧状态） ──

func (g *Gesture) JustTapped() bool                { return g.tapped }               // 本帧是否触发了点击（松开时判定）
func (g *Gesture) TapPos() (float64, float64)      { return g.tapX, g.tapY }         // 点击位置（逻辑坐标）
func (g *Gesture) IsDragging() bool                { return g.dragging }             // 是否正在拖拽中
func (g *Gesture) DragDelta() (float64, float64)   { return g.dragDX, g.dragDY }     // 本帧拖拽增量（逻辑像素）
func (g *Gesture) ScrollDelta() (float64, float64) { return g.scrollDX, g.scrollDY } // 本帧滚轮增量
func (g *Gesture) CursorPos() (float64, float64)   { return g.cursorX, g.cursorY }   // 当前指针位置（逻辑坐标）

// ── 底层指针抽象（统一鼠标和触摸为单一指针） ──

// logicalPos 返回当前指针的逻辑坐标。
// 先取原生坐标（触摸优先于鼠标），再通过 ToLogical 回调转换为 1200x540 逻辑坐标。
func (g *Gesture) logicalPos() (float64, float64) {
	rx, ry := g.rawPos()
	if g.ToLogical != nil {
		return g.ToLogical(rx, ry)
	}
	return rx, ry
}

// gTouchBuf 预分配的触摸 ID 缓冲区。
// Ebitengine 是单线程模型，无需加锁。容量 8 足够覆盖多点触控场景。
var gTouchBuf [8]ebiten.TouchID

// rawPos 返回原生像素坐标（触摸优先于鼠标）。
// 多指触摸时只取第一个手指（塔防不需要多点操作）。
func (g *Gesture) rawPos() (float64, float64) {
	if ids := ebiten.AppendTouchIDs(gTouchBuf[:0]); len(ids) > 0 {
		tx, ty := ebiten.TouchPosition(ids[0])
		return float64(tx), float64(ty)
	}
	mx, my := ebiten.CursorPosition()
	return float64(mx), float64(my)
}

// isDown 判断是否有指针按住（鼠标左键或任意触摸点）。
func (g *Gesture) isDown() bool {
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(gTouchBuf[:0])) > 0
}

// isJustDown 判断本帧是否刚按下（JustPressed，只在按下的第一帧返回 true）。
func (g *Gesture) isJustDown() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(gTouchBuf[:0])) > 0
}
