// pause_menu.go — 暂停菜单覆盖层。
// 半透明遮罩 + 居中面板（PanelBox） + 继续/设置/重新开始/返回主菜单四个按钮。
package hud

import (
	"image/color"

	"defense2/internal/i18n"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// PauseMenuAction 暂停菜单按钮动作。
const (
	PauseNone     = 0
	PauseResume   = 1
	PauseSettings = 2
	PauseRestart  = 3
	PauseQuit     = 4
)

// pauseBtnAbsY 返回第一个按钮的绝对 Y 坐标。
// 推导：PanelBox 内边距(14) + 标题高度(FontPauseTitle+14=38) = 52，
// 再加 28px 间距使按钮组视觉居中 → 面板顶部偏移 80。
func pauseBtnAbsY(panelY float32) float32 {
	return panelY + theme.PanelInnerPad + theme.FontPauseTitle + theme.PanelInnerPad + 28
}

// DrawPauseMenu 渲染暂停菜单覆盖层。
func DrawPauseMenu(screen *ebiten.Image) {
	// 半透明全屏遮罩
	ui.Overlay(screen, 160)

	// 居中面板 + 标题
	sw := float32(theme.CanvasW)
	sh := float32(theme.CanvasH)
	px := (sw - theme.PausePanelW) / 2
	py := (sh - theme.PausePanelH) / 2

	content := ui.PanelBox(screen, px, py, ui.PanelBoxStyle{
		W:         theme.PausePanelW,
		H:         theme.PausePanelH,
		BgColor:   color.RGBA{R: 18, G: 24, B: 42, A: 245},
		Border:    color.RGBA{R: 60, G: 80, B: 120, A: 200},
		Title:     i18n.T("hud.pause.title"),
		TitleFont: theme.FontPauseTitle,
	})

	// 四个按钮
	btnX := (sw - theme.PauseBtnW) / 2
	btnY := pauseBtnAbsY(py)

	buttons := []struct {
		label string
		clr   color.RGBA
	}{
		{i18n.T("hud.pause.resume"), theme.TonePrimary},
		{i18n.T("hud.pause.settings"), theme.BtnSecondary},
		{i18n.T("hud.pause.restart"), color.RGBA{R: 220, G: 160, B: 50, A: 255}},
		{i18n.T("hud.pause.quit"), theme.BtnDanger},
	}

	// Hover detection
	pmx, pmy := draw.CursorPos()
	hoverAction := PauseMenuHitTest(float32(pmx), float32(pmy))

	for i, btn := range buttons {
		bgClr := btn.clr
		if hoverAction == i+1 {
			// Lighten on hover
			bgClr = color.RGBA{
				R: uint8(float64(btn.clr.R) + float64(255-btn.clr.R)*0.15),
				G: uint8(float64(btn.clr.G) + float64(255-btn.clr.G)*0.15),
				B: uint8(float64(btn.clr.B) + float64(255-btn.clr.B)*0.15),
				A: btn.clr.A,
			}
		}
		ui.Button(screen, btnX, btnY, theme.PauseBtnW, theme.PauseBtnH, btn.label, ui.ButtonStyle{
			BgColor:   bgClr,
			TextColor: color.White,
			FontSize:  theme.FontPauseBtn,
			Radius:    theme.PauseBtnR,
			Bold:      true,
		})
		btnY += theme.PauseBtnH + theme.PauseBtnGap
	}

	// 底部快捷键提示
	hintY := float64(content.Y + content.H - 4)
	hintMaxW := float64(content.W)
	ui.Label(screen, i18n.T("hud.pause.hint"), float64(content.X), hintY, hintMaxW, ui.LabelStyle{
		Font:  theme.FontCaption,
		Color: color.RGBA{R: 120, G: 140, B: 170, A: 200},
		Align: ui.AlignCenter,
	})
}

// PauseMenuHitTest 检测暂停菜单点击，返回 PauseResume/PauseRestart/PauseQuit 或 PauseNone。
func PauseMenuHitTest(px, py float32) int {
	sw := float32(theme.CanvasW)
	sh := float32(theme.CanvasH)
	panelY := (sh - theme.PausePanelH) / 2

	btnX := (sw - theme.PauseBtnW) / 2
	btnY := pauseBtnAbsY(panelY)

	for i := 0; i < 4; i++ {
		by := btnY + float32(i)*(theme.PauseBtnH+theme.PauseBtnGap)
		if px >= btnX && px <= btnX+theme.PauseBtnW && py >= by && py <= by+theme.PauseBtnH {
			return i + 1 // PauseResume=1, PauseSettings=2, PauseRestart=3, PauseQuit=4
		}
	}
	return PauseNone
}
