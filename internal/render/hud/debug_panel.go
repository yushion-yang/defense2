// debug_panel.go — 调试面板（仅测试模式）。
// 右侧固定宽度面板，分区标题 + 按钮列表，支持滚动。
package hud

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// DebugAction 调试面板条目。IsSection=true 时为分区标题（不可点击）。
type DebugAction struct {
	Label     string
	Action    func()
	IsSection bool // 分区标题
}

// DebugPanelData 调试面板渲染数据。
type DebugPanelData struct {
	Actions []DebugAction
}

const (
	debugPanelW    = float32(190)
	debugPanelPad  = float32(8)
	debugBtnH      = float32(22)
	debugSecH      = float32(18) // 分区标题高度
	debugBtnGap    = float32(3)
	debugBtnR      = float32(5)
	debugCloseSize = float32(20)
	debugTitleH    = float32(22) // 标题行高度
	debugPanelTop  = float32(54) // 面板顶部 Y
)

// debugScrollY 全局滚动偏移（像素，向下为正）。
var debugScrollY float32

// DebugPanelScroll 接收滚轮增量，更新滚动偏移。由 stage.go 调用。
func DebugPanelScroll(deltaY float64) {
	debugScrollY -= float32(deltaY) * 20 // 每格滚 20px
	if debugScrollY < 0 {
		debugScrollY = 0
	}
}

// debugContentHeight 计算内容总高度。
func debugContentHeight(actions []DebugAction) float32 {
	h := float32(0)
	for _, act := range actions {
		if act.IsSection {
			h += debugSecH + debugBtnGap
		} else {
			h += debugBtnH + debugBtnGap
		}
	}
	return h
}

// debugPanelRect 返回面板可见区域。
func debugPanelRect() (panelX, panelY, panelW, panelH float32) {
	panelX = float32(game.ScreenWidth) - debugPanelW - 8
	panelY = debugPanelTop
	panelW = debugPanelW
	panelH = float32(game.ScreenHeight) - debugPanelTop - 10
	return
}

// DrawDebugPanel 渲染右侧调试面板（支持滚动）。
func DrawDebugPanel(screen *ebiten.Image, d DebugPanelData) {
	fm := render.GlobalFont()
	if fm == nil || len(d.Actions) == 0 {
		return
	}

	panelX, panelY, panelW, panelH := debugPanelRect()
	contentH := debugContentHeight(d.Actions) + debugTitleH + debugPanelPad*2

	// 限制滚动范围
	maxScroll := contentH - panelH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if debugScrollY > maxScroll {
		debugScrollY = maxScroll
	}

	// Background
	draw.RoundRect(screen, panelX, panelY, panelW, panelH, 10, color.RGBA{R: 15, G: 20, B: 35, A: 230})
	draw.StrokeRoundRect(screen, panelX, panelY, panelW, panelH, 10, 1, color.RGBA{R: 60, G: 80, B: 120, A: 200})

	// Title + close button（固定在顶部，不滚动）
	ix := float64(panelX) + float64(debugPanelPad)
	iy := float64(panelY) + float64(debugPanelPad)
	fm.DrawBoldText(screen, "调试面板", ix, iy, theme.FontMD, color.RGBA{R: 120, G: 180, B: 255, A: 255})

	closeX := float64(panelX) + float64(panelW) - float64(debugPanelPad) - float64(debugCloseSize)
	closeY := iy
	draw.RoundRect(screen, float32(closeX), float32(closeY), debugCloseSize, debugCloseSize, 4, color.RGBA{R: 60, G: 40, B: 40, A: 200})
	fm.DrawCenteredVText(screen, "x", closeX+float64(debugCloseSize)/2, closeY+float64(debugCloseSize)/2, 11, color.White)

	// 滚动指示（右侧小条）
	if maxScroll > 0 {
		scrollRatio := debugScrollY / maxScroll
		trackH := panelH - debugTitleH - debugPanelPad*2
		thumbH := trackH * (panelH / contentH)
		if thumbH < 10 {
			thumbH = 10
		}
		thumbY := panelY + debugPanelPad + debugTitleH + (trackH-thumbH)*scrollRatio
		draw.FilledRect(screen, panelX+panelW-4, thumbY, 3, thumbH, color.RGBA{R: 80, G: 100, B: 140, A: 120}, false)
	}

	// Items（带滚动偏移，裁切到面板区域）
	contentTop := panelY + debugPanelPad + debugTitleH
	btnW := panelW - debugPanelPad*2
	sectionClr := color.RGBA{R: 90, G: 110, B: 140, A: 200}
	btnBg := color.RGBA{R: 30, G: 40, B: 65, A: 240}

	itemY := contentTop - debugScrollY
	for _, act := range d.Actions {
		bx := panelX + debugPanelPad
		var h float32
		if act.IsSection {
			h = debugSecH + debugBtnGap
		} else {
			h = debugBtnH + debugBtnGap
		}

		// 裁切：只画在可见区域内的条目
		if itemY+h > contentTop && itemY < panelY+panelH-debugPanelPad {
			if act.IsSection {
				fm.DrawText(screen, "── "+act.Label+" ──", float64(bx), float64(itemY)+1, theme.FontXS, sectionClr)
			} else {
				draw.RoundRect(screen, bx, itemY, btnW, debugBtnH, debugBtnR, btnBg)
				cx := float64(bx) + float64(btnW)/2
				cy := float64(itemY) + float64(debugBtnH)/2 - 5
				fm.DrawCenteredText(screen, act.Label, cx, cy, theme.FontXS, color.White)
			}
		}
		itemY += h
	}
}

// DebugPanelHitTest 检测点击位置。返回按钮索引（跳过 section）或 -1。
// 返回 -2 表示点击了关闭按钮。
func DebugPanelHitTest(px, py float32, actions []DebugAction) int {
	if len(actions) == 0 {
		return -1
	}
	panelX, panelY, panelW, panelH := debugPanelRect()

	// 不在面板范围内
	if px < panelX || px > panelX+panelW || py < panelY || py > panelY+panelH {
		return -1
	}

	// Close button hit test
	closeX := panelX + panelW - debugPanelPad - debugCloseSize
	closeY := panelY + debugPanelPad
	if px >= closeX && px <= closeX+debugCloseSize && py >= closeY && py <= closeY+debugCloseSize {
		return -2 // close
	}

	// Items（带滚动偏移）
	contentTop := panelY + debugPanelPad + debugTitleH
	btnW := panelW - debugPanelPad*2
	itemY := contentTop - debugScrollY

	for i, act := range actions {
		bx := panelX + debugPanelPad
		if act.IsSection {
			itemY += debugSecH + debugBtnGap
			continue
		}
		// 可见区域内才响应点击
		if itemY+debugBtnH > contentTop && itemY < panelY+panelH-debugPanelPad {
			if px >= bx && px <= bx+btnW && py >= itemY && py <= itemY+debugBtnH {
				return i
			}
		}
		itemY += debugBtnH + debugBtnGap
	}
	return -1
}
