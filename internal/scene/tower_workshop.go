// tower_workshop.go — 炮塔工坊场景（蓝图 + 自定义能力管理中心）。
//
// 职责：
//   - 展示玩家已保存的蓝图列表和自定义能力列表（两个 Tab 页）
//   - 提供「新建蓝图」「新建能力」入口，跳转到对应编辑场景
//   - 提供蓝图/能力的编辑、删除功能
//
// 导航流：
//   Select → 炮塔工坊 → TowerWorkshopScene → (ESC) → Select
//                      → 新建蓝图 → BlueprintEditScene → (Save/Cancel) → TowerWorkshopScene
//                      → 编辑蓝图 → BlueprintEditScene → (Save/Cancel) → TowerWorkshopScene
//                      → 新建能力 → AbilityEditScene → (Save/Cancel) → TowerWorkshopScene
//                      → 编辑能力 → AbilityEditScene → (Save/Cancel) → TowerWorkshopScene
//
// 关联：
//   - descriptor.BlueprintStore: 蓝图持久化
//   - descriptor.AbilityStore: 能力持久化
//   - blueprint_edit.go / ability_edit.go: 编辑场景
//   - game.go currentSceneName: 需同步注册场景名
package scene

import (
	"fmt"
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/core/persistence"
	"defense2/internal/core/tower/descriptor"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/hud"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── Tab 枚举 ─────────────────────────────────────

const (
	wsTabBlueprints = 0 // 蓝图列表
	wsTabAbilities  = 1 // 自定义能力列表
)

// ── 布局常量 ─────────────────────────────────────

const (
	// 面板
	wsPanelW      = float32(900)
	wsPanelH      = float32(460)
	wsPanelRadius = float32(14)

	// Tab 按钮
	wsTabW   = float32(120)
	wsTabH   = float32(32)
	wsTabGap = float32(12)

	// 卡片网格
	wsCardW   = float32(180)
	wsCardH   = float32(90)
	wsCardGap = float32(10)
	wsCols    = 4

	// 返回/新建按钮
	wsBackBtnW   = float32(70)
	wsBackBtnH   = float32(30)
	wsCreateBtnW = float32(120)
	wsCreateBtnH = float32(32)
)

// ── TowerWorkshopScene ──────────────────────────

// TowerWorkshopScene 炮塔工坊场景，管理蓝图和自定义能力。
type TowerWorkshopScene struct {
	switcher       Switcher
	blueprintStore *descriptor.BlueprintStore
	abilityStore   *descriptor.AbilityStore

	tab     int // 当前 Tab（0=蓝图，1=能力）
	hover   int // 悬停卡片索引，-1=无
	scrollY float64

	// ── 布局缓存（每帧重建） ──
	cardRects     []ui.Rect // 卡片矩形（用于点击检测）
	createBtnRect ui.Rect   // 「新建」按钮矩形

	bgGrad *draw.CachedGradient
}

// NewTowerWorkshopScene 创建工坊场景。
func NewTowerWorkshopScene(sw Switcher) *TowerWorkshopScene {
	store, err := persistence.DefaultStorage()
	if err != nil {
		store = persistence.NewMemoryStorage()
	}

	return &TowerWorkshopScene{
		switcher:       sw,
		blueprintStore: descriptor.NewBlueprintStore(store),
		abilityStore:   descriptor.NewAbilityStore(store),
		tab:            wsTabBlueprints,
		hover:          -1,
		bgGrad:         draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
}

// ── 坐标辅助 ──────────────────────────────────────

// wsPanelOrigin 返回居中面板左上角坐标。
func wsPanelOrigin() (float32, float32) {
	px := (float32(theme.CanvasW) - wsPanelW) / 2
	py := (float32(theme.CanvasH) - wsPanelH) / 2
	return px, py
}

// wsContentRect 返回面板内容区矩形（Tab 和底部之间）。
func wsContentRect(px, py float32) ui.Rect {
	return ui.Rect{
		X: px + 16,
		Y: py + 90,
		W: wsPanelW - 32,
		H: wsPanelH - 110,
	}
}

// ── Update ──────────────────────────────────────

func (s *TowerWorkshopScene) Update() error {
	// ESC 返回选关场景
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}

	// 悬停检测
	s.hover = -1
	if hx, hy, hov := draw.HoverPos(); hov {
		for i, r := range s.cardRects {
			if r.Contains(hx, hy) {
				s.hover = i
				break
			}
		}
	}

	// 鼠标滚轮
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.scrollY -= wy * 20
		s.clampScroll()
	}

	// 点击检测
	if isTapJustPressed() {
		mx, my := draw.CursorPos()
		s.handleInput(mx, my)
	}

	return nil
}

