// ability_edit.go — 自定义能力编辑场景（完整管线编辑器）。
//
// 用于创建和编辑自定义能力。核心交互：管线增删，每条管线包含
// 触发器、条件列表、目标选择器、效果列表四个槽位。
// 支持 PrimitivePicker 选择组件、Badge 显示已选条件/效果、
// +/- 按钮调参、实时费用计算、保存到 AbilityStore。
//
// 导航流：
//   - returnScene 存储调用方场景，ESC/取消/保存完成后切回。
//   - 从 BlueprintEditScene 步骤 3 的「+创建新能力」进入时，
//     returnScene = BlueprintEditScene（返回蓝图编辑器继续选能力）。
//   - 从工坊（TowerWorkshopScene）直接进入时，
//     returnScene = TowerWorkshopScene（返回工坊）。
//
// 关联：
//   - descriptor.PipelineEditState / ComponentEditState: 管线编辑状态
//   - descriptor.AbilityStore: 能力持久化存储
//   - hud.PrimitivePickerData / DrawPrimitivePicker / PrimitivePickerHitTest: 选择器弹窗
//   - descriptor.AllTriggerMeta / AllConditionMeta / AllSelectorMeta / AllEffectMeta: 元数据
package scene

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strconv"
	"time"

	"defense2/internal/core/game"
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

	// 参数编辑区布局
	aeParamRowH = float32(20) // 参数行高度
	aeParamBtnW = float32(20) // +/- 按钮宽度
	aeParamBtnH = float32(18) // +/- 按钮高度

	// Badge 布局
	aeBadgeH    = float32(18) // Badge 高度
	aeBadgeGap  = float32(4)  // Badge 间距
	aeBadgeDelW = float32(14) // Badge 删除按钮宽度
)

// ── Badge 颜色 ──────────────────────────────────────

var (
	aeBadgeBg       = color.RGBA{R: 45, G: 55, B: 80, A: 220}
	aeBadgeTextClr  = color.RGBA{R: 200, G: 210, B: 230, A: 255}
	aeParamBtnBg    = color.RGBA{R: 50, G: 60, B: 90, A: 220}
	aeParamLabelClr = color.RGBA{R: 160, G: 170, B: 200, A: 255}
	aeParamValueClr = color.RGBA{R: 220, G: 230, B: 245, A: 255}
)

// ── Picker 目标信息 ──────────────────────────────────

// pickerTargetInfo 记录 picker 打开时的目标（点击了哪个管线的哪个槽位）。
type pickerTargetInfo struct {
	pipeIdx  int    // 管线索引
	slotType string // "trigger"/"condition"/"selector"/"effect"
	compIdx  int    // 条件/效果列表中的索引，-1=新增
}

// ── 参数按钮点击区域 ──────────────────────────────────

// paramBtnRect 记录参数 +/- 按钮的位置和关联信息。
type paramBtnRect struct {
	Rect     ui.Rect
	PipeIdx  int    // 管线索引
	SlotType string // "condition"/"selector"/"effect"
	CompIdx  int    // 组件索引（selector 固定 0）
	ParamKey string // 参数键名
	Delta    int    // +1 或 -1
}

// condEffectBadgeRect 记录条件/效果 badge 的点击区域。
type condEffectBadgeRect struct {
	Rect     ui.Rect
	PipeIdx  int
	SlotType string // "condition"/"effect"
	CompIdx  int
}

// scalerToggleRect 记录 scaler mode toggle 的点击区域。
type scalerToggleRect struct {
	Rect     ui.Rect
	PipeIdx  int
	SlotType string
	CompIdx  int
	ParamKey string
}

// stringEnumRect 记录字符串枚举参数的点击区域（click-to-cycle）。
type stringEnumRect struct {
	Rect     ui.Rect
	PipeIdx  int
	SlotType string
	CompIdx  int
	ParamKey string // 参数键名（如 "mode"/"stat"/"buffID"）
	TypeID   string // 所属组件类型 ID（如 "damage"/"buff"），用于查表
}

// ── AbilityEditScene ──────────────────────────────

// AbilityEditScene 自定义能力编辑场景。
type AbilityEditScene struct {
	switcher     Switcher
	abilityStore *descriptor.AbilityStore // 能力持久化存储（可为 nil）
	ability      descriptor.CustomAbility // 正在编辑的能力（值拷贝）
	isNew        bool                     // true=新建, false=编辑已有
	returnScene  Scene                    // 调用方场景（ESC/取消/保存后切回），由构造函数注入

	// ── 管线编辑状态（使用 descriptor 包导出类型，避免类型重复） ──
	pipelines []descriptor.PipelineEditState

	// ── Picker 状态 ──
	picker       hud.PrimitivePickerData // 原语选择器弹窗数据
	pickerRects  []ui.Rect              // picker 各选项的点击区域
	pickerTarget pickerTargetInfo        // picker 打开时的目标上下文

	// ── 展开的参数编辑面板 ──
	// expandedComp 记录当前展开参数编辑的组件，
	// 同一时刻只展开一个（点击 badge 切换）。
	expandedPipe int    // 管线索引，-1=无展开
	expandedSlot string // "condition"/"selector"/"effect"
	expandedComp int    // 组件索引

	// ── UI 状态 ──
	scrollY      float64   // 管线列表滚动偏移
	btnRects     []ui.Rect // 底部按钮的布局矩形
	pipeRects    []ui.Rect // 每条管线卡片的渲染区域
	addPipeRect  ui.Rect   // +添加管线按钮区域
	delPipeRects []ui.Rect // 每条管线的删除按钮区域

	// ── 管线内各槽位的点击区域 ──
	triggerRects   []ui.Rect // 每条管线的触发器区域
	selectorRects  []ui.Rect // 每条管线的选择器区域
	condBtnRects   []ui.Rect // 每条管线的 +条件 按钮区域
	effectBtnRects []ui.Rect // 每条管线的 +效果 按钮区域

	// ── 动态点击区域（每帧重建） ──
	badgeRects      []condEffectBadgeRect // 条件/效果 badge 点击区域
	badgeDelRects   []condEffectBadgeRect // badge 删除按钮（右上角 "×"）区域
	paramBtns       []paramBtnRect        // 参数 +/- 按钮区域
	scalerToggles   []scalerToggleRect    // scaler mode toggle 区域
	stringEnumBtns  []stringEnumRect      // 字符串枚举参数 click-to-cycle 区域

	// ── 名称编辑 ──
	nameEditing bool   // true=名称编辑模式
	nameBackup  string // 编辑前名称（ESC 还原用）
	nameRect    ui.Rect // 名称标签的点击区域

	bgGrad *draw.CachedGradient // 背景渐变缓存
	dirty  bool                  // true=有未保存的更改（任何编辑操作后置 true）
}

// NewAbilityEditScene 创建能力编辑场景。
// ca 为要编辑的能力（nil=新建空能力），store 为持久化存储，returnTo 是返回场景。
func NewAbilityEditScene(sw Switcher, ca *descriptor.CustomAbility, store *descriptor.AbilityStore, returnTo Scene) *AbilityEditScene {
	s := &AbilityEditScene{
		switcher:     sw,
		abilityStore: store,
		isNew:        ca == nil,
		returnScene:  returnTo,
		expandedPipe: -1,
		bgGrad:       draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
	}

	if ca != nil {
		s.ability = *ca // 值拷贝
		// 从已有能力的 descriptor 还原管线编辑状态
		s.pipelines = descriptor.DescriptorToEditState(&ca.Desc)
	} else {
		// 新建：生成默认名称，初始化一条空管线
		count := 0
		if store != nil {
			count = store.Count()
		}
		s.ability = descriptor.CustomAbility{
			Name: fmt.Sprintf("自定义能力-%03d", count+1),
		}
		s.pipelines = []descriptor.PipelineEditState{aeNewEmptyPipeline()}
	}

	return s
}

