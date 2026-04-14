// settings.go — 设置场景。
// 音效/音乐音量滑块 + 画质切换 + 返回按钮。
// 所有改动即时生效并持久化到磁盘。
package scene

import (
	"image/color"
	"math"
	"runtime"
	"strconv"

	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── 布局常量 ──────────────────────────────────────

const (
	settingsPanelW = float32(420)
	settingsPanelH = float32(410)
	settingsRadius = float32(14)

	sliderBarW = float32(200) // 滑块条宽度
	sliderBarH = float32(8)   // 滑块条高度
	sliderKnob = float32(24)  // 滑块手柄直径

	qualityBtnW   = float32(60)
	qualityBtnH   = float32(30)
	qualityBtnGap = float32(12)

	langBtnW   = float32(100)
	langBtnH   = float32(30)
	langBtnGap = float32(16)

	settingsBackW = float32(140)
	settingsBackH = float32(38)
)

// ── SettingsScene ──────────────────────────────────

// SettingsScene 设置场景（音量/画质）。
type SettingsScene struct {
	switcher    Switcher
	returnScene Scene   // 返回时切换到的场景
	sfxEnabled  bool    // 音效总开关
	sfxVol      float64 // 0.0–1.0
	bgmVol      float64 // 0.0–1.0
	quality     int     // 0=High, 1=Medium, 2=Low

	draggingSFX bool // 正在拖动音效滑块
	draggingBGM bool // 正在拖动音乐滑块

	touchID  ebiten.TouchID // 当前触摸 ID（拖动期间追踪）
	hasTouch bool           // 是否正在通过触摸拖动

	bgGrad *draw.CachedGradient // 背景渐变缓存
}

// pointerPos 返回当前指针的逻辑坐标（触摸优先，否则鼠标）。
func (s *SettingsScene) pointerPos() (float64, float64) {
	if s.hasTouch {
		return draw.TouchPos(s.touchID)
	}
	return draw.CursorPos()
}

// NewSettingsScene 创建设置场景。
// returnTo 是按"返回"时切换回的场景（保持引用，不会被 GC）。
func NewSettingsScene(sw Switcher, returnTo Scene) *SettingsScene {
	am := sw.AudioManager()
	sfx := 0.8
	bgm := 0.5
	if am != nil {
		sfx = am.Volume()
		bgm = am.BGMVolume()
	}
	// 从持久化设置读取 sfxEnabled（而非从 Manager，因为 Manager 没有 getter）
	sd := LoadSettings()
	return &SettingsScene{
		switcher:    sw,
		returnScene: returnTo,
		sfxEnabled:  sd.SFXEnabled,
		sfxVol:      sfx,
		bgmVol:      bgm,
		quality:     int(game.CurrentQuality),
		bgGrad:      draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

// ── 坐标辅助 ──────────────────────────────────────

// panelOrigin 返回居中面板左上角。
func panelOrigin() (float32, float32) {
	px := (float32(game.ScreenWidth) - settingsPanelW) / 2
	py := (float32(game.ScreenHeight) - settingsPanelH) / 2
	return px, py
}

// sliderGeom 返回滑块条的 (barX, barY, barW, barH)。
// row: 0=音效, 1=音乐。
func sliderGeom(row int) (float32, float32, float32, float32) {
	px, py := panelOrigin()
	barX := px + (settingsPanelW-sliderBarW)/2 + 20
	barY := py + 90 + float32(row)*50
	return barX, barY, sliderBarW, sliderBarH
}

// sliderValue 将鼠标 X 坐标映射到 [0,1] 滑块值。
func sliderValue(mx float64, barX, barW float32) float64 {
	v := (mx - float64(barX)) / float64(barW)
	return clampF(v, 0, 1)
}

// qualityBtnGeom 返回第 i 个画质按钮的 (x, y, w, h)。
func qualityBtnGeom(i int) (float32, float32, float32, float32) {
	px, py := panelOrigin()
	totalW := 3*qualityBtnW + 2*qualityBtnGap
	startX := px + (settingsPanelW-totalW)/2
	x := startX + float32(i)*(qualityBtnW+qualityBtnGap)
	y := py + 210
	return x, y, qualityBtnW, qualityBtnH
}

// langBtnGeom 返回第 i 个语言按钮的 (x, y, w, h)。
func langBtnGeom(i, count int) (float32, float32, float32, float32) {
	px, py := panelOrigin()
	totalW := float32(count)*langBtnW + float32(count-1)*langBtnGap
	startX := px + (settingsPanelW-totalW)/2
	x := startX + float32(i)*(langBtnW+langBtnGap)
	y := py + 270
	return x, y, langBtnW, langBtnH
}

// backBtnGeom 返回返回按钮的 (x, y, w, h)。
func backBtnGeom() (float32, float32, float32, float32) {
	px, py := panelOrigin()
	x := px + (settingsPanelW-settingsBackW)/2
	y := py + settingsPanelH - settingsBackH - 18
	return x, y, settingsBackW, settingsBackH
}

// ── Update ────────────────────────────────────────

func (s *SettingsScene) Update() error {
	mx, my := s.pointerPos()

	// 按下检测：鼠标或触摸（通过 isTapJustPressed 统一判定）
	if isTapJustPressed() {
		// 捕获触摸 ID（拖动期间持续追踪同一触点）
		if tids := inpututil.JustPressedTouchIDs(); len(tids) > 0 {
			s.touchID = tids[0]
			s.hasTouch = true
			mx, my = draw.TouchPos(s.touchID) // 用触摸坐标覆盖
		}

		// 音效滑块
		if bx, by, bw, bh := sliderGeom(0); s.hitSlider(mx, my, bx, by, bw, bh) {
			s.draggingSFX = true
		}
		// 音乐滑块
		if bx, by, bw, bh := sliderGeom(1); s.hitSlider(mx, my, bx, by, bw, bh) {
			s.draggingBGM = true
		}
		// 画质按钮
		for i := 0; i < 3; i++ {
			qx, qy, qw, qh := qualityBtnGeom(i)
			if mx >= float64(qx) && mx <= float64(qx+qw) && my >= float64(qy) && my <= float64(qy+qh) {
				q := game.QualityLevel(i)
				// WASM 下强制 Low：WebGL uniform 限制
				if runtime.GOOS == "js" && q < game.QualityLow {
					q = game.QualityLow
				}
				if s.quality != int(q) {
					s.quality = int(q)
					game.CurrentQuality = q
					playUIClick(s.switcher)
					s.persist()
				}
			}
		}
		// 语言按钮
		locales := i18n.Available()
		for idx, code := range locales {
			lx, ly, lw, lh := langBtnGeom(idx, len(locales))
			if mx >= float64(lx) && mx <= float64(lx+lw) && my >= float64(ly) && my <= float64(ly+lh) {
				if code != i18n.Locale() {
					i18n.SetLocale(code)
					playUIClick(s.switcher)
					s.persist()
				}
			}
		}
		// 返回按钮
		bkx, bky, bkw, bkh := backBtnGeom()
		if mx >= float64(bkx) && mx <= float64(bkx+bkw) && my >= float64(bky) && my <= float64(bky+bkh) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(s.returnScene)
			return nil
		}
	}

	// 释放检测：鼠标释放 OR 触摸结束 → 结束拖动 + 保存设置
	mouseReleased := !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !s.hasTouch
	touchReleased := s.hasTouch && inpututil.IsTouchJustReleased(s.touchID)
	if mouseReleased || touchReleased {
		if s.draggingSFX || s.draggingBGM {
			s.persist()
		}
		s.draggingSFX = false
		s.draggingBGM = false
		if touchReleased {
			s.hasTouch = false
		}
	}

	// 拖动更新（用 pointerPos 获取当前位置，触摸优先）
	mx, my = s.pointerPos()
	if s.draggingSFX {
		bx, _, bw, _ := sliderGeom(0)
		v := sliderValue(mx, bx, bw)
		if v != s.sfxVol {
			s.sfxVol = v
			if am := s.switcher.AudioManager(); am != nil {
				am.SetVolume(v)
			}
		}
	}
	if s.draggingBGM {
		bx, _, bw, _ := sliderGeom(1)
		v := sliderValue(mx, bx, bw)
		if v != s.bgmVol {
			s.bgmVol = v
			if am := s.switcher.AudioManager(); am != nil {
				am.SetBGMVolume(v)
			}
		}
	}

	// ESC 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.switcher.SwitchScene(s.returnScene)
	}

	return nil
}

// hitSlider 检测点击是否在滑块区域内（含手柄余量）。
func (s *SettingsScene) hitSlider(mx, my float64, bx, by, bw, bh float32) bool {
	margin := float32(sliderKnob)
	return mx >= float64(bx-margin) && mx <= float64(bx+bw+margin) &&
		my >= float64(by-margin) && my <= float64(by+bh+margin)
}

// persist 将当前设置写入磁盘。
func (s *SettingsScene) persist() {
	SaveSettings(SettingsData{
		SFXEnabled: s.sfxEnabled,
		SFXVolume:  s.sfxVol,
		BGMVolume:  s.bgmVol,
		Quality:    s.quality,
		Locale:     i18n.Locale(),
	})
}

// ── Draw ──────────────────────────────────────────

func (s *SettingsScene) Draw(screen *ebiten.Image) {
	// 背景渐变
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	px, py := panelOrigin()

	// 面板背景
	draw.RoundRect(screen, px, py, settingsPanelW, settingsPanelH, settingsRadius, theme.PanelBg)
	draw.StrokeRoundRect(screen, px, py, settingsPanelW, settingsPanelH, settingsRadius, 1, theme.PanelBorder)

	// 标题
	cx := float64(px) + float64(settingsPanelW)/2
	titleY := float64(py) + 24
	fm.DrawCenteredBoldText(screen, i18n.T("settings.title"), cx, titleY, 24, theme.TextTitle)

	// ── 音效音量滑块 ──
	s.drawSlider(screen, fm, 0, i18n.T("settings.sfx_volume"), s.sfxVol)

	// ── 音乐音量滑块 ──
	s.drawSlider(screen, fm, 1, i18n.T("settings.bgm_volume"), s.bgmVol)

	// ── 画质选择 ──
	labelY := float64(py) + 195
	fm.DrawCenteredText(screen, i18n.T("settings.quality"), cx, labelY, theme.FontBody, theme.TextMuted)

	qualityLabels := [3]string{i18n.T("settings.quality.high"), i18n.T("settings.quality.medium"), i18n.T("settings.quality.low")}
	for i := 0; i < 3; i++ {
		qx, qy, qw, qh := qualityBtnGeom(i)
		btnClr := theme.ToneSecondary
		if i == s.quality {
			btnClr = theme.TonePrimary
		}
		ui.Button(screen, qx, qy, qw, qh, qualityLabels[i], ui.ButtonStyle{
			BgColor:   btnClr,
			TextColor: color.White,
			FontSize:  theme.FontBody,
			Radius:    8,
			Bold:      i == s.quality,
		})
	}

	// ── 语言选择 ──
	langLabelY := float64(py) + 255
	fm.DrawCenteredText(screen, i18n.T("settings.language"), cx, langLabelY, theme.FontBody, theme.TextMuted)

	locales := i18n.Available()
	curLocale := i18n.Locale()
	for idx, code := range locales {
		lx, ly, lw, lh := langBtnGeom(idx, len(locales))
		btnClr := theme.ToneSecondary
		if code == curLocale {
			btnClr = theme.TonePrimary
		}
		displayName := i18n.TFromLocale(code, "_meta.name")
		ui.Button(screen, lx, ly, lw, lh, displayName, ui.ButtonStyle{
			BgColor:   btnClr,
			TextColor: color.White,
			FontSize:  theme.FontBody,
			Radius:    8,
			Bold:      code == curLocale,
		})
	}

	// ── 返回按钮 ──
	bkx, bky, bkw, bkh := backBtnGeom()
	ui.Button(screen, bkx, bky, bkw, bkh, i18n.T("settings.back"), ui.ButtonStyle{
		BgColor:   theme.BtnSecondary,
		TextColor: color.White,
		FontSize:  theme.FontH1,
		Radius:    theme.ButtonRadius,
		Bold:      true,
	})

	// ── 作者联系方式（面板底部） ──
	contactY := float64(bky) + float64(bkh) + 12
	contactAddr := "892544825@qq.com"
	fm.DrawCenteredText(screen, i18n.TF("settings.contact_info", contactAddr), cx, contactY, 10, theme.TextMuted)
}

// drawSlider 绘制一个音量滑块行（标签 + 滑块条 + 百分比文字）。
func (s *SettingsScene) drawSlider(screen *ebiten.Image, fm *render.FontManager, row int, label string, value float64) {
	barX, barY, barW, barH := sliderGeom(row)
	px, _ := panelOrigin()

	// 标签（左侧）
	labelX := float64(px) + 30
	labelY := float64(barY) + float64(barH)/2 - float64(theme.FontBody)/2
	fm.DrawText(screen, label, labelX, labelY, theme.FontBody, theme.TextBody)

	// 滑块背景条
	bgClr := color.RGBA{R: 40, G: 50, B: 70, A: 200}
	draw.RoundRect(screen, barX, barY, barW, barH, barH/2, bgClr)

	// 填充条
	fillW := float32(value) * barW
	if fillW > 0 {
		draw.RoundRect(screen, barX, barY, fillW, barH, barH/2, theme.TonePrimary)
	}

	// 手柄（圆形）
	knobX := float64(barX) + float64(fillW)
	knobY := float64(barY) + float64(barH)/2
	knobR := float64(sliderKnob) / 2
	draw.FilledCircle(screen, float32(knobX), float32(knobY), float32(knobR), color.White)

	// 百分比文字（右侧）
	pct := int(math.Round(value * 100))
	pctText := strconv.Itoa(pct) + "%"
	pctX := float64(barX) + float64(barW) + 16
	pctY := labelY
	fm.DrawText(screen, pctText, pctX, pctY, theme.FontBody, theme.TextBody)
}
