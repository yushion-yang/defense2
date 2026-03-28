// parser.go — SVG 解析器。
// 将 SVG 字节数据解析并光栅化为 ebiten.Image，支持指定输出尺寸。
// 底层使用 oksvg（SVG 路径解析）+ rasterx（光栅化）。
package svg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// Parse 将 SVG 字节数据解析为指定尺寸的 ebiten.Image。
// w, h 为目标像素尺寸，SVG 内容会等比缩放填充。
func Parse(data []byte, w, h int) (*ebiten.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("svg parse: %w", err)
	}

	// 设置输出尺寸并光栅化
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1.0)

	// 转为 ebiten.Image
	return ebiten.NewImageFromImage(rgba), nil
}

// ParseWithPadding 解析 SVG 并在四周添加透明边距。
// padding 为单侧边距像素数。
func ParseWithPadding(data []byte, w, h, padding int) (*ebiten.Image, error) {
	innerW := w - padding*2
	innerH := h - padding*2
	if innerW <= 0 || innerH <= 0 {
		return nil, fmt.Errorf("svg: padding too large for %dx%d", w, h)
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("svg parse: %w", err)
	}

	icon.SetTarget(0, 0, float64(innerW), float64(innerH))
	inner := image.NewRGBA(image.Rect(0, 0, innerW, innerH))
	scanner := rasterx.NewScannerGV(innerW, innerH, inner, inner.Bounds())
	dasher := rasterx.NewDasher(innerW, innerH, scanner)
	icon.Draw(dasher, 1.0)

	// 将内层图绘制���带边距的外层图上
	outer := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(outer, image.Rect(padding, padding, padding+innerW, padding+innerH),
		inner, image.Point{}, draw.Over)

	return ebiten.NewImageFromImage(outer), nil
}
