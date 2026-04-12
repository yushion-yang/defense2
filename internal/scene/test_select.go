// test_select.go — 测试模式场景选择器。
// 提供预设测试场景 + 自定义 JSON 场景，按类别筛选，卡片式网格布局。
package scene

import (
	"image/color"

	"defense2/internal/config"
	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 测试场景数据 ────────────────────────────────

type testCategory struct {
	ID   string
	Name string
}

var testCategories = []testCategory{
	{"all", "全部"},
	{"tower", "炮塔测试"},
	{"enemy", "怪物测试"},
	{"combo", "综合测试"},
	{"ability", "能力测试"},
	{"dps", "DPS测试"},
	{"bench", "基准测试"},
	{"custom", "自定义"},
}

type testScenario struct {
	ID          string
	Name        string
	Icon        string
	Description string
	Category    string // matches testCategory.ID
	MapID       string
	Gold        int
	Lives       int
	Waves       int
	Color       color.RGBA // card accent color
	EnemyFilter string     // ground-only/flying-only/elite-only/boss-only/all-static/mixed/stress/dummy/none
	ManualWave  bool       // 仅手动开波
}

var testScenarios = []testScenario{
	{"tower-core", "核心炮塔", "tower-laser", "全部核心炮塔，基础战斗测试", "tower", "map_test_large", 9999, 999, 10, color.RGBA{R: 100, G: 150, B: 220, A: 255}, "mixed", false},
	{"enemy-ground", "地面怪物", "stat-movspd", "仅出地面普通怪，路径测试", "enemy", "map_test_large", 9999, 999, 10, color.RGBA{R: 220, G: 120, B: 80, A: 255}, "ground-only", false},
	{"enemy-flying", "飞行怪物", "stat-range", "仅出飞行怪，防空能力测试", "enemy", "map_test_large", 9999, 999, 10, color.RGBA{R: 100, G: 180, B: 220, A: 255}, "flying-only", false},
	{"enemy-boss", "Boss 怪物", "execute", "仅出Boss，机制和伤害上限", "enemy", "map_test_large", 99999, 999, 8, color.RGBA{R: 220, G: 80, B: 80, A: 255}, "boss-only", false},

	{"combo-static", "全怪静止展示", "stat-target", "所有怪物静止排列展示", "combo", "map_test_large", 9999, 999, 0, color.RGBA{R: 80, G: 180, B: 120, A: 255}, "all-static", false},
	{"combo-mixed", "混合波次", "bounce", "地面+飞行+精英混合波次", "combo", "map_test_large", 9999, 999, 12, color.RGBA{R: 80, G: 140, B: 200, A: 255}, "mixed", false},
	{"combo-stress", "压力测试", "stat-splash", "大量怪物高速刷出，性能极限", "combo", "map_test_large", 99999, 99999, 5, color.RGBA{R: 220, G: 100, B: 60, A: 255}, "stress", false},
	{"combo-sandbox", "沙盒模式", "multishot", "无限金币，手动开波，自由测试", "combo", "map_test_large", 99999, 99999, 0, color.RGBA{R: 200, G: 180, B: 80, A: 255}, "none", true},

	{"ability-zone", "区域控制塔", "slow", "减速/范围DOT效果测试", "ability", "map_test_large", 9999, 999, 8, color.RGBA{R: 120, G: 160, B: 200, A: 255}, "mixed", false},
	{"ability-periodic", "周期释放塔", "thunder", "周期AoE/增益/变异效果测试", "ability", "map_test_large", 9999, 999, 8, color.RGBA{R: 180, G: 120, B: 180, A: 255}, "mixed", false},
	{"ability-aura", "光环体系", "tower-aura", "多种光环叠加效果测试", "ability", "map_test_large", 9999, 999, 10, color.RGBA{R: 220, G: 180, B: 80, A: 255}, "mixed", false},
	{"ability-silence", "沉默 vs Boss", "stun", "沉默塔对Boss伤害上限影响", "ability", "map_test_large", 99999, 999, 5, color.RGBA{R: 180, G: 100, B: 100, A: 255}, "boss-only", false},

	{"dps-dummy", "木桩靶场", "hunterInstinct", "超高HP木桩怪，DPS输出测试", "dps", "map_test_large", 99999, 999, 99, color.RGBA{R: 220, G: 160, B: 60, A: 255}, "dummy", false},
	{"bench-lineup", "阵容编辑器", "armorPen", "手动放塔升级，保存阵容仿真", "bench", "map_test_large", 99999, 20, 12, color.RGBA{R: 140, G: 160, B: 180, A: 255}, "mixed", false},
	{"vfx-preview", "特效预览", "stat-splash", "VFX 特效预览与调试工具", "bench", "", 0, 0, 0, color.RGBA{R: 200, G: 100, B: 255, A: 255}, "", false},
	{"audio-preview", "音效预览", "stat-splash", "音效(SFX+BGM)预览与试听工具", "bench", "", 0, 0, 0, color.RGBA{R: 100, G: 200, B: 255, A: 255}, "", false},
	{"wave-preview", "波次预览", "stat-target", "各地图波次出怪组合查看工具", "bench", "", 0, 0, 0, color.RGBA{R: 120, G: 200, B: 160, A: 255}, "", false},
	{"map-editor", "地图编辑", "stat-target", "塔位布局可视化编辑工具", "bench", "", 0, 0, 0, color.RGBA{R: 180, G: 200, B: 100, A: 255}, "", false},
}

// ── 布局常量 ────────────────────────────────────

const (
	tsCardW   = 210.0
	tsCardH   = 100.0
	tsCardGap = 12.0
	tsCols    = 5
	tsCardY0  = 120.0 // 第一行卡片 Y
	tsTabH    = 28.0
	tsTabGap  = 8.0
	tsTabY    = 78.0
	tsBtnW    = 220.0
	tsBtnH    = 38.0
)

// ── TestSelectScene ──────────────────────────────

// TestSelectScene 测试场景选择器。
type TestSelectScene struct {
	switcher        Switcher
	selectedIdx     int // 选中的场景索引 (-1 = 未选)，>=len(testScenarios) 为自定义场景
	hoverIdx        int // 鼠标悬停场景索引
	activeTab       int // 当前活跃的分类标签索引 (0=全部)
	hoverTab        int // 鼠标悬停标签索引
	hoverStart      bool
	filtered        []int          // 当前筛选后的场景索引列表
	customScenarios []testScenario // 从 JSON 加载的自定义场景
}

// NewTestSelectScene 创建测试场景选择器。
func NewTestSelectScene(sw Switcher) *TestSelectScene {
	s := &TestSelectScene{
		switcher:    sw,
		selectedIdx: -1,
		hoverIdx:    -1,
		hoverTab:    -1,
	}
	s.loadCustomScenarios()
	s.updateFilter()
	return s
}

// loadCustomScenarios 从 config/scenarios/ 加载自定义场景。
func (s *TestSelectScene) loadCustomScenarios() {
	scenarios, err := config.LoadScenarios()
	if err != nil || len(scenarios) == 0 {
		return
	}
	for _, sd := range scenarios {
		s.customScenarios = append(s.customScenarios, testScenario{
			ID:          sd.ID,
			Name:        sd.Name,
			Icon:        "multishot",
			Description: sd.Description,
			Category:    "custom",
			MapID:       sd.MapID,
			Gold:        sd.Gold,
			Lives:       sd.Lives,
			Waves:       sd.Waves,
			Color:       color.RGBA{R: 180, G: 140, B: 220, A: 255},
			EnemyFilter: sd.EnemyFilter,
			ManualWave:  sd.ManualWave,
		})
	}
}

// scenarioAt returns the scenario at the given combined index
// (0..len(testScenarios)-1 = builtin, len(testScenarios).. = custom).
func (s *TestSelectScene) scenarioAt(idx int) *testScenario {
	if idx < 0 {
		return nil
	}
	if idx < len(testScenarios) {
		return &testScenarios[idx]
	}
	ci := idx - len(testScenarios)
	if ci < len(s.customScenarios) {
		return &s.customScenarios[ci]
	}
	return nil
}

func (s *TestSelectScene) updateFilter() {
	cat := testCategories[s.activeTab].ID
	s.filtered = nil
	for i, sc := range testScenarios {
		if cat == "all" || sc.Category == cat {
			s.filtered = append(s.filtered, i)
		}
	}
	// Append custom scenarios with offset indices
	base := len(testScenarios)
	for i, sc := range s.customScenarios {
		if cat == "all" || sc.Category == cat {
			s.filtered = append(s.filtered, base+i)
		}
	}
	// 切换分类后清除选择
	s.selectedIdx = -1
}

// ── Update ──────────────────────────────────────

func (s *TestSelectScene) Update() error {
	mxf, myf := draw.CursorPos()

	// 鼠标悬停
	s.hoverTab = s.hitTestTabs(mxf, myf)
	s.hoverIdx = s.hitTestCards(mxf, myf)
	s.hoverStart = s.hitTestStartBtn(mxf, myf)

	// 鼠标/触摸点击
	if isTapJustPressed() {
		// 返回按钮
		if mxf >= 20 && mxf <= 90 && myf >= 16 && myf <= 44 {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}
		if idx := s.hitTestTabs(mxf, myf); idx >= 0 {
			s.activeTab = idx
			s.updateFilter()
			playUIClick(s.switcher)
		}
		if idx := s.hitTestCards(mxf, myf); idx >= 0 {
			s.selectedIdx = s.filtered[idx]
			playUIClick(s.switcher)
		}
		if s.hoverStart && s.selectedIdx >= 0 {
			playUIClick(s.switcher)
			s.startScenario()
		}
	}

	return nil
}

func (s *TestSelectScene) startScenario() {
	sc := s.scenarioAt(s.selectedIdx)
	if sc == nil {
		return
	}
	// VFX preview is a standalone scene — no StageScene needed.
	if sc.ID == "vfx-preview" {
		s.switcher.SwitchScene(NewVFXPreviewScene(s.switcher))
		return
	}
	// Audio preview is a standalone scene — no StageScene needed.
	if sc.ID == "audio-preview" {
		s.switcher.SwitchScene(NewAudioPreviewScene(s.switcher))
		return
	}
	// Wave preview is a standalone scene — no StageScene needed.
	if sc.ID == "wave-preview" {
		s.switcher.SwitchScene(NewWavePreviewScene(s.switcher))
		return
	}
	// Map editor is a standalone scene — no StageScene needed.
	if sc.ID == "map-editor" {
		s.switcher.SwitchScene(NewMapEditorScene(s.switcher))
		return
	}
	s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
		MapID:       sc.MapID,
		WardenType:  "", // 在 Stage 内第一次开波时选择战灵
		Gold:        sc.Gold,
		Lives:       sc.Lives,
		Waves:       sc.Waves,
		TestMode:    true,
		ScenarioID:  sc.ID,
		EnemyFilter: sc.EnemyFilter,
		ManualWave:  sc.ManualWave,
	}))
}

