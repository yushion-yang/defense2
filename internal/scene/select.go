// select.go — 模式选择场景（卡片式 UI）。
//
// 职责：
//   - 展示 5 种游戏模式卡片（战役/无尽/限时/首领/挑战），其中仅战役可玩，其余显示"敬请期待"
//   - 4 档难度选择（简单/普通/困难/极难），默认普通
//   - 根据选中模式分发：campaign→CampaignSelect, test→TestSelect, 其他→直接进 Stage
//   - 右上角设置按钮、右下角图鉴按钮
//   - DevMode 下额外追加 test 模式卡片
//
// UI 布局：标题(呼吸脉冲) → 模式卡片行 → 开始按钮 → 难度标签+按钮行 → 地图名 → 设置(右上)/图鉴(右下) → 底部提示
package scene

import (
	"image/color"
	"math"

	gameAudio "defense2/internal/audio"
	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/particle"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── 游戏模式定义 ────────────────────────────────
// 模式列表在 initGameModes() 中延迟初始化（依赖 i18n 翻译加载完毕）。

type gameModeUI struct {
	ID          string
	Name        string
	Icon        string // 图标字符（SVG id 或 Unicode 符号）
	Description string
	DefaultMap  string
	ComingSoon  bool // true = 显示"敬请期待"锁定卡片，不可选
}

var gameModes []gameModeUI
var gameModesLocale string // 缓存构建时的语言，语言变化时重建

func initGameModes() {
	if len(gameModes) > 0 && gameModesLocale == i18n.Locale() {
		return
	}
	gameModes = []gameModeUI{
		{"campaign", i18n.T("scene.select.mode.campaign"), "stat-damage", i18n.T("scene.select.mode.campaign_desc"), "map_01", true},
		{"classic", i18n.T("scene.select.mode.classic"), "★", i18n.T("scene.select.mode.classic_desc"), "map_01", false},
		{"endless", i18n.T("scene.select.mode.endless"), "∞", i18n.T("scene.select.mode.endless_desc"), "map_01", true},
		{"timedDefense", i18n.T("scene.select.mode.timed"), "stat-atkspd", i18n.T("scene.select.mode.timed_desc"), "map_02", true},
		{"bossRush", i18n.T("scene.select.mode.boss"), "execute", i18n.T("scene.select.mode.boss_desc"), "map_03", true},
		{"challenge", i18n.T("scene.select.mode.challenge"), "★", i18n.T("scene.select.mode.challenge_desc"), "map_04", true},
	}
	if game.DevMode {
		gameModes = append(gameModes, gameModeUI{
			ID: "test", Name: i18n.T("mode.test.name"), Icon: "⚙", Description: i18n.T("scene.test.title"),
			DefaultMap: "map_01", ComingSoon: false,
		})
	}
	gameModesLocale = i18n.Locale()
}

// ── 难度定义 ────────────────────────────────────

type difficultyUI struct {
	ID          string
	Name        string
	Description string
}

var defaultDifficulties []difficultyUI
var defaultDiffLocale string

func initDefaultDifficulties() {
	if len(defaultDifficulties) > 0 && defaultDiffLocale == i18n.Locale() {
		return
	}
	defaultDifficulties = []difficultyUI{
		{"easy", i18n.T("scene.select.diff.easy"), i18n.T("scene.select.diff.easy_desc")},
		{"normal", i18n.T("scene.select.diff.normal"), i18n.T("scene.select.diff.normal_desc")},
		{"hard", i18n.T("scene.select.diff.hard"), i18n.T("scene.select.diff.hard_desc")},
		{"extreme", i18n.T("scene.select.diff.extreme"), i18n.T("scene.select.diff.extreme_desc")},
	}
	defaultDiffLocale = i18n.Locale()
}

// ResetLocaleCache 语言切换时清空缓存，下次访问自动重建。
func ResetLocaleCache() {
	gameModes = nil
	gameModesLocale = ""
	defaultDifficulties = nil
	defaultDiffLocale = ""
}

// ── 布局常量 ────────────────────────────────────
// 所有 UI 元素按逻辑坐标(1200×540)定位，draw 包内部自动处理 HiDPI 缩放。

const (
	scW = float64(game.ScreenWidth)
	scH = float64(game.ScreenHeight)

	// 模式卡片（一行 5 张，居中排列）
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
	diffBtnH   = 40.0
	diffBtnGap = 12.0
	diffLabelY = 280.0
	diffBtnY   = 300.0
)

// rowStartX 计算一行等宽元素居中排列时的起始 X 坐标。
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

	bgGrad *draw.CachedGradient // 背景渐变缓存
}

