// blueprint_edit.go — 蓝图编辑场景（4步向导）。
//
// 用于创建和编辑自定义炮塔蓝图。
// 4 个步骤：1.攻击模式 2.属性档位+专精 3.能力选择 4.预览+保存。
// 由建塔面板「+新建」或蓝图管理菜单进入。
//
// 关联：
//   - descriptor.TowerBlueprint: 蓝图数据结构
//   - game.go currentSceneName: 需同步注册场景名
//   - 各步骤的具体 UI 将在后续任务中实现（Task 5-8）
package scene

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/core/tower/descriptor"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── 步骤常量 ──────────────────────────────────────

const (
	bpStepAttackStyle = 0 // 步骤 1：选择攻击模式
	bpStepTiers       = 1 // 步骤 2：属性档位 + 专精
	bpStepAbilities   = 2 // 步骤 3：能力选择
	bpStepPreview     = 3 // 步骤 4：预览 + 保存
	bpStepCount       = 4 // 总步骤数
)

// bpStepLabels 每个步骤的标签文本。
var bpStepLabels = [bpStepCount]string{
	"攻击模式",
	"属性档位",
	"能力选择",
	"预览保存",
}

// ── 布局常量 ──────────────────────────────────────

const (
	bpPanelW      = float32(700) // 主面板宽度
	bpPanelH      = float32(420) // 主面板高度
	bpPanelRadius = float32(14)  // 面板圆角

	bpTitleH      = float32(50) // 标题区高度
	bpStepIndH    = float32(40) // 步骤指示器高度
	bpContentTopY = float32(100) // 内容区起始 Y（相对面板顶部）
	bpBtnAreaH    = float32(50) // 底部按钮区高度
	bpBtnH        = float32(34) // 按钮高度
	bpBtnGap      = float32(12) // 按钮间距

	bpDotRadius  = float32(5) // 步骤圆点半径
	bpDotGap     = float32(60) // 步骤圆点间距
)

// ── BlueprintEditScene ──────────────────────────────

// BlueprintEditScene 蓝图编辑场景，4 步向导。
type BlueprintEditScene struct {
	switcher    Switcher
	blueprint   descriptor.TowerBlueprint
	isNew       bool  // true=新建, false=编辑已有
	step        int   // 0-3（attackStyle / tiers / abilities / preview）
	returnScene Scene // 返回时切换到的场景

	bgGrad *draw.CachedGradient // 背景渐变缓存

	// 底部按钮的布局结果（用于点击检测）
	btnRects []ui.Rect
}

// NewBlueprintEditScene 创建蓝图编辑场景。
// bp 为要编辑的蓝图（nil=新建空蓝图），returnTo 是按取消/完成时返回的场景。
func NewBlueprintEditScene(sw Switcher, bp *descriptor.TowerBlueprint, returnTo Scene) *BlueprintEditScene {
	s := &BlueprintEditScene{
		switcher:    sw,
		isNew:       bp == nil,
		step:        bpStepAttackStyle,
		returnScene: returnTo,
		bgGrad:      draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}
	if bp != nil {
		s.blueprint = *bp // 值拷贝，编辑不影响原始数据
	}
	return s
}

// ── 坐标辅助 ──────────────────────────────────────

// bpPanelOrigin 返回居中面板左上角坐标。
func bpPanelOrigin() (float32, float32) {
	px := (float32(theme.CanvasW) - bpPanelW) / 2
	py := (float32(theme.CanvasH) - bpPanelH) / 2
	return px, py
}

// ── Update ────────────────────────────────────────

func (s *BlueprintEditScene) Update() error {
	// ESC 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(s.returnScene)
		return nil
	}

	// 点击检测
	if isTapJustPressed() {
		mx, my := draw.CursorPos()

		// 检查底部按钮
		for i, r := range s.btnRects {
			if r.Contains(mx, my) {
				s.handleBtnClick(i)
				break
			}
		}
	}

	return nil
}

// handleBtnClick 处理底部按钮点击。
//
// 按钮布局随步骤变化：
//   - step=0: [取消(0)] [下一步(1)]            — 无上一步
//   - step=1~2: [取消(0)] [上一步(1)] [下一步(2)]
//   - step=3: [取消(0)] [上一步(1)] [保存(2)]
//
// 因此最后一个按钮始终是"前进"操作（下一步/保存），
// idx == btnCount-1 即为前进，step>0 时 idx==1 为后退。
func (s *BlueprintEditScene) handleBtnClick(idx int) {
	playUIClick(s.switcher)

	btnCount := len(s.btnRects)
	switch {
	case idx == 0:
		// 取消
		s.switcher.SwitchScene(s.returnScene)
	case idx == btnCount-1:
		// 最后一个按钮：下一步 / 保存
		if s.step < bpStepPreview {
			s.step++
		} else {
			// 保存（后续任务实现，暂时返回上级场景）
			s.switcher.SwitchScene(s.returnScene)
		}
	case idx == 1 && s.step > 0:
		// 上一步
		s.step--
	}
}

// ── Draw ──────────────────────────────────────────

