// select.go — 选关场景（卡片式 UI）。
// 提供游戏模式选择、难度选择和开始按钮。
package scene

import (
	"image/color"
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/particle"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 游戏模式定义 ────────────────────────────────

type gameModeUI struct {
	ID          string
	Name        string
	Icon        string // 图标字符
	Description string
	DefaultMap  string
}

var gameModes = []gameModeUI{
	{"campaign", "战役", "stat-damage", "清除所有波次", "map_01"},
	{"endless", "无尽", "∞", "坚持越久越好", "map_01"},
	{"timedDefense", "限时", "stat-atkspd", "存活5分钟", "map_02"},
	{"bossRush", "Boss", "execute", "连续挑战Boss", "map_03"},
	{"challenge", "挑战", "★", "特殊规则", "map_04"},
	{"test", "测试", "stat-dps", "所有怪物静止排列", "map_test"},
}

// ── 难度定义 ────────────────────────────────────

type difficultyUI struct {
	ID          string
	Name        string
	Description string
}

var defaultDifficulties = []difficultyUI{
	{"easy", "简单", "怪物较弱，金币充足"},
	{"normal", "普通", "标准难度"},
	{"hard", "困难", "怪物更强，金币更少"},
	{"extreme", "极限", "地狱难度"},
}

// ── 布局常量 ────────────────────────────────────

const (
	scW = float64(game.ScreenWidth)
	scH = float64(game.ScreenHeight)

	// 模式卡片
	cardW   = 140.0
	cardH   = 110.0
	cardGap = 16.0
	cardY   = 90.0

	// 开始按钮
	btnW = 220.0
	btnH = 42.0
	btnY = 225.0

	// 难度按钮
	diffBtnW   = 80.0
	diffBtnH   = 28.0
	diffBtnGap = 12.0
	diffLabelY = 280.0
	diffBtnY   = 300.0
)

// 计算居中行起始 X
func rowStartX(itemW, gap float64, count int) float64 {
	totalW := float64(count)*itemW + float64(count-1)*gap
	return (scW - totalW) / 2
}

// ── SelectScene ─────────────────────────────────

// SelectScene 选关场景。
type SelectScene struct {
	switcher     Switcher
	progressMgr  *persistence.ProgressManager
	fontMgr      *render.FontManager
	selectedMode int // 0-5
	selectedDiff int // 0-3
	hoverMode    int // 鼠标悬停 (-1=无)
	hoverDiff    int // 鼠标悬停 (-1=无)
	hoverStart   bool
	difficulties []difficultyUI
	mapNames     map[string]string // mapID → 显示名

	// Ambient atmosphere
	particlePool *particle.Pool // 环境粒子池
	ambientTimer float64        // 粒子发射计时器
	frame        int            // 帧计数（用于动画）
}

// NewSelectScene 创建选关场景。
func NewSelectScene(sw Switcher) *SelectScene {
	store, _ := persistence.DefaultStorage()
	pm := persistence.NewProgressManager(store)

	// 加载难度配置
	diffs := loadDifficulties()

	// 缓存地图名称
	mapNames := make(map[string]string)
	for _, m := range gameModes {
		id := m.DefaultMap
		if _, ok := mapNames[id]; ok {
			continue
		}
		cfg, err := config.LoadMap(id)
		if err != nil {
			mapNames[id] = id
		} else {
			mapNames[id] = cfg.Name
		}
	}

	// BGM: 播放菜单音乐
	sw.AudioManager().PlayBGM(gameAudio.BGMMenu)

	return &SelectScene{
		switcher:     sw,
		progressMgr:  pm,
		fontMgr:      render.GlobalFont(),
		selectedMode: 0,
		selectedDiff: 1, // 默认普通
		hoverMode:    -1,
		hoverDiff:    -1,
		difficulties: diffs,
		mapNames:     mapNames,
		particlePool: particle.NewPool(),
	}
}

func loadDifficulties() []difficultyUI {
	modes, _, err := config.LoadDifficultyModes()
	if err != nil {
		return defaultDifficulties
	}
	// 按固定顺序构建
	order := []string{"easy", "normal", "hard", "extreme"}
	diffs := make([]difficultyUI, 0, len(order))
	for _, id := range order {
		m, ok := modes[id]
		if !ok {
			continue
		}
		diffs = append(diffs, difficultyUI{
			ID:          id,
			Name:        m.Label,
			Description: m.Label + "难度",
		})
	}
	if len(diffs) == 0 {
		return defaultDifficulties
	}
	return diffs
}

// ── Update ──────────────────────────────────────

func (s *SelectScene) Update() error {
	s.frame++
	const dt = 1.0 / 60.0

	// Ambient particles (denser than stage: every 0.5s)
	s.ambientTimer += dt
	if s.ambientTimer >= 0.5 {
		s.ambientTimer -= 0.5
		particle.EmitAmbient(s.particlePool, float64(game.ScreenWidth), float64(game.ScreenHeight))
	}
	s.particlePool.Update(dt)

	// 鼠标悬停检测
	mxf, myf := draw.CursorPos()
	s.hoverMode = s.hitTestModeCards(mxf, myf)
	s.hoverDiff = s.hitTestDiffButtons(mxf, myf)
	s.hoverStart = s.hitTestStartButton(mxf, myf)

	// 鼠标/触摸点击
	if isTapJustPressed() {
		if idx := s.hitTestModeCards(mxf, myf); idx >= 0 {
			s.selectedMode = idx
			playUIClick(s.switcher)
		}
		if idx := s.hitTestDiffButtons(mxf, myf); idx >= 0 {
			s.selectedDiff = idx
			playUIClick(s.switcher)
		}
		if s.hitTestStartButton(mxf, myf) {
			playUIClick(s.switcher)
			s.startGame()
		}
		if s.hitTestSettingsButton(mxf, myf) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSettingsScene(s.switcher, s))
		}
	}

	return nil
}

