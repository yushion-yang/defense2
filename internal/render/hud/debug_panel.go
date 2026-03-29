// debug_panel.go — 调试面板（仅测试模式）。
// 右侧固定宽度面板，分区标题 + 按钮列表，自适应高度。
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
)

// debugPanelLayout 计算面板高度和 Y 坐标（Draw 和 HitTest 共享）。
func debugPanelLayout(actions []DebugAction) (panelY, panelH float32) {
	contentH := float32(0)
	for _, act := range actions {
		if act.IsSection {
			contentH += debugSecH + debugBtnGap
		} else {
			contentH += debugBtnH + debugBtnGap
		}
	}
	panelH = debugPanelPad*2 + debugTitleH + contentH
	panelY = float32(54)
	// 面板超出屏幕底部时上移，保证完整显示
	maxBottom := float32(game.ScreenHeight) - 10
	if panelY+panelH > maxBottom {
		panelY = maxBottom - panelH
		if panelY < 4 {
			panelY = 4
		}
	}
	return
}

// DrawDebugPanel 渲染右侧调试面板。
func DrawDebugPanel(screen *ebiten.Image, d DebugPanelData) {
	fm := render.GlobalFont()
	if fm == nil || len(d.Actions) == 0 {
		return
	}

	panelY, panelH := debugPanelLayout(d.Actions)
	panelX := float32(game.ScreenWidth) - debugPanelW - 8

	// Background
	draw.RoundRect(screen, panelX, panelY, debugPanelW, panelH, 10, color.RGBA{R: 15, G: 20, B: 35, A: 230})
	draw.StrokeRoundRect(screen, panelX, panelY, debugPanelW, panelH, 10, 1, color.RGBA{R: 60, G: 80, B: 120, A: 200})

	// Title + close button
	ix := float64(panelX) + float64(debugPanelPad)
	iy := float64(panelY) + float64(debugPanelPad)
	fm.DrawBoldText(screen, "调试面板", ix, iy, theme.FontMD, color.RGBA{R: 120, G: 180, B: 255, A: 255})

	closeX := float64(panelX) + float64(debugPanelW) - float64(debugPanelPad) - float64(debugCloseSize)
	closeY := iy
	draw.RoundRect(screen, float32(closeX), float32(closeY), debugCloseSize, debugCloseSize, 4, color.RGBA{R: 60, G: 40, B: 40, A: 200})
	fm.DrawCenteredVText(screen, "x", closeX+float64(debugCloseSize)/2, closeY+float64(debugCloseSize)/2, 11, color.White)
	iy += float64(debugTitleH)

	// Items
	btnW := debugPanelW - debugPanelPad*2
	sectionClr := color.RGBA{R: 90, G: 110, B: 140, A: 200}
	btnBg := color.RGBA{R: 30, G: 40, B: 65, A: 240}

	for _, act := range d.Actions {
		bx := panelX + debugPanelPad
		if act.IsSection {
			fm.DrawText(screen, "── "+act.Label+" ──", float64(bx), float64(iy)+1, theme.FontXS, sectionClr)
			iy += float64(debugSecH + debugBtnGap)
		} else {
			by := float32(iy)
			draw.RoundRect(screen, bx, by, btnW, debugBtnH, debugBtnR, btnBg)
			cx := float64(bx) + float64(btnW)/2
			cy := float64(by) + float64(debugBtnH)/2 - 5
			fm.DrawCenteredText(screen, act.Label, cx, cy, theme.FontXS, color.White)
			iy += float64(debugBtnH + debugBtnGap)
		}
	}
}

// DebugPanelHitTest 检测点击位置。返回按钮索引（跳过 section）或 -1。
// 返回 -2 表示点击了关闭按钮。
func DebugPanelHitTest(px, py float32, actions []DebugAction) int {
	if len(actions) == 0 {
		return -1
	}
	panelY, _ := debugPanelLayout(actions)
	panelX := float32(game.ScreenWidth) - debugPanelW - 8
	btnW := debugPanelW - debugPanelPad*2

	// Close button hit test
	closeX := panelX + debugPanelW - debugPanelPad - debugCloseSize
	closeY := panelY + debugPanelPad
	if px >= closeX && px <= closeX+debugCloseSize && py >= closeY && py <= closeY+debugCloseSize {
		return -2 // close
	}

	iy := panelY + debugPanelPad + debugTitleH
	for i, act := range actions {
		bx := panelX + debugPanelPad
		if act.IsSection {
			iy += debugSecH + debugBtnGap
			continue
		}
		by := iy
		if px >= bx && px <= bx+btnW && py >= by && py <= by+debugBtnH {
			return i
		}
		iy += debugBtnH + debugBtnGap
	}
	return -1
}
