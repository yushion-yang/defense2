// blueprint_edit.go — 蓝图编辑场景（4步向导）。
//
// 用于创建和编辑自定义炮塔蓝图。
// 4 个步骤：1.攻击模式 2.属性档位+专精 3.能力选择 4.预览+保存。
// 由建塔面板「+新建」或蓝图管理菜单进入。
//
// 关联：
//   - descriptor.TowerBlueprint: 蓝图数据结构
//   - descriptor.BudgetRules / CalcBudget: 预算计算
//   - descriptor.BlueprintStore: 蓝图持久化
//   - game.go currentSceneName: 需同步注册场景名
package scene

import (
	"fmt"
	"image/color"
	"log"
	"sort"
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

	bpTitleH      = float32(50)  // 标题区高度
	bpStepIndH    = float32(40)  // 步骤指示器高度
	bpContentTopY = float32(100) // 内容区起始 Y（相对面板顶部）
	bpBtnAreaH    = float32(50)  // 底部按钮区高度
	bpBtnH        = float32(34)  // 按钮高度
	bpBtnGap      = float32(12)  // 按钮间距

	bpDotRadius = float32(5)  // 步骤圆点半径
	bpDotGap    = float32(60) // 步骤圆点间距
)

// ── 攻击模式选项 ──────────────────────────────────

// attackOption 攻击模式选项（步骤 1 用）。
type attackOption struct {
	ID    string // 描述符 ID（""=默认弹射）
	Label string // 中文标签
	Desc  string // 简短描述
	Cost  int    // 预算费用
	Style string // attackStyle 值
}

// ── 能力选项 ──────────────────────────────────────

// abilityOption 能力选项（步骤 3 用）。
type abilityOption struct {
	ID       string // 描述符 ID（预制能力）或自定义能力 ID（ca_xxx）
	Label    string // 中文标签
	Cost     int    // 预算费用
	Tag      string // 首个分类标签
	IsCustom bool   // true=来自 AbilityStore 的自定义能力
}

// ── BlueprintEditScene ──────────────────────────────

// BlueprintEditScene 蓝图编辑场景，4 步向导。
type BlueprintEditScene struct {
	switcher       Switcher
	blueprint      descriptor.TowerBlueprint
	isNew          bool                      // true=新建, false=编辑已有
	step           int                       // 0-3（attackStyle / tiers / abilities / preview）
	returnScene    Scene                     // 返回时切换到的场景
	blueprintStore *descriptor.BlueprintStore // 蓝图持久化存储
	abilityStore   *descriptor.AbilityStore   // 自定义能力存储（编辑器可能引用自定义能力）

	bgGrad *draw.CachedGradient // 背景渐变缓存

	// 底部按钮的布局结果（用于点击检测）
	btnRects []ui.Rect

	// ── 步骤 1 状态 ──
	attackOptions  []attackOption // 缓存的攻击模式选项
	selectedAttack int           // 选中索引，-1=无

	// ── 步骤 2 状态 ──
	// tiers/specialty 直接存储在 s.blueprint 中

	// ── 步骤 3 状态 ──
	availableAbilities []abilityOption // 可选能力列表（自定义 + 预制）
	lastCustomCount    int            // 上次构建时 abilityStore 中的自定义能力数量（脏检查）

	// ── 步骤 3 卡片布局缓存 ──
	step3AvailRects    []ui.Rect // 可选能力卡片矩形
	step3SelRects      []ui.Rect // 已选能力卡片矩形
	step3CreateBtnRect ui.Rect   // "+创建新能力" 按钮矩形

	// ── 步骤 1 卡片布局缓存 ──
	step1CardRects []ui.Rect // 攻击模式卡片矩形

	// ── 步骤 2 布局缓存 ──
	step2TierBtnRects     [3][3]ui.Rect // [属性][档位] 的按钮矩形
	step2SpecBtnRects     [3]ui.Rect    // 专精按钮矩形
	step2NoSpecBtnRect    ui.Rect       // 无专精按钮矩形
	step2TierBtnsReady    bool          // 按钮矩形是否已计算

	// ── 名称编辑 ──
	nameEditing bool   // true=名称编辑模式
	nameBackup  string // 编辑前名称（ESC 还原用）

	// ── 步骤 4 布局缓存 ──
	step4NameRect ui.Rect // 名称标签的点击区域

	// ── 共享 ──
	budgetRules *descriptor.BudgetRules // 预算规则
	dirty       bool                    // true=有未保存的更改（任何编辑操作后置 true）
}

// NewBlueprintEditScene 创建蓝图编辑场景。
// bp 为要编辑的蓝图（nil=新建空蓝图），returnTo 是按取消/完成时返回的场景。
// store 为蓝图持久化存储（可为 nil，此时保存功能不可用）。
// abStore 为自定义能力存储（可为 nil）。
func NewBlueprintEditScene(sw Switcher, bp *descriptor.TowerBlueprint, returnTo Scene, store *descriptor.BlueprintStore, abStore *descriptor.AbilityStore) *BlueprintEditScene {
	s := &BlueprintEditScene{
		switcher:       sw,
		isNew:          bp == nil,
		step:           bpStepAttackStyle,
		returnScene:    returnTo,
		blueprintStore: store,
		abilityStore:   abStore,
		bgGrad:         draw.NewCachedGradient(game.ScreenWidth, game.ScreenHeight, theme.SelectGradTop, theme.SelectGradBot),
		budgetRules:    descriptor.DefaultBudgetRules(),
		selectedAttack: -1,
	}
	if bp != nil {
		s.blueprint = *bp // 值拷贝，编辑不影响原始数据
	} else {
		// 新建蓝图：初始化默认值
		s.blueprint = descriptor.TowerBlueprint{
			AttackStyle: "projectile",
			Tiers: map[string]string{
				"damage":   "B",
				"atkSpeed": "B",
				"range":    "B",
			},
		}
		s.selectedAttack = 0 // 默认选中"默认弹射"
	}

	s.buildAttackOptions()
	s.buildAbilityOptions()

	// 编辑已有蓝图时，同步 selectedAttack
	if bp != nil {
		s.syncSelectedAttack()
	}

	return s
}

// ── 数据初始化 ──────────────────────────────────────

