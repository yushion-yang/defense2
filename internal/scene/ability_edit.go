// ability_edit.go — 自定义能力编辑场景（管线列表管理）。
//
// 用于创建和编辑自定义能力。核心交互：管线的增删，每条管线包含
// 触发器、条件列表、目标选择器、效果列表 四个槽位。
// 本阶段仅实现骨架：管线增删 + 占位文本，触发/条件/选择/效果的
// 实际选择在后续 Task 中实现。
//
// 关联：
//   - descriptor.CustomAbility: 自定义能力数据结构
//   - descriptor.AbilityStore: 能力持久化存储
//   - game.go currentSceneName: 需同步注册场景名
package scene

import (
	"fmt"
	"image/color"
	"log"

	"defense2/internal/core/game"
	"defense2/internal/core/tower/descriptor"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ── 布局常量 ──────────────────────────────────────

const (
	aePanelW      = float32(760) // 主面板宽度
	aePanelH      = float32(460) // 主面板高度
	aePanelRadius = float32(14)  // 面板圆角

	aeTitleH   = float32(44) // 标题区高度
	aeBtnAreaH = float32(50) // 底部按钮区高度
	aeBtnH     = float32(34) // 按钮高度
	aeBtnGap   = float32(12) // 按钮间距

	// 管线卡片布局
	aePipeCardW      = float32(680) // 管线卡片宽度
	aePipeCardH      = float32(120) // 管线卡片高度（4 行 × ~28px + padding）
	aePipeCardGap    = float32(10)  // 管线卡片间距
	aePipeCardRadius = float32(10)  // 管线卡片圆角
	aePipeRowH       = float32(24)  // 管线内每行高度

	aeAddPipeBtnW = float32(180) // +添加管线按钮宽度
	aeAddPipeBtnH = float32(32)  // +添加管线按钮高度

	aeDelBtnSize = float32(24) // 删除按钮尺寸（方形）
)

// ── 编辑状态类型 ──────────────────────────────────

// pipelineEditState 单条管线的编辑状态。
// 存储用户在 UI 中选择的各组件类型 ID 和参数值。
type pipelineEditState struct {
	TriggerID      string             // 触发器类型 ID（如 "onHit"），空=未选择
	Conditions     []componentEditState // 条件列表
	SelectorID     string             // 目标选择器类型 ID（如 "currentTarget"），空=未选择
	SelectorParams map[string]float64 // 选择器参数
	Effects        []componentEditState // 效果列表
}

// componentEditState 单个条件或效果组件的编辑状态。
type componentEditState struct {
	TypeID string             // 组件类型 ID
	Params map[string]float64 // 可调参数
}

// ── AbilityEditScene ──────────────────────────────

// AbilityEditScene 自定义能力编辑场景。
type AbilityEditScene struct {
	switcher     Switcher
	abilityStore *descriptor.AbilityStore // 能力持久化存储（可为 nil）
	ability      descriptor.CustomAbility // 正在编辑的能力（值拷贝）
	isNew        bool                     // true=新建, false=编辑已有
	returnScene  Scene                    // 返回时切换到的场景

	// ── 管线编辑状态 ──
	pipelines []pipelineEditState

	// ── UI 状态 ──
	scrollY      float64   // 管线列表滚动偏移
	btnRects     []ui.Rect // 底部按钮的布局矩形
	pipeRects    []ui.Rect // 每条管线卡片的渲染区域
	addPipeRect  ui.Rect   // +添加管线按钮区域
	delPipeRects []ui.Rect // 每条管线的删除按钮区域

	// ── 管线内各槽位的点击区域（为后续 Task 预留） ──
	triggerRects  []ui.Rect // 每条管线的触发器区域
	selectorRects []ui.Rect // 每条管线的选择器区域
	condBtnRects  []ui.Rect // 每条管线的 +条件 按钮区域
	effectBtnRects []ui.Rect // 每条管线的 +效果 按钮区域

	bgGrad *draw.CachedGradient // 背景渐变缓存
}

// NewAbilityEditScene 创建能力编辑场景。
// ca 为要编辑的能力（nil=新建空能力），store 为持久化存储，returnTo 是返回场景。
func NewAbilityEditScene(sw Switcher, ca *descriptor.CustomAbility, store *descriptor.AbilityStore, returnTo Scene) *AbilityEditScene {
	s := &AbilityEditScene{
		switcher:     sw,
		abilityStore: store,
		isNew:        ca == nil,
		returnScene:  returnTo,
		bgGrad:       draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}

	if ca != nil {
		s.ability = *ca // 值拷贝
		// 从已有能力的 descriptor 还原管线编辑状态
		s.pipelines = pipelinesFromDescriptor(ca.Desc)
	} else {
		// 新建：生成默认名称，初始化一条空管线
		count := 0
		if store != nil {
			count = store.Count()
		}
		s.ability = descriptor.CustomAbility{
			Name: fmt.Sprintf("自定义能力-%03d", count+1),
		}
		s.pipelines = []pipelineEditState{newEmptyPipeline()}
	}

	return s
}

// newEmptyPipeline 创建一条空管线（所有字段为零值/空）。
func newEmptyPipeline() pipelineEditState {
	return pipelineEditState{
		SelectorParams: map[string]float64{},
	}
}

// pipelinesFromDescriptor 从 AbilityDescriptor 还原管线编辑状态。
// 将已编译的 Pipeline 结构反序列化回编辑器可用的字符串 ID 形式。
func pipelinesFromDescriptor(desc descriptor.AbilityDescriptor) []pipelineEditState {
	if len(desc.Pipelines) == 0 {
		return []pipelineEditState{newEmptyPipeline()}
	}
	result := make([]pipelineEditState, len(desc.Pipelines))
	for i, p := range desc.Pipelines {
		pe := pipelineEditState{
			TriggerID:      p.Trigger.String(),
			SelectorParams: map[string]float64{},
		}
		// 条件
		for _, c := range p.Conditions {
			pe.Conditions = append(pe.Conditions, componentEditState{
				TypeID: fmt.Sprintf("%T", c),
				Params: map[string]float64{},
			})
		}
		// 选择器：用类型名作为占位 ID
		if p.Selector != nil {
			pe.SelectorID = fmt.Sprintf("%T", p.Selector)
		}
		// 效果
		for _, e := range p.Effects {
			pe.Effects = append(pe.Effects, componentEditState{
				TypeID: fmt.Sprintf("%T", e),
				Params: map[string]float64{},
			})
		}
		result[i] = pe
	}
	return result
}

// ── 坐标辅助 ──────────────────────────────────────

// aePanelOrigin 返回居中面板左上角坐标。
func aePanelOrigin() (float32, float32) {
	px := (float32(theme.CanvasW) - aePanelW) / 2
	py := (float32(theme.CanvasH) - aePanelH) / 2
	return px, py
}

// aeContentRect 返回面板内容区矩形（标题和底部按钮之间的区域）。
func aeContentRect(px, py float32) ui.Rect {
	return ui.Rect{
		X: px + 20,
		Y: py + aeTitleH,
		W: aePanelW - 40,
		H: aePanelH - aeTitleH - aeBtnAreaH - 10,
	}
}

// ── Update ────────────────────────────────────────

func (s *AbilityEditScene) Update() error {
	// ESC 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		s.switcher.SwitchScene(s.returnScene)
		return nil
	}

	// 鼠标滚轮滚动管线列表
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
//  1. 管线删除按钮（[x]）
//  2. 添加管线按钮
//  3. 管线内各槽位（为后续 Task 预留，当前无操作）
//  4. 底部按钮（取消/保存）
func (s *AbilityEditScene) handleInput(mx, my float64) {
	// 1. 检查删除管线按钮
	for i, r := range s.delPipeRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.deletePipeline(i)
			return
		}
	}

	// 2. 检查添加管线按钮
	if s.addPipeRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.addPipeline()
		return
	}

	// 3. 管线内槽位点击（预留：触发器、选择器、+条件、+效果）
	// 后续 Task 5-7 实现

	// 4. 底部按钮
	for i, r := range s.btnRects {
		if r.Contains(mx, my) {
			s.handleBtnClick(i)
			return
		}
	}
}

