// tower_workshop.go — 炮塔工坊场景（游戏外蓝图管理中心）。
//
// 职责：
//   - 展示玩家已创建的蓝图列表和自定义能力列表
//   - 提供新建/编辑/删除蓝图和能力的入口
//   - 蓝图编辑和能力编辑完成后自动返回此场景
//
// 关联：
//   - blueprint_edit.go — 蓝图编辑场景（4步向导），从本场景启动
//   - ability_edit.go — 能力编辑场景，从本场景启动
//   - descriptor.BlueprintStore — 蓝图 CRUD
//   - descriptor.AbilityStore — 自定义能力 CRUD
//   - game.go currentSceneName — 需同步注册场景名 "tower_workshop"
package scene

import (
	"fmt"
	"image/color"
	"log"

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

// ── 布局常量 ──────────────────────────────────────

const (
	twPanelW      = float32(900) // 主面板宽度
	twPanelH      = float32(460) // 主面板高度
	twPanelRadius = float32(14)  // 面板圆角

	twHeaderH = float32(50) // 顶栏高度（返回按钮 + 标题）

	twSectionGap = float32(14) // 蓝图区和能力区之间的间距

	// 蓝图/能力卡片
	twCardW      = float32(160) // 卡片宽度
	twCardH      = float32(100) // 卡片高度
	twCardGap    = float32(10)  // 卡片间距
	twCardCols   = 5            // 每行列数
	twCardRadius = float32(10)  // 卡片圆角

	// 卡片内编辑/删除按钮
	twCardBtnW = float32(50) // 卡片底部小按钮宽度
	twCardBtnH = float32(20) // 卡片底部小按钮高度
	twCardBtnR = float32(6)  // 小按钮圆角

	// +新建 按钮
	twNewBtnW = float32(100)
	twNewBtnH = float32(28)
)

// ── TowerWorkshopScene ──────────────────────────

// TowerWorkshopScene 炮塔工坊场景，管理蓝图和自定义能力。
type TowerWorkshopScene struct {
	switcher       Switcher
	blueprintStore *descriptor.BlueprintStore
	abilityStore   *descriptor.AbilityStore

	// 缓存的数据快照（进入场景时加载，操作后刷新）
	blueprints []descriptor.TowerBlueprint
	abilities  []descriptor.CustomAbility

	// UI 状态
	bgGrad *draw.CachedGradient

	// 蓝图卡片点击区域
	bpCardRects []ui.Rect // 每张蓝图卡片
	bpEditRects []ui.Rect // 每张卡片的编辑按钮
	bpDelRects  []ui.Rect // 每张卡片的删除按钮
	newBpRect   ui.Rect   // +新建蓝图 按钮

	// 能力卡片点击区域
	abCardRects []ui.Rect // 每张能力卡片
	abEditRects []ui.Rect // 每张卡片的编辑按钮
	abDelRects  []ui.Rect // 每张卡片的删除按钮
	newAbRect   ui.Rect   // +新建能力 按钮

	backRect ui.Rect // 返回按钮
}

// NewTowerWorkshopScene 创建炮塔工坊场景。
func NewTowerWorkshopScene(sw Switcher) *TowerWorkshopScene {
	store, _ := persistence.DefaultStorage()
	s := &TowerWorkshopScene{
		switcher:       sw,
		blueprintStore: descriptor.NewBlueprintStore(store),
		abilityStore:   descriptor.NewAbilityStore(store),
		bgGrad:         draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
	s.refreshData()
	return s
}

// refreshData 从 store 重新加载蓝图和能力列表快照。
func (s *TowerWorkshopScene) refreshData() {
	s.blueprints = s.blueprintStore.List()
	s.abilities = s.abilityStore.List()
}

// ── 坐标辅助 ──────────────────────────────────────

// twPanelOrigin 返回居中面板左上角坐标。
func twPanelOrigin() (float32, float32) {
	px := (float32(theme.CanvasW) - twPanelW) / 2
	py := (float32(theme.CanvasH) - twPanelH) / 2
	return px, py
}

// ── Update ────────────────────────────────────────

func (s *TowerWorkshopScene) Update() error {
	// 每次 Update 刷新数据（从编辑器返回后数据可能已变）
	s.refreshData()

	// ESC 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}

	if !isTapJustPressed() {
		return nil
	}

	mx, my := draw.CursorPos()

	// 返回按钮
	if s.backRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewSelectScene(s.switcher))
		return nil
	}

	// +新建蓝图
	if s.newBpRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewBlueprintEditScene(s.switcher, nil, s, s.blueprintStore, s.abilityStore))
		return nil
	}

	// +新建能力
	if s.newAbRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(NewAbilityEditScene(s.switcher, nil, s.abilityStore, s))
		return nil
	}

	// 蓝图编辑按钮
	for i, r := range s.bpEditRects {
		if r.Contains(mx, my) && i < len(s.blueprints) {
			playUIClick(s.switcher)
			bp := s.blueprints[i]
			s.switcher.SwitchScene(NewBlueprintEditScene(s.switcher, &bp, s, s.blueprintStore, s.abilityStore))
			return nil
		}
	}

	// 蓝图删除按钮
	for i, r := range s.bpDelRects {
		if r.Contains(mx, my) && i < len(s.blueprints) {
			playUIClick(s.switcher)
			if err := s.blueprintStore.Delete(s.blueprints[i].ID); err != nil {
				log.Printf("[TowerWorkshop] delete blueprint error: %v", err)
			} else {
				hud.ShowToast("蓝图已删除")
			}
			s.refreshData()
			return nil
		}
	}

	// 能力编辑按钮
	for i, r := range s.abEditRects {
		if r.Contains(mx, my) && i < len(s.abilities) {
			playUIClick(s.switcher)
			ca := s.abilities[i]
			s.switcher.SwitchScene(NewAbilityEditScene(s.switcher, &ca, s.abilityStore, s))
			return nil
		}
	}

	// 能力删除按钮
	for i, r := range s.abDelRects {
		if r.Contains(mx, my) && i < len(s.abilities) {
			playUIClick(s.switcher)
			if err := s.abilityStore.Delete(s.abilities[i].ID); err != nil {
				log.Printf("[TowerWorkshop] delete ability error: %v", err)
			} else {
				hud.ShowToast("能力已删除")
			}
			s.refreshData()
			return nil
		}
	}

	return nil
}