func (s *SelectScene) startGame() {
	mode := gameModes[s.selectedMode]
	diff := s.difficulties[s.selectedDiff]
	// 战役模式进入关卡选择
	if mode.ID == "campaign" {
		s.switcher.SwitchScene(NewCampaignSelectScene(s.switcher))
		return
	}
	// 测试模式进入专用场景选择器
	if mode.ID == "test" {
		s.switcher.SwitchScene(NewTestSelectScene(s.switcher))
		return
	}
	// 其他模式直接进入 Stage（战灵在 Stage 内第一波倒计时结束时选择）
	s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
		MapID:        mode.DefaultMap,
		ModeID:       mode.ID,
		DifficultyID: diff.ID,
	}))
}

// ── 碰撞检测 ────────────────────────────────────

func (s *SelectScene) hitTestModeCards(mx, my float64) int {
	startX := rowStartX(cardW, cardGap, len(gameModes))
	for i := range gameModes {
		x := startX + float64(i)*(cardW+cardGap)
		if mx >= x && mx <= x+cardW && my >= cardY && my <= cardY+cardH {
			return i
		}
	}
	return -1
}

func (s *SelectScene) hitTestDiffButtons(mx, my float64) int {
	startX := rowStartX(diffBtnW, diffBtnGap, len(s.difficulties))
	for i := range s.difficulties {
		x := startX + float64(i)*(diffBtnW+diffBtnGap)
		if mx >= x && mx <= x+diffBtnW && my >= diffBtnY && my <= diffBtnY+diffBtnH {
			return i
		}
	}
	return -1
}

func (s *SelectScene) hitTestStartButton(mx, my float64) bool {
	x := (scW - btnW) / 2
	return mx >= x && mx <= x+btnW && my >= btnY && my <= btnY+btnH
}

// settingsBtn 布局常量（右下角）。
const (
	settingsBtnW = 70.0
	settingsBtnH = 28.0
	settingsBtnX = scW - settingsBtnW - 16
	settingsBtnY = scH - settingsBtnH - 16
)

func (s *SelectScene) hitTestSettingsButton(mx, my float64) bool {
	return mx >= settingsBtnX && mx <= settingsBtnX+settingsBtnW &&
		my >= settingsBtnY && my <= settingsBtnY+settingsBtnH
}

// ── Draw ────────────────────────────────────────

var (
	bgColor       = color.RGBA{R: 18, G: 25, B: 45, A: 255}
	cardBg        = color.RGBA{R: 30, G: 38, B: 60, A: 255}
	cardHoverBg   = color.RGBA{R: 40, G: 50, B: 75, A: 255}
	cardBorder    = color.RGBA{R: 60, G: 70, B: 95, A: 255}
	greenAccent   = color.RGBA{R: 76, G: 175, B: 80, A: 255}
	greenBtnHover = color.RGBA{R: 100, G: 200, B: 100, A: 255}
	diffBtnBg     = color.RGBA{R: 35, G: 42, B: 68, A: 255}
	diffBtnBorder = color.RGBA{R: 70, G: 80, B: 110, A: 255}
	diffSelBorder = color.RGBA{R: 100, G: 150, B: 220, A: 255}
	textWhite     = color.RGBA{R: 230, G: 230, B: 235, A: 255}
	textGray      = color.RGBA{R: 140, G: 145, B: 160, A: 255}
	textDim       = color.RGBA{R: 90, G: 95, B: 110, A: 255}
)