// handleInput 处理点击事件。
//
// 检查顺序：
//  1. 返回按钮
//  2. Tab 按钮
//  3. 新建按钮
//  4. 卡片点击（编辑）
func (s *TowerWorkshopScene) handleInput(mx, my float64) {
	px, py := wsPanelOrigin()

	// 1. 返回按钮（左上角）
	backX := float64(px) + 16
	backY := float64(py) + 12
	if mx >= backX && mx <= backX+float64(wsBackBtnW) &&
		my >= backY && my <= backY+float64(wsBackBtnH) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return
	}

	// 2. Tab 按钮
	for i := 0; i < 2; i++ {
		tx, ty := s.tabGeom(i)
		if mx >= float64(tx) && mx <= float64(tx+wsTabW) &&
			my >= float64(ty) && my <= float64(ty+wsTabH) {
			if s.tab != i {
				playUIClick(s.switcher)
				s.tab = i
				s.scrollY = 0
				s.hover = -1
			}
			return
		}
	}

	// 3. 新建按钮
	if s.createBtnRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.handleCreate()
		return
	}

	// 4. 卡片点击（编辑）
	for i, r := range s.cardRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleCardClick(i)
			return
		}
	}
}

// handleCreate 根据当前 Tab 创建新蓝图或新能力。
func (s *TowerWorkshopScene) handleCreate() {
	switch s.tab {
	case wsTabBlueprints:
		if s.blueprintStore.Count() >= 20 {
			hud.ShowToast("蓝图数量已达上限 (20)")
			return
		}
		scene := NewBlueprintEditScene(s.switcher, nil, s, s.blueprintStore, s.abilityStore)
		s.switcher.SwitchScene(scene)
	case wsTabAbilities:
		if s.abilityStore.Count() >= 50 {
			hud.ShowToast("自定义能力已达上限 (50)")
			return
		}
		scene := NewAbilityEditScene(s.switcher, nil, s.abilityStore, s)
		s.switcher.SwitchScene(scene)
	}
}

// handleCardClick 处理卡片点击 — 打开编辑器。
func (s *TowerWorkshopScene) handleCardClick(idx int) {
	switch s.tab {
	case wsTabBlueprints:
		bps := s.blueprintStore.List()
		if idx < 0 || idx >= len(bps) {
			return
		}
		bp := bps[idx]
		scene := NewBlueprintEditScene(s.switcher, &bp, s, s.blueprintStore, s.abilityStore)
		s.switcher.SwitchScene(scene)
	case wsTabAbilities:
		cas := s.abilityStore.List()
		if idx < 0 || idx >= len(cas) {
			return
		}
		ca := cas[idx]
		scene := NewAbilityEditScene(s.switcher, &ca, s.abilityStore, s)
		s.switcher.SwitchScene(scene)
	}
}

// tabGeom 返回 Tab 按钮的 (x, y) 坐标。
func (s *TowerWorkshopScene) tabGeom(idx int) (float32, float32) {
	px, py := wsPanelOrigin()
	totalW := 2*wsTabW + wsTabGap
	startX := px + (wsPanelW-totalW)/2
	return startX + float32(idx)*(wsTabW+wsTabGap), py + 50
}

// clampScroll 将滚动偏移限制在合法范围内。
func (s *TowerWorkshopScene) clampScroll() {
	var count int
	switch s.tab {
	case wsTabBlueprints:
		count = s.blueprintStore.Count()
	case wsTabAbilities:
		count = s.abilityStore.Count()
	}

	rows := (count + wsCols - 1) / wsCols
	contentH := float64(rows)*float64(wsCardH+wsCardGap) + float64(wsCreateBtnH) + 20

	_, py := wsPanelOrigin()
	cr := wsContentRect(0, py)
	viewH := float64(cr.H)

	maxScroll := contentH - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	if s.scrollY > maxScroll {
		s.scrollY = maxScroll
	}
}

// ── Draw ────────────────────────────────────────