// buildAttackOptions 从全局描述符表构建攻击模式选项列表。
func (s *BlueprintEditScene) buildAttackOptions() {
	// 默认弹射（无描述符，cost=0）
	s.attackOptions = []attackOption{
		{ID: "", Label: "默认弹射", Desc: "基础单体弹射", Cost: 0, Style: "projectile"},
	}

	table := descriptor.GlobalDescriptorTable()
	if table == nil {
		return
	}

	// 收集所有 attack 标签的描述符
	var ids []string
	for id, desc := range table {
		if hasTag(desc.Tags, "attack") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	for _, id := range ids {
		desc := table[id]
		cost := s.budgetRules.AttackStyleCosts[desc.AttackStyle]
		s.attackOptions = append(s.attackOptions, attackOption{
			ID:    id,
			Label: desc.Label,
			Desc:  bpAttackDesc(id),
			Cost:  cost,
			Style: desc.AttackStyle,
		})
	}
}

// buildAbilityOptions 构建可选能力列表：自定义能力在前，预制能力在后。
//
// 自定义能力来自 abilityStore，预制能力来自全局描述符表（排除 attack 标签）。
// 同时记录 lastCustomCount 用于后续脏检查（从能力编辑器返回后自动刷新）。
func (s *BlueprintEditScene) buildAbilityOptions() {
	s.availableAbilities = nil

	// ── 自定义能力（来自 AbilityStore） ──
	if s.abilityStore != nil {
		customs := s.abilityStore.List()
		s.lastCustomCount = len(customs)
		for _, ca := range customs {
			tag := "自定义"
			if len(ca.Desc.Tags) > 0 {
				tag = ca.Desc.Tags[0]
			}
			s.availableAbilities = append(s.availableAbilities, abilityOption{
				ID:       ca.ID,
				Label:    ca.Name,
				Cost:     ca.Desc.Cost,
				Tag:      tag,
				IsCustom: true,
			})
		}
	}

	// ── 预制能力（来自全局描述符表，排除 attack 标签） ──
	table := descriptor.GlobalDescriptorTable()
	if table != nil {
		var ids []string
		for id, desc := range table {
			if !hasTag(desc.Tags, "attack") {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)

		for _, id := range ids {
			desc := table[id]
			tag := ""
			if len(desc.Tags) > 0 {
				tag = desc.Tags[0]
			}
			s.availableAbilities = append(s.availableAbilities, abilityOption{
				ID:    id,
				Label: desc.Label,
				Cost:  desc.Cost,
				Tag:   tag,
			})
		}
	}
}

// refreshAbilitiesIfNeeded 检查 abilityStore 数量变化，必要时重建可选列表。
// 用于从能力编辑器返回后懒刷新，避免每帧重建。
func (s *BlueprintEditScene) refreshAbilitiesIfNeeded() {
	if s.abilityStore == nil {
		return
	}
	if s.abilityStore.Count() != s.lastCustomCount {
		s.buildAbilityOptions()
	}
}

// syncSelectedAttack 根据 blueprint.AttackStyle 找到 attackOptions 中的对应索引。
func (s *BlueprintEditScene) syncSelectedAttack() {
	// 先尝试匹配能力 ID（编辑已有蓝图时 abilities 中可能有攻击能力）
	for i, opt := range s.attackOptions {
		if opt.Style == s.blueprint.AttackStyle {
			s.selectedAttack = i
			return
		}
	}
	s.selectedAttack = 0 // 回退到默认
}

// hasTag 检查标签列表是否包含指定标签。
func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

// bpAttackDesc 返回攻击模式简短描述。
func bpAttackDesc(id string) string {
	descs := map[string]string{
		"enhance":     "强化弹射伤害",
		"scatter":     "扇形散弹攻击",
		"wideBeam":    "贯穿直线光束",
		"spinAoe":     "旋风范围攻击",
		"bounce":      "弹射多目标",
		"splash":      "溅射范围伤害",
		"multiTarget": "同时攻击多目标",
		"radial":      "环形弹幕",
		"barrage":     "连续快射",
	}
	if d, ok := descs[id]; ok {
		return d
	}
	return ""
}

// ── 坐标辅助 ──────────────────────────────────────

// bpPanelOrigin 返回居中面板左上角坐标。
func bpPanelOrigin() (float32, float32) {
	px := (float32(theme.CanvasW) - bpPanelW) / 2
	py := (float32(theme.CanvasH) - bpPanelH) / 2
	return px, py
}

// bpContentRect 返回内容区矩形。
func bpContentRect(px, py float32) ui.Rect {
	return ui.Rect{
		X: px + 20,
		Y: py + bpContentTopY,
		W: bpPanelW - 40,
		H: bpPanelH - bpContentTopY - bpBtnAreaH - 10,
	}
}

// ── 预算辅助 ──────────────────────────────────────

// calcCurrentBudget 计算当前蓝图的预算使用情况。
// 合并预制能力和自定义能力的费用表，确保两种能力的费用都被正确计算。
func (s *BlueprintEditScene) calcCurrentBudget() descriptor.BudgetResult {
	return descriptor.CalcBudget(s.blueprint, *s.budgetRules, s.mergedAbilityCosts())
}

// mergedAbilityCosts 返回预制能力 + 自定义能力的合并费用表。
// GlobalAbilityCosts 仅包含预制能力，自定义能力费用从 abilityStore 补充。
func (s *BlueprintEditScene) mergedAbilityCosts() map[string]int {
	costs := descriptor.GlobalAbilityCosts()
	if s.abilityStore != nil {
		for _, ca := range s.abilityStore.List() {
			costs[ca.ID] = ca.Desc.Cost
		}
	}
	return costs
}

// ── Update ────────────────────────────────────────

func (s *BlueprintEditScene) Update() error {
	// 名称编辑模式：优先处理文本输入
	if s.nameEditing {
		s.updateNameEditing()
		return nil
	}

	// ESC 返回（有未保存更改时提示）
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		playUIClick(s.switcher)
		if s.dirty {
			hud.ShowToast("未保存的更改已丢弃") // TODO: i18n — blueprint.unsaved_discard
		}
		s.switcher.SwitchScene(s.returnScene)
		return nil
	}

	// 点击检测
	if isTapJustPressed() {
		mx, my := draw.CursorPos()

		// 先检查当前步骤内的点击
		s.handleStepInput(mx, my)

		// 再检查底部按钮
		for i, r := range s.btnRects {
			if r.Contains(mx, my) {
				s.handleBtnClick(i)
				break
			}
		}
	}

	return nil
}

// handleStepInput 处理当前步骤内的点击。
func (s *BlueprintEditScene) handleStepInput(mx, my float64) {
	switch s.step {
	case bpStepAttackStyle:
		s.handleAttackStyleInput(mx, my)
	case bpStepTiers:
		s.handleTiersInput(mx, my)
	case bpStepAbilities:
		s.handleAbilitiesInput(mx, my)
	case bpStepPreview:
		// 预览步骤无交互（保存按钮在底部按钮行处理）
	}
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
		// 取消（有未保存更改时提示）
		if s.dirty {
			hud.ShowToast("未保存的更改已丢弃") // TODO: i18n — blueprint.unsaved_discard
		}
		s.switcher.SwitchScene(s.returnScene)
	case idx == btnCount-1:
		// 最后一个按钮：下一步 / 保存
		if s.step < bpStepPreview {
			s.step++
		} else {
			s.saveBlueprint()
		}
	case idx == 1 && s.step > 0:
		// 上一步
		s.step--
	}
}

