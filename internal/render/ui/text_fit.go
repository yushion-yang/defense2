// text_fit.go — 自适应文本工具集。
// 提供截断(TruncateText)、字号缩小(ShrinkFontSize)、换行(WrapText) 三种策略，
// 确保 HUD 文本不溢出容器，支持中英双语 i18n。
package ui

import (
	"image/color"

	"defense2/internal/render"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 截断 ─────────────────────────────────────────

// TruncateText 返回不超过 maxW 的文本，超宽时在尾部添加 "..."。
// 快速路径：文本不超宽时直接返回原字符串（零分配）。
func TruncateText(fm *render.FontManager, text string, maxW float64, fontSize float64) string {
	if fm == nil || text == "" || maxW <= 0 {
		return text
	}
	if fm.MeasureText(text, fontSize) <= maxW {
		return text
	}
	const ellipsis = "..."
	ellipsisW := fm.MeasureText(ellipsis, fontSize)
	avail := maxW - ellipsisW
	if avail <= 0 {
		return ellipsis
	}

	// 二分查找最长不超宽的 rune 边界
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		w := fm.MeasureText(string(runes[:mid]), fontSize)
		if w <= avail {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		return ellipsis
	}
	return string(runes[:lo]) + ellipsis
}

// ── 字号缩小 ────────────────────────────────────

// ShrinkFontSize 返回使文本适应 maxW 的字号。
// 从 baseSize 开始逐步缩小（步长 1），直到适应或达到 minSize。
func ShrinkFontSize(fm *render.FontManager, text string, maxW float64, baseSize, minSize float64) float64 {
	if fm == nil || text == "" || maxW <= 0 {
		return baseSize
	}
	for size := baseSize; size >= minSize; size-- {
		if fm.MeasureText(text, size) <= maxW {
			return size
		}
	}
	return minSize
}

// ── 换行 ─────────────────────────────────────────

// WrapText 按像素宽度拆行（逐字符，对中英文混排友好）。
// 返回各行文本。单行不超宽时返回单元素切片。
func WrapText(fm *render.FontManager, text string, maxW float64, fontSize float64) []string {
	if fm == nil || text == "" {
		return nil
	}
	runes := []rune(text)
	var lines []string
	start := 0
	for start < len(runes) {
		end := start
		for end < len(runes) {
			w := fm.MeasureText(string(runes[start:end+1]), fontSize)
			if w > maxW && end > start {
				break
			}
			end++
		}
		lines = append(lines, string(runes[start:end]))
		start = end
	}
	return lines
}

// WrapLineHeight 返回换行文本的行高（与 DrawWrappedText 一致）。
const WrapLineHeight = 3 // fontSize + WrapLineHeight = 实际行高

// DrawWrappedText 在 maxW 内自动换行绘制文本，返回实际行数。
func DrawWrappedText(screen *ebiten.Image, fm *render.FontManager,
	text string, x, y, maxW, fontSize float64, clr color.Color) int {

	lines := WrapText(fm, text, maxW, fontSize)
	lineH := fontSize + WrapLineHeight
	for i, line := range lines {
		fm.DrawText(screen, line, x, y+float64(i)*lineH, fontSize, clr)
	}
	return len(lines)
}

// ── 多色段落换行 ─────────────────────────────────

// TextSegment 描述一段带颜色的文本片段。
type TextSegment struct {
	Text  string
	Color color.Color
}

// DrawSegmentsWrapped 渲染多色 TextSegment 列表，自动换行。
func DrawSegmentsWrapped(screen *ebiten.Image, fm *render.FontManager,
	segs []TextSegment, x, y, maxW, fontSize float64) {

	lineH := fontSize + WrapLineHeight
	curX := x
	curY := y

	for _, seg := range segs {
		segW := fm.MeasureText(seg.Text, fontSize)
		// 换行检测
		if curX+segW > x+maxW && curX > x {
			curX = x
			curY += lineH
		}
		clr := seg.Color
		if clr == nil {
			clr = color.White
		}
		fm.DrawText(screen, seg.Text, curX, curY, fontSize, clr)
		curX += segW
	}
}