func (s *TowerWorkshopScene) Draw(screen *ebiten.Image) {
	// 背景渐变
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	px, py := wsPanelOrigin()

	// 面板背景
	ui.Panel(screen, px, py, wsPanelW, wsPanelH, ui.PanelStyle{
		BgColor:     theme.PanelBg,
		BorderColor: theme.PanelBorder,
		Radius:      wsPanelRadius,
	})

	// ── 标题 ──
	cx := float64(px) + float64(wsPanelW)/2
	fm.DrawCenteredBoldText(screen, "炮塔工坊", cx, float64(py)+20, theme.FontOverlayTitle, theme.TextTitle)

	// ── 返回按钮（左上角） ──
	ui.Button(screen, px+16, py+12, wsBackBtnW, wsBackBtnH, "← 返回", ui.ButtonStyle{
		BgColor:  theme.BtnSecondary,
		FontSize: theme.FontBody,
		Radius:   8,
	})

	// ── Tab 按钮 ──
	tabLabels := [2]string{"蓝图", "自定义能力"}
	for i := 0; i < 2; i++ {
		tx, ty := s.tabGeom(i)
		btnClr := color.Color(theme.ToneSecondary)
		if i == s.tab {
			btnClr = theme.TonePrimary
		}
		label := fmt.Sprintf("%s (%d)", tabLabels[i], s.tabCount(i))
		ui.Button(screen, tx, ty, wsTabW, wsTabH, label, ui.ButtonStyle{
			BgColor:  btnClr,
			FontSize: theme.FontBody,
			Radius:   8,
			Bold:     i == s.tab,
		})
	}

	// ── 内容区 ──
	cr := wsContentRect(px, py)

	// 内容区背景
	draw.RoundRect(screen, cr.X, cr.Y, cr.W, cr.H, 10,
		color.RGBA{R: 10, G: 15, B: 30, A: 150})

	// 绘制卡片列表
	s.cardRects = s.cardRects[:0]
	s.createBtnRect = ui.Rect{}

	switch s.tab {
	case wsTabBlueprints:
		s.drawBlueprintCards(screen, fm, cr)
	case wsTabAbilities:
		s.drawAbilityCards(screen, fm, cr)
	}
}

// tabCount 返回指定 Tab 的项目数量。
func (s *TowerWorkshopScene) tabCount(tab int) int {
	switch tab {
	case wsTabBlueprints:
		return s.blueprintStore.Count()
	case wsTabAbilities:
		return s.abilityStore.Count()
	}
	return 0
}

// ── 蓝图卡片列表 ──────────────────────────────────

func (s *TowerWorkshopScene) drawBlueprintCards(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	bps := s.blueprintStore.List()

	startY := float32(float64(cr.Y) + 8 - s.scrollY)
	gridW, _ := ui.CardGridSize(len(bps), ui.CardGridStyle{Cols: wsCols, CardW: wsCardW, CardH: wsCardH, Gap: wsCardGap})
	cardX := cr.X + (cr.W-gridW)/2
	if len(bps) < wsCols {
		cardX = cr.X + 12
	}

	for i, bp := range bps {
		col := i % wsCols
		row := i / wsCols
		x := cardX + float32(col)*(wsCardW+wsCardGap)
		y := startY + float32(row)*(wsCardH+wsCardGap)

		r := ui.Rect{X: x, Y: y, W: wsCardW, H: wsCardH}
		s.cardRects = append(s.cardRects, r)

		// 裁剪检查
		if float64(y+wsCardH) < float64(cr.Y) || float64(y) > float64(cr.Y+cr.H) {
			continue
		}

		hovered := i == s.hover
		bgClr := color.RGBA{R: 25, G: 32, B: 55, A: 220}
		borderClr := color.RGBA{R: 60, G: 70, B: 95, A: 200}
		if hovered {
			bgClr = color.RGBA{R: 35, G: 45, B: 70, A: 230}
			borderClr = color.RGBA{R: 80, G: 100, B: 140, A: 220}
		}

		ui.Card(screen, x, y, wsCardW, wsCardH, ui.CardStyle{
			BgColor:     bgClr,
			BorderColor: borderClr,
			Radius:      10,
			BorderWidth: 1.5,
		})

		cx := float64(x) + float64(wsCardW)/2

		// 蓝图名称
		name := bp.Name
		if name == "" {
			name = bp.ID
		}
		ui.LabelV(screen, name, cx, float64(y)+18, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
		})

		// 攻击方式
		atkLabel := bpAttackStyleLabel(bp.AttackStyle)
		ui.LabelV(screen, atkLabel, cx, float64(y)+38, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextMuted,
		})

		// 档位摘要
		tierText := fmt.Sprintf("伤:%s 速:%s 距:%s", bp.Tiers["damage"], bp.Tiers["atkSpeed"], bp.Tiers["range"])
		ui.LabelV(screen, tierText, cx, float64(y)+54, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextMuted,
		})

		// 能力数量
		abText := fmt.Sprintf("能力: %d", len(bp.Abilities))
		ui.LabelV(screen, abText, cx, float64(y)+70, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontXS, Color: theme.TextLocked,
		})
	}

	// 「新建蓝图」按钮
	s.drawCreateButton(screen, cr, startY, len(bps), "＋ 新建蓝图")
}

