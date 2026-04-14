// tooltip.go — 浮动提示面板组件。
// 在指定位置渲染 tooltip 背景，自动避免超出屏幕边界。
// 返回内容区 Rect，调用方在内容区内用 Label 等组件填充。
package ui

import (
	"image/color"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TooltipStyle 浮动提示样式。
type TooltipStyle struct {
	W       float32     // 宽度，0 → theme.TooltipBuildW
	Radius  float32     // 圆角，0 → theme.TooltipRadius
	Pad     float32     // 内边距，0 → theme.TooltipPad
	BgColor color.Color // 背景色，nil → 默认深蓝
	Border  color.Color // 边框色，nil → 默认灰蓝
}

// Tooltip 在 (anchorX, anchorY) 附近绘制浮动提示背景。
// contentH 为内容区高度（不含 padding）。
// 自动避免超出屏幕：优先向右下展开，空间不足时向左上。
// 返回内容区 Rect。
func Tooltip(screen *ebiten.Image, anchorX, anchorY float32, contentH float32, style TooltipStyle) Rect {
	w := style.W
	if w <= 0 {
		w = theme.TooltipBuildW
	}
	r := style.Radius
	if r <= 0 {
		r = theme.TooltipRadius
	}
	pad := style.Pad
	if pad <= 0 {
		pad = theme.TooltipPad
	}
	bg := style.BgColor
	if bg == nil {
		bg = color.RGBA{R: 18, G: 24, B: 42, A: 240}
	}
	border := style.Border
	if border == nil {
		border = color.RGBA{R: 60, G: 80, B: 120, A: 200}
	}

	h := contentH + pad*2
	tx := anchorX + 8
	ty := anchorY + 8

	// 屏幕边界修正
	canvasW := float32(theme.CanvasW)
	canvasH := float32(theme.CanvasH)
	if tx+w > canvasW {
		tx = anchorX - w - 8
	}
	if ty+h > canvasH {
		ty = canvasH - h - 4
	}
	if tx < 0 {
		tx = 4
	}
	if ty < 0 {
		ty = 4
	}

	draw.RoundRect(screen, tx, ty, w, h, r, bg)
	draw.StrokeRoundRect(screen, tx, ty, w, h, r, 1, border)

	return Rect{
		X: tx + pad,
		Y: ty + pad,
		W: w - pad*2,
		H: contentH,
	}
}
