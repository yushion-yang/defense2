// label.go — 安全单行文本组件。
// 所有文本渲染必须通过 Label，内置 TruncateText 溢出保护。
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

// Label 在 maxW 宽度内绘制安全单行文本。
// 超宽时自动截断并添加 "..."。maxW <= 0 时不截断（仅用于已知安全的短文本）。
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

	// 溢出保护
	display := text
	if maxW > 0 {
		display = TruncateText(fm, text, maxW, fontSize)
	}

	switch style.Align {
	case AlignCenter:
		cx := x + maxW/2
		if style.Bold {
			fm.DrawCenteredBoldText(screen, display, cx, y, fontSize, clr)
		} else {
			fm.DrawCenteredText(screen, display, cx, y, fontSize, clr)
		}
	case AlignRight:
		rx := x + maxW
		if style.Bold {
			fm.DrawRightBoldText(screen, display, rx, y, fontSize, clr)
		} else {
			fm.DrawRightText(screen, display, rx, y, fontSize, clr)
		}
	default: // AlignLeft
		if style.Bold {
			fm.DrawBoldText(screen, display, x, y, fontSize, clr)
		} else {
			fm.DrawText(screen, display, x, y, fontSize, clr)
		}
	}
}

// LabelV 在矩形中心垂直居中绘制安全单行文本（常用于按钮内文字）。
// cx, cy 为中心点坐标。
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

	display := text
	if maxW > 0 {
		display = TruncateText(fm, text, maxW, fontSize)
	}

	if style.Bold {
		fm.DrawCenteredVBoldText(screen, display, cx, cy, fontSize, clr)
	} else {
		fm.DrawCenteredVText(screen, display, cx, cy, fontSize, clr)
	}
}
