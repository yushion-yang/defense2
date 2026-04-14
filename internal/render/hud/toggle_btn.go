// toggle_btn.go — 左下/右下角的收起/展开切换按钮。
// 小圆角方形按钮，点击切换关联面板的显示/隐藏。
package hud

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	toggleBtnSize   = float32(44)
	toggleBtnRadius = float32(10)
	toggleBtnMargin = float32(12)
)

// toggleBtnRect 返回切换按钮的矩形。left=true 左下角，false=右下角。
func toggleBtnRect(left bool) (x, y float32) {
	y = float32(game.ScreenHeight) - toggleBtnSize - toggleBtnMargin
	if left {
		x = toggleBtnMargin
	} else {
		x = float32(game.ScreenWidth) - toggleBtnSize - toggleBtnMargin
	}
	return
}

// DrawToggleButton 绘制一个收起/展开切换按钮。
// left: true=左下角（波次面板），false=右下角（战灵面板）
// expanded: 当前是否展开
// icon: 按钮显示文字（如 ">" 或 "⚡"）
func DrawToggleButton(screen *ebiten.Image, left, expanded bool, icon string) {
	DrawToggleButtonWithSprite(screen, left, expanded, icon, nil)
}

// DrawToggleButtonWithSprite 绘制带精灵图标的切换按钮。sprite 为 nil 时回退为文字。
func DrawToggleButtonWithSprite(screen *ebiten.Image, left, expanded bool, icon string, sprite *ebiten.Image) {
	x, y := toggleBtnRect(left)

	bg := color.RGBA{R: 30, G: 38, B: 60, A: 200}
	if expanded {
		bg = color.RGBA{R: 40, G: 55, B: 85, A: 220}
	}
	draw.RoundRect(screen, x, y, toggleBtnSize, toggleBtnSize, toggleBtnRadius, bg)
	draw.StrokeRoundRect(screen, x, y, toggleBtnSize, toggleBtnSize, toggleBtnRadius, 1, theme.PanelBorder)

	cx := float64(x) + float64(toggleBtnSize)/2
	cy := float64(y) + float64(toggleBtnSize)/2

	if sprite != nil {
		draw.Sprite(screen, sprite, cx, cy, float64(toggleBtnSize-8))
	} else if fm := render.GlobalFont(); fm != nil {
		label := icon
		if expanded && left {
			label = "<"
		}
		fm.DrawCenteredVText(screen, label, cx, cy, 16, color.White)
	}
}

// ToggleButtonHitTest 检查点击是否命中收起/展开按钮。
// left: true=左下角, false=右下角。
func ToggleButtonHitTest(px, py float32, left bool) bool {
	x, y := toggleBtnRect(left)
	return px >= x && px <= x+toggleBtnSize && py >= y && py <= y+toggleBtnSize
}
