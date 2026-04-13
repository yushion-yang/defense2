// title.go — 标题场景。
// 游戏启动首屏，显示标题和脉冲提示文本。
package scene

import (
	"image/color"
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// isTapJustPressed 检测本帧是否有鼠标左键或触摸点击。
// 长按悬浮释放不算点击。
func isTapJustPressed() bool {
	if draw.LongPressConsumed() {
		return false
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	return len(inpututil.JustPressedTouchIDs()) > 0
}

// playUIClick 播放 UI 点击音效（各场景共用）。
func playUIClick(sw Switcher) {
	if am := sw.AudioManager(); am != nil {
		am.PlaySafe(gameAudio.SFXUIClick)
	}
}

// TitleScene 标题场景。
type TitleScene struct {
	switcher  Switcher
	fontMgr   *render.FontManager
	pulseTime float64 // 脉冲动画计时器
}

// NewTitleScene 创建标题场景。
func NewTitleScene(sw Switcher) *TitleScene {
	return &TitleScene{
		switcher: sw,
		fontMgr:  render.GlobalFont(),
	}
}

func (s *TitleScene) Update() error {
	s.pulseTime += 1.0 / 60.0

	// 点击/触摸进入选关
	if isTapJustPressed() {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
	}

	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	screen.Fill(bgColor) // 深蓝背景

	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// 装饰：几何图案
	drawTitleDecorations(screen, sw, sh)

	if s.fontMgr == nil {
		return
	}
	fm := s.fontMgr

	// 游戏标题
	fm.DrawCenteredBoldText(screen, i18n.T("scene.title.game_name"), sw/2, sh/2-40, 32, textWhite)

	// 脉冲提示
	pulse := 0.5 + 0.5*math.Sin(s.pulseTime*3)
	alpha := uint8(100 + 155*pulse)
	pulseClr := color.RGBA{R: 200, G: 210, B: 220, A: alpha}
	fm.DrawCenteredText(screen, i18n.T("scene.title.tap_to_start"), sw/2, sh/2+40, 14, pulseClr)

	// 版本
	fm.DrawCenteredText(screen, game.Version, sw/2, sh-20, 10, textDim)
}

// drawTitleDecorations 绘制标题画面的几何装饰。
func drawTitleDecorations(screen *ebiten.Image, sw, sh float64) {
	// 上下渐变线
	lineClr := color.RGBA{R: 50, G: 60, B: 90, A: 100}
	y1 := float32(sh/2 - 90)
	y2 := float32(sh/2 + 70)
	cx := float32(sw / 2)
	halfW := float32(200)
	draw.Line(screen, cx-halfW, y1, cx+halfW, y1, 1, lineClr, false)
	draw.Line(screen, cx-halfW, y2, cx+halfW, y2, 1, lineClr, false)

	// 角落小菱形
	dClr := color.RGBA{R: 76, G: 175, B: 80, A: 60}
	r := float32(4)
	for _, pos := range [][2]float32{{cx - halfW, y1}, {cx + halfW, y1}, {cx - halfW, y2}, {cx + halfW, y2}} {
		px, py := pos[0], pos[1]
		draw.Line(screen, px, py-r, px+r, py, 1, dClr, false)
		draw.Line(screen, px+r, py, px, py+r, 1, dClr, false)
		draw.Line(screen, px, py+r, px-r, py, 1, dClr, false)
		draw.Line(screen, px-r, py, px, py-r, 1, dClr, false)
	}
}