// ── 碰撞检测 ────────────────────────────────────

func (s *TestSelectScene) hitTestTabs(mx, my float64) int {
	sw := float64(game.ScreenWidth)
	tabW := 80.0
	totalW := float64(len(testCategories))*tabW + float64(len(testCategories)-1)*tsTabGap
	startX := (sw - totalW) / 2
	for i := range testCategories {
		x := startX + float64(i)*(tabW+tsTabGap)
		if mx >= x && mx <= x+tabW && my >= tsTabY && my <= tsTabY+tsTabH {
			return i
		}
	}
	return -1
}

func (s *TestSelectScene) hitTestCards(mx, my float64) int {
	sw := float64(game.ScreenWidth)
	cols := tsCols
	if len(s.filtered) < cols {
		cols = len(s.filtered)
	}
	totalW := float64(cols)*tsCardW + float64(cols-1)*tsCardGap
	startX := (sw - totalW) / 2

	for i := range s.filtered {
		col := i % tsCols
		row := i / tsCols
		x := startX + float64(col)*(tsCardW+tsCardGap)
		y := tsCardY0 + float64(row)*(tsCardH+tsCardGap)
		if mx >= x && mx <= x+tsCardW && my >= y && my <= y+tsCardH {
			return i
		}
	}
	return -1
}