// saveBlueprint 执行蓝图保存逻辑。
// 保存前必须通过校验门控，校验未通过时阻止保存并提示用户。
func (s *BlueprintEditScene) saveBlueprint() {
	// 校验门控：保存前验证蓝图合法性
	errs := descriptor.ValidateBlueprint(s.blueprint, *s.budgetRules, s.mergedAbilityCosts())
	if len(errs) > 0 {
		hud.ShowToast("校验未通过，请检查") // TODO: i18n — blueprint.validation_failed
		return
	}

	// 生成 ID 和元数据（新建时）
	if s.isNew {
		s.blueprint.ID = fmt.Sprintf("bp_%d", time.Now().UnixMilli())
		s.blueprint.Author = "player"
		s.blueprint.CreatedAt = time.Now().Format(time.RFC3339)
	}

	// 自动生成名称（如果为空）
	if s.blueprint.Name == "" {
		count := 0
		if s.blueprintStore != nil {
			count = s.blueprintStore.Count()
		}
		s.blueprint.Name = generateBlueprintName(s.blueprint.AttackStyle, count)
	}

	// 计算建造费用
	budget := s.calcCurrentBudget()
	s.blueprint.BuildCost = descriptor.CalcBuildCost(budget.Used, *s.budgetRules)

	// 保存
	if s.blueprintStore != nil {
		if err := s.blueprintStore.Save(s.blueprint); err != nil {
			hud.ShowToast("保存失败") // TODO: i18n — blueprint.save_error
			log.Printf("[BlueprintEdit] save error: %v", err)
			return
		}
	}
	hud.ShowToast("蓝图已保存") // TODO: i18n — blueprint.saved

	s.switcher.SwitchScene(s.returnScene)
}

// ── 名称编辑 ──────────────────────────────────────

const bpNameMaxLen = 20 // 名称最大字符数

// updateNameEditing 处理名称编辑模式的键盘输入。
// 使用 ebiten.AppendInputChars 支持 IME/中文输入。
func (s *BlueprintEditScene) updateNameEditing() {
	// Enter：确认
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.nameEditing = false
		s.dirty = true
		return
	}

	// ESC：还原并退出
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.blueprint.Name = s.nameBackup
		s.nameEditing = false
		return
	}

	// 点击名称区域外：确认并退出
	if isTapJustPressed() {
		mx, my := draw.CursorPos()
		if !s.step4NameRect.Contains(mx, my) {
			s.nameEditing = false
			s.dirty = true
			return
		}
	}

	// Backspace：删除末尾字符
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.blueprint.Name) > 0 {
		runes := []rune(s.blueprint.Name)
		s.blueprint.Name = string(runes[:len(runes)-1])
	}

	// 追加输入字符
	chars := ebiten.AppendInputChars(nil)
	for _, ch := range chars {
		if len([]rune(s.blueprint.Name)) < bpNameMaxLen {
			s.blueprint.Name += string(ch)
		}
	}
}

// generateBlueprintName 根据攻击方式自动生成蓝图名称。
func generateBlueprintName(attackStyle string, count int) string {
	styleNames := map[string]string{
		"projectile": "弹射",
		"scatter":    "散射",
		"wideBeam":   "光束",
		"spin_aoe":   "旋风",
		"radial":     "环射",
		"barrage":    "连击",
	}
	name := styleNames[attackStyle]
	if name == "" {
		name = "自定义"
	}
	return fmt.Sprintf("%s-%03d", name, count+1)
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

	// ── 内容区 ──
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

// drawStepContent 根据当前步骤分发绘制。
func (s *BlueprintEditScene) drawStepContent(screen *ebiten.Image, fm *render.FontManager, px, py float32) {
	cr := bpContentRect(px, py)

	// 内容区背景（略深于面板）
	draw.RoundRect(screen, cr.X, cr.Y, cr.W, cr.H, 10,
		color.RGBA{R: 10, G: 15, B: 30, A: 150})

	switch s.step {
	case bpStepAttackStyle:
		s.drawAttackStyleStep(screen, fm, cr)
	case bpStepTiers:
		s.drawTiersStep(screen, fm, cr)
	case bpStepAbilities:
		s.drawAbilitiesStep(screen, fm, cr)
	case bpStepPreview:
		s.drawPreviewStep(screen, fm, cr)
	}
}

// ── 步骤 1：攻击模式选择 ──────────────────────────

// drawAttackStyleStep 绘制攻击模式选择步骤。
func (s *BlueprintEditScene) drawAttackStyleStep(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	// 标题
	ui.Label(screen, "选择攻击模式", float64(cr.X)+10, float64(cr.Y)+8, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// 卡片网格：5 列
	const (
		cardW = float32(120)
		cardH = float32(72)
		cols  = 5
		gap   = float32(8)
	)

	count := len(s.attackOptions)
	gridW, _ := ui.CardGridSize(count, ui.CardGridStyle{Cols: cols, CardW: cardW, CardH: cardH, Gap: gap})
	startX := cr.X + (cr.W-gridW)/2
	startY := cr.Y + 30

	result := ui.CardGrid(screen, startX, startY, count, ui.CardGridStyle{
		Cols: cols, CardW: cardW, CardH: cardH, Gap: gap,
	}, func(screen *ebiten.Image, idx int, r ui.Rect) {
		opt := s.attackOptions[idx]
		selected := idx == s.selectedAttack

		// 卡片背景
		bgClr := color.RGBA{R: 25, G: 32, B: 55, A: 220}
		if selected {
			bgClr = color.RGBA{R: 20, G: 60, B: 40, A: 230}
		}
		selClr := color.RGBA{R: 74, G: 222, B: 128, A: 200}

		ui.Card(screen, r.X, r.Y, r.W, r.H, ui.CardStyle{
			BgColor:       bgClr,
			BorderColor:   color.RGBA{R: 60, G: 70, B: 95, A: 200},
			Radius:        10,
			BorderWidth:   1.5,
			Selected:      selected,
			SelectedColor: selClr,
			HighlightBar:  true,
			BarWidth:       40,
		})

		// 标签
		cx := r.CenterX()
		ui.LabelV(screen, opt.Label, cx, float64(r.Y)+22, float64(r.W)-10, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
		})

		// 描述
		if opt.Desc != "" {
			ui.LabelV(screen, opt.Desc, cx, float64(r.Y)+40, float64(r.W)-10, ui.LabelStyle{
				Font: theme.FontCaption, Color: theme.TextMuted,
			})
		}

		// 费用
		costLabel := "免费"
		costClr := color.Color(theme.StatusGrowth)
		if opt.Cost > 0 {
			costLabel = fmt.Sprintf("费用: %d", opt.Cost)
			costClr = theme.ResGold
		}
		ui.LabelV(screen, costLabel, cx, float64(r.Y)+58, float64(r.W)-10, ui.LabelStyle{
			Font: theme.FontCaption, Color: costClr,
		})
	})

	s.step1CardRects = result.Rects

	// 底部预算提示
	budget := s.calcCurrentBudget()
	budgetText := fmt.Sprintf("预算: %d / %d", budget.Used, budget.Cap)
	ui.Label(screen, budgetText, float64(cr.X)+10, float64(cr.Y+cr.H)-22, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextMuted, Align: ui.AlignRight,
	})
}

