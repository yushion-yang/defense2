// layout.go — 自适应布局组件。
// 提供 Rect、ButtonRow、StatLine、FlexPanel、Anchor 等基础组件，
// 方便后续 HUD 调整。所有坐标为逻辑像素。
package ui

import (
	"image/color"

	"defense2/internal/core/game"
	"defense2/internal/render"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Rect — 通用矩形（布局 + 碰撞检测）
// ---------------------------------------------------------------------------

// Rect 描述一个逻辑像素矩形。
// 自适应：无限制，纯数据结构。
type Rect struct {
	X, Y, W, H float32
}

// Contains 检查点是否在矩形内。
func (r Rect) Contains(px, py float64) bool {
	return float32(px) >= r.X && float32(px) <= r.X+r.W &&
		float32(py) >= r.Y && float32(py) <= r.Y+r.H
}

// CenterX 返回矩形水平中心。
func (r Rect) CenterX() float64 { return float64(r.X) + float64(r.W)/2 }

// CenterY 返回矩形垂直中心。
func (r Rect) CenterY() float64 { return float64(r.Y) + float64(r.H)/2 }

// ---------------------------------------------------------------------------
// Anchor — 锚点定位
// ---------------------------------------------------------------------------

// Anchor 锚点位置。
type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorCenterLeft
	AnchorCenter
	AnchorCenterRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
)

// AnchoredRect 根据锚点和边距计算矩形位置。
// 自适应：自动适配 ScreenWidth/ScreenHeight，不超出屏幕边界。
// 限制：不处理内容溢出（w/h 超过屏幕时不裁剪）。
// margin 分别为 上/右/下/左（类似 CSS），未使用的方向传 0。
func AnchoredRect(anchor Anchor, w, h float32, marginTop, marginRight, marginBottom, marginLeft float32) Rect {
	sw := float32(game.ScreenWidth)
	sh := float32(game.ScreenHeight)

	var x, y float32
	switch anchor {
	case AnchorTopLeft:
		x, y = marginLeft, marginTop
	case AnchorTopCenter:
		x, y = (sw-w)/2, marginTop
	case AnchorTopRight:
		x, y = sw-w-marginRight, marginTop
	case AnchorCenterLeft:
		x, y = marginLeft, (sh-h)/2
	case AnchorCenter:
		x, y = (sw-w)/2, (sh-h)/2
	case AnchorCenterRight:
		x, y = sw-w-marginRight, (sh-h)/2
	case AnchorBottomLeft:
		x, y = marginLeft, sh-h-marginBottom
	case AnchorBottomCenter:
		x, y = (sw-w)/2, sh-h-marginBottom
	case AnchorBottomRight:
		x, y = sw-w-marginRight, sh-h-marginBottom
	}
	return Rect{X: x, Y: y, W: w, H: h}
}

// ---------------------------------------------------------------------------
// ButtonRow — 一排等分按钮
// ---------------------------------------------------------------------------

// ButtonRowItem 按钮行中的一个按钮。
type ButtonRowItem struct {
	Label string
	Color color.Color // 背景色
	Bold  bool
}

// ButtonRowStyle 按钮行样式。
type ButtonRowStyle struct {
	Height   float32 // 按钮高度
	Gap      float32 // 按钮间距
	Radius   float32 // 圆角
	FontSize float64 // 字号
}

// ButtonRowResult 按钮行的布局结果。
type ButtonRowResult struct {
	Rects []Rect // 每个按钮的矩形
}

// DrawButtonRow 在指定矩形内绘制一排等分按钮，返回每个按钮的矩形（用于 hit test）。
// 自适应：按钮宽度自动等分 area.W，数量不限。
// 限制：当按钮过多导致单个宽度 < 40px 时文字会截断；最大建议 6 个按钮。
func DrawButtonRow(screen *ebiten.Image, area Rect, items []ButtonRowItem, style ButtonRowStyle) ButtonRowResult {
	if len(items) == 0 {
		return ButtonRowResult{}
	}
	fm := render.GlobalFont()

	n := float32(len(items))
	gap := style.Gap
	if gap <= 0 {
		gap = 8
	}
	btnW := (area.W - gap*(n-1)) / n
	btnH := style.Height
	if btnH <= 0 {
		btnH = 30
	}
	btnR := style.Radius
	if btnR <= 0 {
		btnR = 10
	}
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}

	rects := make([]Rect, len(items))
	for i, item := range items {
		bx := area.X + float32(i)*(btnW+gap)
		by := area.Y

		rects[i] = Rect{X: bx, Y: by, W: btnW, H: btnH}

		bgClr := item.Color
		if bgClr == nil {
			bgClr = color.RGBA{R: 60, G: 70, B: 95, A: 255}
		}
		draw.RoundRect(screen, bx, by, btnW, btnH, btnR, bgClr)

		if fm != nil {
			cx := float64(bx) + float64(btnW)/2
			cy := float64(by) + float64(btnH)/2
			if item.Bold {
				fm.DrawCenteredVBoldText(screen, item.Label, cx, cy, fontSize, color.White)
			} else {
				fm.DrawCenteredVText(screen, item.Label, cx, cy, fontSize, color.White)
			}
		}
	}
	return ButtonRowResult{Rects: rects}
}

