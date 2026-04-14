// panel_box.go — 面板容器组件。
// 提供背景+边框+标题栏+关闭提示的完整面板框架。
// 返回内容区 Rect，调用方在内容区内用其他组件排列内容。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// PanelBoxStyle 面板容器样式。
type PanelBoxStyle struct {
	W, H      float32     // 面板宽高
	Radius    float32     // 圆角，0 → theme.PanelRadius
	BgColor   color.Color // 背景色，nil → 默认深蓝
	Border    color.Color // 边框色，nil → 无边框
	Title     string      // 标题文本，空=不渲染标题
	TitleFont float64     // 标题字号，0 → theme.FontLG
	CloseHint string      // 右上角提示（如 "ESC"），空=不显示
	Pad       float32     // 内边距，0 → theme.PanelInnerPad
}

// PanelBox 绘制面板容器，返回内容区 Rect（标题下方区域）。
func PanelBox(screen *ebiten.Image, x, y float32, style PanelBoxStyle) Rect {
	fm := render.GlobalFont()

	r := style.Radius
	if r <= 0 {
		r = theme.PanelRadius
	}
	bg := style.BgColor
	if bg == nil {
		bg = color.RGBA{R: 18, G: 24, B: 42, A: 245}
	}
	pad := style.Pad
	if pad <= 0 {
		pad = theme.PanelInnerPad
	}

	// 背景 + 边框
	draw.RoundRect(screen, x, y, style.W, style.H, r, bg)
	if style.Border != nil {
		draw.StrokeRoundRect(screen, x, y, style.W, style.H, r, 1, style.Border)
	}

	titleH := float32(0)

	// 标题
	if style.Title != "" && fm != nil {
		titleFont := style.TitleFont
		if titleFont <= 0 {
			titleFont = theme.FontLG
		}
		titleMaxW := float64(style.W) - float64(pad)*2
		Label(screen, style.Title, float64(x)+float64(pad), float64(y)+float64(pad), titleMaxW, LabelStyle{
			Font: titleFont, Bold: true,
		})
		titleH = float32(titleFont) + pad
	}

	// 关闭提示（右上角）
	if style.CloseHint != "" && fm != nil {
		hintMaxW := float64(style.W) - float64(pad)*2
		Label(screen, style.CloseHint, float64(x)+float64(pad), float64(y)+float64(pad), hintMaxW, LabelStyle{
			Font:  theme.FontCaption,
			Color: color.RGBA{R: 120, G: 140, B: 170, A: 200},
			Align: AlignRight,
		})
	}

	// 返回内容区
	return Rect{
		X: x + pad,
		Y: y + pad + titleH,
		W: style.W - pad*2,
		H: style.H - pad*2 - titleH,
	}
}
