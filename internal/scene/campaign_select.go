// campaign_select.go — 战役模式关卡选择场景。
// 显示 8 张地图卡片（2 行 × 4 列）、关卡描述、难度选择和开始按钮。
package scene

import (
	"image/color"
	"strconv"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/particle"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 布局常量 ────────────────────────────────────

const (
	csCardW   = 160.0
	csCardH   = 110.0
	csCardGap = 16.0
	csCols    = 4
	csCardY0  = 70.0 // 第一行卡片 Y

	csDescY   = 330.0 // 描述区 Y
	csDiffY   = 390.0 // 难度按钮 Y
	csBtnY    = 440.0 // 开始按钮 Y
	csBtnW    = 220.0
	csBtnH    = 42.0
	csDiffW   = 80.0
	csDiffH   = 28.0
	csDiffGap = 12.0
)

// 难度颜色映射
var diffColors = map[string]color.RGBA{
	"easy":    {R: 76, G: 175, B: 80, A: 255},   // 绿
	"normal":  {R: 200, G: 200, B: 210, A: 255}, // 白
	"hard":    {R: 255, G: 165, B: 0, A: 255},   // 橙
	"extreme": {R: 220, G: 60, B: 60, A: 255},   // 红
}

// ── CampaignSelectScene ─────────────────────────

// CampaignSelectScene 战役关卡选择。
type CampaignSelectScene struct {
	switcher     Switcher
	fontMgr      *render.FontManager
	levels       []config.LevelEntry
	difficulties []difficultyUI
	selectedMap  int // 0-based index into levels
	selectedDiff int // 0-based index into difficulties
	hoverMap     int
	hoverDiff    int
	hoverStart   bool
	hoverBack    bool

	particlePool *particle.Pool
	ambientTimer float64
}

// NewCampaignSelectScene 创建战役关卡选择场景。
func NewCampaignSelectScene(sw Switcher) *CampaignSelectScene {
	levels, err := config.LoadLevelList()
	if err != nil {
		levels = nil
	}

	return &CampaignSelectScene{
		switcher:     sw,
		fontMgr:      render.GlobalFont(),
		levels:       levels,
		difficulties: loadDifficulties(),
		selectedMap:  0,
		selectedDiff: 1, // 默认普通
		hoverMap:     -1,
		hoverDiff:    -1,
		particlePool: particle.NewPool(),
	}
}

// ── Update ──────────────────────────────────────

func (s *CampaignSelectScene) Update() error {
	const dt = 1.0 / 60.0

	// 环境粒子
	s.ambientTimer += dt
	if s.ambientTimer >= 0.5 {
		s.ambientTimer -= 0.5
		particle.EmitAmbient(s.particlePool, float64(game.ScreenWidth), float64(game.ScreenHeight))
	}
	s.particlePool.Update(dt)

	mx, my := draw.CursorPos()
	s.hoverMap = s.hitTestMapCards(mx, my)
	s.hoverDiff = s.hitTestDiffBtns(mx, my)
	s.hoverStart = s.hitTestStartBtn(mx, my)
	s.hoverBack = mx >= 20 && mx <= 90 && my >= 16 && my <= 44

	if isTapJustPressed() {
		// 返回
		if s.hoverBack {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}
		if idx := s.hitTestMapCards(mx, my); idx >= 0 {
			s.selectedMap = idx
			playUIClick(s.switcher)
		}
		if idx := s.hitTestDiffBtns(mx, my); idx >= 0 {
			s.selectedDiff = idx
			playUIClick(s.switcher)
		}
		if s.hoverStart && len(s.levels) > 0 {
			playUIClick(s.switcher)
			s.startGame()
		}
	}

	return nil
}

func (s *CampaignSelectScene) startGame() {
	if s.selectedMap < 0 || s.selectedMap >= len(s.levels) {
		return
	}
	level := s.levels[s.selectedMap]
	diff := s.difficulties[s.selectedDiff]
	s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
		MapID:        level.ID,
		ModeID:       "campaign",
		DifficultyID: diff.ID,
	}))
}