// NewSelectScene 创建选关场景。
func NewSelectScene(sw Switcher) *SelectScene {
	initGameModes()
	initDefaultDifficulties()
	store, err := persistence.DefaultStorage()
	if err != nil {
		store = persistence.NewMemoryStorage()
	}
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
		bgGrad:       draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

// loadDifficulties 从 settings.json difficulty.modes 加载难度配置。
// 按 easy→normal→hard→extreme 固定顺序构建 UI 列表，加载失败时回退到硬编码默认值。
func loadDifficulties() []difficultyUI {
	initDefaultDifficulties()
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
			Description: m.Label + i18n.T("scene.select.diff_suffix"),
		})
	}
	if len(diffs) == 0 {
		return defaultDifficulties
	}
	return diffs
}

// ── Update ──────────────────────────────────────

// Update 每帧更新：环境粒子 → 悬停检测 → 点击响应。
// 点击优先级：模式卡片 > 难度按钮 > 开始按钮 > 设置按钮。
func (s *SelectScene) Update() error {
	// 语言切换后 gameModes 被清空，确保重建（Settings 返回复用旧实例）
	initGameModes()

	s.frame++
	const dt = 1.0 / 60.0

	// 环境装饰粒子（比 Stage 密集：每 0.5 秒发射一批）
	s.ambientTimer += dt
	if s.ambientTimer >= 0.5 {
		s.ambientTimer -= 0.5
		particle.EmitAmbient(s.particlePool, float64(game.ScreenWidth), float64(game.ScreenHeight))
	}
	s.particlePool.Update(dt)

	// 悬停检测（桌面=鼠标光标, 触摸=长按）
	if hx, hy, hov := draw.HoverPos(); hov {
		s.hoverMode = s.hitTestModeCards(hx, hy)
		s.hoverStart = s.hitTestStartButton(hx, hy)
	} else {
		s.hoverMode = -1
		s.hoverStart = false
	}

	// 键盘快捷键
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewTitleScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		playUIClick(s.switcher)
		s.startGame()
		return nil
	}

	// 鼠标/触摸点击
	mxf, myf := draw.CursorPos()
	if isTapJustPressed() {
		if idx := s.hitTestModeCards(mxf, myf); idx >= 0 {
			if gameModes[idx].ComingSoon {
				hud.ShowToast(i18n.T("scene.select.coming_soon"))
			} else {
				s.selectedMode = idx
				playUIClick(s.switcher)
			}
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
		if s.hitTestBestiaryButton(mxf, myf) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewBestiaryScene(s.switcher))
		}
	}

	return nil
}

