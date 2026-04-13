// choice_panel.go — 通用选择面板（N 选 1）。
// 居中覆盖层，显示 N 个选项卡片，支持品质配色和鼠标悬停。
// 多选项（>5）时自动切换为双行网格布局+缩小卡片。
package hud

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	"defense2/internal/i18n"
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ChoiceOption 选择项。
type ChoiceOption struct {
	Label       string           // 选项标签
	Description string           // 选项描述（纯文本 fallback）
	Segments    []AbilitySegment // 带颜色的分段描述（优先于 Description）
	Tier        string           // 品质（normal/rare/epic）
	Icon        string           // 图标 key（对应 assets/icons/abilities/{icon}.png）
	Data        any      // 携带数据（调用方自行断言）
}

// IconReader 图标资源读取接口。
type IconReader interface {
	ReadFile(name string) ([]byte, error)
}

// ChoicePanel 通用选择面板（N 选 1）。
type ChoicePanel struct {
	Title       string                          // 面板标题
	Options     []ChoiceOption                  // 选项列表
	OnSelect    func(idx int, opt ChoiceOption) // 选择回调
	Active      bool                            // 是否激活
	Dismissible bool                            // 点击外部是否可关闭（默认 false）
	hovered     int                             // 当前悬停索引(-1=无)
	assetFS     IconReader                      // 资源文件系统（用于加载图标）
	iconCache   map[string]*ebiten.Image        // 图标缓存
}

// 品质颜色映射。
var tierColors = map[string]color.RGBA{
	"normal": {R: 74, G: 222, B: 128, A: 255},  // #4ade80
	"rare":   {R: 96, G: 165, B: 250, A: 255},  // #60a5fa
	"epic":   {R: 192, G: 132, B: 252, A: 255}, // #c084fc
}

// 品质中文名映射。
// tierLabelKeys maps tier names to i18n keys.
var tierLabelKeys = map[string]string{
	"normal": "hud.tier.normal",
	"rare":   "hud.tier.rare",
	"epic":   "hud.tier.epic",
}

// cardLayout 描述卡片布局参数（根据选项数量动态计算）。
type cardLayout struct {
	cardW, cardH float32 // 卡片尺寸
	gap          float32 // 卡片间距
	radius       float32 // 圆角半径
	pad          float32 // 内边距
	cols         int     // 每行列数
	rows         int     // 行数
	labelSize    float64 // 标签字号
	descSize     float64 // 描述字号
	tierSize     float64 // 品质标签字号
}

// cpMaxSingleRow 单行布局最大选项数。
const cpMaxSingleRow = 5

// calcLayout 根据选项数量计算布局参数。
func calcLayout(n int) cardLayout {
	if n <= cpMaxSingleRow {
		return cardLayout{
			cardW: 220, cardH: 200, gap: 16, radius: 10, pad: 10,
			cols: n, rows: 1,
			labelSize: theme.FontLG, descSize: theme.FontXS, tierSize: theme.FontXS,
		}
	}
	// 双行网格布局
	cols := (n + 1) / 2 // 上取整
	return cardLayout{
		cardW: 155, cardH: 150, gap: 10, radius: 8, pad: 8,
		cols: cols, rows: 2,
		labelSize: theme.FontSM, descSize: 9, tierSize: 9,
	}
}

// cardPos 计算第 i 张卡片的位置。
func (l *cardLayout) cardPos(n, i int) (x, y float32) {
	row := i / l.cols
	col := i % l.cols

	// 该行实际有多少个卡片
	rowCount := l.cols
	if row == l.rows-1 && n%l.cols != 0 {
		rowCount = n % l.cols
	}

	totalW := float32(rowCount)*l.cardW + float32(rowCount-1)*l.gap
	startX := (float32(theme.CanvasW) - totalW) / 2
	x = startX + float32(col)*(l.cardW+l.gap)

	totalH := float32(l.rows)*l.cardH + float32(l.rows-1)*l.gap
	startY := (float32(theme.CanvasH) - totalH) / 2
	y = startY + float32(row)*(l.cardH+l.gap)
	return x, y
}

// NewChoicePanel 创建通用选择面板。
func NewChoicePanel() *ChoicePanel {
	return &ChoicePanel{
		hovered:   -1,
		iconCache: make(map[string]*ebiten.Image),
	}
}

// SetAssetFS 设置资源文件系统（用于加载能力图标）。
func (p *ChoicePanel) SetAssetFS(fs IconReader) {
	p.assetFS = fs
}

// loadIcon 加载并缓存图标。
func (p *ChoicePanel) loadIcon(key string) *ebiten.Image {
	if key == "" {
		return nil
	}
	if img, ok := p.iconCache[key]; ok {
		return img
	}
	if p.assetFS == nil {
		return nil
	}
	path := fmt.Sprintf("assets/icons/abilities/%s.png", key)
	data, err := p.assetFS.ReadFile(path)
	if err != nil {
		p.iconCache[key] = nil
		return nil
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		p.iconCache[key] = nil
		return nil
	}
	img := ebiten.NewImageFromImage(decoded)
	p.iconCache[key] = img
	return img
}