// ── 碰撞检测 ────────────────────────────────────

func (s *CampaignSelectScene) hitTestMapCards(mx, my float64) int {
	sw := float64(game.ScreenWidth)
	cols := csCols
	if len(s.levels) < cols {
		cols = len(s.levels)
	}
	totalW := float64(cols)*csCardW + float64(cols-1)*csCardGap
	startX := (sw - totalW) / 2

	for i := range s.levels {
		col := i % csCols
		row := i / csCols
		x := startX + float64(col)*(csCardW+csCardGap)
		y := csCardY0 + float64(row)*(csCardH+csCardGap)
		if mx >= x && mx <= x+csCardW && my >= y && my <= y+csCardH {
			return i
		}
	}
	return -1
}

func (s *CampaignSelectScene) hitTestDiffBtns(mx, my float64) int {
	sw := float64(game.ScreenWidth)
	totalW := float64(len(s.difficulties))*csDiffW + float64(len(s.difficulties)-1)*csDiffGap
	startX := (sw - totalW) / 2
	for i := range s.difficulties {
		x := startX + float64(i)*(csDiffW+csDiffGap)
		if mx >= x && mx <= x+csDiffW && my >= csDiffY && my <= csDiffY+csDiffH {
			return i
		}
	}
	return -1
}

func (s *CampaignSelectScene) hitTestStartBtn(mx, my float64) bool {
	sw := float64(game.ScreenWidth)
	bx := (sw - csBtnW) / 2
	return mx >= bx && mx <= bx+csBtnW && my >= csBtnY && my <= csBtnY+csBtnH
}

// ── Draw ────────────────────────────────────────

func (s *CampaignSelectScene) Draw(screen *ebiten.Image) {
	draw.LinearGradientV(screen, 0, 0, game.ScreenWidth, game.ScreenHeight,
		theme.SelectGradTop, theme.SelectGradBot)

	s.particlePool.Draw(screen)

	fm := s.fontMgr
	if fm == nil {
		return
	}

	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// ── 返回按钮 ──
	backBg := theme.BtnSecondary
	if s.hoverBack {
		backBg = theme.BtnMuted
	}
	draw.RoundRect(screen, 20, 16, 70, 28, 12, backBg)
	fm.DrawCenteredText(screen, "<- 返回", 55, 22, theme.FontMD, theme.TextBody)

	// ── 标题 ──
	fm.DrawCenteredText(screen, "战役模式 — 选择关卡", sw/2, 20, 22, theme.TextTitle)

	// ── 地图卡片 ──
	s.drawMapCards(screen, fm)

	// ── 选中关卡描述 ──
	if s.selectedMap >= 0 && s.selectedMap < len(s.levels) {
		desc := s.levels[s.selectedMap].Description
		if desc != "" {
			fm.DrawCenteredText(screen, desc, sw/2, csDescY, 12, theme.TextMuted)
		}
	}

	// ── 难度标签 ──
	fm.DrawCenteredText(screen, "难度", sw/2, csDiffY-18, 12, theme.TextMuted)

	// ── 难度按钮 ──
	s.drawDiffBtns(screen, fm)

	// ── 开始按钮 ──
	bx := float32((sw - csBtnW) / 2)
	by := float32(csBtnY)
	btnClr := greenAccent
	if s.hoverStart {
		btnClr = greenBtnHover
	}
	ui.Button(screen, bx, by, float32(csBtnW), float32(csBtnH), "开始游戏", ui.ButtonStyle{
		BgColor:  btnClr,
		FontSize: 18,
		Radius:   20,
		Bold:     true,
	})

	// ── 底部提示 ──
	fm.DrawCenteredText(screen, "点击卡片选择关卡", sw/2, sh-30, 10, textDim)
	fm.DrawCenteredText(screen, "v0.1.0", sw/2, sh-12, 9, color.RGBA{R: 60, G: 65, B: 80, A: 255})
}