// aeNewEmptyPipeline 创建一条空管线（所有字段为零值/空）。
func aeNewEmptyPipeline() descriptor.PipelineEditState {
	return descriptor.PipelineEditState{
		SelectorParams: map[string]float64{},
	}
}

// ── 元数据查找辅助 ──────────────────────────────────

// aeFindMeta 从元数据列表中按 ID 查找。未找到返回 nil。
func aeFindMeta(metas []descriptor.PrimitiveMeta, id string) *descriptor.PrimitiveMeta {
	for i := range metas {
		if metas[i].ID == id {
			return &metas[i]
		}
	}
	return nil
}

// aeMetaLabel 返回元数据的中文标签，未找到时返回原始 ID。
func aeMetaLabel(metas []descriptor.PrimitiveMeta, id string) string {
	if m := aeFindMeta(metas, id); m != nil {
		return m.Label
	}
	return id
}

// aeDefaultParams 从参数元数据列表生成默认参数 map。
// 对 scaler 类型参数，生成 "value" 键（fixed 模式）。
func aeDefaultParams(params []descriptor.ParamMeta) map[string]float64 {
	m := make(map[string]float64, len(params))
	// 统计 scaler 类型参数数量，决定是否需要前缀
	scalerCount := 0
	for _, p := range params {
		if p.Type == "scaler" {
			scalerCount++
		}
	}
	for _, p := range params {
		switch p.Type {
		case "scaler":
			if scalerCount == 1 {
				// 单 scaler：直接用 value 键（与 buildScaler 的 FixedScaler 解析兼容）
				m["value"] = p.Default
			} else {
				// 多 scaler：用 Key 作前缀（与 extractPrefixParams 兼容）
				// 例如 slow 的 factor + duration → factor_value=0.3, duration_value=1.0
				m[p.Key+"_value"] = p.Default
			}
		case "float", "int":
			m[p.Key] = p.Default
		case "string":
			m[p.Key] = 0 // 默认选第一个枚举选项
		}
	}
	return m
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

// ── 费用计算 ──────────────────────────────────────

// aeTotalCost 计算所有管线的总费用。
// 从各组件的 PrimitiveMeta.Cost 中累加。
func (s *AbilityEditScene) aeTotalCost() int {
	cost := 0
	condMetas := descriptor.AllConditionMeta()
	selMetas := descriptor.AllSelectorMeta()
	effMetas := descriptor.AllEffectMeta()

	for _, p := range s.pipelines {
		// 条件费用
		for _, c := range p.Conditions {
			if m := aeFindMeta(condMetas, c.TypeID); m != nil {
				cost += m.Cost
			}
		}
		// 选择器费用
		if m := aeFindMeta(selMetas, p.SelectorID); m != nil {
			cost += m.Cost
		}
		// 效果费用
		for _, e := range p.Effects {
			if m := aeFindMeta(effMetas, e.TypeID); m != nil {
				cost += m.Cost
			}
		}
	}
	return cost
}

// ── 参数调整步长 ──────────────────────────────────

// aeParamStep 根据 min/max 计算合理步长。
// step = (max-min)/20, 然后对齐到 nice values。
func aeParamStep(pm descriptor.ParamMeta) float64 {
	if pm.Max <= pm.Min {
		return 1
	}
	raw := (pm.Max - pm.Min) / 20
	if pm.Type == "int" {
		// 整数参数步长至少 1
		s := math.Round(raw)
		if s < 1 {
			s = 1
		}
		return s
	}
	// 浮点数：对齐到 0.01/0.05/0.1/0.5/1/5/10 等
	niceSteps := []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 50, 100}
	for _, ns := range niceSteps {
		if raw <= ns*1.5 {
			return ns
		}
	}
	return raw
}

// ── Update ────────────────────────────────────────

func (s *AbilityEditScene) Update() error {
	// 名称编辑模式：优先处理文本输入
	if s.nameEditing {
		s.updateNameEditing()
		return nil
	}

	// ESC：picker 打开时先关闭 picker；否则返回上层场景（有未保存更改时提示）
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		if s.picker.Visible {
			s.picker.Visible = false
			return nil
		}
		if s.dirty {
			hud.ShowToast("未保存的更改已丢弃") // TODO: i18n — ability.unsaved_discard
		}
		s.switcher.SwitchScene(s.returnScene)
		return nil
	}

	// 鼠标滚轮滚动管线列表。
	// 直接使用 ebiten.Wheel() 而非 input.Gesture.ScrollDelta()，
	// 因为 AbilityEditScene 不持有 Gesture 实例。
	// 这与 vfx_preview.go/audio_preview.go/wave_preview.go 保持一致。
	_, wy := ebiten.Wheel()
	if wy != 0 {
		s.scrollY -= wy * 20
		s.clampScroll()
	}

	// Picker hover 更新
	if s.picker.Visible && len(s.pickerRects) > 0 {
		mx, my := draw.CursorPos()
		s.picker.HoverIdx = hud.PrimitivePickerHoverTest(mx, my, s.pickerRects)
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
//  1. Picker 弹窗（最顶层，优先消费）
//  2. 参数 +/- 按钮
//  3. Scaler mode toggle
//  4a. 条件/效果 badge 删除按钮（×）
//  4b. 条件/效果 badge 点击（展开参数面板）
//  5. 管线删除按钮（[x]）
//  6. 添加管线按钮
//  7. 管线内各槽位（触发器、选择器、+条件、+效果）
//  8. 名称标签点击 → 进入编辑模式
//  9. 底部按钮（取消/保存）
func (s *AbilityEditScene) handleInput(mx, my float64) {
	// 1. Picker 弹窗命中检测（如果可见）
	if s.picker.Visible {
		hit := hud.PrimitivePickerHitTest(mx, my, s.pickerRects, s.picker)
		if hit >= 0 {
			// 选中某个选项
			playUIClick(s.switcher)
			s.handlePickerSelect(hit)
			return
		}
		if hit == -2 {
			// 点击在弹窗内但不在选项上（标题栏区域），忽略
			return
		}
		// hit == -1：点击弹窗外，关闭 picker
		s.picker.Visible = false
		// 不 return，允许点击穿透到下层
	}

	// 2. 参数 +/- 按钮
	for _, pb := range s.paramBtns {
		if pb.Rect.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleParamAdjust(pb)
			return
		}
	}

	// 3. Scaler mode toggle
	for _, st := range s.scalerToggles {
		if st.Rect.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleScalerToggle(st)
			return
		}
	}

	// 3b. 字符串枚举参数 click-to-cycle
	for _, se := range s.stringEnumBtns {
		if se.Rect.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleStringEnumCycle(se)
			return
		}
	}

	// 4a. 条件/效果 badge 删除按钮 → 移除组件
	for _, dr := range s.badgeDelRects {
		if dr.Rect.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleBadgeDelete(dr)
			return
		}
	}

	// 4b. 条件/效果 badge 点击 → 展开/收起参数面板或打开 picker 替换
	for _, br := range s.badgeRects {
		if br.Rect.Contains(mx, my) {
			playUIClick(s.switcher)
			s.handleBadgeClick(br)
			return
		}
	}

	// 5. 检查删除管线按钮
	for i, r := range s.delPipeRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.deletePipeline(i)
			return
		}
	}

	// 6. 检查添加管线按钮
	if s.addPipeRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.addPipeline()
		return
	}

	// 7. 管线内槽位点击
	for i, r := range s.triggerRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.openPicker(i, "trigger", -1, r)
			return
		}
	}
	for i, r := range s.selectorRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			pipe := s.pipelines[i]
			// 选择器已设置且有参数：检查点击位置区分「展开参数」和「更换选择器」
			if pipe.SelectorID != "" {
				selMeta := aeFindMeta(descriptor.AllSelectorMeta(), pipe.SelectorID)
				if selMeta != nil && len(selMeta.Params) > 0 {
					// 右侧 50px 区域为更换按钮，其余为参数展开
					changeX := r.X + r.W - 50
					if mx >= float64(changeX) {
						s.openPicker(i, "selector", -1, r)
					} else {
						// 切换参数面板展开
						if s.expandedSlot == "selector" && s.expandedPipe == i {
							s.expandedPipe = -1
							s.expandedSlot = ""
						} else {
							s.expandedSlot = "selector"
							s.expandedPipe = i
							s.expandedComp = 0
						}
					}
					return
				}
			}
			// 选择器未设置或无参数：直接打开 picker
			s.openPicker(i, "selector", -1, r)
			return
		}
	}
	for i, r := range s.condBtnRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.openPicker(i, "condition", -1, r)
			return
		}
	}
	for i, r := range s.effectBtnRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.openPicker(i, "effect", -1, r)
			return
		}
	}

	// 8. 名称标签点击 → 进入编辑模式
	if s.nameRect.W > 0 && s.nameRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.nameBackup = s.ability.Name
		s.nameEditing = true
		return
	}

	// 9. 底部按钮
	for i, r := range s.btnRects {
		if r.Contains(mx, my) {
			s.handleBtnClick(i)
			return
		}
	}
}