// handleBtnClick 处理底部按钮点击。
// 按钮布局：[取消(0)] [保存(1)]
func (s *AbilityEditScene) handleBtnClick(idx int) {
	playUIClick(s.switcher)
	switch idx {
	case 0:
		// 取消
		s.switcher.SwitchScene(s.returnScene)
	case 1:
		// 保存（当前为占位，仅日志输出）
		s.saveAbility()
	}
}

// addPipeline 添加一条新的空管线。
func (s *AbilityEditScene) addPipeline() {
	s.pipelines = append(s.pipelines, newEmptyPipeline())
}

// deletePipeline 删除指定索引的管线。至少保留一条。
func (s *AbilityEditScene) deletePipeline(idx int) {
	if len(s.pipelines) <= 1 {
		return // 至少保留一条管线
	}
	if idx < 0 || idx >= len(s.pipelines) {
		return
	}
	s.pipelines = append(s.pipelines[:idx], s.pipelines[idx+1:]...)
}

// saveAbility 保存能力（Task 9 实现完整逻辑，当前仅日志占位）。
func (s *AbilityEditScene) saveAbility() {
	log.Printf("[AbilityEditScene] save placeholder: name=%q, pipelines=%d", s.ability.Name, len(s.pipelines))
	// TODO(Task 9): 将 pipelineEditState 转换为 descriptor.Pipeline 并保存到 AbilityStore
}