// ── Draw ──────────────────────────────────────────

func (s *TowerWorkshopScene) Draw(screen *ebiten.Image) {
	// 背景渐变
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	px, py := twPanelOrigin()

	// 面板背景
	ui.Panel(screen, px, py, twPanelW, twPanelH, ui.PanelStyle{
		BgColor:     theme.PanelBg,
		BorderColor: theme.PanelBorder,
		Radius:      twPanelRadius,
	})

	// ── 顶栏：返回按钮 + 标题 ──
	s.drawHeader(screen, fm, px, py)

	// ── 蓝图区 ──
	bpSectionY := py + twHeaderH + 8
	bpSectionH := s.drawBlueprintSection(screen, fm, px, bpSectionY)

	// ── 分隔线 ──
	divY := bpSectionY + bpSectionH + twSectionGap/2
	ui.Divider(screen, px+30, divY, twPanelW-60, nil)

	// ── 能力区 ──
	abSectionY := divY + twSectionGap/2
	s.drawAbilitySection(screen, fm, px, abSectionY)
}

// drawHeader 绘制顶栏（返回按钮 + 居中标题）。
func (s *TowerWorkshopScene) drawHeader(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	// 分隔线
	ui.Divider(screen, px+20, py+twHeaderH-1, twPanelW-40, nil)

	// 返回按钮（左上角）
	backX := px + 16
	backY := py + 10
	backW := float32(70)
	backH := float32(34)
	s.backRect = ui.Rect{X: backX, Y: backY, W: backW, H: backH}

	ui.Button(screen, backX, backY, backW, backH, "← 返回", ui.ButtonStyle{
		BgColor:  theme.BtnSecondary,
		FontSize: theme.FontBody,
		Radius:   10,
	})

	// 居中标题
	cx := float64(px) + float64(twPanelW)/2
	fm.DrawCenteredBoldText(screen, "炮塔工坊", cx, float64(py)+18, theme.FontOverlayTitle, theme.TextTitle)
}

// ── 蓝图区 ──────────────────────────────────────