// handleAttackStyleInput 处理攻击模式步骤的点击。
func (s *BlueprintEditScene) handleAttackStyleInput(mx, my float64) {
	for i, r := range s.step1CardRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.selectedAttack = i
			opt := s.attackOptions[i]
			s.blueprint.AttackStyle = opt.Style
			s.dirty = true
			break
		}
	}
}

// ── 步骤 2：属性档位 + 专精选择 ──────────────────

// bpAttrKeys 属性键名顺序。
var bpAttrKeys = [3]string{"damage", "atkSpeed", "range"}

// bpAttrLabels 属性中文标签。
var bpAttrLabels = [3]string{"伤害", "攻速", "射程"}

// bpTierLabels 档位标签。
var bpTierLabels = [3]string{"S", "B", "D"}

// bpSpecLabels 专精标签。
var bpSpecLabels = [3]string{"伤害专精", "攻速专精", "射程专精"}

// drawTiersStep 绘制属性档位 + 专精选择步骤。
//
// 布局：
//   - 3 行属性（每行：标签 + 3 个档位按钮 + 费用）
//   - 分隔线
//   - 专精选择（4 个按钮：3 属性 + 无专精）
//   - 底部预算条
func (s *BlueprintEditScene) drawTiersStep(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	// 标题
	ui.Label(screen, "选择属性档位 & 专精", float64(cr.X)+10, float64(cr.Y)+8, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// 属性档位区
	const (
		rowH     = float32(36)
		btnW     = float32(50)
		btnH     = float32(28)
		btnGap   = float32(8)
		labelW   = float32(60)
		costW    = float32(50)
		startY   = float32(34) // 相对于 cr.Y
	)

	attrColors := [3]color.Color{theme.InfoAttrDamage, theme.InfoAttrAtkSpd, theme.InfoAttrRange}

	// 居中整行：labelW + 3*btnW + 2*btnGap + costW
	totalRowW := labelW + 3*btnW + 2*btnGap + costW + 20
	rowStartX := cr.X + (cr.W-totalRowW)/2

	s.step2TierBtnsReady = true

	for row := 0; row < 3; row++ {
		ry := cr.Y + startY + float32(row)*rowH
		attr := bpAttrKeys[row]
		currentTier := s.blueprint.Tiers[attr]

		// 属性标签
		ui.Label(screen, bpAttrLabels[row], float64(rowStartX), float64(ry)+4, float64(labelW), ui.LabelStyle{
			Font: theme.FontBody, Color: attrColors[row], Bold: true,
		})

		// 3 个档位按钮
		btnStartX := rowStartX + labelW + 10
		for col := 0; col < 3; col++ {
			tier := bpTierLabels[col]
			bx := btnStartX + float32(col)*(btnW+btnGap)
			by := ry

			selected := currentTier == tier
			bgClr := color.Color(theme.BtnMuted)
			if selected {
				bgClr = theme.TonePrimary
			}

			ui.Button(screen, bx, by, btnW, btnH, tier, ui.ButtonStyle{
				BgColor:  bgClr,
				FontSize: theme.FontH2,
				Radius:   8,
				Bold:     selected,
			})

			s.step2TierBtnRects[row][col] = ui.Rect{X: bx, Y: by, W: btnW, H: btnH}
		}

		// 费用显示
		tierCost := s.budgetRules.TierCosts[currentTier]
		costText := fmt.Sprintf("(%d)", tierCost)
		costX := btnStartX + 3*(btnW+btnGap) + 4
		ui.Label(screen, costText, float64(costX), float64(ry)+6, float64(costW), ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextMuted,
		})
	}

	// 分隔线
	divY := cr.Y + startY + 3*rowH + 4
	ui.Divider(screen, cr.X+20, divY, cr.W-40, nil)

	// 专精选择标题
	specTitleY := divY + 12
	ui.Label(screen, "专精选择", float64(cr.X)+10, float64(specTitleY), float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
	})
	specCostText := fmt.Sprintf("(费用: %d)", s.budgetRules.SpecialtyCost)
	ui.Label(screen, specCostText, float64(cr.X)+10, float64(specTitleY), float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted, Align: ui.AlignRight,
	})

	// 专精按钮行
	specY := specTitleY + 22
	const specBtnW = float32(100)
	const specBtnH = float32(28)
	const specGap = float32(10)
	totalSpecW := 4*specBtnW + 3*specGap
	specStartX := cr.X + (cr.W-totalSpecW)/2

	for i := 0; i < 3; i++ {
		bx := specStartX + float32(i)*(specBtnW+specGap)
		by := specY
		selected := s.blueprint.Specialty == bpAttrKeys[i]
		bgClr := color.Color(theme.BtnMuted)
		if selected {
			bgClr = theme.ToneAccent
		}

		ui.Button(screen, bx, by, specBtnW, specBtnH, bpSpecLabels[i], ui.ButtonStyle{
			BgColor:  bgClr,
			FontSize: theme.FontBody,
			Radius:   8,
			Bold:     selected,
		})

		s.step2SpecBtnRects[i] = ui.Rect{X: bx, Y: by, W: specBtnW, H: specBtnH}
	}

	// 无专精按钮
	noSpecX := specStartX + 3*(specBtnW+specGap)
	noSpecSel := s.blueprint.Specialty == ""
	noSpecBg := color.Color(theme.BtnMuted)
	if noSpecSel {
		noSpecBg = theme.BtnSecondary
	}
	noSpecLabel := "无专精"
	ui.Button(screen, noSpecX, specY, specBtnW, specBtnH, noSpecLabel, ui.ButtonStyle{
		BgColor:  noSpecBg,
		FontSize: theme.FontBody,
		Radius:   8,
		Bold:     noSpecSel,
	})
	s.step2NoSpecBtnRect = ui.Rect{X: noSpecX, Y: specY, W: specBtnW, H: specBtnH}

	// 预算条
	budget := s.calcCurrentBudget()
	s.drawBudgetBar(screen, fm, cr, budget)
}