// HitTestButtonRow 检查点 (px,py) 命中了哪个按钮，返回索引或 -1。
func HitTestButtonRow(rects []Rect, px, py float64) int {
	for i, r := range rects {
		if r.Contains(px, py) {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// StatLine — 键值对显示行
// ---------------------------------------------------------------------------

// StatLineItem 一个属性项。
type StatLineItem struct {
	Label string
	Value string
	Color color.Color // 值颜色（nil=默认白色）
}

// DrawStatLine 在指定 Y 坐标绘制一行键值对，等分列宽。
// 自适应：列宽自动等分 totalW。
// 限制：不换行，label+value 超出列宽时会重叠；最大建议 4 列。
func DrawStatLine(screen *ebiten.Image, x, y float64, totalW float64, items []StatLineItem, labelSize, valueSize float64) {
	fm := render.GlobalFont()
	if fm == nil || len(items) == 0 {
		return
	}
	colW := totalW / float64(len(items))
	for i, item := range items {
		cx := x + float64(i)*colW
		fm.DrawText(screen, item.Label, cx, y, labelSize, color.RGBA{R: 140, G: 145, B: 160, A: 255})
		valClr := item.Color
		if valClr == nil {
			valClr = color.White
		}
		fm.DrawText(screen, item.Value, cx+fm.MeasureText(item.Label, labelSize)+4, y, valueSize, valClr)
	}
}

// ---------------------------------------------------------------------------
// FlexPanel — 内容自适应面板
// ---------------------------------------------------------------------------

// FlexPanel 内容驱动高度的面板。调用 Add* 方法添加行，最后 Draw 渲染。
// 自适应：高度由内容行累加决定，宽度固定。
// 限制：不支持滚动（内容超出屏幕高度时会溢出）；宽度固定不自适应。
// 最大建议高度：ScreenHeight * 0.8（~432px）。
type FlexPanel struct {
	X, Y    float32     // 左上角位置
	W       float32     // 固定宽度
	Pad     float32     // 内边距
	Radius  float32     // 圆角
	BgColor color.Color // 背景色
	Border  color.Color // 边框色（nil=无边框）

	rows    []flexRow // 内容行
	cursorY float32   // 当前 Y 偏移（相对面板顶部）
}

type flexRow struct {
	drawFn func(screen *ebiten.Image, x, y float64, w float64) // 渲染函数
	height float32
}

// NewFlexPanel 创建自适应面板。
func NewFlexPanel(x, y, w, pad float32) *FlexPanel {
	return &FlexPanel{
		X: x, Y: y, W: w, Pad: pad,
		Radius:  14,
		BgColor: color.RGBA{R: 15, G: 23, B: 42, A: 224},
		cursorY: pad,
	}
}

// AddSpace 添加空白间距。
func (p *FlexPanel) AddSpace(h float32) {
	p.cursorY += h
}

// AddRow 添加自定义渲染行。
func (p *FlexPanel) AddRow(height float32, fn func(screen *ebiten.Image, x, y float64, w float64)) {
	p.rows = append(p.rows, flexRow{drawFn: fn, height: height})
	p.cursorY += height
}

// Height 返回面板总高度。
func (p *FlexPanel) Height() float32 {
	return p.cursorY + p.Pad
}

// ContentWidth 返回内容区宽度（面板宽度减去两侧内边距）。
func (p *FlexPanel) ContentWidth() float32 {
	return p.W - p.Pad*2
}

// Draw 渲染面板及其所有内容行。
func (p *FlexPanel) Draw(screen *ebiten.Image) {
	h := p.Height()

	// 面板背景
	draw.RoundRect(screen, p.X, p.Y, p.W, h, p.Radius, p.BgColor)
	if p.Border != nil {
		draw.StrokeRoundRect(screen, p.X, p.Y, p.W, h, p.Radius, 1, p.Border)
	}

	// 渲染每一行
	cy := p.Pad
	ix := float64(p.X) + float64(p.Pad)
	cw := float64(p.ContentWidth())
	for _, row := range p.rows {
		row.drawFn(screen, ix, float64(p.Y)+float64(cy), cw)
		cy += row.height
	}
}

// Rect 返回面板的矩形（用于 hit test）。
func (p *FlexPanel) Rect() Rect {
	return Rect{X: p.X, Y: p.Y, W: p.W, H: p.Height()}
}
