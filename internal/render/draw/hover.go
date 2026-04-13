// hover.go — 全局长按悬浮追踪器，为触摸设备模拟鼠标悬浮行为。
//
// 问题：触摸设备没有鼠标悬浮(hover)概念，但游戏 UI 大量依赖悬浮来显示
// tooltip、高亮可交互元素。
//
// 方案：长按 500ms 不移动 = 模拟鼠标悬浮。手指位置等同光标位置，
// 松开时标记为"已消费"防止触发点击。
//
// 使用方式：
//   - 桌面端(鼠标)：HoverPos() 直接返回 CursorPos()，无额外逻辑
//   - 触摸端：手指按下→等待→达到阈值→进入悬浮态→松开消费
//
// 调用位置：TickHover() 必须在每帧 Game.Update() 最顶部调用，先于所有场景 Update。
package draw

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// 长按悬浮的判定阈值。
const (
	longPressFrames    = 30   // 60fps 下约 500ms，达到此帧数触发悬浮
	longPressMoveLimit = 10.0 // 逻辑像素——手指移动超过此距离则取消判定，视为拖拽
)

// hoverPhase 长按悬浮的三阶段状态机。
type hoverPhase int

const (
	hvIdle    hoverPhase = iota // 空闲：无触摸或使用鼠标
	hvWaiting                   // 等待：手指按下，正在计数帧数
	hvActive                    // 激活：长按达标，手指位置等同鼠标悬浮
)

// 全局悬浮状态（包级变量，Ebitengine 单线程模型无需加锁）。
var (
	hvPhase    hoverPhase // 当前阶段
	hvStartX   float64    // 按下时的逻辑坐标（用于计算移动距离）
	hvStartY   float64
	hvX, hvY   float64           // 当前悬浮位置（逻辑坐标）
	hvFrames   int               // 已持续帧数（未超出移动限制）
	hvConsumed bool              // 松开后标记 1 帧，防止释放事件被当作点击
	hvTouchID  ebiten.TouchID    // 正在追踪的触摸 ID
	hvTracking bool              // hvTouchID 是否有效
	hvTouchBuf [8]ebiten.TouchID // 复用缓冲，避免 AppendTouchIDs 每帧分配
	hvJPBuf    [8]ebiten.TouchID // 复用缓冲，避免 AppendJustPressedTouchIDs 每帧分配
)

// ResetHover 清除长按悬浮追踪器的残留状态。
// 在关卡切换、场景跳转时调用，防止上一局的触摸状态泄漏到下一局。
func ResetHover() {
	hvPhase = hvIdle
	hvFrames = 0
	hvConsumed = false
	hvTracking = false
}

// TickHover 每帧更新长按追踪器（三阶段状态机）。
// 必须在 Game.Update() 最顶部调用，先于所有场景 Update，
// 这样场景代码调用 HoverPos() 时能拿到当前帧的最新状态。
func TickHover() {
	// 清除上一帧的"已消费"标记（只保留 1 帧）。
	hvConsumed = false

	// 检测当前活跃触摸（复用固定缓冲，避免每帧堆分配）。
	ids := ebiten.AppendTouchIDs(hvTouchBuf[:0])
	touching := len(ids) > 0

	switch hvPhase {
	case hvIdle:
		if !touching {
			return
		}
		// 新触摸按下——开始追踪，记录起始位置。
		justPressed := inpututil.AppendJustPressedTouchIDs(hvJPBuf[:0])
		if len(justPressed) == 0 {
			return
		}
		hvTouchID = justPressed[0]
		hvTracking = true
		tx, ty := ebiten.TouchPosition(hvTouchID)
		hvStartX = float64(tx) / Scale
		hvStartY = float64(ty) / Scale
		hvX, hvY = hvStartX, hvStartY
		hvFrames = 0
		hvPhase = hvWaiting

	case hvWaiting:
		if !hvTouchActive(ids) {
			// 在达到阈值前松开——普通点击，不触发悬浮。
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		tx, ty := ebiten.TouchPosition(hvTouchID)
		lx, ly := float64(tx)/Scale, float64(ty)/Scale
		dist := math.Hypot(lx-hvStartX, ly-hvStartY)
		if dist > longPressMoveLimit {
			// 手指移动超过阈值——取消悬浮判定，交给手势系统处理拖拽。
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		hvFrames++
		if hvFrames >= longPressFrames {
			// 达到长按阈值——进入悬浮激活态。
			hvPhase = hvActive
			hvX, hvY = lx, ly
		}

	case hvActive:
		if !hvTouchActive(ids) {
			// 悬浮态松开——标记为"已消费"，防止此次释放被场景当作点击处理。
			hvConsumed = true
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		// 悬浮态移动——持续更新悬浮位置，允许手指滑动查看不同元素的 tooltip。
		tx, ty := ebiten.TouchPosition(hvTouchID)
		hvX, hvY = float64(tx)/Scale, float64(ty)/Scale
	}
}

// HoverPos 返回当前帧的有效悬浮位置，供 UI 高亮/tooltip 使用。
//
// 返回值 (x, y, active)：
//   - 桌面端(鼠标)：始终返回 (光标位置, true)
//   - 触摸端悬浮激活：返回 (长按位置, true)
//   - 触摸端未悬浮：返回 (-1, -1, false)
//
// 场景代码只需检查 active 即可统一处理桌面和触摸两种输入。
func HoverPos() (float64, float64, bool) {
	// 触摸模式下（正在追踪或刚消费），使用长按追踪器的结果。
	if hvTracking || hvConsumed {
		if hvPhase == hvActive {
			return hvX, hvY, true
		}
		return -1, -1, false
	}
	// 桌面模式——鼠标光标始终是有效的悬浮源。
	x, y := CursorPos()
	return x, y, true
}

// LongPressConsumed 返回长按悬浮释放后的 1 帧标记。
// 场景的点击处理应检查此标记，避免将悬浮释放误判为普通点击。
func LongPressConsumed() bool {
	return hvConsumed
}

// hvTouchActive 检查被追踪的触摸 ID 是否仍在屏幕上。
func hvTouchActive(ids []ebiten.TouchID) bool {
	if !hvTracking {
		return false
	}
	for _, id := range ids {
		if id == hvTouchID {
			return true
		}
	}
	return false
}
