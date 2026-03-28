// select.go — 选关场景（卡片式 UI）。
// 提供游戏模式选择、难度选择和开始按钮。
package scene

import (
	"image/color"
	"log"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	{"campaign", "战役", "⚔", "清除所有波次", "map_01"},
	{"endless", "无尽", "∞", "坚持越久越好", "map_01"},
	{"timedDefense", "限时", "⏱", "存活5分钟", "map_02"},
	{"bossRush", "Boss", "♛", "连续挑战Boss", "map_03"},
	{"challenge", "挑战", "★", "特殊规则", "map_04"},
	{"test", "测试", "✎", "所有怪物静止排列", "map_test"},
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
}

// NewSelectScene 创建选关场景。
func NewSelectScene(sw Switcher) *SelectScene {
	store, _ := persistence.DefaultStorage()
	pm := persistence.NewProgressManager(store)

	// 加载字体
	fm := loadFont()

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

	return &SelectScene{
		switcher:     sw,
		progressMgr:  pm,
		fontMgr:      fm,
		selectedMode: 0,
		selectedDiff: 1, // 默认普通
		hoverMode:    -1,
		hoverDiff:    -1,
		difficulties: diffs,
		mapNames:     mapNames,
	}
}

func loadFont() *render.FontManager {
	assetFS := config.GetAssetFS()
	if assetFS == nil {
		return nil
	}
	data, err := assetFS.ReadFile("assets/fonts/NotoSans-Regular.ttf")
	if err != nil {
		log.Printf("字体加载失败: %v", err)
		return nil
	}
	fm, err := render.NewFontManager(data)
	if err != nil {
		log.Printf("字体解析失败: %v", err)
		return nil
	}
	return fm
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
	// 键盘：左右切换模式
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		s.selectedMode--
		if s.selectedMode < 0 {
			s.selectedMode = len(gameModes) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		s.selectedMode++
		if s.selectedMode >= len(gameModes) {
			s.selectedMode = 0
		}
	}

	// 键盘：上下切换难度
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		s.selectedDiff--
		if s.selectedDiff < 0 {
			s.selectedDiff = len(s.difficulties) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		s.selectedDiff++
		if s.selectedDiff >= len(s.difficulties) {
			s.selectedDiff = 0
		}
	}

	// Enter/Space 开始游戏
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.startGame()
		return nil
	}

	// 数字键快选模式
	for i := 0; i < len(gameModes) && i < 6; i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			s.selectedMode = i
		}
	}

	// 鼠标悬停检测
	mx, my := ebiten.CursorPosition()
	mxf, myf := float64(mx), float64(my)
	s.hoverMode = s.hitTestModeCards(mxf, myf)
	s.hoverDiff = s.hitTestDiffButtons(mxf, myf)
	s.hoverStart = s.hitTestStartButton(mxf, myf)

	// 鼠标点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if idx := s.hitTestModeCards(mxf, myf); idx >= 0 {
			s.selectedMode = idx
		}
		if idx := s.hitTestDiffButtons(mxf, myf); idx >= 0 {
			s.selectedDiff = idx
		}
		if s.hitTestStartButton(mxf, myf) {
			s.startGame()
		}
	}

	return nil
}

func (s *SelectScene) startGame() {
	mapID := gameModes[s.selectedMode].DefaultMap
	s.switcher.SwitchScene(NewWardenSelectScene(s.switcher, mapID))
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

// ── Draw ────────────────────────────────────────

var (
	bgColor        = color.RGBA{R: 18, G: 25, B: 45, A: 255}
	cardBg         = color.RGBA{R: 30, G: 38, B: 60, A: 255}
	cardHoverBg    = color.RGBA{R: 40, G: 50, B: 75, A: 255}
	cardBorder     = color.RGBA{R: 60, G: 70, B: 95, A: 255}
	greenAccent    = color.RGBA{R: 76, G: 175, B: 80, A: 255}
	greenBtnHover  = color.RGBA{R: 100, G: 200, B: 100, A: 255}
	diffBtnBg      = color.RGBA{R: 35, G: 42, B: 68, A: 255}
	diffBtnBorder  = color.RGBA{R: 70, G: 80, B: 110, A: 255}
	diffSelBorder  = color.RGBA{R: 100, G: 150, B: 220, A: 255}
	textWhite      = color.RGBA{R: 230, G: 230, B: 235, A: 255}
	textGray       = color.RGBA{R: 140, G: 145, B: 160, A: 255}
	textDim        = color.RGBA{R: 90, G: 95, B: 110, A: 255}
)

func (s *SelectScene) Draw(screen *ebiten.Image) {
	draw.LinearGradientV(screen, 0, 0, game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot)

	if s.fontMgr == nil {
		// 字体加载失败回退
		return
	}

	fm := s.fontMgr

	// ── 标题 ──
	fm.DrawCenteredText(screen, "Mini Tower Defense", scW/2, 28, 28, textWhite)
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

		// 卡片背景
		bg := cardBg
		if hovered && !selected {
			bg = cardHoverBg
		}
		vector.DrawFilledRect(screen, x, y, w, h, bg, false)

		// 卡片边框
		border := cardBorder
		if selected {
			border = greenAccent
		}
		strokeRect(screen, x, y, w, h, 2, border)

		// 选中底部高亮条
		if selected {
			barW := float32(40)
			barH := float32(3)
			vector.DrawFilledRect(screen, x+(w-barW)/2, y+h-8, barW, barH, greenAccent, false)
		}

		// 图标（大号居中）
		cx := float64(x) + float64(w)/2
		fm.DrawCenteredText(screen, mode.Icon, cx, float64(y)+18, 22, textWhite)

		// 名称
		fm.DrawCenteredText(screen, mode.Name, cx, float64(y)+55, 14, textWhite)

		// 描述
		fm.DrawCenteredText(screen, mode.Description, cx, float64(y)+78, 10, textGray)
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
	vector.DrawFilledRect(screen, bx, by, bw, bh, btnClr, false)
	fm.DrawCenteredText(screen, "开始游戏", scW/2, float64(by)+10, 18, textWhite)

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
		vector.DrawFilledRect(screen, dx, dy, dw, dh, bg, false)

		border := diffBtnBorder
		if selected {
			border = diffSelBorder
		}
		strokeRect(screen, dx, dy, dw, dh, 1.5, border)

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

	// ── 底部提示 ──
	fm.DrawCenteredText(screen, "快捷键: ←→选模式 · ↑↓选难度 · Enter开始 · 1-6快选模式", scW/2, scH-30, 10, textDim)
	fm.DrawCenteredText(screen, "Mini Tower Defense v1.0", scW/2, scH-14, 10, textDim)
}

// strokeRect 绘制矩形边框。
func strokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.Color) {
	vector.StrokeLine(screen, x, y, x+w, y, width, clr, false)
	vector.StrokeLine(screen, x+w, y, x+w, y+h, width, clr, false)
	vector.StrokeLine(screen, x+w, y+h, x, y+h, width, clr, false)
	vector.StrokeLine(screen, x, y+h, x, y, width, clr, false)
}