// clampScroll 将滚动偏移限制在合法范围内。
func (s *AbilityEditScene) clampScroll() {
	// 计算内容总高度
	contentH := float64(len(s.pipelines)) * float64(aePipeCardH+aePipeCardGap)
	contentH += float64(aeAddPipeBtnH) + 16 // +添加管线 按钮 + 间距
	_, py := aePanelOrigin()
	cr := aeContentRect(0, py) // 只需要高度
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

// ── Draw ──────────────────────────────────────────

func (s *AbilityEditScene) Draw(screen *ebiten.Image) {
	// 背景渐变
	s.bgGrad.Draw(screen, 0, 0)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	px, py := aePanelOrigin()

	// 面板背景
	ui.Panel(screen, px, py, aePanelW, aePanelH, ui.PanelStyle{
		BgColor:     theme.PanelBg,
		BorderColor: theme.PanelBorder,
		Radius:      aePanelRadius,
	})

	// ── 标题栏 ──
	s.drawTitleBar(screen, fm, px, py)

	// ── 内容区（管线列表） ──
	cr := aeContentRect(px, py)
	s.drawPipelineList(screen, fm, cr)

	// ── 底部按钮 ──
	s.drawBottomButtons(screen, px, py)
}

// drawTitleBar 绘制标题栏：能力名称 + 费用。
func (s *AbilityEditScene) drawTitleBar(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	// 标题区背景分隔线
	divY := py + aeTitleH - 1
	ui.Divider(screen, px+20, divY, aePanelW-40, nil)

	// 左侧：能力名称
	nameLabel := fmt.Sprintf("能力名称: %s", s.ability.Name)
	ui.Label(screen, nameLabel, float64(px)+24, float64(py)+14, float64(aePanelW)/2-30, ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// 右侧：费用（占位）
	costLabel := "费用: --"
	ui.Label(screen, costLabel, float64(px)+24, float64(py)+14, float64(aePanelW)-48, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextMuted, Align: ui.AlignRight,
	})
}

// drawPipelineList 绘制管线列表区域（带滚动裁剪）。
//
// 每条管线渲染为一个卡片，内含 4 行：
//   1. 触发: [触发器名称 或 "未选择"]
//   2. 条件: [条件列表 或 "(未设置)"]  [+条件]
//   3. 目标: [选择器名称 或 "未选择"]
//   4. 效果: [效果列表 或 "(未设置)"]  [+效果]
func (s *AbilityEditScene) drawPipelineList(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	// 内容区背景（略深于面板，区分层次）
	draw.RoundRect(screen, cr.X, cr.Y, cr.W, cr.H, 10,
		color.RGBA{R: 10, G: 15, B: 30, A: 150})

	// 重置布局缓存
	s.pipeRects = s.pipeRects[:0]
	s.delPipeRects = s.delPipeRects[:0]
	s.triggerRects = s.triggerRects[:0]
	s.selectorRects = s.selectorRects[:0]
	s.condBtnRects = s.condBtnRects[:0]
	s.effectBtnRects = s.effectBtnRects[:0]

	// 管线卡片起始坐标
	cardX := cr.X + (cr.W-aePipeCardW)/2
	startY := float32(float64(cr.Y) + 8 - s.scrollY)

	for i, pipe := range s.pipelines {
		cardY := startY + float32(i)*float32(aePipeCardH+aePipeCardGap)

		// 裁剪检查：仅绘制在可视区域内的卡片
		if float64(cardY+aePipeCardH) < float64(cr.Y) || float64(cardY) > float64(cr.Y+cr.H) {
			// 卡片完全在可视区外，仍需记录矩形供点击检测
			s.pipeRects = append(s.pipeRects, ui.Rect{X: cardX, Y: cardY, W: aePipeCardW, H: aePipeCardH})
			s.delPipeRects = append(s.delPipeRects, ui.Rect{})
			s.triggerRects = append(s.triggerRects, ui.Rect{})
			s.selectorRects = append(s.selectorRects, ui.Rect{})
			s.condBtnRects = append(s.condBtnRects, ui.Rect{})
			s.effectBtnRects = append(s.effectBtnRects, ui.Rect{})
			continue
		}

		s.drawPipelineCard(screen, fm, i, pipe, cardX, cardY)
	}

	// +添加管线 按钮
	addY := startY + float32(len(s.pipelines))*float32(aePipeCardH+aePipeCardGap) + 4
	addX := cr.X + (cr.W-aeAddPipeBtnW)/2
	s.addPipeRect = ui.Rect{X: addX, Y: addY, W: aeAddPipeBtnW, H: aeAddPipeBtnH}

	// 仅在可视区域内绘制
	if float64(addY) >= float64(cr.Y) && float64(addY+aeAddPipeBtnH) <= float64(cr.Y+cr.H) {
		ui.Button(screen, addX, addY, aeAddPipeBtnW, aeAddPipeBtnH, "+ 添加管线", ui.ButtonStyle{
			BgColor:  theme.BtnMuted,
			FontSize: theme.FontBody,
			Radius:   8,
		})
	}
}

// drawPipelineCard 绘制单条管线卡片。
func (s *AbilityEditScene) drawPipelineCard(screen *ebiten.Image, fm *render.FontManager, idx int, pipe pipelineEditState, x, y float32) {
	// 卡片矩形
	cardRect := ui.Rect{X: x, Y: y, W: aePipeCardW, H: aePipeCardH}
	s.pipeRects = append(s.pipeRects, cardRect)

	// 卡片背景
	ui.Panel(screen, x, y, aePipeCardW, aePipeCardH, ui.PanelStyle{
		BgColor:     color.RGBA{R: 20, G: 28, B: 50, A: 220},
		BorderColor: color.RGBA{R: 60, G: 70, B: 95, A: 180},
		Radius:      aePipeCardRadius,
	})

	// ── 卡片标题行：管线编号 + 删除按钮 ──
	titleLabel := fmt.Sprintf("管线 %d", idx+1)
	ui.Label(screen, titleLabel, float64(x)+12, float64(y)+6, float64(aePipeCardW)-50, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
	})

	// 删除按钮 [x]
	delX := x + aePipeCardW - aeDelBtnSize - 8
	delY := y + 4
	delRect := ui.Rect{X: delX, Y: delY, W: aeDelBtnSize, H: aeDelBtnSize}
	s.delPipeRects = append(s.delPipeRects, delRect)

	// 只有多于 1 条管线时才显示删除按钮
	if len(s.pipelines) > 1 {
		ui.Button(screen, delX, delY, aeDelBtnSize, aeDelBtnSize, "x", ui.ButtonStyle{
			BgColor:  theme.BtnDanger,
			FontSize: theme.FontCaption,
			Radius:   6,
		})
	}

	// ── 4 行内容 ──
	contentX := x + 12
	contentW := aePipeCardW - 24
	rowY := y + 28 // 标题行下方

	// 行 1: 触发器
	triggerLabel := "未选择"
	triggerClr := color.Color(theme.TextLocked)
	if pipe.TriggerID != "" {
		triggerLabel = pipe.TriggerID
		triggerClr = theme.TextBody
	}
	ui.Label(screen, "触发:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	triggerSlotX := contentX + 42
	triggerSlotW := contentW - 42
	triggerRect := ui.Rect{X: triggerSlotX, Y: rowY, W: triggerSlotW, H: aePipeRowH}
	s.triggerRects = append(s.triggerRects, triggerRect)
	ui.Label(screen, fmt.Sprintf("[%s]", triggerLabel), float64(triggerSlotX), float64(rowY), float64(triggerSlotW), ui.LabelStyle{
		Font: theme.FontCaption, Color: triggerClr,
	})
	rowY += aePipeRowH

	// 行 2: 条件
	condLabel := "(未设置)"
	condClr := color.Color(theme.TextLocked)
	if len(pipe.Conditions) > 0 {
		condLabel = fmt.Sprintf("%d 个条件", len(pipe.Conditions))
		condClr = theme.TextBody
	}
	ui.Label(screen, "条件:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	ui.Label(screen, condLabel, float64(contentX+42), float64(rowY), float64(contentW-120), ui.LabelStyle{
		Font: theme.FontCaption, Color: condClr,
	})
	// +条件 按钮（右侧）
	condBtnX := contentX + contentW - 60
	condBtnRect := ui.Rect{X: condBtnX, Y: rowY, W: 56, H: aePipeRowH - 2}
	s.condBtnRects = append(s.condBtnRects, condBtnRect)
	ui.Button(screen, condBtnX, rowY, 56, aePipeRowH-2, "+条件", ui.ButtonStyle{
		BgColor:  color.RGBA{R: 40, G: 50, B: 75, A: 200},
		FontSize: 10,
		Radius:   6,
	})
	rowY += aePipeRowH

	// 行 3: 目标选择器
	selectorLabel := "未选择"
	selectorClr := color.Color(theme.TextLocked)
	if pipe.SelectorID != "" {
		selectorLabel = pipe.SelectorID
		selectorClr = theme.TextBody
	}
	ui.Label(screen, "目标:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	selectorSlotX := contentX + 42
	selectorSlotW := contentW - 42
	selectorRect := ui.Rect{X: selectorSlotX, Y: rowY, W: selectorSlotW, H: aePipeRowH}
	s.selectorRects = append(s.selectorRects, selectorRect)
	ui.Label(screen, fmt.Sprintf("[%s]", selectorLabel), float64(selectorSlotX), float64(rowY), float64(selectorSlotW), ui.LabelStyle{
		Font: theme.FontCaption, Color: selectorClr,
	})
	rowY += aePipeRowH

	// 行 4: 效果
	effectLabel := "(未设置)"
	effectClr := color.Color(theme.TextLocked)
	if len(pipe.Effects) > 0 {
		effectLabel = fmt.Sprintf("%d 个效果", len(pipe.Effects))
		effectClr = theme.TextBody
	}
	ui.Label(screen, "效果:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	ui.Label(screen, effectLabel, float64(contentX+42), float64(rowY), float64(contentW-120), ui.LabelStyle{
		Font: theme.FontCaption, Color: effectClr,
	})
	// +效果 按钮（右侧）
	effectBtnX := contentX + contentW - 60
	effectBtnRect := ui.Rect{X: effectBtnX, Y: rowY, W: 56, H: aePipeRowH - 2}
	s.effectBtnRects = append(s.effectBtnRects, effectBtnRect)
	ui.Button(screen, effectBtnX, rowY, 56, aePipeRowH-2, "+效果", ui.ButtonStyle{
		BgColor:  color.RGBA{R: 40, G: 50, B: 75, A: 200},
		FontSize: 10,
		Radius:   6,
	})
}

// drawBottomButtons 绘制底部按钮行并缓存布局结果。
func (s *AbilityEditScene) drawBottomButtons(screen *ebiten.Image, px, py float32) {
	btnY := py + aePanelH - aeBtnAreaH - 4

	items := []ui.ButtonRowItem{
		{Label: "取消", Color: theme.BtnSecondary},
		{Label: "保存", Color: theme.BtnPrimary, Bold: true},
	}

	btnAreaW := aePanelW - 40
	area := ui.Rect{
		X: px + 20,
		Y: btnY,
		W: btnAreaW,
		H: aeBtnH,
	}
	result := ui.DrawButtonRowAutoWidth(screen, area, items, ui.ButtonRowStyle{
		Height:   aeBtnH,
		Gap:      aeBtnGap,
		Radius:   theme.ButtonRadius,
		FontSize: theme.FontH2,
	})

	s.btnRects = result.Rects
}
