// label.go — 安全单行文本组件。
// 所有文本渲染必须通过 Label，内置 ShrinkFontSize 溢出保护。
// 文本不截断、不溢出：超宽时自动缩小字号适配容器。
// hud/ 包禁止直接调用 fm.DrawText，改用此组件。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TextAlign 文本水平对齐方式。
type TextAlign int

const (
	AlignLeft   TextAlign = iota // 左对齐（默认）
	AlignCenter                  // 居中
	AlignRight                   // 右对齐
)

// LabelStyle 单行文本样式。零值字段使用合理默认值。
type LabelStyle struct {
	Font  float64     // 字号，0 → theme.FontBody
	Color color.Color // 颜色，nil → white
	Bold  bool        // 是否加粗
	Align TextAlign   // 对齐方式，默认左对齐
}

// labelMinFont 缩放下限：不低于 9px，保证可读性。
const labelMinFont = 9

// Label 在 maxW 宽度内绘制安全单行文本。
// 超宽时自动缩小字号（最多缩 4px，下限 9px），保留完整文本。
// maxW <= 0 时不做溢出保护。
func Label(screen *ebiten.Image, text string, x, y, maxW float64, style LabelStyle) {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}

	// 溢出保护：缩小字号适配容器，不截断
	if maxW > 0 {
		minFS := fontSize - 4
		if minFS < labelMinFont {
			minFS = labelMinFont
		}
		fontSize = ShrinkFontSize(fm, text, maxW, fontSize, minFS)
	}

	switch style.Align {
	case AlignCenter:
		cx := x + maxW/2
		if style.Bold {
			fm.DrawCenteredBoldText(screen, text, cx, y, fontSize, clr)
		} else {
			fm.DrawCenteredText(screen, text, cx, y, fontSize, clr)
		}
	case AlignRight:
		rx := x + maxW
		if style.Bold {
			fm.DrawRightBoldText(screen, text, rx, y, fontSize, clr)
		} else {
			fm.DrawRightText(screen, text, rx, y, fontSize, clr)
		}
	default: // AlignLeft
		if style.Bold {
			fm.DrawBoldText(screen, text, x, y, fontSize, clr)
		} else {
			fm.DrawText(screen, text, x, y, fontSize, clr)
		}
	}
}

// LabelV 在矩形中心垂直居中绘制安全单行文本（常用于按钮内文字）。
// cx, cy 为中心点坐标。超宽时自动缩小字号。
func LabelV(screen *ebiten.Image, text string, cx, cy, maxW float64, style LabelStyle) {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}

	// 溢出保护
	if maxW > 0 {
		minFS := fontSize - 4
		if minFS < labelMinFont {
			minFS = labelMinFont
		}
		fontSize = ShrinkFontSize(fm, text, maxW, fontSize, minFS)
	}

	if style.Bold {
		fm.DrawCenteredVBoldText(screen, text, cx, cy, fontSize, clr)
	} else {
		fm.DrawCenteredVText(screen, text, cx, cy, fontSize, clr)
	}
}
