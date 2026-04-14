// widget.go — 可复用 UI 组件。
// 基于 draw 包原语，提供高层级的卡片、按钮、面板、进度条等组件。
// 所有坐标均为逻辑像素（1200×540），HiDPI 缩放由 draw 包自动处理。
package ui

import (
	"image/color"
	"math"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Card — 圆角卡片（模式选择、战灵选择等）
// ---------------------------------------------------------------------------

// CardStyle 卡片样式。
// 自适应：宽高由调用者指定，内部元素自动居中。
// 限制：不自动换行，高亮条位置固定在底部 8px。
type CardStyle struct {
	BgColor      color.Color // 背景色
	BorderColor  color.Color // 边框色
	Radius       float32     // 圆角半径（0=直角）
	BorderWidth  float32     // 边框宽度
	Selected     bool        // 是否选中
	SelectedColor color.Color // 选中边框/高亮色
	HighlightBar bool        // 选中时是否在底部画高亮条
	BarWidth     float32     // 高亮条宽度
}

// DefaultCardStyle 默认卡片样式。
func DefaultCardStyle() CardStyle {
	return CardStyle{
		BgColor:      color.RGBA{R: 30, G: 38, B: 60, A: 255},
		BorderColor:  color.RGBA{R: 60, G: 70, B: 95, A: 255},
		Radius:       12,
		BorderWidth:  1.5,
		HighlightBar: true,
		BarWidth:     40,
	}
}

// Card 绘制一个圆角卡片。
func Card(screen *ebiten.Image, x, y, w, h float32, style CardStyle) {
	r := style.Radius

	// 背景
	draw.RoundRect(screen, x, y, w, h, r, style.BgColor)

	// 边框
	borderClr := style.BorderColor
	if style.Selected && style.SelectedColor != nil {
		borderClr = style.SelectedColor
	}
	if borderClr != nil {
		bw := style.BorderWidth
		if bw <= 0 {
			bw = 1.5
		}
		draw.StrokeRoundRect(screen, x, y, w, h, r, bw, borderClr)
	}

	// 选中底部高亮条
	if style.Selected && style.HighlightBar && style.SelectedColor != nil {
		barW := style.BarWidth
		if barW <= 0 {
			barW = 40
		}
		barH := float32(3)
		draw.RoundRect(screen, x+(w-barW)/2, y+h-8, barW, barH, barH/2, style.SelectedColor)
	}
}

// ---------------------------------------------------------------------------
// Button — 圆角按钮
// ---------------------------------------------------------------------------

// ButtonStyle 按钮样式。
// 自适应：文字自动垂直居中；Radius=0 时自动 Pill 形状（h/2）。
// 文字超出按钮宽度时自动缩小字号（最多缩 4px，下限 10px）。
type ButtonStyle struct {
	BgColor   color.Color // 背景色
	TextColor color.Color // 文字颜色（默认白色）
	FontSize  float64     // 文字大小
	Radius    float32     // 圆角半径（0=Pill 自动半圆角）
	Bold      bool        // 文字是否加粗
}

// Button 绘制一个圆角按钮 + 居中文字。
func Button(screen *ebiten.Image, x, y, w, h float32, label string, style ButtonStyle) {
	r := style.Radius
	if r <= 0 {
		r = h / 2 // Pill shape
	}
	draw.RoundRect(screen, x, y, w, h, r, style.BgColor)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	textClr := style.TextColor
	if textClr == nil {
		textClr = color.White
	}
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 14
	}
	// 自动缩小字号适应按钮宽度（最多缩 4px，下限 10px）
	avail := float64(w) - 12 // 6px padding each side
	minFS := fontSize - 4
	if minFS < 10 {
		minFS = 10
	}
	fontSize = ShrinkFontSize(fm, label, avail, fontSize, minFS)
	cx := float64(x) + float64(w)/2
	cy := float64(y) + float64(h)/2
	if style.Bold {
		fm.DrawCenteredVBoldText(screen, label, cx, cy, fontSize, textClr)
	} else {
		fm.DrawCenteredVText(screen, label, cx, cy, fontSize, textClr)
	}
}

// ---------------------------------------------------------------------------
// ButtonState — 按钮交互动画状态
// ---------------------------------------------------------------------------

// ButtonState tracks interactive animation state for a button.
type ButtonState struct {
	ScaleT     float64 // animation timer (1.0 = just pressed, decays to 0)
	ShakeT     float64 // disabled-tap shake timer
	Pressed    bool    // currently pressed
	wasPressed bool    // previous frame state (for detecting release)
}

// Update advances the button animation state.
// hovered: mouse is over the button. pressed: mouse button is down on the button.
func (bs *ButtonState) Update(dt float64, hovered, pressed bool) {
	// Detect press edge
	if pressed && !bs.wasPressed {
		bs.ScaleT = 1.0
	}
	bs.wasPressed = pressed
	bs.Pressed = pressed

	// Decay animation timers
	const decaySpeed = 8.0
	if bs.ScaleT > 0 {
		bs.ScaleT -= dt * decaySpeed
		if bs.ScaleT < 0 {
			bs.ScaleT = 0
		}
	}
	if bs.ShakeT > 0 {
		bs.ShakeT -= dt * decaySpeed
		if bs.ShakeT < 0 {
			bs.ShakeT = 0
		}
	}
}