// ── Picker 管理 ──────────────────────────────────

// openPicker 打开原语选择器弹窗。
// slotRect 用于定位弹窗（紧贴槽位右下方）。
func (s *AbilityEditScene) openPicker(pipeIdx int, slotType string, compIdx int, slotRect ui.Rect) {
	var title string
	var options []descriptor.PrimitiveMeta
	var current string

	pipe := s.pipelines[pipeIdx]
	switch slotType {
	case "trigger":
		title = "选择触发器"
		options = descriptor.AllTriggerMeta()
		current = pipe.TriggerID
	case "condition":
		title = "选择条件"
		options = descriptor.AllConditionMeta()
		if compIdx >= 0 && compIdx < len(pipe.Conditions) {
			current = pipe.Conditions[compIdx].TypeID
		}
	case "selector":
		title = "选择目标"
		options = descriptor.AllSelectorMeta()
		current = pipe.SelectorID
	case "effect":
		title = "选择效果"
		options = descriptor.AllEffectMeta()
		if compIdx >= 0 && compIdx < len(pipe.Effects) {
			current = pipe.Effects[compIdx].TypeID
		}
	}

	s.picker = hud.PrimitivePickerData{
		Title:    title,
		Options:  options,
		Current:  current,
		X:        slotRect.X + slotRect.W + 4,
		Y:        slotRect.Y,
		Visible:  true,
		HoverIdx: -1,
	}
	s.pickerTarget = pickerTargetInfo{
		pipeIdx:  pipeIdx,
		slotType: slotType,
		compIdx:  compIdx,
	}
}

// handlePickerSelect 处理 picker 选择结果。
func (s *AbilityEditScene) handlePickerSelect(optionIdx int) {
	if optionIdx < 0 || optionIdx >= len(s.picker.Options) {
		s.picker.Visible = false
		return
	}

	meta := s.picker.Options[optionIdx]
	pt := s.pickerTarget
	pipe := &s.pipelines[pt.pipeIdx]

	switch pt.slotType {
	case "trigger":
		pipe.TriggerID = meta.ID

	case "selector":
		pipe.SelectorID = meta.ID
		pipe.SelectorParams = aeDefaultParams(meta.Params)

	case "condition":
		comp := descriptor.ComponentEditState{
			TypeID: meta.ID,
			Params: aeDefaultParams(meta.Params),
		}
		if pt.compIdx >= 0 && pt.compIdx < len(pipe.Conditions) {
			// 替换已有条件
			pipe.Conditions[pt.compIdx] = comp
		} else {
			// 新增条件
			pipe.Conditions = append(pipe.Conditions, comp)
		}

	case "effect":
		comp := descriptor.ComponentEditState{
			TypeID: meta.ID,
			Params: aeDefaultParams(meta.Params),
		}
		if pt.compIdx >= 0 && pt.compIdx < len(pipe.Effects) {
			// 替换已有效果
			pipe.Effects[pt.compIdx] = comp
		} else {
			// 新增效果
			pipe.Effects = append(pipe.Effects, comp)
		}
	}

	s.picker.Visible = false
	s.dirty = true
}

// ── Badge 点击处理 ──────────────────────────────────

// handleBadgeClick 处理条件/效果 badge 点击。
// 如果已展开该组件的参数面板则收起，否则展开。
func (s *AbilityEditScene) handleBadgeClick(br condEffectBadgeRect) {
	// 已展开同一个组件 → 收起
	if s.expandedPipe == br.PipeIdx && s.expandedSlot == br.SlotType && s.expandedComp == br.CompIdx {
		s.expandedPipe = -1
		return
	}
	// 展开新的组件
	s.expandedPipe = br.PipeIdx
	s.expandedSlot = br.SlotType
	s.expandedComp = br.CompIdx
}

// handleBadgeDelete 处理条件/效果 badge 删除。
// 从对应管线中移除指定索引的条件或效果。
func (s *AbilityEditScene) handleBadgeDelete(dr condEffectBadgeRect) {
	if dr.PipeIdx < 0 || dr.PipeIdx >= len(s.pipelines) {
		return
	}
	pipe := &s.pipelines[dr.PipeIdx]

	switch dr.SlotType {
	case "condition":
		if dr.CompIdx >= 0 && dr.CompIdx < len(pipe.Conditions) {
			pipe.Conditions = append(pipe.Conditions[:dr.CompIdx], pipe.Conditions[dr.CompIdx+1:]...)
		}
	case "effect":
		if dr.CompIdx >= 0 && dr.CompIdx < len(pipe.Effects) {
			pipe.Effects = append(pipe.Effects[:dr.CompIdx], pipe.Effects[dr.CompIdx+1:]...)
		}
	}

	// 如果删除的组件正在展开参数面板，收起
	if s.expandedPipe == dr.PipeIdx && s.expandedSlot == dr.SlotType && s.expandedComp == dr.CompIdx {
		s.expandedPipe = -1
	}
	s.dirty = true
}

// ── 参数调整 ──────────────────────────────────────

