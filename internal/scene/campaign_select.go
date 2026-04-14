// campaign_select.go — 战役模式关卡选择场景。
//
// 职责：
//   - 展示 8 张地图卡片（2 行 × 4 列），逐步解锁（通关前一关解锁下一关）
//   - 锁定关卡显示解锁条件，已解锁关卡显示编号/名称/波数/难度/星级/最高分
//   - 星级评定按当前选中难度读取历史成绩（3星=全通/2星≥80%/1星=胜利）
//   - 难度选择（4档）和开始按钮，选中锁定关卡时按钮灰显
//   - 左上角返回按钮回到模式选择 (SelectScene)
//
// 解锁机制：由 persistence.ProgressManager.IsMapUnlocked() 驱动，
// map_01 默认可玩，后续关卡需通关前置关卡才解锁。
package scene

import (
	"image/color"
	"strconv"

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

// ── 布局常量 ────────────────────────────────────
// 以 "cs" 前缀区分 campaign_select 专属常量，避免与 select.go 冲突。

const (
	csCardW   = 160.0 // 地图卡片宽度
	csCardH   = 110.0 // 地图卡片高度
	csCardGap = 16.0  // 卡片间距
	csCols    = 4     // 每行列数
	csCardY0  = 70.0  // 第一行卡片 Y

	csDescY   = 330.0 // 描述区 Y
	csDiffY   = 390.0 // 难度按钮 Y
	csBtnY    = 440.0 // 开始按钮 Y
	csBtnW    = 220.0
	csBtnH    = 42.0
	csDiffW   = 80.0
	csDiffH   = 40.0
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

// CampaignSelectScene 战役关卡选择（campaign/classic 共用）。
type CampaignSelectScene struct {
	switcher     Switcher
	modeID       string // 游戏模式 ID（"casual" 或 "classic"）
	fontMgr      *render.FontManager
	progressMgr  *persistence.ProgressManager
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

	bgGrad *draw.CachedGradient // 背景渐变缓存
}

// NewCampaignSelectScene 创建战役关卡选择场景。
// modeID 可选：默认 "casual"，传 "classic" 则进入经典模式。
func NewCampaignSelectScene(sw Switcher, modeIDs ...string) *CampaignSelectScene {
	modeID := "casual"
	if len(modeIDs) > 0 && modeIDs[0] != "" {
		modeID = modeIDs[0]
	}
	_ = modeID // 下方赋值
	levels, err := config.LoadLevelList()
	if err != nil {
		levels = nil
	}

	store, err := persistence.DefaultStorage()
	if err != nil {
		store = persistence.NewMemoryStorage()
	}
	pm := persistence.NewProgressManager(store)

	return &CampaignSelectScene{
		switcher:     sw,
		modeID:       modeID,
		fontMgr:      render.GlobalFont(),
		progressMgr:  pm,
		levels:       levels,
		difficulties: loadDifficulties(),
		selectedMap:  0,
		selectedDiff: 1, // 默认普通
		hoverMap:     -1,
		hoverDiff:    -1,
		particlePool: particle.NewPool(),
		bgGrad:       draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

// ── Update ──────────────────────────────────────

// Update 每帧更新：环境粒子 → 悬停检测 → 点击响应。
// 点击优先级：返回按钮 > 地图卡片 > 难度按钮 > 开始按钮。
// 点击锁定关卡时显示 Toast 提示解锁条件，不会选中。
func (s *CampaignSelectScene) Update() error {
	const dt = 1.0 / 60.0

	// 环境粒子
	s.ambientTimer += dt
	if s.ambientTimer >= 0.5 {
		s.ambientTimer -= 0.5
		particle.EmitAmbient(s.particlePool, float64(game.ScreenWidth), float64(game.ScreenHeight))
	}
	s.particlePool.Update(dt)

	// 悬停检测（桌面=鼠标光标, 触摸=长按）
	if hx, hy, hov := draw.HoverPos(); hov {
		s.hoverMap = s.hitTestMapCards(hx, hy)
		s.hoverDiff = s.hitTestDiffBtns(hx, hy)
		s.hoverStart = s.hitTestStartBtn(hx, hy)
		s.hoverBack = hitTestNavBackBtn(hx, hy)
	} else {
		s.hoverMap, s.hoverDiff = -1, -1
		s.hoverStart = false
		s.hoverBack = false
	}

	// 键盘快捷键
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(s.levels) > 0 && s.selectedMap < len(s.levels) && s.progressMgr.IsMapUnlocked(s.modeID,s.levels[s.selectedMap].ID) {
			playUIClick(s.switcher)
			s.startGame()
			return nil
		}
	}

	mx, my := draw.CursorPos()
	if isTapJustPressed() {
		// 返回
		if hitTestNavBackBtn(mx, my) {
			playUIClick(s.switcher)
			s.switcher.SwitchScene(NewSelectScene(s.switcher))
			return nil
		}
		if idx := s.hitTestMapCards(mx, my); idx >= 0 {
			if idx < len(s.levels) && s.progressMgr.IsMapUnlocked(s.modeID,s.levels[idx].ID) {
				s.selectedMap = idx
				playUIClick(s.switcher)
			} else if idx < len(s.levels) {
				// 点击锁定关卡：显示解锁条件
				req := persistence.UnlockRequirement("map", s.levels[idx].ID)
				if req != "" {
					hud.ShowToast(req)
				} else {
					hud.ShowToast(i18n.T("scene.campaign.locked"))
				}
			}
		}
		if idx := s.hitTestDiffBtns(mx, my); idx >= 0 {
			s.selectedDiff = idx
			playUIClick(s.switcher)
		}
		if s.hoverStart && len(s.levels) > 0 {
			// 检查选中关卡是否已解锁
			if s.selectedMap < len(s.levels) && s.progressMgr.IsMapUnlocked(s.modeID,s.levels[s.selectedMap].ID) {
				playUIClick(s.switcher)
				s.startGame()
			}
		}
	}

	return nil
}

// startGame 创建 StageScene 进入战斗，传递选中的地图 ID 和难度 ID。
// 战灵选择在 Stage 内第一波倒计时结束时触发。
func (s *CampaignSelectScene) startGame() {
	if s.selectedMap < 0 || s.selectedMap >= len(s.levels) {
		return
	}
	level := s.levels[s.selectedMap]
	diff := s.difficulties[s.selectedDiff]
	s.switcher.SwitchScene(NewStageSceneWithOpts(s.switcher, StageOptions{
		MapID:        level.ID,
		ModeID:       s.modeID,
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

// Draw 绘制关卡选择场景。
// 渲染顺序：渐变背景 → 粒子 → 返回按钮 → 标题 → 地图卡片(2行4列)
//
//	→ 选中关卡描述 → 难度区域 → 开始按钮 → 底部提示。
func (s *CampaignSelectScene) Draw(screen *ebiten.Image) {
	s.bgGrad.Draw(screen, 0, 0)

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
	draw.RoundRect(screen, 20, 8, 70, 44, 12, backBg)
	fm.DrawCenteredText(screen, i18n.T("scene.common.back"), 55, 22, theme.FontMD, theme.TextBody)

	// ── 标题 ──
	fm.DrawCenteredText(screen, i18n.T("scene.campaign.title"), sw/2, 20, 22, theme.TextTitle)

	// ── 地图卡片 ──
	s.drawMapCards(screen, fm)

	// ── 选中关卡描述 ──
	if s.selectedMap >= 0 && s.selectedMap < len(s.levels) {
		level := s.levels[s.selectedMap]
		if s.progressMgr.IsMapUnlocked(s.modeID,level.ID) {
			if level.Description != "" {
				fm.DrawCenteredText(screen, level.Description, sw/2, csDescY, 12, theme.TextMuted)
			}
		} else {
			req := persistence.UnlockRequirement("map", level.ID)
			if req == "" {
				req = i18n.T("scene.campaign.locked")
			}
			fm.DrawCenteredText(screen, req, sw/2, csDescY, 12, theme.TextLocked)
		}
	}

	// ── 难度标签 ──
	fm.DrawCenteredText(screen, i18n.T("scene.select.difficulty"), sw/2, csDiffY-18, 12, theme.TextMuted)

	// ── 难度按钮 ──
	s.drawDiffBtns(screen, fm)

	// ── 开始按钮 ──
	bx := float32((sw - csBtnW) / 2)
	by := float32(csBtnY)
	selectedLocked := s.selectedMap < len(s.levels) && !s.progressMgr.IsMapUnlocked(s.modeID,s.levels[s.selectedMap].ID)
	btnClr := greenAccent
	if selectedLocked {
		btnClr = color.RGBA{R: 60, G: 70, B: 85, A: 255} // 灰色禁用
	} else if s.hoverStart {
		btnClr = greenBtnHover
	}
	ui.Button(screen, bx, by, float32(csBtnW), float32(csBtnH), i18n.T("scene.select.start_game"), ui.ButtonStyle{
		BgColor:  btnClr,
		FontSize: 18,
		Radius:   20,
		Bold:     true,
	})

	// ── 底部提示 ──
	fm.DrawCenteredText(screen, i18n.T("scene.campaign.hint"), sw/2, sh-30, 10, textDim)
	fm.DrawCenteredText(screen, game.Version, sw/2, sh-12, 9, color.RGBA{R: 60, G: 65, B: 80, A: 255})
}

// drawMapCards 绘制地图卡片网格。
// 锁定卡片：暗色背景 + 锁图标 + 解锁条件文字。
// 已解锁卡片：编号(左上) + 星级(右上) + 名称(居中) + 波数+难度(底部) + 最高分(右下)。
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
		locked := !s.progressMgr.IsMapUnlocked(s.modeID,level.ID)

		// 卡片背景
		bg := cardBg
		if locked {
			bg = color.RGBA{R: 20, G: 25, B: 40, A: 255} // 更暗的锁定背景
		} else if hovered && !selected {
			bg = cardHoverBg
		}

		borderClr := cardBorder
		if locked {
			borderClr = color.RGBA{R: 40, G: 45, B: 60, A: 255} // 暗淡边框
		}
		ui.Card(screen, x, y, w, h, ui.CardStyle{
			BgColor:       bg,
			BorderColor:   borderClr,
			Radius:        12,
			BorderWidth:   1.5,
			Selected:      selected && !locked,
			SelectedColor: greenAccent,
			HighlightBar:  !locked,
			BarWidth:      40,
		})

		cx := float64(x) + float64(w)/2

		if locked {
			// 锁定状态：显示锁图标和解锁条件
			fm.DrawCenteredBoldText(screen, level.Name, cx, float64(y)+28, 13, theme.TextLocked)
			fm.DrawCenteredText(screen, i18n.T("scene.campaign.locked_tag"), cx, float64(y)+52, 14, theme.TextLocked)
			req := persistence.UnlockRequirement("map", level.ID)
			if req != "" {
				fm.DrawCenteredText(screen, req, cx, float64(y)+74, 10, theme.TextLocked)
			}
		} else {
			// 编号（左上）
			numStr := strconv.Itoa(i + 1)
			if i+1 < 10 {
				numStr = "0" + numStr
			}
			fm.DrawBoldText(screen, numStr, float64(x)+10, float64(y)+8, 12, theme.TextMuted)

			// 星级（右上）— 从持久化读取当前难度下的星级
			diffID := s.difficulties[s.selectedDiff].ID
			rec := s.progressMgr.GetMapRecord(s.modeID, diffID, level.ID)
			starStr := "\u2606\u2606\u2606" // 默认三空星
			starClr := theme.TextLocked
			if rec != nil && rec.Stars > 0 {
				filled := rec.Stars
				starStr = ""
				for si := 0; si < 3; si++ {
					if si < filled {
						starStr += "\u2605" // ★
					} else {
						starStr += "\u2606" // ☆
					}
				}
				starClr = theme.ToneGold
			}
			fm.DrawText(screen, starStr, float64(x)+float64(w)-50, float64(y)+8, 11, starClr)

			// 名称（居中）
			fm.DrawCenteredBoldText(screen, level.Name, cx, float64(y)+42, 14, theme.TextTitle)

			// 波数 + 难度标签（底部）
			waveTxt := i18n.TF("scene.campaign.waves", level.Waves)
			diffTxt := diffLabel(level.Difficulty)
			infoTxt := waveTxt + "  " + diffTxt
			diffClr := diffLabelColor(level.Difficulty)
			// 用两段绘制：波数白色，难度着色
			waveW := fm.MeasureText(waveTxt+"  ", 11)
			infoX := cx - fm.MeasureText(infoTxt, 11)/2
			fm.DrawText(screen, waveTxt+"  ", infoX, float64(y)+float64(h)-24, 11, theme.TextBody)
			fm.DrawText(screen, diffTxt, infoX+waveW, float64(y)+float64(h)-24, 11, diffClr)

			// 最高分（底部右下角）
			if rec != nil && rec.BestScore > 0 {
				scoreStr := strconv.Itoa(rec.BestScore)
				fm.DrawRightText(screen, scoreStr, float64(x)+float64(w)-8, float64(y)+float64(h)-24, 10, theme.ToneGold)
			}
		}
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
		fm.DrawCenteredText(screen, diff.Name, cx, float64(dy)+12, 12, txtClr)
	}
}

// ── 辅助函数 ──────────────────────────────────────

// diffLabel 根据难度 ID 返回 i18n 翻译后的显示名称。
func diffLabel(id string) string {
	key := "scene.select.diff." + id
	label := i18n.T(key)
	if label == key {
		return id
	}
	return label
}

// diffLabelColor 返回难度对应的颜色（easy=绿 normal=白 hard=橙 extreme=红）。
func diffLabelColor(id string) color.RGBA {
	if c, ok := diffColors[id]; ok {
		return c
	}
	return color.RGBA{R: 200, G: 200, B: 210, A: 255}
}