// drawBlueprintSection 绘制蓝图区域，返回区域总高度。
func (s *TowerWorkshopScene) drawBlueprintSection(screen *ebiten.Image, fm *render.FontManager, px, sectionY float32) float32 {
	contentX := px + 24
	contentW := twPanelW - 48

	// 标题行：「我的蓝图 (n/20)」 + [+新建蓝图]
	countText := fmt.Sprintf("我的蓝图 (%d/%d)", len(s.blueprints), 20)
	ui.Label(screen, countText, float64(contentX), float64(sectionY)+2, float64(contentW), ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// +新建蓝图 按钮（右侧）
	newBpX := contentX + contentW - twNewBtnW
	newBpY := sectionY
	s.newBpRect = ui.Rect{X: newBpX, Y: newBpY, W: twNewBtnW, H: twNewBtnH}
	newBpBg := color.Color(theme.TonePrimary)
	if len(s.blueprints) >= 20 {
		newBpBg = theme.ToneDisabled
	}
	ui.Button(screen, newBpX, newBpY, twNewBtnW, twNewBtnH, "+新建蓝图", ui.ButtonStyle{
		BgColor:  newBpBg,
		FontSize: theme.FontBody,
		Radius:   8,
		Bold:     true,
	})

	// 卡片网格
	cardStartY := sectionY + twNewBtnH + 8
	s.bpCardRects = nil
	s.bpEditRects = nil
	s.bpDelRects = nil

	if len(s.blueprints) == 0 {
		// 空态提示
		ui.LabelV(screen, "暂无蓝图，点击右上角新建", float64(px)+float64(twPanelW)/2, float64(cardStartY)+40, float64(contentW), ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextLocked,
		})
		return twNewBtnH + 8 + 80
	}

	result := ui.CardGrid(screen, contentX, cardStartY, len(s.blueprints), ui.CardGridStyle{
		Cols: twCardCols, CardW: twCardW, CardH: twCardH, Gap: twCardGap,
	}, func(screen *ebiten.Image, idx int, r ui.Rect) {
		bp := s.blueprints[idx]
		s.drawBlueprintCard(screen, fm, bp, r)
	})

	s.bpCardRects = result.Rects

	// 计算区域高度
	rows := (len(s.blueprints) + twCardCols - 1) / twCardCols
	gridH := float32(rows)*(twCardH+twCardGap) - twCardGap
	return twNewBtnH + 8 + gridH
}

// drawBlueprintCard 绘制单张蓝图卡片。
//
// 内容布局：
//   - 名称（加粗，第一行）
//   - 攻击方式标签（第二行）
//   - 费用 + 档位摘要（第三行）
//   - [编辑] [删除] 按钮（底部行）
func (s *TowerWorkshopScene) drawBlueprintCard(screen *ebiten.Image, fm *render.FontManager, bp descriptor.TowerBlueprint, r ui.Rect) {
	// 卡片背景
	ui.Card(screen, r.X, r.Y, r.W, r.H, ui.CardStyle{
		BgColor:     color.RGBA{R: 25, G: 32, B: 55, A: 220},
		BorderColor: color.RGBA{R: 60, G: 70, B: 95, A: 200},
		Radius:      twCardRadius,
		BorderWidth: 1.5,
	})

	cx := r.CenterX()
	padX := float64(r.X) + 8
	cardW := float64(r.W) - 16

	// 名称（加粗居中）
	name := bp.Name
	if name == "" {
		name = bp.ID
	}
	ui.LabelV(screen, name, cx, float64(r.Y)+14, cardW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
	})

	// 攻击方式
	atkLabel := bpAttackStyleLabel(bp.AttackStyle)
	ui.LabelV(screen, atkLabel, cx, float64(r.Y)+32, cardW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})

	// 费用 + 档位摘要
	tierSummary := twTierSummary(bp.Tiers)
	costLine := fmt.Sprintf("费用:%d  %s", bp.BuildCost, tierSummary)
	ui.LabelV(screen, costLine, cx, float64(r.Y)+48, cardW, ui.LabelStyle{
		Font: theme.FontXS, Color: theme.TextMuted,
	})

	// 能力数量
	abCount := fmt.Sprintf("能力×%d", len(bp.Abilities))
	ui.Label(screen, abCount, padX, float64(r.Y)+60, cardW, ui.LabelStyle{
		Font: theme.FontXS, Color: theme.TextMuted, Align: ui.AlignRight,
	})

	// [编辑] [删除] 按钮
	btnY := r.Y + r.H - twCardBtnH - 4
	editX := r.X + 6
	delX := r.X + r.W - twCardBtnW - 6

	editR := ui.Rect{X: editX, Y: btnY, W: twCardBtnW, H: twCardBtnH}
	delR := ui.Rect{X: delX, Y: btnY, W: twCardBtnW, H: twCardBtnH}
	s.bpEditRects = append(s.bpEditRects, editR)
	s.bpDelRects = append(s.bpDelRects, delR)

	ui.Button(screen, editX, btnY, twCardBtnW, twCardBtnH, "编辑", ui.ButtonStyle{
		BgColor:  theme.BtnSecondary,
		FontSize: theme.FontXS,
		Radius:   twCardBtnR,
	})
	ui.Button(screen, delX, btnY, twCardBtnW, twCardBtnH, "删除", ui.ButtonStyle{
		BgColor:  theme.BtnDanger,
		FontSize: theme.FontXS,
		Radius:   twCardBtnR,
	})
}

// twTierSummary 生成档位摘要字符串（如 "S/B/D"）。
func twTierSummary(tiers map[string]string) string {
	d := tiers["damage"]
	a := tiers["atkSpeed"]
	r := tiers["range"]
	if d == "" {
		d = "B"
	}
	if a == "" {
		a = "B"
	}
	if r == "" {
		r = "B"
	}
	return d + "/" + a + "/" + r
}