// handleParamAdjust 处理参数 +/- 按钮点击。
func (s *AbilityEditScene) handleParamAdjust(pb paramBtnRect) {
	pipe := &s.pipelines[pb.PipeIdx]

	// 获取参数 map
	var params map[string]float64
	var paramMetas []descriptor.ParamMeta

	switch pb.SlotType {
	case "condition":
		if pb.CompIdx < 0 || pb.CompIdx >= len(pipe.Conditions) {
			return
		}
		params = pipe.Conditions[pb.CompIdx].Params
		if params == nil {
			params = map[string]float64{}
			pipe.Conditions[pb.CompIdx].Params = params
		}
		paramMetas = aeGetParamMetas(descriptor.AllConditionMeta(), pipe.Conditions[pb.CompIdx].TypeID)
	case "selector":
		params = pipe.SelectorParams
		if params == nil {
			params = map[string]float64{}
			pipe.SelectorParams = params
		}
		paramMetas = aeGetParamMetas(descriptor.AllSelectorMeta(), pipe.SelectorID)
	case "effect":
		if pb.CompIdx < 0 || pb.CompIdx >= len(pipe.Effects) {
			return
		}
		params = pipe.Effects[pb.CompIdx].Params
		if params == nil {
			params = map[string]float64{}
			pipe.Effects[pb.CompIdx].Params = params
		}
		paramMetas = aeGetParamMetas(descriptor.AllEffectMeta(), pipe.Effects[pb.CompIdx].TypeID)
	}

	// 查找对应参数的元数据。
	// 对 scaler 参数，ParamKey 可能是 "value"/"base"/"potential"/"k"/"cap"，
	// 也可能带前缀如 "factor_base"/"duration_k"。
	// 需要匹配 scaler 类型的 ParamMeta（而非按 key 精确匹配）。
	var pm *descriptor.ParamMeta
	scalerSuffixes := []string{"value", "base", "potential", "k", "cap"}
	isScalerSubKey := false
	for _, sfx := range scalerSuffixes {
		if pb.ParamKey == sfx {
			isScalerSubKey = true
			break
		}
		// 检查带前缀的形式（如 "factor_base"）
		if len(pb.ParamKey) > len(sfx)+1 && pb.ParamKey[len(pb.ParamKey)-len(sfx)-1] == '_' && pb.ParamKey[len(pb.ParamKey)-len(sfx):] == sfx {
			isScalerSubKey = true
			break
		}
	}

	for i := range paramMetas {
		if isScalerSubKey && paramMetas[i].Type == "scaler" {
			pm = &paramMetas[i]
			break
		}
		if paramMetas[i].Key == pb.ParamKey {
			pm = &paramMetas[i]
			break
		}
	}
	if pm == nil {
		return
	}

	step := aeParamStep(*pm)
	val := params[pb.ParamKey]
	val += step * float64(pb.Delta)

	// 钳制到 [min, max]（potential/k/cap 有自定义范围）
	// 支持带前缀的键名（如 "factor_potential"、"duration_k"）
	minVal := pm.Min
	maxVal := pm.Max
	keySuffix := pb.ParamKey
	if idx := len(pb.ParamKey) - 1; idx > 0 {
		for _, sfx := range []string{"potential", "k", "cap"} {
			if len(pb.ParamKey) > len(sfx) && pb.ParamKey[len(pb.ParamKey)-len(sfx):] == sfx {
				keySuffix = sfx
				break
			}
		}
	}
	switch keySuffix {
	case "potential":
		minVal = 0
	case "k":
		// diminishing k 参数：值越大衰减越缓，合理范围 1~500
		minVal = 1
		maxVal = 500
	case "cap":
		// capped 上限：最小为 min，最大为 max*10
		minVal = pm.Min
		maxVal = pm.Max * 10
	}
	if val < minVal {
		val = minVal
	}
	if val > maxVal {
		val = maxVal
	}

	// 整数类型取整
	if pm.Type == "int" {
		val = math.Round(val)
	}

	params[pb.ParamKey] = val
	s.dirty = true
}

// handleScalerToggle 处理 scaler mode 切换。
// fixed → linear → diminishing → capped → fixed 循环。
func (s *AbilityEditScene) handleScalerToggle(st scalerToggleRect) {
	pipe := &s.pipelines[st.PipeIdx]

	var params map[string]float64
	switch st.SlotType {
	case "condition":
		if st.CompIdx >= 0 && st.CompIdx < len(pipe.Conditions) {
			params = pipe.Conditions[st.CompIdx].Params
		}
	case "selector":
		params = pipe.SelectorParams
	case "effect":
		if st.CompIdx >= 0 && st.CompIdx < len(pipe.Effects) {
			params = pipe.Effects[st.CompIdx].Params
		}
	}
	if params == nil {
		return
	}

	// 多 scaler 效果用 Key 前缀（factor_value / duration_base 等）
	prefix := aeScalerPrefix(params, st.ParamKey)

	_, hasValue := params[prefix+"value"]
	_, hasBase := params[prefix+"base"]
	_, hasK := params[prefix+"k"]
	_, hasCap := params[prefix+"cap"]

	switch {
	case hasValue && !hasBase:
		// fixed → linear: value → base, 新增 potential=0
		base := params[prefix+"value"]
		delete(params, prefix+"value")
		params[prefix+"base"] = base
		params[prefix+"potential"] = 0

	case hasBase && !hasK && !hasCap:
		// linear → diminishing: 保留 base/potential, 新增 k=50
		params[prefix+"k"] = 50

	case hasBase && hasK:
		// diminishing → capped: 删除 k, 新增 cap=base*3
		capVal := params[prefix+"base"] * 3
		if capVal < 1 {
			capVal = 1
		}
		delete(params, prefix+"k")
		params[prefix+"cap"] = capVal

	case hasBase && hasCap:
		// capped → fixed: base → value, 删除 potential/cap
		val := params[prefix+"base"]
		delete(params, prefix+"base")
		delete(params, prefix+"potential")
		delete(params, prefix+"cap")
		params[prefix+"value"] = val
	}
	s.dirty = true
}

// aeScalerPrefix 判断给定 params 中 scaler 是否使用前缀模式。
// 如果 params 中有 "key_value" 或 "key_base" 形式的 key，返回 "key_"；否则返回 ""。
func aeScalerPrefix(params map[string]float64, paramKey string) string {
	prefix := paramKey + "_"
	for k := range params {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			return prefix
		}
	}
	return "" // 单 scaler，无前缀
}

// aeGetParamMetas 从元数据列表中查找指定 typeID 的参数定义。
func aeGetParamMetas(metas []descriptor.PrimitiveMeta, typeID string) []descriptor.ParamMeta {
	if m := aeFindMeta(metas, typeID); m != nil {
		return m.Params
	}
	return nil
}

// ── 管线增删 ──────────────────────────────────────

// handleBtnClick 处理底部按钮点击。
// 按钮布局：[取消(0)] [保存(1)]
func (s *AbilityEditScene) handleBtnClick(idx int) {
	playUIClick(s.switcher)
	switch idx {
	case 0:
		// 取消（有未保存更改时提示）
		if s.dirty {
			hud.ShowToast("未保存的更改已丢弃") // TODO: i18n — ability.unsaved_discard
		}
		s.switcher.SwitchScene(s.returnScene)
	case 1:
		// 保存
		s.saveAbility()
	}
}

// ── 名称编辑 ──────────────────────────────────────

const aeNameMaxLen = 20 // 名称最大字符数

// updateNameEditing 处理名称编辑模式的键盘输入。
// 使用 ebiten.AppendInputChars 支持 IME/中文输入。
func (s *AbilityEditScene) updateNameEditing() {
	// Enter：确认
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.nameEditing = false
		s.dirty = true
		return
	}

	// ESC：还原并退出
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.ability.Name = s.nameBackup
		s.nameEditing = false
		return
	}

	// 点击名称区域外：确认并退出
	if isTapJustPressed() {
		mx, my := draw.CursorPos()
		if !s.nameRect.Contains(mx, my) {
			s.nameEditing = false
			s.dirty = true
			return
		}
	}

	// Backspace：删除末尾字符
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.ability.Name) > 0 {
		runes := []rune(s.ability.Name)
		s.ability.Name = string(runes[:len(runes)-1])
	}

	// 追加输入字符
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if len([]rune(s.ability.Name)) < aeNameMaxLen {
			s.ability.Name += string(ch)
		}
	}
}

// addPipeline 添加一条新的空管线。
func (s *AbilityEditScene) addPipeline() {
	s.pipelines = append(s.pipelines, aeNewEmptyPipeline())
	s.dirty = true
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
	s.dirty = true
	// 如果展开的参数面板所属管线被删，收起
	if s.expandedPipe == idx {
		s.expandedPipe = -1
	} else if s.expandedPipe > idx {
		s.expandedPipe--
	}
}

// ── 保存 ──────────────────────────────────────────