// handleTiersInput 处理属性档位步骤的点击。
func (s *BlueprintEditScene) handleTiersInput(mx, my float64) {
	if !s.step2TierBtnsReady {
		return
	}

	// 档位按钮
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if s.step2TierBtnRects[row][col].Contains(mx, my) {
				playUIClick(s.switcher)
				attr := bpAttrKeys[row]
				if s.blueprint.Tiers == nil {
					s.blueprint.Tiers = map[string]string{}
				}
				s.blueprint.Tiers[attr] = bpTierLabels[col]
				s.dirty = true
				return
			}
		}
	}

	// 专精按钮
	for i := 0; i < 3; i++ {
		if s.step2SpecBtnRects[i].Contains(mx, my) {
			playUIClick(s.switcher)
			s.blueprint.Specialty = bpAttrKeys[i]
			s.dirty = true
			return
		}
	}

	// 无专精按钮
	if s.step2NoSpecBtnRect.Contains(mx, my) {
		playUIClick(s.switcher)
		s.blueprint.Specialty = ""
		s.dirty = true
	}
}

// ── 步骤 3：能力选择 ──────────────────────────────

// drawAbilitiesStep 绘制能力选择步骤。
//
// 布局：左侧可选列表（自定义能力分区 + 预制能力分区），右侧已选列表。
// 自定义能力来自 AbilityStore，以紫色边框区分。预制能力使用标准灰色边框。
// 自定义能力分区底部有 "+创建新能力" 按钮，点击后启动 AbilityEditScene。
func (s *BlueprintEditScene) drawAbilitiesStep(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	// 从能力编辑器返回后，懒刷新可选列表
	s.refreshAbilitiesIfNeeded()

	// 标题
	ui.Label(screen, "选择能力", float64(cr.X)+10, float64(cr.Y)+8, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	// 已选数量提示
	slotsText := fmt.Sprintf("已选: %d / %d", len(s.blueprint.Abilities), s.budgetRules.MaxSlots)
	ui.Label(screen, slotsText, float64(cr.X)+10, float64(cr.Y)+8, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextMuted, Align: ui.AlignRight,
	})

	// 两列布局
	leftW := cr.W*0.58 - 10
	rightW := cr.W*0.42 - 10
	leftX := cr.X + 8
	rightX := cr.X + leftW + 20
	bodyY := cr.Y + 28

	budget := s.calcCurrentBudget()

	// ── 左列：可选能力（分区布局） ──
	s.step3AvailRects = nil
	s.step3CreateBtnRect = ui.Rect{}

	curY := bodyY

	// 分离自定义能力和预制能力
	var customOpts []int // 索引列表
	var prebuiltOpts []int
	for i, opt := range s.availableAbilities {
		if opt.IsCustom {
			customOpts = append(customOpts, i)
		} else {
			prebuiltOpts = append(prebuiltOpts, i)
		}
	}

	const (
		aCols  = 3
		aCardW = float32(116)
		aCardH = float32(50)
		aGap   = float32(6)
	)

	// ── 自定义能力分区 ──
	hasCustomSection := s.abilityStore != nil
	if hasCustomSection {
		// 分区标题
		customHeaderClr := color.RGBA{R: 147, G: 130, B: 220, A: 255} // 紫色调
		ui.Label(screen, "我的自定义能力", float64(leftX), float64(curY), float64(leftW), ui.LabelStyle{
			Font: theme.FontCaption, Color: customHeaderClr, Bold: true,
		})
		curY += 16

		// 自定义能力卡片
		if len(customOpts) > 0 {
			result := ui.CardGrid(screen, leftX, curY, len(customOpts), ui.CardGridStyle{
				Cols: aCols, CardW: aCardW, CardH: aCardH, Gap: aGap,
			}, func(screen *ebiten.Image, idx int, r ui.Rect) {
				optIdx := customOpts[idx]
				opt := s.availableAbilities[optIdx]
				s.drawAbilityCard(screen, opt, r, budget, true)
			})

			// 记录每张卡片在 availableAbilities 中的真实索引
			for _, r := range result.Rects {
				s.step3AvailRects = append(s.step3AvailRects, r)
			}

			// 计算卡片区域底部 Y
			rows := (len(customOpts) + aCols - 1) / aCols
			curY += float32(rows)*(aCardH+aGap) + 2
		}

		// "+创建新能力" 按钮
		const createBtnW = float32(140)
		const createBtnH = float32(28)
		createX := leftX
		createY := curY
		s.step3CreateBtnRect = ui.Rect{X: createX, Y: createY, W: createBtnW, H: createBtnH}

		ui.Button(screen, createX, createY, createBtnW, createBtnH, "+ 创建新能力", ui.ButtonStyle{
			BgColor:  color.RGBA{R: 60, G: 50, B: 90, A: 220},
			FontSize: theme.FontCaption,
			Radius:   8,
		})

		curY += createBtnH + 8
	}

	// ── 预制能力分区 ──
	ui.Label(screen, "预制能力", float64(leftX), float64(curY), float64(leftW), ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted, Bold: true,
	})
	curY += 16

	if len(prebuiltOpts) > 0 {
		result := ui.CardGrid(screen, leftX, curY, len(prebuiltOpts), ui.CardGridStyle{
			Cols: aCols, CardW: aCardW, CardH: aCardH, Gap: aGap,
		}, func(screen *ebiten.Image, idx int, r ui.Rect) {
			optIdx := prebuiltOpts[idx]
			opt := s.availableAbilities[optIdx]
			s.drawAbilityCard(screen, opt, r, budget, false)
		})

		for _, r := range result.Rects {
			s.step3AvailRects = append(s.step3AvailRects, r)
		}
	}

	// ── 右列：已选能力列表 ──
	ui.Label(screen, "已选能力", float64(rightX), float64(bodyY), float64(rightW), ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted, Bold: true,
	})

	selY := bodyY + 18
	s.step3SelRects = nil

	if len(s.blueprint.Abilities) == 0 {
		// 空提示
		ui.LabelV(screen, "点击左侧添加", float64(rightX)+float64(rightW)/2, float64(selY)+40, float64(rightW)-10, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.TextLocked,
		})
	} else {
		const selItemH = float32(28)
		const selGap = float32(4)
		for i, abID := range s.blueprint.Abilities {
			iy := selY + float32(i)*(selItemH+selGap)
			r := ui.Rect{X: rightX, Y: iy, W: rightW, H: selItemH}
			s.step3SelRects = append(s.step3SelRects, r)

			// 查找名称和费用：先查预制描述符，再查自定义能力
			label, cost := s.lookupAbilityInfo(abID)

			// 小卡片背景：自定义能力用紫色调
			bgClr := color.RGBA{R: 30, G: 50, B: 40, A: 200}
			borderClr := color.RGBA{R: 74, G: 222, B: 128, A: 120}
			if s.isCustomAbilityID(abID) {
				bgClr = color.RGBA{R: 40, G: 35, B: 55, A: 200}
				borderClr = color.RGBA{R: 147, G: 130, B: 220, A: 150}
			}

			ui.Card(screen, r.X, r.Y, r.W, r.H, ui.CardStyle{
				BgColor:     bgClr,
				BorderColor: borderClr,
				Radius:      6,
				BorderWidth: 1,
			})

			// 名称
			ui.Label(screen, label, float64(r.X)+8, float64(r.Y)+6, float64(r.W)-50, ui.LabelStyle{
				Font: theme.FontCaption, Color: theme.TextTitle, Bold: true,
			})

			// 费用 + "×" 移除提示
			removeText := fmt.Sprintf("%d  ×", cost)
			ui.Label(screen, removeText, float64(r.X)+8, float64(r.Y)+6, float64(r.W)-12, ui.LabelStyle{
				Font: theme.FontCaption, Color: theme.TextMuted, Align: ui.AlignRight,
			})
		}
	}

	// 预算条
	s.drawBudgetBar(screen, fm, cr, budget)
}