// startGame 根据选中的模式执行跳转：
//   - campaign → CampaignSelectScene（关卡选择）
//   - test → TestSelectScene（测试场景选择器）
//   - 其他模式 → 直接创建 StageScene（战灵在 Stage 内第一波倒计时时选择）
func (s *SelectScene) startGame() {
	mode := gameModes[s.selectedMode]
	diff := s.difficulties[s.selectedDiff]
	// 战役/经典模式进入关卡选择
	if mode.ID == "campaign" || mode.ID == "classic" {
		s.switcher.SwitchScene(NewCampaignSelectScene(s.switcher, mode.ID))
		return
	}
	// 测试模式进入测试场景选择器
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

// ── 碰撞检测（UI 元素矩形命中判定）────────────────

// hitTestModeCards 返回鼠标所在的模式卡片索引，无命中返回 -1。
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

// settingsBtn 布局常量（右上角）。
const (
	settingsBtnW = 70.0
	settingsBtnH = 40.0
	settingsBtnX = scW - settingsBtnW - 16
	settingsBtnY = 14.0
)

func (s *SelectScene) hitTestSettingsButton(mx, my float64) bool {
	return mx >= settingsBtnX && mx <= settingsBtnX+settingsBtnW &&
		my >= settingsBtnY && my <= settingsBtnY+settingsBtnH
}

// bestiaryBtn 布局常量（右下角）。
const (
	bestiaryBtnW = 70.0
	bestiaryBtnH = 40.0
	bestiaryBtnX = scW - bestiaryBtnW - 16
	bestiaryBtnY = scH - bestiaryBtnH - 16
)

func (s *SelectScene) hitTestBestiaryButton(mx, my float64) bool {
	return mx >= bestiaryBtnX && mx <= bestiaryBtnX+bestiaryBtnW &&
		my >= bestiaryBtnY && my <= bestiaryBtnY+bestiaryBtnH
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

// Draw 绘制模式选择场景。
// 渲染顺序：渐变背景 → 粒子 → 标题(呼吸缩放) → 模式卡片 → 开始按钮 → 难度区域 → 设置按钮 → 版本号。
func (s *SelectScene) Draw(screen *ebiten.Image) {
	s.bgGrad.Draw(screen, 0, 0)

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
	fm.DrawCenteredBoldText(screen, i18n.T("scene.title.game_name"), scW/2, 28, titleSize, textWhite)
	fm.DrawCenteredText(screen, i18n.T("scene.select.choose_mode"), scW/2, 60, 14, textGray)

	// ── 模式卡片 ──
	startX := rowStartX(cardW, cardGap, len(gameModes))
	for i, mode := range gameModes {
		x := float32(startX + float64(i)*(cardW+cardGap))
		y := float32(cardY)
		w := float32(cardW)
		h := float32(cardH)
		selected := i == s.selectedMode
		hovered := i == s.hoverMode

		if mode.ComingSoon {
			// 锁定模式：暗色卡片 + "敬请期待"
			lockedBg := color.RGBA{R: 20, G: 25, B: 40, A: 255}
			lockedBorder := color.RGBA{R: 40, G: 45, B: 60, A: 255}
			if hovered {
				lockedBg = color.RGBA{R: 28, G: 33, B: 52, A: 255}
			}
			ui.IconCard(screen, x, y, w, h, mode.Icon, mode.Name, i18n.T("scene.select.coming_soon"), ui.IconCardStyle{
				CardStyle: ui.CardStyle{
					BgColor:     lockedBg,
					BorderColor: lockedBorder,
					Radius:      12,
					BorderWidth: 1.5,
				},
				NameColor: theme.TextLocked,
				NameBold:  true,
				DescColor: textDim,
			})
			// 右上角锁定标记
			lockClr := color.RGBA{R: 80, G: 85, B: 100, A: 180}
			fm.DrawText(screen, i18n.T("scene.campaign.locked_tag"), float64(x+w)-30, float64(y)+4, theme.FontXS, lockClr)
		} else {
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
	ui.Button(screen, bx, by, bw, bh, i18n.T("scene.select.start_game"), ui.ButtonStyle{
		BgColor:  btnClr,
		FontSize: 18,
		Radius:   20,
		Bold:     true,
	})

	// ── 设置按钮（右上角） ──
	ui.Button(screen, float32(settingsBtnX), float32(settingsBtnY), float32(settingsBtnW), float32(settingsBtnH), i18n.T("scene.select.settings"), ui.ButtonStyle{
		BgColor:   theme.BtnSecondary,
		TextColor: textGray,
		FontSize:  12,
		Radius:    8,
	})

	// ── 图鉴按钮（右下角） ──
	ui.Button(screen, float32(bestiaryBtnX), float32(bestiaryBtnY), float32(bestiaryBtnW), float32(bestiaryBtnH), i18n.T("scene.title.bestiary"), ui.ButtonStyle{
		BgColor:   theme.BtnSecondary,
		TextColor: textGray,
		FontSize:  12,
		Radius:    8,
	})

	// ── 底部提示 ──
	fm.DrawCenteredText(screen, i18n.T("scene.select.hint"), scW/2, scH-30, 10, textDim)
	// Version text with muted color
	versionColor := color.RGBA{R: 60, G: 65, B: 80, A: 255}
	fm.DrawCenteredText(screen, game.Version, scW/2, scH-12, 9, versionColor)
}

// strokeRect 绘制矩形边框。
func strokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.Color) {
	draw.Line(screen, x, y, x+w, y, width, clr, false)
	draw.Line(screen, x+w, y, x+w, y+h, width, clr, false)
	draw.Line(screen, x+w, y+h, x, y+h, width, clr, false)
	draw.Line(screen, x, y+h, x, y, width, clr, false)
}
