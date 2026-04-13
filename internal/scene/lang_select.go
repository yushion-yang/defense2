// lang_select.go — 首次语言选择场景。
// 首次启动时显示，让玩家选择界面语言后进入标题场景。
package scene

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	langCardW   = float32(160)
	langCardH   = float32(80)
	langCardGap = float32(40)
)

// LangSelectScene 首次语言选择场景。
type LangSelectScene struct {
	switcher Switcher
	locales  []string // 可用语言代码
	hover    int      // 悬停索引，-1=无
	bgGrad   *draw.CachedGradient
}

// NewLangSelectScene 创建首次语言选择场景。
func NewLangSelectScene(sw Switcher) *LangSelectScene {
	return &LangSelectScene{
		switcher: sw,
		locales:  i18n.Available(),
		hover:    -1,
		bgGrad:   draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

func (s *LangSelectScene) langCardGeom(idx int) (float32, float32, float32, float32) {
	count := len(s.locales)
	totalW := float32(count)*langCardW + float32(count-1)*langCardGap
	startX := (float32(game.ScreenWidth) - totalW) / 2
	x := startX + float32(idx)*(langCardW+langCardGap)
	y := float32(game.ScreenHeight)/2 - langCardH/2 + 20
	return x, y, langCardW, langCardH
}

func (s *LangSelectScene) Update() error {
	mx, my := draw.CursorPos()

	// 悬停检测
	s.hover = -1
	for idx := range s.locales {
		x, y, w, h := s.langCardGeom(idx)
		if mx >= float64(x) && mx <= float64(x+w) && my >= float64(y) && my <= float64(y+h) {
			s.hover = idx
		}
	}

	// 点击选择
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.JustPressedTouchIDs()) > 0 {
		for idx, code := range s.locales {
			x, y, w, h := s.langCardGeom(idx)
			if mx >= float64(x) && mx <= float64(x+w) && my >= float64(y) && my <= float64(y+h) {
				// 设置语言
				i18n.SetLocale(code)
				// 保存设置
				SaveSettings(SettingsData{
					SFXEnabled: true,
					SFXVolume:  0.8,
					BGMVolume:  0.5,
					Quality:    0,
					Locale:     code,
				})
				// 标记首次运行完成
				pm := persistence.DefaultProgressManager()
				pm.SetFirstRunDone()
				// 进入标题场景
				playUIClick(s.switcher)
				s.switcher.SwitchScene(NewTitleScene(s.switcher))
				return nil
			}
		}
	}

	return nil
}

func (s *LangSelectScene) Draw(screen *ebiten.Image) {
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	cx := float64(game.ScreenWidth) / 2
	cy := float64(game.ScreenHeight) / 2

	// 双语标题
	fm.DrawCenteredBoldText(screen, "选择语言 / Select Language", cx, cy-80, 28, theme.TextTitle)

	// 语言卡片
	for idx, code := range s.locales {
		x, y, w, h := s.langCardGeom(idx)
		displayName := i18n.TFromLocale(code, "_meta.name")

		bgClr := theme.ToneSecondary
		if idx == s.hover {
			bgClr = theme.TonePrimary
		}

		ui.Button(screen, x, y, w, h, displayName, ui.ButtonStyle{
			BgColor:   bgClr,
			TextColor: color.White,
			FontSize:  22,
			Radius:    12,
			Bold:      true,
		})
	}
}