func (s *CampaignSelectScene) drawMapCards(screen *ebiten.Image, fm *render.FontManager) {
	sw := float64(game.ScreenWidth)
	cols := csCols
	if len(s.levels) < cols {
		cols = len(s.levels)
	}
	totalW := float64(cols)*csCardW + float64(cols-1)*csCardGap
	startX := (sw - totalW) / 2

	for i, level := range s.levels {
		col := i % csCols
		row := i / csCols
		x := float32(startX + float64(col)*(csCardW+csCardGap))
		y := float32(csCardY0 + float64(row)*(csCardH+csCardGap))
		w := float32(csCardW)
		h := float32(csCardH)
		selected := i == s.selectedMap
		hovered := i == s.hoverMap

		// 卡片背景
		bg := cardBg
		if hovered && !selected {
			bg = cardHoverBg
		}
		ui.Card(screen, x, y, w, h, ui.CardStyle{
			BgColor:       bg,
			BorderColor:   cardBorder,
			Radius:        12,
			BorderWidth:   1.5,
			Selected:      selected,
			SelectedColor: greenAccent,
			HighlightBar:  true,
			BarWidth:      40,
		})

		cx := float64(x) + float64(w)/2

		// 编号（左上）
		numStr := strconv.Itoa(i + 1)
		if i+1 < 10 {
			numStr = "0" + numStr
		}
		fm.DrawBoldText(screen, numStr, float64(x)+10, float64(y)+8, 12, theme.TextMuted)

		// 星级占位（右上）
		fm.DrawText(screen, "\u2606\u2606\u2606", float64(x)+float64(w)-50, float64(y)+8, 11, theme.TextLocked)

		// 名称（居中）
		fm.DrawCenteredBoldText(screen, level.Name, cx, float64(y)+42, 14, theme.TextTitle)

		// 波数 + 难度标签（底部）
		waveTxt := strconv.Itoa(level.Waves) + "波"
		diffTxt := diffLabel(level.Difficulty)
		infoTxt := waveTxt + "  " + diffTxt
		diffClr := diffLabelColor(level.Difficulty)
		// 用两段绘制：波数白色，难度着色
		waveW := fm.MeasureText(waveTxt+"  ", 11)
		infoX := cx - fm.MeasureText(infoTxt, 11)/2
		fm.DrawText(screen, waveTxt+"  ", infoX, float64(y)+float64(h)-24, 11, theme.TextBody)
		fm.DrawText(screen, diffTxt, infoX+waveW, float64(y)+float64(h)-24, 11, diffClr)
	}
}

func (s *CampaignSelectScene) drawDiffBtns(screen *ebiten.Image, fm *render.FontManager) {
	sw := float64(game.ScreenWidth)
	totalW := float64(len(s.difficulties))*csDiffW + float64(len(s.difficulties)-1)*csDiffGap
	startX := (sw - totalW) / 2

	for i, diff := range s.difficulties {
		dx := float32(startX + float64(i)*(csDiffW+csDiffGap))
		dy := float32(csDiffY)
		dw := float32(csDiffW)
		dh := float32(csDiffH)
		selected := i == s.selectedDiff
		hovered := i == s.hoverDiff

		bg := diffBtnBg
		if hovered && !selected {
			bg = cardHoverBg
		}
		border := diffBtnBorder
		if selected {
			border = diffSelBorder
		}
		ui.Card(screen, dx, dy, dw, dh, ui.CardStyle{
			BgColor:     bg,
			BorderColor: border,
			Radius:      8,
			BorderWidth: 1.5,
		})

		txtClr := textGray
		if selected {
			txtClr = textWhite
		}
		cx := float64(dx) + float64(dw)/2
		fm.DrawCenteredText(screen, diff.Name, cx, float64(dy)+6, 12, txtClr)
	}
}

// ── 辅助 ────────────────────────────────────────

func diffLabel(id string) string {
	switch id {
	case "easy":
		return "简单"
	case "normal":
		return "普通"
	case "hard":
		return "困难"
	case "extreme":
		return "极限"
	default:
		return id
	}
}

func diffLabelColor(id string) color.RGBA {
	if c, ok := diffColors[id]; ok {
		return c
	}
	return color.RGBA{R: 200, G: 200, B: 210, A: 255}
}