// Trigger starts the press scale animation programmatically (e.g., item drop completed).
func (bs *ButtonState) Trigger() {
	bs.ScaleT = 1.0
}

// TriggerDisabledShake starts the shake animation for a disabled button tap.
func (bs *ButtonState) TriggerDisabledShake() {
	bs.ShakeT = 1.0
}

// Scale returns the current scale factor for the button.
func (bs *ButtonState) Scale() float64 {
	if bs.Pressed {
		return 1.0 - 0.07*bs.ScaleT // press-down: shrink
	}
	return 1.0 + 0.03*bs.ScaleT // release bounce: slight grow
}

// ShakeOffsetX returns horizontal shake offset for disabled buttons.
func (bs *ButtonState) ShakeOffsetX() float64 {
	if bs.ShakeT <= 0 {
		return 0
	}
	return math.Sin(bs.ShakeT*math.Pi*4) * 3.0 * bs.ShakeT // decaying oscillation
}

// ButtonWithState draws a button with interactive animation applied.
func ButtonWithState(screen *ebiten.Image, x, y, w, h float32, label string, style ButtonStyle, state *ButtonState) {
	if state == nil {
		Button(screen, x, y, w, h, label, style)
		return
	}

	scale := float32(state.Scale())
	shakeX := float32(state.ShakeOffsetX())

	// Apply scale transform (centered)
	scaledW := w * scale
	scaledH := h * scale
	scaledX := x + (w-scaledW)/2 + shakeX
	scaledY := y + (h-scaledH)/2

	// Darken color when pressed
	drawStyle := style
	if state.Pressed {
		drawStyle.BgColor = darkenColor(style.BgColor, 0.15)
	}

	Button(screen, scaledX, scaledY, scaledW, scaledH, label, drawStyle)
}

// darkenColor reduces the brightness of a color by the given factor.
func darkenColor(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()
	mult := 1.0 - factor
	return color.RGBA{
		R: uint8(float64(r>>8) * mult),
		G: uint8(float64(g>>8) * mult),
		B: uint8(float64(b>>8) * mult),
		A: uint8(a >> 8),
	}
}

// lightenColor increases the brightness of a color toward white by the given factor.
func lightenColor(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()
	r8, g8, b8 := float64(r>>8), float64(g>>8), float64(b>>8)
	return color.RGBA{
		R: uint8(r8 + (255-r8)*factor),
		G: uint8(g8 + (255-g8)*factor),
		B: uint8(b8 + (255-b8)*factor),
		A: uint8(a >> 8),
	}
}

// ---------------------------------------------------------------------------
// Panel — 圆角面板（半透明背景 + 可选边框）
// ---------------------------------------------------------------------------

// PanelStyle 面板样式。
// 自适应：宽高由调用者指定，默认圆角 14px。
// 限制：不管理内部内容布局，仅绘制背景+边框。
type PanelStyle struct {
	BgColor     color.Color // 背景色（通常半透明）
	BorderColor color.Color // 边框色（nil=无边框）
	Radius      float32     // 圆角半径
	BorderWidth float32     // 边框宽度
}

// Panel 绘制一个圆角面板。
func Panel(screen *ebiten.Image, x, y, w, h float32, style PanelStyle) {
	r := style.Radius
	if r <= 0 {
		r = 14
	}
	draw.RoundRect(screen, x, y, w, h, r, style.BgColor)
	if style.BorderColor != nil {
		bw := style.BorderWidth
		if bw <= 0 {
			bw = 1
		}
		draw.StrokeRoundRect(screen, x, y, w, h, r, bw, style.BorderColor)
	}
}

// ---------------------------------------------------------------------------
// ProgressBar — 圆角进度条（HP/XP/加载等）
// ---------------------------------------------------------------------------

// ProgressBarStyle 进度条样式。
// 自适应：宽度由调用者指定，填充宽度按 ratio 计算；Radius=0 时自动 Pill（h/2）。
// 限制：最小填充宽度 = 2*radius，ratio < 极小值时显示为空。
type ProgressBarStyle struct {
	BgColor   color.Color // 槽背景色
	FillColor color.Color // 填充色
	Radius    float32     // 圆角半径（0=自动 h/2）
}

// ProgressBar 绘制一个圆角进度条。ratio 为 0.0~1.0。
func ProgressBar(screen *ebiten.Image, x, y, w, h float32, ratio float64, style ProgressBarStyle) {
	r := style.Radius
	if r <= 0 {
		r = h / 2
	}
	// 背景槽
	draw.RoundRect(screen, x, y, w, h, r, style.BgColor)
	// 填充
	if ratio > 0 {
		if ratio > 1 {
			ratio = 1
		}
		fillW := w * float32(ratio)
		if fillW < r*2 {
			fillW = r * 2 // 最小宽度保证圆角可见
		}
		draw.RoundRect(screen, x, y, fillW, h, r, style.FillColor)
	}
}