// ── 自定义能力卡片列表 ──────────────────────────────

func (s *TowerWorkshopScene) drawAbilityCards(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	cas := s.abilityStore.List()

	startY := float32(float64(cr.Y) + 8 - s.scrollY)
	gridW, _ := ui.CardGridSize(len(cas), ui.CardGridStyle{Cols: wsCols, CardW: wsCardW, CardH: wsCardH, Gap: wsCardGap})
	cardX := cr.X + (cr.W-gridW)/2
	if len(cas) < wsCols {
		cardX = cr.X + 12
	}

	for i, ca := range cas {
		col := i % wsCols
		row := i / wsCols
		x := cardX + float32(col)*(wsCardW+wsCardGap)
		y := startY + float32(row)*(wsCardH+wsCardGap)

		r := ui.Rect{X: x, Y: y, W: wsCardW, H: wsCardH}
		s.cardRects = append(s.cardRects, r)

		// 裁剪检查
		if float64(y+wsCardH) < float64(cr.Y) || float64(y) > float64(cr.Y+cr.H) {
			continue
		}

		hovered := i == s.hover
		bgClr := color.RGBA{R: 35, G: 28, B: 55, A: 220}
		borderClr := color.RGBA{R: 120, G: 100, B: 180, A: 200}
		if hovered {
			bgClr = color.RGBA{R: 45, G: 38, B: 70, A: 230}
			borderClr = color.RGBA{R: 147, G: 130, B: 220, A: 220}
		}

		ui.Card(screen, x, y, wsCardW, wsCardH, ui.CardStyle{
			BgColor:     bgClr,
			BorderColor: borderClr,
			Radius:      10,
			BorderWidth: 1.5,
		})

		cx := float64(x) + float64(wsCardW)/2

		// 能力名称
		ui.LabelV(screen, ca.Name, cx, float64(y)+20, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
		})

		// 标签
		tag := "自定义"
		if len(ca.Desc.Tags) > 0 {
			tag = ca.Desc.Tags[0]
		}
		ui.LabelV(screen, tag, cx, float64(y)+40, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontCaption, Color: color.RGBA{R: 147, G: 130, B: 220, A: 255},
		})

		// 费用
		costText := fmt.Sprintf("费用: %d", ca.Desc.Cost)
		ui.LabelV(screen, costText, cx, float64(y)+58, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextMuted,
		})

		// 管线数量
		pipeText := fmt.Sprintf("管线: %d", len(ca.Desc.Pipelines))
		ui.LabelV(screen, pipeText, cx, float64(y)+74, float64(wsCardW)-16, ui.LabelStyle{
			Font: theme.FontXS, Color: theme.TextLocked,
		})
	}

	// 「新建能力」按钮
	s.drawCreateButton(screen, cr, startY, len(cas), "＋ 新建能力")
}

// drawCreateButton 在卡片列表末尾绘制新建按钮。
func (s *TowerWorkshopScene) drawCreateButton(screen *ebiten.Image, cr ui.Rect, startY float32, count int, label string) {
	rows := (count + wsCols - 1) / wsCols
	if count == 0 {
		rows = 0
	}
	btnY := startY + float32(rows)*(wsCardH+wsCardGap) + 8
	btnX := cr.X + (cr.W-wsCreateBtnW)/2

	s.createBtnRect = ui.Rect{X: btnX, Y: btnY, W: wsCreateBtnW, H: wsCreateBtnH}

	// 仅在可视区域内绘制
	if float64(btnY) >= float64(cr.Y) && float64(btnY+wsCreateBtnH) <= float64(cr.Y+cr.H) {
		ui.Button(screen, btnX, btnY, wsCreateBtnW, wsCreateBtnH, label, ui.ButtonStyle{
			BgColor:  theme.BtnPrimary,
			FontSize: theme.FontBody,
			Radius:   8,
			Bold:     true,
		})
	}

	// 空列表提示
	if count == 0 {
		tipY := float64(cr.Y) + float64(cr.H)/2 - 30
		ui.LabelV(screen, "暂无内容，点击下方按钮创建", float64(cr.X)+float64(cr.W)/2, tipY, float64(cr.W)-40, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextLocked,
		})
	}
}