func (s *TestSelectScene) hitTestStartBtn(mx, my float64) bool {
	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)
	bx := (sw - tsBtnW) / 2
	by := sh - 50
	return mx >= bx && mx <= bx+tsBtnW && my >= by && my <= by+tsBtnH
}

// ── Draw ────────────────────────────────────────

func (s *TestSelectScene) Draw(screen *ebiten.Image) {
	draw.LinearGradientV(screen, 0, 0, game.ScreenWidth, game.ScreenHeight,
		theme.SelectGradTop, theme.SelectGradBot)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	sw := float64(game.ScreenWidth)
	sh := float64(game.ScreenHeight)

	// ── 返回按钮 ──
	backX, backY := float32(20), float32(16)
	draw.RoundRect(screen, backX, backY, 70, 28, 12, theme.BtnSecondary)
	fm.DrawCenteredText(screen, "<- 返回", float64(backX)+35, float64(backY)+6, theme.FontMD, theme.TextBody)

	// ── 标题 ──
	fm.DrawCenteredText(screen, "测试模式 — 选择场景", sw/2, 20, 22, theme.TextTitle)

	// ── 分类标签 ──
	tabW := 80.0
	totalTabW := float64(len(testCategories))*tabW + float64(len(testCategories)-1)*tsTabGap
	tabStartX := (sw - totalTabW) / 2
	for i, cat := range testCategories {
		x := float32(tabStartX + float64(i)*(tabW+tsTabGap))
		y := float32(tsTabY)
		w := float32(tabW)
		h := float32(tsTabH)
		active := i == s.activeTab
		hovered := i == s.hoverTab

		bg := theme.BtnMuted
		if active {
			bg = theme.BtnPrimary
		} else if hovered {
			bg = theme.BtnSecondary
		}
		draw.RoundRect(screen, x, y, w, h, 10, bg)
		txtClr := theme.TextMuted
		if active {
			txtClr = theme.TextTitle
		}
		fm.DrawCenteredText(screen, cat.Name, float64(x)+float64(w)/2, float64(y)+6, theme.FontSM, txtClr)
	}

	// ── 卡片网格 ──
	cols := tsCols
	if len(s.filtered) < cols {
		cols = len(s.filtered)
	}
	totalCardW := float64(cols)*tsCardW + float64(cols-1)*tsCardGap
	cardStartX := (sw - totalCardW) / 2

	for i, scIdx := range s.filtered {
		sc := s.scenarioAt(scIdx)
		if sc == nil {
			continue
		}
		col := i % tsCols
		row := i / tsCols
		x := float32(cardStartX + float64(col)*(tsCardW+tsCardGap))
		y := float32(tsCardY0 + float64(row)*(tsCardH+tsCardGap))
		w := float32(tsCardW)
		h := float32(tsCardH)
		selected := scIdx == s.selectedIdx
		hovered := i == s.hoverIdx

		// 卡片背景
		bg := theme.PanelBg
		if hovered && !selected {
			bg = color.RGBA{R: 35, G: 45, B: 70, A: 230}
		}
		draw.RoundRect(screen, x, y, w, h, 10, bg)

		// 边框
		borderClr := theme.PanelBorder
		if selected {
			borderClr = sc.Color
		}
		draw.StrokeRoundRect(screen, x, y, w, h, 10, 1.5, borderClr)

		// 选中底部高亮条
		if selected {
			barW := float32(40)
			barH := float32(2)
			draw.FilledRect(screen, x+(w-barW)/2, y+h-6, barW, barH, sc.Color, false)
		}

		// 图标（优先 PNG，回退文本）
		iconDrawn := false
		if im := render.GlobalIcons(); im != nil {
			if img := im.Get(sc.Icon); img != nil {
				draw.Sprite(screen, img, float64(x)+16, float64(y)+16, 16)
				iconDrawn = true
			}
		}
		if !iconDrawn {
			fm.DrawText(screen, sc.Icon, float64(x)+8, float64(y)+6, 16, theme.TextTitle)
		}

		// 卡片内部使用固定字号（不受全局 theme 字号影响）
		const (
			cardNameSize = 13.0
			cardDescSize = 10.0
			cardTagSize  = 10.0
		)

		// 名称
		fm.DrawBoldText(screen, sc.Name, float64(x)+36, float64(y)+8, cardNameSize, theme.TextTitle)

		// 描述（源数据已限定字数，无需截断）
		fm.DrawText(screen, sc.Description, float64(x)+10, float64(y)+32, cardDescSize, theme.TextMuted)

		// 分类标签
		catName := ""
		for _, c := range testCategories {
			if c.ID == sc.Category {
				catName = c.Name
				break
			}
		}
		catClr := dimColor(sc.Color, 0.7)
		fm.DrawText(screen, catName, float64(x)+10, float64(y)+float64(h)-18, cardTagSize, catClr)
	}

	// ── 底部按钮 ──
	bx := float32((sw - tsBtnW) / 2)
	by := float32(sh - 50)
	bw := float32(tsBtnW)
	bh := float32(tsBtnH)

	if s.selectedIdx >= 0 {
		btnClr := theme.TonePrimary
		if s.hoverStart {
			btnClr = color.RGBA{R: 80, G: 200, B: 100, A: 255}
		}
		draw.RoundRect(screen, bx, by, bw, bh, 14, btnClr)
		fm.DrawCenteredText(screen, "开始场景", sw/2, float64(by)+9, theme.FontLG, theme.TextTitle)
	} else {
		draw.RoundRect(screen, bx, by, bw, bh, 14, theme.BtnMuted)
		fm.DrawCenteredText(screen, "请选择场景", sw/2, float64(by)+9, theme.FontLG, theme.TextLocked)
	}
}

// dimColor 降低颜色亮度。
func dimColor(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
		A: c.A,
	}
}