// saveAbility 将编辑状态转换为 AbilityDescriptor 并保存。
//
// 流程：
//  1. 生成或复用 ID
//  2. EditStateToDescriptor 编译管线
//  3. 构造 CustomAbility 保存到 AbilityStore
//  4. 成功则切回上层场景
func (s *AbilityEditScene) saveAbility() {
	id := s.ability.ID
	if id == "" {
		id = fmt.Sprintf("ca_%d", time.Now().UnixMilli())
	}
	name := s.ability.Name

	desc, err := descriptor.EditStateToDescriptor(id, name, s.pipelines)
	if err != nil {
		hud.ShowToast("保存失败") // TODO: i18n — ability.save_error
		log.Printf("[AbilityEdit] save error: %v", err)
		return
	}

	ca := descriptor.CustomAbility{
		ID:        id,
		Name:      name,
		Desc:      *desc,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	if s.abilityStore != nil {
		if err := s.abilityStore.Save(ca); err != nil {
			hud.ShowToast("保存失败") // TODO: i18n — ability.save_error
			log.Printf("[AbilityEdit] store.Save error: %v", err)
			return
		}
	}

	hud.ShowToast("能力已保存") // TODO: i18n — ability.saved
	log.Printf("[AbilityEdit] saved ability %q (id=%s, pipelines=%d)", name, id, len(s.pipelines))
	s.switcher.SwitchScene(s.returnScene)
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

	// ── 展开的参数编辑面板（浮动在卡片上方） ──
	if s.expandedPipe >= 0 {
		s.drawExpandedParamPanel(screen, fm)
	}

	// ── Picker 弹窗（最顶层绘制） ──
	if s.picker.Visible {
		s.pickerRects = hud.DrawPrimitivePicker(screen, s.picker)
	}
}

// drawTitleBar 绘制标题栏：能力名称（可点击编辑） + 实时费用。
func (s *AbilityEditScene) drawTitleBar(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	// 标题区背景分隔线
	divY := py + aeTitleH - 1
	ui.Divider(screen, px+20, divY, aePanelW-40, nil)

	// 左侧：能力名称（点击可编辑）
	nameX := float64(px) + 24
	nameY := float64(py) + 14
	nameW := float64(aePanelW)/2 - 30
	s.nameRect = ui.Rect{X: float32(nameX), Y: float32(nameY) - 2, W: float32(nameW), H: aeTitleH - 10}

	if s.nameEditing {
		// 编辑模式：高亮背景 + 光标闪烁
		draw.RoundRect(screen, float32(nameX)-2, float32(nameY)-4, float32(nameW)+4, aeTitleH-8, 4,
			color.RGBA{R: 15, G: 18, B: 30, A: 255}) //nolint:hud
		draw.StrokeRoundRect(screen, float32(nameX)-2, float32(nameY)-4, float32(nameW)+4, aeTitleH-8, 4, 1,
			color.RGBA{R: 60, G: 80, B: 140, A: 200})
		display := s.ability.Name
		if int(time.Now().UnixMilli()/500)%2 == 0 {
			display += "|"
		}
		ui.Label(screen, display, nameX, nameY, nameW, ui.LabelStyle{
			Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
		})
	} else {
		nameLabel := fmt.Sprintf("能力: %s", s.ability.Name)
		ui.Label(screen, nameLabel, nameX, nameY, nameW, ui.LabelStyle{
			Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
		})
	}

	// 右侧：实时费用
	cost := s.aeTotalCost()
	costLabel := fmt.Sprintf("费用: %d", cost)
	ui.Label(screen, costLabel, float64(px)+24, float64(py)+14, float64(aePanelW)-48, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextMuted, Align: ui.AlignRight,
	})
}

// drawPipelineList 绘制管线列表区域（带滚动裁剪）。
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
	s.badgeRects = s.badgeRects[:0]
	s.badgeDelRects = s.badgeDelRects[:0]
	s.paramBtns = s.paramBtns[:0]
	s.scalerToggles = s.scalerToggles[:0]
	s.stringEnumBtns = s.stringEnumBtns[:0]

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
// 内含触发器行、条件行（带 badge）、目标行、效果行（带 badge）。
func (s *AbilityEditScene) drawPipelineCard(screen *ebiten.Image, fm *render.FontManager, idx int, pipe descriptor.PipelineEditState, x, y float32) {
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
	s.drawTriggerRow(screen, fm, idx, pipe, contentX, contentW, rowY)
	rowY += aePipeRowH

	// 行 2: 条件
	s.drawConditionRow(screen, fm, idx, pipe, contentX, contentW, rowY)
	rowY += aePipeRowH

	// 行 3: 目标选择器
	s.drawSelectorRow(screen, fm, idx, pipe, contentX, contentW, rowY)
	rowY += aePipeRowH

	// 行 4: 效果
	s.drawEffectRow(screen, fm, idx, pipe, contentX, contentW, rowY)
}

// ── 行渲染：触发器 ──────────────────────────────────

