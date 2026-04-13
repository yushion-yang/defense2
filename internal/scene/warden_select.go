// warden_select.go — 战灵选择场景。
// 左侧列表 + 右侧详情面板布局，匹配 JS 版设计。
package scene

import (
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// wardenOptions 从配置加载战灵选项列表（懒加载缓存在 stage_warden_vm.go）。
func wardenOptions() []hud.WardenOption {
	return GetWardenOptions()
}

// ── 布局常量 ────────────────────────────────────

const (
	wListX   = 40.0  // 左侧列表 X
	wListY   = 80.0  // 列表起始 Y
	wListW   = 180.0 // 列表项宽度
	wListH   = 44.0  // 列表项高度
	wListGap = 4.0   // 列表项间距
	wDetailX = 250.0 // 右侧详情面板 X
	wDetailY = 80.0  // 详情面板 Y
	wDetailW = 910.0 // 详情面板宽度
	wDetailH = 380.0 // 详情面板高度
	wBtnY    = 480.0 // 底部按钮 Y
	wBtnW    = 200.0
	wBtnH    = 36.0
	wBtnGap  = 20.0
)

// ── WardenSelectScene ──────────────────────────────

type WardenSelectScene struct {
	switcher       Switcher
	fontMgr        *render.FontManager
	wardenRenderer *render.WardenRenderer
	mapID          string
	modeID         string // 游戏模式 ID
	diffID         string // 难度 ID
	selectedIdx    int
	hoverIdx       int // 列表悬停
	hoverConfirm   bool
	hoverSkip      bool

	bgGrad *draw.CachedGradient // 背景渐变缓存
}

func NewWardenSelectScene(sw Switcher, mapID, modeID, diffID string) *WardenSelectScene {
	return &WardenSelectScene{
		switcher:       sw,
		fontMgr:        render.GlobalFont(),
		wardenRenderer: render.NewWardenRenderer(config.GetAssetFS()),
		mapID:          mapID,
		modeID:         modeID,
		diffID:         diffID,
		selectedIdx:    0,
		hoverIdx:       -1,
		bgGrad:         draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

func (s *WardenSelectScene) Update() error {
	// 悬停检测（桌面=鼠标光标, 触摸=长按）
	if hx, hy, hov := draw.HoverPos(); hov {
		s.hoverIdx = s.hitTestList(hx, hy)
		s.hoverConfirm = s.hitTestBtn(hx, hy, 0)
		s.hoverSkip = s.hitTestBtn(hx, hy, 1)
	} else {
		s.hoverIdx = -1
		s.hoverConfirm = false
		s.hoverSkip = false
	}

	// 键盘快捷键
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		playUIClick(s.switcher)
		s.confirm()
		return nil
	}

	mx, my := draw.CursorPos()
	if isTapJustPressed() {
		// 返回
		if hitTestNavBackBtn(mx, my) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}
		// 列表选择
		if idx := s.hitTestList(mx, my); idx >= 0 {
			s.selectedIdx = idx
			playUIClick(s.switcher)
		}
		// 确认按钮
		if s.hitTestBtn(mx, my, 0) {
			playUIClick(s.switcher)
			s.confirm()
		}
		// 跳过按钮
		if s.hitTestBtn(mx, my, 1) {
			playUIClick(s.switcher)
			s.selectedIdx = len(wardenOptions()) - 1 // "none"
			s.confirm()
		}
	}

	return nil
}

func (s *WardenSelectScene) confirm() {
	opt := wardenOptions()[s.selectedIdx]
	wType := opt.Key
	if wType == "none" {
		wType = ""
	}
	s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
		MapID:        s.mapID,
		WardenType:   wType,
		ModeID:       s.modeID,
		DifficultyID: s.diffID,
	}))
}

func (s *WardenSelectScene) hitTestList(mx, my float64) int {
	for i := range wardenOptions() {
		y := wListY + float64(i)*(wListH+wListGap)
		if mx >= wListX && mx <= wListX+wListW && my >= y && my <= y+wListH {
			return i
		}
	}
	return -1
}

func (s *WardenSelectScene) hitTestBtn(mx, my float64, btnIdx int) bool {
	sw := float64(game.ScreenWidth)
	totalW := wBtnW*2 + wBtnGap
	startX := (sw - totalW) / 2
	bx := startX + float64(btnIdx)*(wBtnW+wBtnGap)
	return mx >= bx && mx <= bx+wBtnW && my >= wBtnY && my <= wBtnY+wBtnH
}

