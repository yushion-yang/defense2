// gradient.go — 渐变图像生成与缓存。
//
// 提供两种使用方式：
//   - LinearGradientV：即时生成并绘制，适合一次性使用（每帧会重新分配内存）
//   - CachedGradient：预烘焙为 *ebiten.Image 缓存复用，适合多帧不变的背景
//
// CachedGradient 被 6 个非 Stage 场景（Title/Select/CampaignSelect/TestSelect/
// Settings/Result）用于背景渐变，避免每帧重新计算像素数据。
//
// 实现细节：Ebitengine 的 WritePixels 要求预乘 alpha (premultiplied alpha)，
// 即 R/G/B 通道值必须 <= A 通道值。本文件在像素填充时手动执行预乘。
package draw

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LinearGradientV 即时生成并绘制垂直线性渐变。
// 从 top 颜色到 bottom 颜色逐行插值。注意每次调用都会 NewImage+WritePixels，
// 仅适合一次性绘制。频繁使用请改用 CachedGradient。
func LinearGradientV(screen *ebiten.Image, x, y, w, h int, top, bottom color.RGBA) {
	// 逻辑尺寸 → 物理像素尺寸（HiDPI 适配）
	sw := int(float64(w) * Scale)
	sh := int(float64(h) * Scale)
	if sw <= 0 || sh <= 0 {
		return
	}

	img := ebiten.NewImage(sw, sh)
	w, h = sw, sh
	pix := make([]byte, w*h*4)

	for row := 0; row < h; row++ {
		t := float64(row) / float64(h-1) // 插值参数 0→1
		if h == 1 {
			t = 0
		}
		r := lerp8(top.R, bottom.R, t)
		g := lerp8(top.G, bottom.G, t)
		b := lerp8(top.B, bottom.B, t)
		a := lerp8(top.A, bottom.A, t)

		// 预乘 alpha（Ebitengine WritePixels 要求）
		pr := uint8(uint16(r) * uint16(a) / 255)
		pg := uint8(uint16(g) * uint16(a) / 255)
		pb := uint8(uint16(b) * uint16(a) / 255)

		// 同一行所有像素颜色相同（垂直渐变）
		for col := 0; col < w; col++ {
			off := (row*w + col) * 4
			pix[off+0] = pr
			pix[off+1] = pg
			pix[off+2] = pb
			pix[off+3] = a
		}
	}

	img.WritePixels(pix)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(S(float64(x)), S(float64(y)))
	screen.DrawImage(img, op)
}

// CachedGradient 预烘焙的垂直渐变图像，创建一次后可跨帧复用。
// 避免每帧重新计算像素并分配内存，适合背景等静态渐变。
type CachedGradient struct {
	img *ebiten.Image
}

// NewCachedGradient 创建预烘焙的垂直渐变图像。
// 返回值可长期持有，通过 Draw() 方法绘制到屏幕。
func NewCachedGradient(w, h int, top, bottom color.RGBA) *CachedGradient {
	sw := int(float64(w) * Scale)
	sh := int(float64(h) * Scale)
	if sw <= 0 || sh <= 0 {
		return &CachedGradient{img: ebiten.NewImage(1, 1)} // 退化情况返回 1x1 占位
	}

	w, h = sw, sh
	img := ebiten.NewImage(w, h)
	pix := make([]byte, w*h*4)

	for row := 0; row < h; row++ {
		t := float64(row) / float64(h-1)
		if h == 1 {
			t = 0
		}
		r := lerp8(top.R, bottom.R, t)
		g := lerp8(top.G, bottom.G, t)
		b := lerp8(top.B, bottom.B, t)
		a := lerp8(top.A, bottom.A, t)

		// 预乘 alpha
		pr := uint8(uint16(r) * uint16(a) / 255)
		pg := uint8(uint16(g) * uint16(a) / 255)
		pb := uint8(uint16(b) * uint16(a) / 255)

		for col := 0; col < w; col++ {
			off := (row*w + col) * 4
			pix[off+0] = pr
			pix[off+1] = pg
			pix[off+2] = pb
			pix[off+3] = a
		}
	}

	img.WritePixels(pix)
	return &CachedGradient{img: img}
}

// Draw 将缓存的渐变图像绘制到 screen 的逻辑坐标 (x, y) 位置。
func (cg *CachedGradient) Draw(screen *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(S(x), S(y))
	screen.DrawImage(cg.img, op)
}

// Image 返回底层预渲染的渐变图像（供需要直接操作的场景使用）。
func (cg *CachedGradient) Image() *ebiten.Image {
	return cg.img
}

// lerp8 对两个 uint8 值做线性插值。t 范围 [0,1]，t=0 返回 a，t=1 返回 b。
func lerp8(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t)
}