// ── 能力区 ──────────────────────────────────────

// drawAbilitySection 绘制自定义能力区域。
func (s *TowerWorkshopScene) drawAbilitySection(screen *ebiten.Image, fm *render.FontManager, px, sectionY float32) {
	contentX := px + 24
	contentW := twPanelW - 48

	// 标题行：「自定义能力 (n/50)」 + [+新建能力]
	countText := fmt.Sprintf("自定义能力 (%d/%d)", len(s.abilities), 50)
	ui.Label(screen, countText, float64(contentX), float64(sectionY)+2, float64(contentW), ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// +新建能力 按钮（右侧）
	newAbX := contentX + contentW - twNewBtnW
	newAbY := sectionY
	s.newAbRect = ui.Rect{X: newAbX, Y: newAbY, W: twNewBtnW, H: twNewBtnH}
	newAbBg := color.Color(theme.ToneAccent)
	if len(s.abilities) >= 50 {
		newAbBg = theme.ToneDisabled
	}
	ui.Button(screen, newAbX, newAbY, twNewBtnW, twNewBtnH, "+新建能力", ui.ButtonStyle{
		BgColor:  newAbBg,
		FontSize: theme.FontBody,
		Radius:   8,
		Bold:     true,
	})

	// 卡片网格
	cardStartY := sectionY + twNewBtnH + 8
	s.abCardRects = nil
	s.abEditRects = nil
	s.abDelRects = nil

	if len(s.abilities) == 0 {
		// 空态提示
		ui.LabelV(screen, "暂无自定义能力，点击右上角新建", float64(px)+float64(twPanelW)/2, float64(cardStartY)+30, float64(contentW), ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextLocked,
		})
		return
	}

	// 能力卡片比蓝图矮一些
	const abCardH = float32(80)

	result := ui.CardGrid(screen, contentX, cardStartY, len(s.abilities), ui.CardGridStyle{
		Cols: twCardCols, CardW: twCardW, CardH: abCardH, Gap: twCardGap,
	}, func(screen *ebiten.Image, idx int, r ui.Rect) {
		ca := s.abilities[idx]
		s.drawAbilityCard(screen, fm, ca, r)
	})

	s.abCardRects = result.Rects
}

// drawAbilityCard 绘制单张能力卡片。
//
// 内容布局：
//   - 名称（加粗，第一行）
//   - 管线数 + 费用（第二行）
//   - [编辑] [删除] 按钮（底部行）
func (s *TowerWorkshopScene) drawAbilityCard(screen *ebiten.Image, fm *render.FontManager, ca descriptor.CustomAbility, r ui.Rect) {
	// 卡片背景（紫色调区分自定义能力）
	ui.Card(screen, r.X, r.Y, r.W, r.H, ui.CardStyle{
		BgColor:     color.RGBA{R: 35, G: 28, B: 55, A: 220},
		BorderColor: color.RGBA{R: 120, G: 100, B: 180, A: 200},
		Radius:      twCardRadius,
		BorderWidth: 1.5,
	})

	cx := r.CenterX()
	cardW := float64(r.W) - 16

	// 名称（加粗居中）
	ui.LabelV(screen, ca.Name, cx, float64(r.Y)+14, cardW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
	})

	// 管线数 + 费用
	pipeCount := len(ca.Desc.Pipelines)
	costText := fmt.Sprintf("管线×%d  费用:%d", pipeCount, ca.Desc.Cost)
	ui.LabelV(screen, costText, cx, float64(r.Y)+32, cardW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})

	// [编辑] [删除] 按钮
	btnY := r.Y + r.H - twCardBtnH - 4
	editX := r.X + 6
	delX := r.X + r.W - twCardBtnW - 6

	editR := ui.Rect{X: editX, Y: btnY, W: twCardBtnW, H: twCardBtnH}
	delR := ui.Rect{X: delX, Y: btnY, W: twCardBtnW, H: twCardBtnH}
	s.abEditRects = append(s.abEditRects, editR)
	s.abDelRects = append(s.abDelRects, delR)

	ui.Button(screen, editX, btnY, twCardBtnW, twCardBtnH, "编辑", ui.ButtonStyle{
		BgColor:  theme.BtnSecondary,
		FontSize: theme.FontXS,
		Radius:   twCardBtnR,
	})
	ui.Button(screen, delX, btnY, twCardBtnW, twCardBtnH, "删除", ui.ButtonStyle{
		BgColor:  theme.BtnDanger,
		FontSize: theme.FontXS,
		Radius:   twCardBtnR,
	})
}

// ── Layout ────────────────────────────────────────

// Layout 返回逻辑分辨率，非 Stage 场景固定 1200×540。
func (s *TowerWorkshopScene) Layout(_, _ int) (int, int) {
	return game.ScreenWidth, game.ScreenHeight
}