// drawAbilityCard 绘制单个能力卡片（自定义能力和预制能力共用）。
// isCustom 为 true 时使用紫色边框区分。
func (s *BlueprintEditScene) drawAbilityCard(screen *ebiten.Image, opt abilityOption, r ui.Rect, budget descriptor.BudgetResult, isCustom bool) {
	isSelected := s.isAbilitySelected(opt.ID)
	overBudget := budget.Used+opt.Cost > budget.Cap && !isSelected
	full := len(s.blueprint.Abilities) >= s.budgetRules.MaxSlots && !isSelected
	greyed := overBudget || full || isSelected

	// 卡片背景色：自定义=紫色调，预制=标准蓝灰
	bgClr := color.RGBA{R: 25, G: 32, B: 55, A: 220}
	borderClr := color.RGBA{R: 60, G: 70, B: 95, A: 200}
	if isCustom {
		bgClr = color.RGBA{R: 35, G: 28, B: 55, A: 220}
		borderClr = color.RGBA{R: 120, G: 100, B: 180, A: 200}
	}
	if greyed {
		bgClr = color.RGBA{R: 20, G: 22, B: 35, A: 180}
		borderClr = color.RGBA{R: 40, G: 45, B: 60, A: 140}
	}

	ui.Card(screen, r.X, r.Y, r.W, r.H, ui.CardStyle{
		BgColor:     bgClr,
		BorderColor: borderClr,
		Radius:      8,
		BorderWidth: 1,
	})

	cx := r.CenterX()
	textClr := color.Color(theme.TextTitle)
	if greyed {
		textClr = theme.TextLocked
	}

	// 名称
	ui.LabelV(screen, opt.Label, cx, float64(r.Y)+16, float64(r.W)-8, ui.LabelStyle{
		Font: theme.FontCaption, Color: textClr, Bold: true,
	})

	// 标签 + 费用
	tagCostText := fmt.Sprintf("%s | %d", bpTagLabel(opt.Tag), opt.Cost)
	tagClr := color.Color(theme.TextMuted)
	if greyed {
		tagClr = theme.TextLocked
	}
	ui.LabelV(screen, tagCostText, cx, float64(r.Y)+34, float64(r.W)-8, ui.LabelStyle{
		Font: theme.FontCaption, Color: tagClr,
	})
}