// ── Draw ────────────────────────────────────────

func (s *WardenSelectScene) Draw(screen *ebiten.Image) {
	s.bgGrad.Draw(screen, 0, 0)

	fm := s.fontMgr
	if fm == nil {
		return
	}
	sw := float64(game.ScreenWidth)

	// ── 标题 ──
	fm.DrawCenteredBoldText(screen, i18n.T("scene.warden.title"), sw/2, 20, 22, theme.TextTitle)
	fm.DrawCenteredText(screen, i18n.T("scene.warden.subtitle"), sw/2, 50, 11, theme.TextMuted)

	// ── 返回按钮 ──
	draw.RoundRect(screen, 20, 8, 70, 44, 12, theme.BtnSecondary)
	fm.DrawCenteredText(screen, i18n.T("scene.common.back"), 55, 22, 12, theme.TextBody)

	// ── 左侧列表 ──
	for i, opt := range wardenOptions() {
		x := float32(wListX)
		y := float32(wListY + float64(i)*(wListH+wListGap))
		w := float32(wListW)
		h := float32(wListH)
		selected := i == s.selectedIdx
		hovered := i == s.hoverIdx

		bg := theme.PanelBg
		if selected {
			bg = opt.Color
			bg.A = 180
		} else if hovered {
			bg = color.RGBA{R: 40, G: 50, B: 75, A: 230}
		}
		draw.RoundRect(screen, x, y, w, h, 8, bg)

		if selected {
			draw.StrokeRoundRect(screen, x, y, w, h, 8, 2, opt.Color)
		}

		// 名称
		nameClr := theme.TextBody
		if selected {
			nameClr = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		fm.DrawBoldText(screen, opt.Name, float64(x)+12, float64(y)+8, theme.FontLG, nameClr)

		// 类别
		if opt.Category != "-" {
			catClr := theme.TextMuted
			if opt.Category == i18n.T("game.warden.cat.mobile") {
				catClr = color.RGBA{R: 100, G: 200, B: 130, A: 200}
			} else {
				catClr = color.RGBA{R: 200, G: 160, B: 100, A: 200}
			}
			fm.DrawText(screen, opt.Category, float64(x)+12, float64(y)+26, theme.FontXS, catClr)
		}
	}

	// ── 右侧详情面板 ──
	opt := wardenOptions()[s.selectedIdx]
	s.drawDetail(screen, fm, opt)

	// ── 底部按钮 ──
	totalBtnW := wBtnW*2 + wBtnGap
	btnStartX := (sw - totalBtnW) / 2

	// 确认按钮
	confirmClr := theme.TonePrimary
	if s.hoverConfirm {
		confirmClr = color.RGBA{R: 60, G: 180, B: 100, A: 255}
	}
	draw.RoundRect(screen, float32(btnStartX), float32(wBtnY), float32(wBtnW), float32(wBtnH), 14, confirmClr)
	confirmLabel := i18n.TF("scene.warden.confirm", opt.Name)
	fm.DrawCenteredBoldText(screen, confirmLabel, btnStartX+wBtnW/2, wBtnY+9, theme.FontLG, theme.TextTitle)

	// 跳过按钮
	skipClr := theme.BtnSecondary
	if s.hoverSkip {
		skipClr = theme.BtnMuted
	}
	skipX := btnStartX + wBtnW + wBtnGap
	draw.RoundRect(screen, float32(skipX), float32(wBtnY), float32(wBtnW), float32(wBtnH), 14, skipClr)
	fm.DrawCenteredText(screen, i18n.T("scene.warden.skip"), skipX+wBtnW/2, wBtnY+10, theme.FontMD, theme.TextMuted)
}

func (s *WardenSelectScene) drawDetail(screen *ebiten.Image, fm *render.FontManager, opt hud.WardenOption) {
	x := float32(wDetailX)
	y := float32(wDetailY)
	w := float32(wDetailW)
	h := float32(wDetailH)

	// 面板背景
	draw.RoundRect(screen, x, y, w, h, 12, theme.PanelBg)
	draw.StrokeRoundRect(screen, x, y, w, h, 12, 1, theme.PanelBorder)

	if opt.Key == "none" {
		fm.DrawCenteredText(screen, i18n.T("scene.warden.none_desc"), float64(x)+float64(w)/2, float64(y)+float64(h)/2-10, theme.FontLG, theme.TextMuted)
		return
	}

	px := float64(x) + 20 // padding
	py := float64(y) + 16

	// 战灵精灵预览（右上角）
	if s.wardenRenderer != nil {
		if img := s.wardenRenderer.GetSprite(opt.Key); img != nil {
			previewX := float64(x) + float64(w) - 84
			previewY := float64(y) + 20
			draw.Sprite(screen, img, previewX, previewY, 64)
		}
	}

	// 标题行
	fm.DrawBoldText(screen, i18n.TF("scene.warden.detail_title", opt.Name), px, py, 18, theme.TextTitle)
	py += 24

	// 描述
	fm.DrawText(screen, opt.Description, px, py, theme.FontMD, theme.TextBody)
	py += 28

	// 分割线
	draw.Line(screen, float32(px), float32(py), float32(px)+w-40, float32(py), 1, theme.PanelBorder, false)
	py += 12

	// 攻击模式
	if opt.AttackName != "" {
		fm.DrawBoldText(screen, ">> "+opt.AttackName, px, py, theme.FontMD, theme.TextTitle)
		py += 16
		fm.DrawText(screen, opt.AttackDesc, px+16, py, theme.FontSM, theme.TextBody)
		py += 18
	}

	// 特殊能力
	if opt.SpecialName != "" {
		fm.DrawBoldText(screen, ">> "+opt.SpecialName, px, py, theme.FontMD, color.RGBA{R: 255, G: 180, B: 60, A: 255})
		py += 16
		fm.DrawText(screen, opt.SpecialDesc, px+16, py, theme.FontSM, theme.TextBody)
		py += 18
	}

	// Tips
	for _, tip := range opt.Tips {
		fm.DrawText(screen, tip, px+16, py, theme.FontXS, theme.TextMuted)
		py += 14
	}

	py += 10
	draw.Line(screen, float32(px), float32(py), float32(px)+w-40, float32(py), 1, theme.PanelBorder, false)
	py += 12

	// Lv.1 属性网格 (2行3列)
	fm.DrawBoldText(screen, i18n.T("scene.warden.lv1_stats"), px, py, theme.FontMD, theme.TextTitle)
	py += 20

	attrClr := theme.TextBody
	valClr := theme.TextTitle
	colW := 150.0

	attrs := []struct{ iconName, label, value string }{
		{"stat-damage", i18n.T("game.attr.damage"), opt.Damage},
		{"stat-atkspd", i18n.T("game.attr.interval"), opt.Interval},
		{"stat-movspd", i18n.T("game.attr.speed"), opt.Speed},
		{"stat-splash", i18n.T("game.attr.range"), opt.AoE},
		{"stat-atkspd", i18n.T("game.attr.duration"), opt.Duration},
		{"burn", i18n.T("game.attr.dot"), opt.DoT},
	}

	im := render.GlobalIcons()
	for i, a := range attrs {
		if a.value == "-" || a.value == "" {
			continue
		}
		col := i % 3
		row := i / 3
		ax := px + float64(col)*colW
		ay := py + float64(row)*16
		// 绘制属性图标
		if im != nil {
			if img := im.Get(a.iconName); img != nil {
				draw.Sprite(screen, img, ax+5, ay+5, 10)
			}
		}
		fm.DrawText(screen, a.label, ax+14, ay, theme.FontSM, attrClr)
		fm.DrawBoldText(screen, a.value, ax+44, ay, theme.FontSM, valClr)
	}

	// 成长
	py += 40
	fm.DrawBoldText(screen, i18n.T("scene.warden.growth"), px, py, theme.FontMD, theme.TextTitle)
	py += 18
	growthClr := color.RGBA{R: 76, G: 175, B: 80, A: 255}
	growthText := ""
	if opt.GrowthKill != "-" && opt.GrowthKill != "" {
		growthText += i18n.TF("scene.warden.growth_kill", opt.GrowthKill)
	}
	if opt.GrowthWave != "-" && opt.GrowthWave != "" {
		if growthText != "" {
			growthText += "    "
		}
		growthText += i18n.TF("scene.warden.growth_wave", opt.GrowthWave)
	}
	if growthText != "" {
		fm.DrawText(screen, growthText, px, py, theme.FontSM, growthClr)
	}
}