// ---------------------------------------------------------------------------
// Badge — 小标签/徽章（Pill 形状）
// ---------------------------------------------------------------------------

// BadgeStyle 徽章样式。
// 自适应：宽度根据文字长度自动计算（MeasureText + padding）。
// 限制：单行文字，不支持换行；最大建议字符数 ~20。
type BadgeStyle struct {
	BgColor   color.Color
	TextColor color.Color
	FontSize  float64
}

// Badge 绘制一个 Pill 形状的小标签。自动计算宽度。
func Badge(screen *ebiten.Image, cx, cy float64, label string, style BadgeStyle) {
	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 10
	}
	tw := fm.MeasureText(label, fontSize)
	padX := float32(8)
	padY := float32(4)
	h := float32(fontSize) + padY*2
	w := float32(tw) + padX*2
	x := float32(cx) - w/2
	y := float32(cy) - h/2

	draw.RoundRect(screen, x, y, w, h, h/2, style.BgColor)

	textClr := style.TextColor
	if textClr == nil {
		textClr = color.White
	}
	fm.DrawCenteredText(screen, label, cx, float64(y)+float64(padY), fontSize, textClr)
}

// ---------------------------------------------------------------------------
// IconCard — 带图标的卡片（选关/战灵选择场景）
// ---------------------------------------------------------------------------

// IconCardStyle 图标卡片样式。
// 自适应：图标/名称/描述垂直排列自动居中，优先使用 PNG 图标回退到文字。
// 限制：固定垂直布局（icon@y+18, name@y+55, desc@y+78），卡片高度需 ≥ 100px。
type IconCardStyle struct {
	CardStyle                  // 嵌入卡片样式
	IconSize     float64       // 图标字号
	IconColor    color.Color   // 图标颜色
	NameSize     float64       // 名称字号
	NameColor    color.Color   // 名称颜色
	NameBold     bool          // 名称是否加粗
	DescSize     float64       // 描述字号
	DescColor    color.Color   // 描述颜色
}

// IconCard 绘制带图标、名称、描述的卡片。
func IconCard(screen *ebiten.Image, x, y, w, h float32, icon, name, desc string, style IconCardStyle) {
	// 卡片底板
	Card(screen, x, y, w, h, style.CardStyle)

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	cx := float64(x) + float64(w)/2

	// 图标：优先尝试 PNG 图标（icon 字符串匹配已注册的图标名），回退到文本渲染
	iconSize := style.IconSize
	if iconSize <= 0 {
		iconSize = 22
	}
	iconDrawn := false
	if im := render.GlobalIcons(); im != nil {
		if img := im.Get(icon); img != nil {
			draw.Sprite(screen, img, cx, float64(y)+18+iconSize/2, iconSize)
			iconDrawn = true
		}
	}
	if !iconDrawn {
		iconClr := style.IconColor
		if iconClr == nil {
			iconClr = color.White
		}
		fm.DrawCenteredText(screen, icon, cx, float64(y)+18, iconSize, iconClr)
	}

	// 名称（超宽时截断）
	nameSize := style.NameSize
	if nameSize <= 0 {
		nameSize = 14
	}
	nameClr := style.NameColor
	if nameClr == nil {
		nameClr = color.White
	}
	nameMaxW := float64(w) - 16 // 8px padding each side
	displayName := TruncateText(fm, name, nameMaxW, nameSize)
	if style.NameBold {
		fm.DrawCenteredBoldText(screen, displayName, cx, float64(y)+55, nameSize, nameClr)
	} else {
		fm.DrawCenteredText(screen, displayName, cx, float64(y)+55, nameSize, nameClr)
	}

	// 描述（超宽时截断）
	if desc != "" {
		descSize := style.DescSize
		if descSize <= 0 {
			descSize = 10
		}
		descClr := style.DescColor
		if descClr == nil {
			descClr = color.RGBA{R: 140, G: 145, B: 160, A: 255}
		}
		displayDesc := TruncateText(fm, desc, nameMaxW, descSize)
		fm.DrawCenteredText(screen, displayDesc, cx, float64(y)+78, descSize, descClr)
	}
}

// ---------------------------------------------------------------------------
// Overlay — 全屏半透明遮罩
// ---------------------------------------------------------------------------

// Overlay 绘制全屏半透明遮罩。用于 pause/choice/spawn/warden_select 等场景。
func Overlay(screen *ebiten.Image, alpha uint8) {
	sw := float32(theme.CanvasW)
	sh := float32(theme.CanvasH)
	draw.FilledRect(screen, 0, 0, sw, sh, color.RGBA{A: alpha}, false)
}

// ---------------------------------------------------------------------------
// Divider — 水平分隔线
// ---------------------------------------------------------------------------

// Divider 在 (x,y) 处绘制一条宽 w 的水平分隔线。
func Divider(screen *ebiten.Image, x, y, w float32, clr color.Color) {
	if clr == nil {
		clr = color.RGBA{R: 60, G: 70, B: 90, A: 180}
	}
	draw.Line(screen, x, y, x+w, y, 1, clr, false)
}