// handleAbilitiesInput 处理能力选择步骤的点击。
//
// 检查顺序：
//  1. "+创建新能力" 按钮 → 启动 AbilityEditScene
//  2. 已选列表 → 点击移除
//  3. 可选列表 → 点击添加（自定义 + 预制混合索引）
func (s *BlueprintEditScene) handleAbilitiesInput(mx, my float64) {
	// 1. "+创建新能力" 按钮
	if s.abilityStore != nil && s.step3CreateBtnRect.Contains(mx, my) {
		playUIClick(s.switcher)
		scene := NewAbilityEditScene(s.switcher, nil, s.abilityStore, s)
		s.switcher.SwitchScene(scene)
		return
	}

	budget := s.calcCurrentBudget()

	// 2. 点击已选列表：移除
	for i, r := range s.step3SelRects {
		if r.Contains(mx, my) {
			playUIClick(s.switcher)
			s.blueprint.Abilities = append(s.blueprint.Abilities[:i], s.blueprint.Abilities[i+1:]...)
			s.dirty = true
			return
		}
	}

	// 3. 点击可选列表：添加
	// step3AvailRects 的顺序与 drawAbilitiesStep 中绘制的卡片对应：
	// 先 customOpts（自定义能力），后 prebuiltOpts（预制能力）。
	// 需要将 rect 索引映射回 availableAbilities 的真实索引。
	customCount, prebuiltIndices := s.splitAbilityIndices()
	for i, r := range s.step3AvailRects {
		if r.Contains(mx, my) {
			// 映射回 availableAbilities 索引
			var optIdx int
			if i < customCount {
				optIdx = i // 自定义能力在 availableAbilities 前部
			} else {
				pIdx := i - customCount
				if pIdx >= len(prebuiltIndices) {
					return
				}
				optIdx = prebuiltIndices[pIdx]
			}

			opt := s.availableAbilities[optIdx]
			isSelected := s.isAbilitySelected(opt.ID)
			overBudget := budget.Used+opt.Cost > budget.Cap
			full := len(s.blueprint.Abilities) >= s.budgetRules.MaxSlots

			if !isSelected && !overBudget && !full {
				playUIClick(s.switcher)
				s.blueprint.Abilities = append(s.blueprint.Abilities, opt.ID)
				s.dirty = true
			}
			return
		}
	}
}

// handlePreviewInput 处理预览步骤的点击。
// 名称标签可点击进入编辑模式。
func (s *BlueprintEditScene) handlePreviewInput(mx, my float64) {
	if s.step4NameRect.Contains(mx, my) {
		playUIClick(s.switcher)
		// 进入名称编辑模式前，若名称为空则先填充自动名称
		if s.blueprint.Name == "" {
			count := 0
			if s.blueprintStore != nil {
				count = s.blueprintStore.Count()
			}
			s.blueprint.Name = generateBlueprintName(s.blueprint.AttackStyle, count)
		}
		s.nameBackup = s.blueprint.Name
		s.nameEditing = true
	}
}

// splitAbilityIndices 返回自定义能力数量和预制能力在 availableAbilities 中的索引列表。
// 与 drawAbilitiesStep 中的分离逻辑保持一致。
func (s *BlueprintEditScene) splitAbilityIndices() (customCount int, prebuiltIndices []int) {
	for i, opt := range s.availableAbilities {
		if opt.IsCustom {
			customCount++
		} else {
			prebuiltIndices = append(prebuiltIndices, i)
		}
	}
	return
}

// isAbilitySelected 检查能力是否已在已选列表中。
func (s *BlueprintEditScene) isAbilitySelected(id string) bool {
	for _, ab := range s.blueprint.Abilities {
		if ab == id {
			return true
		}
	}
	return false
}

// lookupAbilityInfo 查找能力的显示名称和费用。
// 先查全局描述符表（预制能力），未找到则查 abilityStore（自定义能力）。
func (s *BlueprintEditScene) lookupAbilityInfo(abID string) (label string, cost int) {
	// 预制能力
	if desc, ok := descriptor.LookupDescriptor(abID); ok {
		return desc.Label, desc.Cost
	}
	// 自定义能力
	if s.abilityStore != nil {
		if ca, err := s.abilityStore.Get(abID); err == nil {
			return ca.Name, ca.Desc.Cost
		}
	}
	return abID, 0
}

// isCustomAbilityID 判断能力 ID 是否属于自定义能力。
func (s *BlueprintEditScene) isCustomAbilityID(abID string) bool {
	if s.abilityStore == nil {
		return false
	}
	_, err := s.abilityStore.Get(abID)
	return err == nil
}

// bpTagLabel 返回标签中文名。
func bpTagLabel(tag string) string {
	labels := map[string]string{
		"cc":     "控制",
		"damage": "伤害",
		"buff":   "增益",
		"dot":    "持续",
		"zone":   "区域",
	}
	if l, ok := labels[tag]; ok {
		return l
	}
	return tag
}

// ── 步骤 4：预览 + 保存 ──────────────────────────

