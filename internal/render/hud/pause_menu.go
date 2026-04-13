// pause_menu.go — 暂停菜单覆盖层。
// 半透明遮罩 + 居中面板 + 继续/设置/重新开始/返回主菜单四个按钮。
package hud

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
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

const (
	pausePanelW = float32(360)
	pausePanelH = float32(360)
	pauseBtnW   = float32(260)
	pauseBtnH   = float32(46)
	pauseBtnGap = float32(12)
	pauseBtnR   = float32(12)
)

// DrawPauseMenu 渲染暂停菜单覆盖层。
func DrawPauseMenu(screen *ebiten.Image) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	sw := float32(game.ScreenWidth)
	sh := float32(game.ScreenHeight)

	// 半透明全屏遮罩
	draw.RoundRect(screen, 0, 0, sw, sh, 0, color.RGBA{R: 0, G: 0, B: 0, A: 160})

	// 居中面板
	px := (sw - pausePanelW) / 2
	py := (sh - pausePanelH) / 2
	draw.RoundRect(screen, px, py, pausePanelW, pausePanelH, 14, color.RGBA{R: 18, G: 24, B: 42, A: 245})
	draw.StrokeRoundRect(screen, px, py, pausePanelW, pausePanelH, 14, 1, color.RGBA{R: 60, G: 80, B: 120, A: 200})

	// 标题
	cx := float64(sw) / 2
	titleY := float64(py) + 20
	fm.DrawCenteredBoldText(screen, i18n.T("hud.pause.title"), cx, titleY, 24, color.White)

	// 四个按钮
	btnX := (sw - pauseBtnW) / 2
	btnY := py + 80

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
		ui.Button(screen, float32(btnX), float32(btnY), float32(pauseBtnW), float32(pauseBtnH), btn.label, ui.ButtonStyle{
			BgColor:   bgClr,
			TextColor: color.White,
			FontSize:  18,
			Radius:    pauseBtnR,
			Bold:      true,
		})
		btnY += pauseBtnH + pauseBtnGap
	}

	// 底部快捷键提示
	hintY := float64(py) + float64(pausePanelH) - 16
	fm.DrawCenteredText(screen, i18n.T("hud.pause.hint"), cx, hintY, 11, color.RGBA{R: 120, G: 140, B: 170, A: 200})
}

// PauseMenuHitTest 检测暂停菜单点击，返回 PauseResume/PauseRestart/PauseQuit 或 PauseNone。
func PauseMenuHitTest(px, py float32) int {
	sw := float32(game.ScreenWidth)
	sh := float32(game.ScreenHeight)
	panelY := (sh - pausePanelH) / 2

	btnX := (sw - pauseBtnW) / 2
	btnY := panelY + 80

	for i := 0; i < 4; i++ {
		by := btnY + float32(i)*(pauseBtnH+pauseBtnGap)
		if px >= btnX && px <= btnX+pauseBtnW && py >= by && py <= by+pauseBtnH {
			return i + 1 // PauseResume=1, PauseSettings=2, PauseRestart=3, PauseQuit=4
		}
	}
	return PauseNone
}