func (s *BlueprintEditScene) Draw(screen *ebiten.Image) {
	// 背景渐变
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	px, py := bpPanelOrigin()

	// 面板背景
	ui.Panel(screen, px, py, bpPanelW, bpPanelH, ui.PanelStyle{
		BgColor:     theme.PanelBg,
		BorderColor: theme.PanelBorder,
		Radius:      bpPanelRadius,
	})

	// ── 标题 ──
	cx := float64(px) + float64(bpPanelW)/2
	titleY := float64(py) + 20
	title := "新建蓝图"
	if !s.isNew {
		title = "编辑蓝图"
	}
	fm.DrawCenteredBoldText(screen, title, cx, titleY, theme.FontOverlayTitle, theme.TextTitle)

	// ── 步骤指示器 ──
	s.drawStepIndicator(screen, fm, px, py)

	// ── 内容区占位 ──
	s.drawStepContent(screen, fm, px, py)

	// ── 底部按钮 ──
	s.drawBottomButtons(screen, px, py)
}

// drawStepIndicator 绘制步骤指示器（圆点 + 标签）。
func (s *BlueprintEditScene) drawStepIndicator(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	// 指示器整体居中
	cx := float64(px) + float64(bpPanelW)/2
	indicatorY := float64(py) + float64(bpTitleH) + 8

	// 圆点总宽度
	totalW := float64(bpStepCount-1) * float64(bpDotGap)
	startX := cx - totalW/2

	for i := 0; i < bpStepCount; i++ {
		dotX := startX + float64(i)*float64(bpDotGap)
		dotY := indicatorY

		// 圆点颜色：当前步骤=主色，已完成=主色半透明，未完成=灰色
		var dotClr color.Color
		if i == s.step {
			dotClr = theme.TonePrimary
		} else if i < s.step {
			dotClr = color.RGBA{R: 34, G: 197, B: 94, A: 150}
		} else {
			dotClr = color.RGBA{R: 100, G: 116, B: 139, A: 180}
		}
		draw.FilledCircle(screen, float32(dotX), float32(dotY), bpDotRadius, dotClr)

		// 连接线（圆点之间）
		if i < bpStepCount-1 {
			lineStartX := float32(dotX) + bpDotRadius + 4
			lineEndX := float32(dotX) + float32(bpDotGap) - bpDotRadius - 4
			lineClr := color.RGBA{R: 100, G: 116, B: 139, A: 80}
			if i < s.step {
				lineClr = color.RGBA{R: 34, G: 197, B: 94, A: 120}
			}
			draw.Line(screen, lineStartX, float32(dotY), lineEndX, float32(dotY), 1.5, lineClr, false)
		}

		// 步骤标签（圆点下方）
		labelY := dotY + float64(bpDotRadius) + 8
		labelClr := theme.TextMuted
		if i == s.step {
			labelClr = theme.TextTitle
		}
		fm.DrawCenteredText(screen, bpStepLabels[i], dotX, labelY, theme.FontCaption, labelClr)
	}
}

// drawStepContent 绘制当前步骤的内容区（占位）。
func (s *BlueprintEditScene) drawStepContent(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	// 内容区域
	contentX := px + 20
	contentY := py + bpContentTopY
	contentW := bpPanelW - 40
	contentH := bpPanelH - bpContentTopY - bpBtnAreaH - 10

	// 内容区背景（略深于面板）
	draw.RoundRect(screen, contentX, contentY, contentW, contentH, 10,
		color.RGBA{R: 10, G: 15, B: 30, A: 150})

	// 占位文本
	cx := float64(contentX) + float64(contentW)/2
	cy := float64(contentY) + float64(contentH)/2
	placeholder := bpStepLabels[s.step] + " (待实现)"
	fm.DrawCenteredVText(screen, placeholder, cx, cy, theme.FontH2, theme.TextMuted)
}

// drawBottomButtons 绘制底部按钮行并缓存布局结果。
func (s *BlueprintEditScene) drawBottomButtons(screen *ebiten.Image, px, py float32) {
	btnY := py + bpPanelH - bpBtnAreaH - 4

	// 构建按钮列表
	var items []ui.ButtonRowItem

	// 取消按钮（始终显示）
	items = append(items, ui.ButtonRowItem{
		Label: "取消",
		Color: theme.BtnSecondary,
	})

	// 上一步（step > 0 时显示）
	if s.step > 0 {
		items = append(items, ui.ButtonRowItem{
			Label: "上一步",
			Color: theme.BtnMuted,
		})
	}

	// 下一步 / 保存
	if s.step < bpStepPreview {
		items = append(items, ui.ButtonRowItem{
			Label: "下一步",
			Color: theme.BtnPrimary,
			Bold:  true,
		})
	} else {
		items = append(items, ui.ButtonRowItem{
			Label: "保存",
			Color: theme.TonePrimary,
			Bold:  true,
		})
	}

	// 按钮区域（面板底部居中）
	btnAreaW := bpPanelW - 40
	area := ui.Rect{
		X: px + 20,
		Y: btnY,
		W: btnAreaW,
		H: bpBtnH,
	}
	result := ui.DrawButtonRowAutoWidth(screen, area, items, ui.ButtonRowStyle{
		Height:   bpBtnH,
		Gap:      bpBtnGap,
		Radius:   theme.ButtonRadius,
		FontSize: theme.FontH2,
	})

	// 缓存按钮矩形用于点击检测
	s.btnRects = result.Rects
}