// drawPreviewStep 绘制预览 + 保存步骤。
//
// 显示蓝图完整摘要：攻击方式/档位/专精/能力/预算/建造费用/校验错误。
func (s *BlueprintEditScene) drawPreviewStep(screen *ebiten.Image, fm *render.FontManager, cr ui.Rect) {
	// 标题
	ui.Label(screen, "蓝图预览", float64(cr.X)+10, float64(cr.Y)+8, float64(cr.W)-20, ui.LabelStyle{
		Font: theme.FontH2, Color: theme.TextTitle, Bold: true,
	})

	budget := s.calcCurrentBudget()
	buildCost := descriptor.CalcBuildCost(budget.Used, *s.budgetRules)

	// 两列布局
	leftW := float64(cr.W) * 0.5
	rightW := float64(cr.W) * 0.5 - 20
	leftX := float64(cr.X) + 10
	rightX := float64(cr.X) + leftW + 10
	y := float64(cr.Y) + 30

	lineH := 18.0

	// ── 左列 ──

	// 蓝图名称（点击可编辑）
	name := s.blueprint.Name
	if name == "" {
		count := 0
		if s.blueprintStore != nil {
			count = s.blueprintStore.Count()
		}
		name = generateBlueprintName(s.blueprint.AttackStyle, count)
	}
	ui.Label(screen, "名称（点击编辑）", leftX, y, leftW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	y += lineH - 4

	nameFieldW := leftW - 10
	nameFieldH := lineH + 2
	s.step4NameRect = ui.Rect{X: float32(leftX), Y: float32(y - 2), W: float32(nameFieldW), H: float32(nameFieldH)}

	if s.nameEditing {
		// 编辑模式：高亮背景 + 光标闪烁
		draw.RoundRect(screen, float32(leftX)-2, float32(y)-3, float32(nameFieldW)+4, float32(nameFieldH)+2, 4,
			color.RGBA{R: 15, G: 18, B: 30, A: 255}) //nolint:hud
		draw.StrokeRoundRect(screen, float32(leftX)-2, float32(y)-3, float32(nameFieldW)+4, float32(nameFieldH)+2, 4, 1,
			color.RGBA{R: 60, G: 80, B: 140, A: 200})
		display := s.blueprint.Name
		if int(time.Now().UnixMilli()/500)%2 == 0 {
			display += "|"
		}
		ui.Label(screen, display, leftX, y, nameFieldW, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
		})
	} else {
		ui.Label(screen, name, leftX, y, nameFieldW, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
		})
	}
	y += lineH

	// 攻击方式
	ui.Label(screen, "攻击方式", leftX, y, leftW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	y += lineH - 4
	atkLabel := bpAttackStyleLabel(s.blueprint.AttackStyle)
	ui.Label(screen, atkLabel, leftX, y, leftW-10, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle,
	})
	y += lineH

	// 属性档位
	ui.Label(screen, "属性档位", leftX, y, leftW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	y += lineH - 4
	attrColors := [3]color.Color{theme.InfoAttrDamage, theme.InfoAttrAtkSpd, theme.InfoAttrRange}
	for i, key := range bpAttrKeys {
		tier := s.blueprint.Tiers[key]
		if tier == "" {
			tier = "B"
		}
		text := fmt.Sprintf("%s: %s", bpAttrLabels[i], tier)
		ui.Label(screen, text, leftX+float64(i)*70, y, 65, ui.LabelStyle{
			Font: theme.FontBody, Color: attrColors[i], Bold: true,
		})
	}
	y += lineH

	// 专精
	ui.Label(screen, "专精", leftX, y, leftW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	y += lineH - 4
	specLabel := "无"
	for i, key := range bpAttrKeys {
		if s.blueprint.Specialty == key {
			specLabel = bpSpecLabels[i]
			break
		}
	}
	ui.Label(screen, specLabel, leftX, y, leftW-10, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.StatusSkill,
	})

	// ── 右列 ──

	ry := float64(cr.Y) + 30

	// 能力列表
	ui.Label(screen, "能力列表", rightX, ry, rightW, ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted,
	})
	ry += lineH - 4
	if len(s.blueprint.Abilities) == 0 {
		ui.Label(screen, "(无)", rightX, ry, rightW, ui.LabelStyle{
			Font: theme.FontBody, Color: theme.TextLocked,
		})
		ry += lineH
	} else {
		for _, abID := range s.blueprint.Abilities {
			label, cost := s.lookupAbilityInfo(abID)
			text := fmt.Sprintf("• %s (%d)", label, cost)
			ui.Label(screen, text, rightX, ry, rightW-10, ui.LabelStyle{
				Font: theme.FontCaption, Color: theme.TextBody,
			})
			ry += lineH - 4
		}
	}

	ry += 6

	// 费用汇总
	ui.Divider(screen, float32(rightX), float32(ry), float32(rightW)-10, nil)
	ry += 10

	ui.Label(screen, fmt.Sprintf("预算: %d / %d", budget.Used, budget.Cap), rightX, ry, rightW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.TextTitle, Bold: true,
	})
	ry += lineH

	ui.Label(screen, fmt.Sprintf("建造费用: %d 金", buildCost), rightX, ry, rightW, ui.LabelStyle{
		Font: theme.FontBody, Color: theme.ResGold, Bold: true,
	})
	ry += lineH + 4

	// 校验错误
	errs := descriptor.ValidateBlueprint(s.blueprint, *s.budgetRules, s.mergedAbilityCosts())
	if len(errs) > 0 {
		ui.Label(screen, "校验问题:", rightX, ry, rightW, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.BtnDanger, Bold: true,
		})
		ry += lineH - 4
		for _, e := range errs {
			ui.Label(screen, "• "+e.Error(), rightX, ry, rightW-10, ui.LabelStyle{
				Font: theme.FontCaption, Color: theme.BtnDanger,
			})
			ry += lineH - 4
		}
	} else {
		ui.Label(screen, "校验通过", rightX, ry, rightW, ui.LabelStyle{
			Font: theme.FontCaption, Color: theme.StatusGrowth, Bold: true,
		})
	}
}

// bpAttackStyleLabel 返回攻击方式的中文标签。
func bpAttackStyleLabel(style string) string {
	labels := map[string]string{
		"projectile": "弹射",
		"scatter":    "散射",
		"wideBeam":   "贯穿光束",
		"spin_aoe":   "旋风",
		"radial":     "环射",
		"barrage":    "连击",
	}
	if l, ok := labels[style]; ok {
		return l
	}
	return style
}

// ── 共享 UI 组件 ──────────────────────────────────

// drawBudgetBar 在内容区底部绘制预算进度条。
func (s *BlueprintEditScene) drawBudgetBar(screen *ebiten.Image, _ *render.FontManager, cr ui.Rect, budget descriptor.BudgetResult) {
	barY := cr.Y + cr.H - 22
	barX := cr.X + 10
	barW := cr.W - 20
	barH := float32(10)

	ratio := 0.0
	if budget.Cap > 0 {
		ratio = float64(budget.Used) / float64(budget.Cap)
	}

	// 进度条颜色：正常=绿，接近上限=橙，超出=红
	fillClr := color.Color(theme.TonePrimary)
	if ratio > 1.0 {
		fillClr = theme.BtnDanger
	} else if ratio > 0.8 {
		fillClr = theme.ResGold
	}

	ui.ProgressBar(screen, barX, barY, barW, barH, ratio, ui.ProgressBarStyle{
		BgColor:   color.RGBA{R: 30, G: 35, B: 50, A: 200},
		FillColor: fillClr,
		Radius:    barH / 2,
	})

	// 预算文本
	budgetText := fmt.Sprintf("预算: %d / %d", budget.Used, budget.Cap)
	ui.Label(screen, budgetText, float64(barX), float64(barY)-14, float64(barW), ui.LabelStyle{
		Font: theme.FontCaption, Color: theme.TextMuted, Align: ui.AlignRight,
	})
}

// ── 底部按钮 ──────────────────────────────────────

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
		// 保存按钮：校验失败时变灰
		saveClr := color.Color(theme.TonePrimary)
		errs := descriptor.ValidateBlueprint(s.blueprint, *s.budgetRules, s.mergedAbilityCosts())
		if len(errs) > 0 {
			saveClr = theme.ToneDisabled
		}
		items = append(items, ui.ButtonRowItem{
			Label: "保存",
			Color: saveClr,
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