func (s *AbilityEditScene) drawTriggerRow(screen *ebiten.Image, fm *render.FontManager, pipeIdx int, pipe descriptor.PipelineEditState, contentX, contentW, rowY float32) {
	triggerLabel := "未选择"
	triggerClr := color.Color(theme.TextLocked)
	if pipe.TriggerID != "" {
		triggerLabel = aeMetaLabel(descriptor.AllTriggerMeta(), pipe.TriggerID)
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
}

// ── 行渲染：条件 ──────────────────────────────────

func (s *AbilityEditScene) drawConditionRow(screen *ebiten.Image, fm *render.FontManager, pipeIdx int, pipe descriptor.PipelineEditState, contentX, contentW, rowY float32) {
	ui.Label(screen, "条件:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})

	// 已有条件 badge 列表
	condMetas := descriptor.AllConditionMeta()
	badgeX := contentX + 42
	for ci, cond := range pipe.Conditions {
		label := aeMetaLabel(condMetas, cond.TypeID)
		// 附加主要参数值的缩写
		label = aeCompactBadgeLabel(label, cond.Params)

		bw := aeMeasureBadgeWidth(fm, label) + aeBadgeDelW // 额外预留删除按钮宽度
		if badgeX+bw > contentX+contentW-64 {
			break // 超出可用宽度，不再画
		}

		badgeRect := ui.Rect{X: badgeX, Y: rowY + 1, W: bw, H: aeBadgeH}
		s.badgeRects = append(s.badgeRects, condEffectBadgeRect{
			Rect: badgeRect, PipeIdx: pipeIdx, SlotType: "condition", CompIdx: ci,
		})

		// 高亮当前展开的 badge
		bg := aeBadgeBg
		if s.expandedPipe == pipeIdx && s.expandedSlot == "condition" && s.expandedComp == ci {
			bg = color.RGBA{R: 50, G: 75, B: 110, A: 230}
		}
		draw.RoundRect(screen, badgeX, rowY+1, bw, aeBadgeH, aeBadgeH/2, bg) //nolint:hud
		ui.Label(screen, label, float64(badgeX)+4, float64(rowY)+3, float64(bw-aeBadgeDelW)-4, ui.LabelStyle{
			Font: theme.FontXS, Color: aeBadgeTextClr,
		})

		// "×" 删除按钮（badge 右侧）
		delX := badgeX + bw - aeBadgeDelW
		delRect := ui.Rect{X: delX, Y: rowY + 1, W: aeBadgeDelW, H: aeBadgeH}
		s.badgeDelRects = append(s.badgeDelRects, condEffectBadgeRect{
			Rect: delRect, PipeIdx: pipeIdx, SlotType: "condition", CompIdx: ci,
		})
		ui.Label(screen, "×", float64(delX)+1, float64(rowY)+3, float64(aeBadgeDelW)-2, ui.LabelStyle{
			Font: theme.FontXS, Color: color.RGBA{R: 200, G: 100, B: 100, A: 220},
		})

		badgeX += bw + aeBadgeGap
	}

	// +条件 按钮（右侧）
	condBtnX := contentX + contentW - 56
	condBtnRect := ui.Rect{X: condBtnX, Y: rowY, W: 52, H: aePipeRowH - 2}
	s.condBtnRects = append(s.condBtnRects, condBtnRect)
	ui.Button(screen, condBtnX, rowY, 52, aePipeRowH-2, "+条件", ui.ButtonStyle{
		BgColor:  color.RGBA{R: 40, G: 50, B: 75, A: 200},
		FontSize: 10,
		Radius:   6,
	})
}

// ── 行渲染：选择器 ──────────────────────────────────

func (s *AbilityEditScene) drawSelectorRow(screen *ebiten.Image, fm *render.FontManager, pipeIdx int, pipe descriptor.PipelineEditState, contentX, contentW, rowY float32) {
	selLabel := "未选择"
	selClr := color.Color(theme.TextLocked)
	hasParams := false
	if pipe.SelectorID != "" {
		selLabel = aeMetaLabel(descriptor.AllSelectorMeta(), pipe.SelectorID)
		selClr = theme.TextBody
		if m := aeFindMeta(descriptor.AllSelectorMeta(), pipe.SelectorID); m != nil && len(m.Params) > 0 {
			hasParams = true
		}
	}
	ui.Label(screen, "目标:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	selectorSlotX := contentX + 42
	selectorSlotW := contentW - 42
	selectorRect := ui.Rect{X: selectorSlotX, Y: rowY, W: selectorSlotW, H: aePipeRowH}
	s.selectorRects = append(s.selectorRects, selectorRect)

	// 高亮当前展开参数的选择器
	displayLabel := fmt.Sprintf("[%s]", selLabel)
	if hasParams && s.expandedPipe == pipeIdx && s.expandedSlot == "selector" {
		displayLabel = fmt.Sprintf("[%s] ▲", selLabel)
	} else if hasParams {
		displayLabel = fmt.Sprintf("[%s] ▼", selLabel)
	}
	ui.Label(screen, displayLabel, float64(selectorSlotX), float64(rowY), float64(selectorSlotW)-54, ui.LabelStyle{
		Font: theme.FontCaption, Color: selClr,
	})

	// 有参数时，右侧显示「换」按钮区域提示
	if hasParams {
		changeBtnX := selectorSlotX + selectorSlotW - 50
		ui.Label(screen, "[换]", float64(changeBtnX), float64(rowY), 46, ui.LabelStyle{
			Font: theme.FontXS, Color: theme.TextMuted,
		})
	}
}

// ── 行渲染：效果 ──────────────────────────────────

func (s *AbilityEditScene) drawEffectRow(screen *ebiten.Image, fm *render.FontManager, pipeIdx int, pipe descriptor.PipelineEditState, contentX, contentW, rowY float32) {
	ui.Label(screen, "效果:", float64(contentX), float64(rowY), 40, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})

	// 已有效果 badge 列表
	effMetas := descriptor.AllEffectMeta()
	badgeX := contentX + 42
	for ei, eff := range pipe.Effects {
		label := aeMetaLabel(effMetas, eff.TypeID)
		label = aeCompactBadgeLabel(label, eff.Params)

		bw := aeMeasureBadgeWidth(fm, label) + aeBadgeDelW // 额外预留删除按钮宽度
		if badgeX+bw > contentX+contentW-64 {
			break
		}

		badgeRect := ui.Rect{X: badgeX, Y: rowY + 1, W: bw, H: aeBadgeH}
		s.badgeRects = append(s.badgeRects, condEffectBadgeRect{
			Rect: badgeRect, PipeIdx: pipeIdx, SlotType: "effect", CompIdx: ei,
		})

		bg := aeBadgeBg
		if s.expandedPipe == pipeIdx && s.expandedSlot == "effect" && s.expandedComp == ei {
			bg = color.RGBA{R: 50, G: 75, B: 110, A: 230}
		}
		draw.RoundRect(screen, badgeX, rowY+1, bw, aeBadgeH, aeBadgeH/2, bg) //nolint:hud
		ui.Label(screen, label, float64(badgeX)+4, float64(rowY)+3, float64(bw-aeBadgeDelW)-4, ui.LabelStyle{
			Font: theme.FontXS, Color: aeBadgeTextClr,
		})

		// "×" 删除按钮（badge 右侧）
		delX := badgeX + bw - aeBadgeDelW
		delRect := ui.Rect{X: delX, Y: rowY + 1, W: aeBadgeDelW, H: aeBadgeH}
		s.badgeDelRects = append(s.badgeDelRects, condEffectBadgeRect{
			Rect: delRect, PipeIdx: pipeIdx, SlotType: "effect", CompIdx: ei,
		})
		ui.Label(screen, "×", float64(delX)+1, float64(rowY)+3, float64(aeBadgeDelW)-2, ui.LabelStyle{
			Font: theme.FontXS, Color: color.RGBA{R: 200, G: 100, B: 100, A: 220},
		})

		badgeX += bw + aeBadgeGap
	}

	// +效果 按钮（右侧）
	effectBtnX := contentX + contentW - 56
	effectBtnRect := ui.Rect{X: effectBtnX, Y: rowY, W: 52, H: aePipeRowH - 2}
	s.effectBtnRects = append(s.effectBtnRects, effectBtnRect)
	ui.Button(screen, effectBtnX, rowY, 52, aePipeRowH-2, "+效果", ui.ButtonStyle{
		BgColor:  color.RGBA{R: 40, G: 50, B: 75, A: 200},
		FontSize: 10,
		Radius:   6,
	})
}

// ── Badge 辅助 ──────────────────────────────────────

// aeCompactBadgeLabel 在标签后附加主要参数值的缩写。
// 例如 "概率" + {value:0.3} → "概率 0.3"
func aeCompactBadgeLabel(label string, params map[string]float64) string {
	if len(params) == 0 {
		return label
	}
	// 取第一个有意义的值
	if v, ok := params["value"]; ok {
		return label + " " + aeFormatFloat(v)
	}
	if v, ok := params["base"]; ok {
		return label + " " + aeFormatFloat(v)
	}
	// 从 params 中取任意第一个值
	for _, v := range params {
		return label + " " + aeFormatFloat(v)
	}
	return label
}

// aeFormatFloat 格式化浮点数用于显示，去除不必要的尾零。
func aeFormatFloat(v float64) string {
	if v == math.Floor(v) && math.Abs(v) < 10000 {
		return strconv.Itoa(int(v))
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	// 去除尾零：1.50 → 1.5, 1.00 → 1
	for len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] != '.' {
		s = s[:len(s)-1]
	}
	if len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] == '.' {
		s = s[:len(s)-2]
	}
	return s
}

// aeMeasureBadgeWidth 测量 badge 所需宽度。
func aeMeasureBadgeWidth(fm *render.FontManager, label string) float32 {
	if fm == nil {
		return 50
	}
	tw := fm.MeasureText(label, theme.FontXS)
	w := float32(tw) + 12
	if w < 30 {
		w = 30
	}
	return w
}

// ── 展开参数编辑面板 ──────────────────────────────

// drawExpandedParamPanel 绘制当前展开的条件/效果/选择器参数面板。
// 浮动在对应 badge 下方，显示参数名称 + 当前值 + [-][+] 按钮。
// 对 scaler 类型参数，额外显示 mode toggle（fixed/linear）。
func (s *AbilityEditScene) drawExpandedParamPanel(screen *ebiten.Image, fm *render.FontManager) {
	if s.expandedPipe < 0 || s.expandedPipe >= len(s.pipelines) {
		return
	}
	pipe := &s.pipelines[s.expandedPipe]

	// 确定参数来源和元数据
	var params map[string]float64
	var paramMetas []descriptor.ParamMeta
	var typeID string

	switch s.expandedSlot {
	case "condition":
		if s.expandedComp < 0 || s.expandedComp >= len(pipe.Conditions) {
			return
		}
		c := &pipe.Conditions[s.expandedComp]
		params = c.Params
		typeID = c.TypeID
		paramMetas = aeGetParamMetas(descriptor.AllConditionMeta(), typeID)
	case "selector":
		params = pipe.SelectorParams
		typeID = pipe.SelectorID
		paramMetas = aeGetParamMetas(descriptor.AllSelectorMeta(), typeID)
	case "effect":
		if s.expandedComp < 0 || s.expandedComp >= len(pipe.Effects) {
			return
		}
		e := &pipe.Effects[s.expandedComp]
		params = e.Params
		typeID = e.TypeID
		paramMetas = aeGetParamMetas(descriptor.AllEffectMeta(), typeID)
	default:
		return
	}

	if len(paramMetas) == 0 {
		return
	}

	// 查找对应 badge 的位置，作为面板锚点
	var anchorRect ui.Rect
	found := false
	for _, br := range s.badgeRects {
		if br.PipeIdx == s.expandedPipe && br.SlotType == s.expandedSlot && br.CompIdx == s.expandedComp {
			anchorRect = br.Rect
			found = true
			break
		}
	}
	if !found {
		// 对 selector 无 badge，使用 selector rect 作为锚点
		if s.expandedSlot == "selector" && s.expandedPipe < len(s.selectorRects) {
			anchorRect = s.selectorRects[s.expandedPipe]
			found = true
		}
	}
	if !found {
		return
	}

	// 计算面板尺寸和位置
	panelW := float32(280)
	rowCount := 0
	for _, pm := range paramMetas {
		if pm.Type == "bool" {
			continue // 暂不支持布尔参数编辑
		}
		if pm.Type == "string" {
			rowCount++ // 字符串枚举选择行
			continue
		}
		if pm.Type == "scaler" {
			// scaler: mode toggle 行 + value 行（fixed）或 base+potential+[k|cap] 行（linear/diminishing/capped）
			prefix := aeScalerPrefix(params, pm.Key)
			_, hasBase := params[prefix+"base"]
			_, hasK := params[prefix+"k"]
			_, hasCap := params[prefix+"cap"]
			if hasBase {
				rows := 3 // mode + base + potential
				if hasK || hasCap {
					rows = 4 // mode + base + potential + k/cap
				}
				rowCount += rows
			} else {
				rowCount += 2 // mode + value
			}
		} else {
			rowCount++
		}
	}
	if rowCount == 0 {
		return
	}

	panelH := float32(rowCount)*aeParamRowH + 12 // 上下 6px padding
	panelX := anchorRect.X
	panelY := anchorRect.Y + anchorRect.H + 4

	// 屏幕边界修正
	if panelX+panelW > float32(theme.CanvasW)-4 {
		panelX = float32(theme.CanvasW) - panelW - 4
	}
	if panelY+panelH > float32(theme.CanvasH)-4 {
		panelY = anchorRect.Y - panelH - 4 // 上方弹出
	}

	// 面板背景
	ui.Panel(screen, panelX, panelY, panelW, panelH, ui.PanelStyle{
		BgColor:     color.RGBA{R: 15, G: 22, B: 42, A: 245},
		BorderColor: color.RGBA{R: 70, G: 85, B: 120, A: 200},
		Radius:      8,
		BorderWidth: 1,
	})

	// 逐行绘制参数
	curY := panelY + 6
	for _, pm := range paramMetas {
		if pm.Type == "bool" {
			continue
		}

		if pm.Type == "string" {
			s.drawStringParamRow(screen, fm, panelX, curY, panelW, pm, params, typeID)
			curY += aeParamRowH
			continue
		}

		if pm.Type == "scaler" {
			curY = s.drawScalerParamRows(screen, fm, panelX, curY, panelW, pm, params)
		} else {
			s.drawSimpleParamRow(screen, fm, panelX, curY, panelW, pm, params)
			curY += aeParamRowH
		}
	}
}

// drawSimpleParamRow 绘制一个简单参数行：标签 + 值 + [-] [+]
func (s *AbilityEditScene) drawSimpleParamRow(screen *ebiten.Image, fm *render.FontManager, panelX, rowY, panelW float32, pm descriptor.ParamMeta, params map[string]float64) {
	padX := float32(8)
	val := params[pm.Key]

	// 标签
	ui.Label(screen, pm.Label+":", float64(panelX+padX), float64(rowY)+2, 80, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamLabelClr,
	})

	// 值
	valStr := aeFormatFloat(val)
	ui.Label(screen, valStr, float64(panelX)+100, float64(rowY)+2, 60, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamValueClr, Bold: true,
	})

	// [-] 按钮
	minusBtnX := panelX + panelW - padX - aeParamBtnW*2 - 6
	minusRect := ui.Rect{X: minusBtnX, Y: rowY + 1, W: aeParamBtnW, H: aeParamBtnH}
	s.paramBtns = append(s.paramBtns, paramBtnRect{
		Rect: minusRect, PipeIdx: s.expandedPipe, SlotType: s.expandedSlot,
		CompIdx: s.expandedComp, ParamKey: pm.Key, Delta: -1,
	})
	ui.Button(screen, minusBtnX, rowY+1, aeParamBtnW, aeParamBtnH, "-", ui.ButtonStyle{
		BgColor: aeParamBtnBg, FontSize: theme.FontXS, Radius: 4,
	})

	// [+] 按钮
	plusBtnX := panelX + panelW - padX - aeParamBtnW
	plusRect := ui.Rect{X: plusBtnX, Y: rowY + 1, W: aeParamBtnW, H: aeParamBtnH}
	s.paramBtns = append(s.paramBtns, paramBtnRect{
		Rect: plusRect, PipeIdx: s.expandedPipe, SlotType: s.expandedSlot,
		CompIdx: s.expandedComp, ParamKey: pm.Key, Delta: 1,
	})
	ui.Button(screen, plusBtnX, rowY+1, aeParamBtnW, aeParamBtnH, "+", ui.ButtonStyle{
		BgColor: aeParamBtnBg, FontSize: theme.FontXS, Radius: 4,
	})
}