// Show 显示选择面板。
func (p *ChoicePanel) Show(title string, options []ChoiceOption, onSelect func(int, ChoiceOption)) {
	p.Title = title
	p.Options = options
	p.OnSelect = onSelect
	p.Active = true
	p.Dismissible = true
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
	n := len(p.Options)
	lay := calcLayout(n)

	// 悬停检测
	p.hovered = -1
	for i := 0; i < n; i++ {
		cx, cy := lay.cardPos(n, i)
		if fmx >= cx && fmx <= cx+lay.cardW && fmy >= cy && fmy <= cy+lay.cardH {
			p.hovered = i
			break
		}
	}

	// 点击选择（命中卡片）或点击外部取消
	if clicked {
		if p.hovered >= 0 && p.OnSelect != nil {
			opt := p.Options[p.hovered]
			prevTitle := p.Title
			p.OnSelect(p.hovered, opt)
			// OnSelect 回调可能重新 Show() 了面板（如二级选择），
			// 通过 Title 变化判断：Title 变了说明回调内调了 Show()，保持打开。
			if p.Active && p.Title == prevTitle {
				p.Close()
			}
			return
		}
		// Only dismiss on click-outside if Dismissible is true
		if p.Dismissible {
			p.Close()
		}
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
	lay := calcLayout(n)

	// 半透明遮罩
	draw.RoundRect(screen, 0, 0, float32(theme.CanvasW), float32(theme.CanvasH), 0, theme.HUDGameOverlay)

	// 标题
	_, cardY := lay.cardPos(n, 0)
	titleY := float64(cardY) - 32
	fm.DrawCenteredBoldText(screen, p.Title,
		float64(theme.CanvasW)/2, titleY, theme.FontXL, theme.TextTitle)

	// 选项卡片
	for i, opt := range p.Options {
		cx, cy := lay.cardPos(n, i)
		hovered := i == p.hovered

		// 卡片背景
		cardBg := theme.PanelBg
		if hovered {
			cardBg = theme.TonePrimary
		}
		draw.RoundRect(screen, cx, cy, lay.cardW, lay.cardH, lay.radius, cardBg)

		// 品质描边
		tierClr := tierColor(opt.Tier)
		borderW := float32(1.5)
		if hovered {
			borderW = 2.5
		}
		draw.StrokeRoundRect(screen, cx, cy, lay.cardW, lay.cardH, lay.radius, borderW, tierClr)

		centerX := float64(cx) + float64(lay.cardW)/2

		// 图标（卡片顶部居中）
		iconSize := float32(28)
		if lay.rows > 1 {
			iconSize = 22
		}
		iconY := float64(cy) + 8
		if icon := p.loadIcon(opt.Icon); icon != nil {
			draw.Sprite(screen, icon, centerX, iconY+float64(iconSize)/2, float64(iconSize))
		}

		// 品质标签（图标下方）
		tierLabelY := iconY + float64(iconSize) + 4
		tierText := opt.Tier
		if key, ok := tierLabelKeys[opt.Tier]; ok {
			tierText = i18n.T(key)
		}
		fm.DrawCenteredText(screen, tierText, centerX, tierLabelY, lay.tierSize, tierClr)

		// 标签
		labelY := tierLabelY + 14
		fm.DrawCenteredBoldText(screen, opt.Label, centerX, labelY, lay.labelSize, theme.TextTitle)

		// 描述（卡片下部）
		descY := labelY + 18
		if len(opt.Segments) > 0 {
			// 带颜色的分段描述（与炮塔 HUD 一致）
			ui.DrawSegmentsWrapped(screen, fm, abilitySegsToTextSegs(opt.Segments),
				float64(cx)+float64(lay.pad), descY, float64(lay.cardW-lay.pad*2), lay.descSize)
		} else {
			maxW := float64(lay.cardW - lay.pad*2)
			ui.DrawWrappedText(screen, fm, opt.Description,
				float64(cx)+float64(lay.pad), descY, maxW, lay.descSize, theme.TextBody)
		}
	}
}

// Close 关闭选择面板。
func (p *ChoicePanel) Close() {
	p.Active = false
	p.Options = nil
	p.OnSelect = nil
	p.hovered = -1
}

// tierColor 返回品质对应的颜色，默认为 normal。
func tierColor(tier string) color.RGBA {
	if clr, ok := tierColors[tier]; ok {
		return clr
	}
	return tierColors["normal"]
}

// abilitySegsToTextSegs converts AbilitySegment slice to ui.TextSegment slice,
// mapping Kind to the appropriate theme color.
func abilitySegsToTextSegs(segs []AbilitySegment) []ui.TextSegment {
	out := make([]ui.TextSegment, len(segs))
	for i, seg := range segs {
		var clr color.Color
		switch seg.Kind {
		case "text":
			clr = theme.TextMuted
		case "base", "total":
			clr = theme.TextBody
		case "scaled":
			clr = seg.Color
			if clr == nil {
				clr = theme.TextBody
			}
		default:
			clr = theme.TextBody
		}
		out[i] = ui.TextSegment{Text: seg.Text, Color: clr}
	}
	return out
}
