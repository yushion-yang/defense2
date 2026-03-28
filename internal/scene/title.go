// title.go — 标题画面场景。
// 显示游戏标题和开始提示，按 Enter/点击/触摸 后切换到游戏主场景。
package scene

import (
	"image/color"

	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TitleScene 标题画面场景。
type TitleScene struct {
	switcher Switcher // 场景切换器引用
}

// NewTitleScene 创建标题画面。
func NewTitleScene(sw Switcher) *TitleScene {
	return &TitleScene{switcher: sw}
}

// Update 检测输入，按 Enter/点击/触摸 时跳转到游戏主场景。
func (s *TitleScene) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.JustPressedTouchIDs()) > 0 {
		s.switcher.SwitchScene(NewStageScene(s.switcher))
	}
	return nil
}

// Draw 渲染标题画面（深绿背景 + 标题文字 + 开始提示）。
func (s *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 35, B: 20, A: 255})
	cx := game.ScreenWidth / 2
	cy := game.ScreenHeight / 2
	ebitenutil.DebugPrintAt(screen, "TOWER DEFENSE", cx-40, cy-20)
	ebitenutil.DebugPrintAt(screen, "ENTER / TAP to start", cx-58, cy+20)
}