// drawScalerParamRows 绘制 scaler 类型参数行：mode toggle + value/base+potential+[k|cap]。
// 返回绘制后的 Y 坐标。
func (s *AbilityEditScene) drawScalerParamRows(screen *ebiten.Image, fm *render.FontManager, panelX, startY, panelW float32, pm descriptor.ParamMeta, params map[string]float64) float32 {
	padX := float32(8)
	curY := startY

	// 多 scaler 效果（slow/dot/weaken）用 Key 前缀区分参数。
	// 例如 slow 有 factor + duration 两个 scaler，存储为 factor_value / duration_base 等。
	// 单 scaler 效果直接用 value/base/potential。
	prefix := s.scalerKeyPrefix(pm.Key)

	// 判断当前模式：fixed / linear / diminishing / capped
	_, hasBase := params[prefix+"base"]
	_, hasK := params[prefix+"k"]
	_, hasCap := params[prefix+"cap"]

	modeLabel := "fixed"
	if hasBase && hasK {
		modeLabel = "diminish"
	} else if hasBase && hasCap {
		modeLabel = "capped"
	} else if hasBase {
		modeLabel = "linear"
	}

	// 行 1: 参数名 + mode toggle
	ui.Label(screen, pm.Label+":", float64(panelX+padX), float64(curY)+2, 80, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamLabelClr,
	})

	// Mode toggle 按钮
	toggleX := panelX + 100
	toggleW := float32(62)
	toggleRect := ui.Rect{X: toggleX, Y: curY + 1, W: toggleW, H: aeParamBtnH}
	s.scalerToggles = append(s.scalerToggles, scalerToggleRect{
		Rect: toggleRect, PipeIdx: s.expandedPipe, SlotType: s.expandedSlot,
		CompIdx: s.expandedComp, ParamKey: pm.Key,
	})
	toggleBg := color.RGBA{R: 40, G: 55, B: 80, A: 220}
	ui.Button(screen, toggleX, curY+1, toggleW, aeParamBtnH, modeLabel, ui.ButtonStyle{
		BgColor: toggleBg, FontSize: theme.FontXS, Radius: 4,
	})
	curY += aeParamRowH

	if hasBase {
		// linear/diminishing/capped: base + potential + 可选 k 或 cap
		s.drawScalerValueRow(screen, fm, panelX, curY, panelW, prefix+"base", params[prefix+"base"], pm)
		curY += aeParamRowH
		s.drawScalerValueRow(screen, fm, panelX, curY, panelW, prefix+"potential", params[prefix+"potential"], pm)
		curY += aeParamRowH

		if hasK {
			// diminishing 模式额外显示 k 参数行
			s.drawScalerValueRow(screen, fm, panelX, curY, panelW, prefix+"k", params[prefix+"k"], pm)
			curY += aeParamRowH
		} else if hasCap {
			// capped 模式额外显示 cap 参数行
			s.drawScalerValueRow(screen, fm, panelX, curY, panelW, prefix+"cap", params[prefix+"cap"], pm)
			curY += aeParamRowH
		}
	} else {
		// fixed: 仅 value 行
		s.drawScalerValueRow(screen, fm, panelX, curY, panelW, prefix+"value", params[prefix+"value"], pm)
		curY += aeParamRowH
	}

	return curY
}