func (s *SelectScene) Draw(screen *ebiten.Image) {
	draw.LinearGradientV(screen, 0, 0, game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot)

	// Ambient particles (behind all UI)
	s.particlePool.Draw(screen)

	if s.fontMgr == nil {
		// 字体加载失败回退
		return
	}

	fm := s.fontMgr

	// ── 标题（breathing pulse） ──
	animTime := float64(s.frame) / 60.0
	pulse := 1.0 + 0.015*math.Sin(animTime*2) // scale oscillates 0.985 - 1.015
	titleSize := 28.0 * pulse
	fm.DrawCenteredBoldText(screen, "Mini Tower Defense", scW/2, 28, titleSize, textWhite)
	fm.DrawCenteredText(screen, "选择游戏模式", scW/2, 60, 14, textGray)

	// ── 模式卡片 ──
	startX := rowStartX(cardW, cardGap, len(gameModes))
	for i, mode := range gameModes {
		x := float32(startX + float64(i)*(cardW+cardGap))
		y := float32(cardY)
		w := float32(cardW)
		h := float32(cardH)
		selected := i == s.selectedMode
		hovered := i == s.hoverMode

		bg := cardBg
		if hovered && !selected {
			bg = cardHoverBg
		}

		ui.IconCard(screen, x, y, w, h, mode.Icon, mode.Name, mode.Description, ui.IconCardStyle{
			CardStyle: ui.CardStyle{
				BgColor:       bg,
				BorderColor:   cardBorder,
				Radius:        12,
				BorderWidth:   1.5,
				Selected:      selected,
				SelectedColor: greenAccent,
				HighlightBar:  true,
				BarWidth:      40,
			},
			NameColor: textWhite,
			NameBold:  true,
			DescColor: textGray,
		})
	}

	// ── 开始按钮 ──
	bx := float32((scW - btnW) / 2)
	by := float32(btnY)
	bw := float32(btnW)
	bh := float32(btnH)
	btnClr := greenAccent
	if s.hoverStart {
		btnClr = greenBtnHover
	}
	ui.Button(screen, bx, by, bw, bh, "开始游戏", ui.ButtonStyle{
		BgColor:  btnClr,
		FontSize: 18,
		Radius:   20,
		Bold:     true,
	})

	// ── 难度标签 ──
	fm.DrawCenteredText(screen, "难度", scW/2, diffLabelY, 12, textGray)

	// ── 难度按钮 ──
	dStartX := rowStartX(diffBtnW, diffBtnGap, len(s.difficulties))
	for i, diff := range s.difficulties {
		dx := float32(dStartX + float64(i)*(diffBtnW+diffBtnGap))
		dy := float32(diffBtnY)
		dw := float32(diffBtnW)
		dh := float32(diffBtnH)
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

	// ── 难度描述 + 地图名 ──
	if s.selectedDiff < len(s.difficulties) {
		desc := s.difficulties[s.selectedDiff].Description
		fm.DrawCenteredText(screen, desc, scW/2, 340, 10, textGray)
	}
	mapID := gameModes[s.selectedMode].DefaultMap
	mapName := s.mapNames[mapID]
	if mapName == "" {
		mapName = mapID
	}
	fm.DrawCenteredText(screen, "地图: "+mapName, scW/2, 358, 10, textGray)

	// ── 设置按钮（右下角） ──
	ui.Button(screen, float32(settingsBtnX), float32(settingsBtnY), float32(settingsBtnW), float32(settingsBtnH), "设置", ui.ButtonStyle{
		BgColor:   theme.BtnSecondary,
		TextColor: textGray,
		FontSize:  12,
		Radius:    8,
	})

	// ── 底部提示 ──
	fm.DrawCenteredText(screen, "点击卡片选择模式和难度", scW/2, scH-30, 10, textDim)
	// Version text with muted color
	versionColor := color.RGBA{R: 60, G: 65, B: 80, A: 255}
	fm.DrawCenteredText(screen, "v0.1.0", scW/2, scH-12, 9, versionColor)
}

// strokeRect 绘制矩形边框。
func strokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.Color) {
	draw.Line(screen, x, y, x+w, y, width, clr, false)
	draw.Line(screen, x+w, y, x+w, y+h, width, clr, false)
	draw.Line(screen, x+w, y+h, x, y+h, width, clr, false)
	draw.Line(screen, x, y+h, x, y, width, clr, false)
}
