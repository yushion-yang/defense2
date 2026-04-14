// paragraph.go — 安全多行换行文本组件。
// 封装 WrapText + 逐行渲染，hud/ 包禁止手动循环 WrapText。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ParagraphStyle 多行文本样式。
type ParagraphStyle struct {
	Font    float64     // 字号，0 → theme.FontBody
	Color   color.Color // 颜色，nil → white
	Bold    bool        // 是否加粗
	LineGap float64     // 行间距，0 → WrapLineHeight(3)
}

// Paragraph 在 maxW 内自动换行绘制文本，返回实际行数。
func Paragraph(screen *ebiten.Image, text string, x, y, maxW float64, style ParagraphStyle) int {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return 0
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}
	lineGap := style.LineGap
	if lineGap <= 0 {
		lineGap = WrapLineHeight
	}

	lines := WrapText(fm, text, maxW, fontSize)
	lineH := fontSize + lineGap
	for i, line := range lines {
		ly := y + float64(i)*lineH
		if style.Bold {
			fm.DrawBoldText(screen, line, x, ly, fontSize, clr)
		} else {
			fm.DrawText(screen, line, x, ly, fontSize, clr)
		}
	}
	return len(lines)
}

// ParagraphHeight 计算多行文本的高度（不渲染）。
func ParagraphHeight(text string, maxW float64, fontSize float64, lineGap float64) float64 {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return 0
	}
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	if lineGap <= 0 {
		lineGap = WrapLineHeight
	}
	lines := WrapText(fm, text, maxW, fontSize)
	if len(lines) == 0 {
		return 0
	}
	return float64(len(lines))*fontSize + float64(len(lines)-1)*lineGap
}