// scalerKeyPrefix 返回 scaler 参数的 key 前缀。
// 多 scaler 效果（如 slow 的 factor/duration）用 "factor_"/"duration_" 前缀。
// 单 scaler 效果无前缀（空字符串）。
func (s *AbilityEditScene) scalerKeyPrefix(paramKey string) string {
	// 获取当前展开组件的 meta 信息
	meta := s.expandedMeta()
	if meta == nil {
		return ""
	}
	scalerCount := 0
	for _, p := range meta.Params {
		if p.Type == "scaler" {
			scalerCount++
		}
	}
	if scalerCount <= 1 {
		return "" // 单 scaler，无前缀
	}
	return paramKey + "_" // 多 scaler，用 key 作前缀
}

// expandedMeta 返回当前展开的组件对应的 PrimitiveMeta。
func (s *AbilityEditScene) expandedMeta() *descriptor.PrimitiveMeta {
	if s.expandedPipe < 0 || s.expandedPipe >= len(s.pipelines) {
		return nil
	}
	p := &s.pipelines[s.expandedPipe]
	var typeID string
	switch s.expandedSlot {
	case "condition":
		if s.expandedComp >= 0 && s.expandedComp < len(p.Conditions) {
			typeID = p.Conditions[s.expandedComp].TypeID
		}
	case "effect":
		if s.expandedComp >= 0 && s.expandedComp < len(p.Effects) {
			typeID = p.Effects[s.expandedComp].TypeID
		}
	case "selector":
		typeID = p.SelectorID
	default:
		return nil
	}
	for _, m := range s.metaForSlot(s.expandedSlot) {
		if m.ID == typeID {
			return &m
		}
	}
	return nil
}

// metaForSlot 返回指定槽位类型的所有元数据。
func (s *AbilityEditScene) metaForSlot(slot string) []descriptor.PrimitiveMeta {
	switch slot {
	case "condition":
		return descriptor.AllConditionMeta()
	case "effect":
		return descriptor.AllEffectMeta()
	case "selector":
		return descriptor.AllSelectorMeta()
	default:
		return nil
	}
}

// drawScalerValueRow 绘制 scaler 的单个值行（base/potential/value）。
func (s *AbilityEditScene) drawScalerValueRow(screen *ebiten.Image, fm *render.FontManager, panelX, rowY, panelW float32, paramKey string, val float64, pm descriptor.ParamMeta) {
	padX := float32(8)

	// 缩进的标签
	keyLabel := "  " + paramKey + ":"
	ui.Label(screen, keyLabel, float64(panelX+padX), float64(rowY)+2, 80, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamLabelClr,
	})

	// 值
	valStr := aeFormatFloat(val)
	ui.Label(screen, valStr, float64(panelX)+100, float64(rowY)+2, 60, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamValueClr, Bold: true,
	})

	// [-] 按钮
	minusBtnX := panelX + panelW - padX - aeParamBtnW*2 - 6
	minusRect := ui.Rect{X: minusBtnX, Y: rowY + 1, W: aeParamBtnW, H: aeParamBtnH}
	s.paramBtns = append(s.paramBtns, paramBtnRect{
		Rect: minusRect, PipeIdx: s.expandedPipe, SlotType: s.expandedSlot,
		CompIdx: s.expandedComp, ParamKey: paramKey, Delta: -1,
	})
	ui.Button(screen, minusBtnX, rowY+1, aeParamBtnW, aeParamBtnH, "-", ui.ButtonStyle{
		BgColor: aeParamBtnBg, FontSize: theme.FontXS, Radius: 4,
	})

	// [+] 按钮
	plusBtnX := panelX + panelW - padX - aeParamBtnW
	plusRect := ui.Rect{X: plusBtnX, Y: rowY + 1, W: aeParamBtnW, H: aeParamBtnH}
	s.paramBtns = append(s.paramBtns, paramBtnRect{
		Rect: plusRect, PipeIdx: s.expandedPipe, SlotType: s.expandedSlot,
		CompIdx: s.expandedComp, ParamKey: paramKey, Delta: 1,
	})
	ui.Button(screen, plusBtnX, rowY+1, aeParamBtnW, aeParamBtnH, "+", ui.ButtonStyle{
		BgColor: aeParamBtnBg, FontSize: theme.FontXS, Radius: 4,
	})
}

// drawStringParamRow 绘制字符串枚举参数行：标签 + [当前值 ▼]（点击循环切换）。
func (s *AbilityEditScene) drawStringParamRow(screen *ebiten.Image, fm *render.FontManager, panelX, rowY, panelW float32, pm descriptor.ParamMeta, params map[string]float64, typeID string) {
	padX := float32(8)

	// 标签
	ui.Label(screen, pm.Label+":", float64(panelX+padX), float64(rowY)+2, 80, ui.LabelStyle{
		Font: theme.FontXS, Color: aeParamLabelClr,
	})

	// 获取当前选中的枚举标签
	lookupKey := typeID + ":" + pm.Key
	opts := descriptor.StringParamOptions[lookupKey]
	idx := int(params[pm.Key])
	displayLabel := "?"
	if idx >= 0 && idx < len(opts) {
		displayLabel = opts[idx].Label
	}
	displayLabel += " ▼"

	// 可点击的枚举值区域
	enumX := panelX + 100
	enumW := panelW - 100 - padX
	enumRect := ui.Rect{X: enumX, Y: rowY + 1, W: enumW, H: aeParamBtnH}
	s.stringEnumBtns = append(s.stringEnumBtns, stringEnumRect{
		Rect:     enumRect,
		PipeIdx:  s.expandedPipe,
		SlotType: s.expandedSlot,
		CompIdx:  s.expandedComp,
		ParamKey: pm.Key,
		TypeID:   typeID,
	})

	// 渲染为按钮样式
	ui.Button(screen, enumX, rowY+1, enumW, aeParamBtnH, displayLabel, ui.ButtonStyle{
		BgColor:  color.RGBA{R: 40, G: 55, B: 80, A: 220},
		FontSize: theme.FontXS,
		Radius:   4,
	})
}

// handleStringEnumCycle 处理字符串枚举参数的 click-to-cycle。
// 点击后索引 +1，超出范围回到 0。
func (s *AbilityEditScene) handleStringEnumCycle(se stringEnumRect) {
	pipe := &s.pipelines[se.PipeIdx]

	var params map[string]float64
	switch se.SlotType {
	case "condition":
		if se.CompIdx >= 0 && se.CompIdx < len(pipe.Conditions) {
			params = pipe.Conditions[se.CompIdx].Params
		}
	case "selector":
		params = pipe.SelectorParams
	case "effect":
		if se.CompIdx >= 0 && se.CompIdx < len(pipe.Effects) {
			params = pipe.Effects[se.CompIdx].Params
		}
	}
	if params == nil {
		return
	}

	lookupKey := se.TypeID + ":" + se.ParamKey
	opts := descriptor.StringParamOptions[lookupKey]
	if len(opts) == 0 {
		return
	}

	cur := int(params[se.ParamKey])
	next := (cur + 1) % len(opts)
	params[se.ParamKey] = float64(next)
	s.dirty = true
}

// ── drawBottomButtons ──────────────────────────────

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
