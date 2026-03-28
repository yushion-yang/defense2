// choice_panel.go — 通用选择面板（3 选 1 等）。
// 居中覆盖层，显示 N 个选项卡片，支持品质配色和鼠标悬停。
package hud

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ChoiceOption 选择项。
type ChoiceOption struct {
	Label       string      // 选项标签
	Description string      // 选项描述
	Tier        string      // 品质（normal/rare/epic）
	Data        interface{} // 携带数据（调用方自行断言）
}

// ChoicePanel 通用选择面板（3 选 1 等）。
type ChoicePanel struct {
	Title    string                              // 面板标题
	Options  []ChoiceOption                      // 选项列表
	OnSelect func(idx int, opt ChoiceOption)     // 选择回调
	Active   bool                                // 是否激活
	hovered  int                                 // 当前悬停索引(-1=无)
}

// 品质颜色映射。
var tierColors = map[string]color.RGBA{
	"normal": {R: 74, G: 222, B: 128, A: 255},  // #4ade80
	"rare":   {R: 96, G: 165, B: 250, A: 255},  // #60a5fa
	"epic":   {R: 192, G: 132, B: 252, A: 255}, // #c084fc
}

// 选项卡片布局常量。
const (
	cpCardW   = float32(180) // 卡片宽度
	cpCardH   = float32(140) // 卡片高度
	cpCardGap = float32(16)  // 卡片间距
	cpCardR   = float32(10)  // 卡片圆角
)

// NewChoicePanel 创建通用选择面板。
func NewChoicePanel() *ChoicePanel {
	return &ChoicePanel{
		hovered: -1,
	}
}

// Show 显示选择面板。
func (p *ChoicePanel) Show(title string, options []ChoiceOption, onSelect func(int, ChoiceOption)) {
	p.Title = title
	p.Options = options
	p.OnSelect = onSelect
	p.Active = true
	p.hovered = -1
}

// IsActive 返回面板是否激活。
func (p *ChoicePanel) IsActive() bool {
	return p.Active
}

// Update 每帧更新：鼠标悬停检测和点击选择。
func (p *ChoicePanel) Update(mx, my float64, clicked bool) {
	if !p.Active || len(p.Options) == 0 {
		return
	}

	fmx, fmy := float32(mx), float32(my)

	// 悬停检测
	p.hovered = -1
	n := len(p.Options)
	for i := 0; i < n; i++ {
		cx, cy := choiceCardPos(n, i)
		if fmx >= cx && fmx <= cx+cpCardW && fmy >= cy && fmy <= cy+cpCardH {
			p.hovered = i
			break
		}
	}

	// 点击选择
	if clicked && p.hovered >= 0 && p.OnSelect != nil {
		opt := p.Options[p.hovered]
		p.OnSelect(p.hovered, opt)
		p.Close()
	}
}

// Draw 绘制选择面板（半透明遮罩 + 标题 + 选项卡片）。
func (p *ChoicePanel) Draw(screen *ebiten.Image) {
	if !p.Active || len(p.Options) == 0 {
		return
	}

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	n := len(p.Options)

	// 半透明遮罩
	draw.RoundRect(screen, 0, 0, float32(theme.CanvasW), float32(theme.CanvasH), 0, theme.HUDGameOverlay)

	// 标题
	_, cardY := choiceCardPos(n, 0)
	titleY := float64(cardY) - 40
	fm.DrawCenteredBoldText(screen, p.Title,
		float64(theme.CanvasW)/2, titleY, theme.FontXL, theme.TextTitle)

	// 选项卡片
	for i, opt := range p.Options {
		cx, cy := choiceCardPos(n, i)
		hovered := i == p.hovered

		// 卡片背景
		cardBg := theme.PanelBg
		if hovered {
			cardBg = theme.TonePrimary
		}
		draw.RoundRect(screen, cx, cy, cpCardW, cpCardH, cpCardR, cardBg)

		// 品质描边
		tierClr := tierColor(opt.Tier)
		borderW := float32(1.5)
		if hovered {
			borderW = 2.5
		}
		draw.StrokeRoundRect(screen, cx, cy, cpCardW, cpCardH, cpCardR, borderW, tierClr)

		// 品质标签（顶部）
		tierLabelY := float64(cy) + 12
		tierLabelClr := tierClr
		fm.DrawCenteredText(screen, opt.Tier,
			float64(cx)+float64(cpCardW)/2, tierLabelY, theme.FontXS, tierLabelClr)

		// 标签（卡片中部偏上）
		labelY := float64(cy) + 40
		fm.DrawCenteredBoldText(screen, opt.Label,
			float64(cx)+float64(cpCardW)/2, labelY, theme.FontLG, theme.TextTitle)

		// 描述（卡片下部）
		descY := float64(cy) + 72
		fm.DrawCenteredText(screen, opt.Description,
			float64(cx)+float64(cpCardW)/2, descY, theme.FontSM, theme.TextBody)
	}
}

// Close 关闭选择面板。
func (p *ChoicePanel) Close() {
	p.Active = false
	p.Options = nil
	p.OnSelect = nil
	p.hovered = -1
}

// choiceCardPos 计算第 i 张选项卡片的位置（居中布局）。
func choiceCardPos(n, i int) (x, y float32) {
	totalW := float32(n)*cpCardW + float32(n-1)*cpCardGap
	startX := (float32(theme.CanvasW) - totalW) / 2
	x = startX + float32(i)*(cpCardW+cpCardGap)
	y = (float32(theme.CanvasH) - cpCardH) / 2
	return x, y
}

// tierColor 返回品质对应的颜色，默认为 normal。
func tierColor(tier string) color.RGBA {
	if clr, ok := tierColors[tier]; ok {
		return clr
	}
	return tierColors["normal"]
}
