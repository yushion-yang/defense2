package draw

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LinearGradientV draws a vertical linear gradient directly onto screen,
// blending from top to bottom color row-by-row.
func LinearGradientV(screen *ebiten.Image, x, y, w, h int, top, bottom color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}

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

		// Pre-multiply alpha for WritePixels.
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

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, op)
}

// CachedGradient pre-renders a vertical gradient to an *ebiten.Image for reuse
// across frames without re-computing pixels every tick.
type CachedGradient struct {
	img *ebiten.Image
}

// NewCachedGradient creates a cached vertical gradient image.
func NewCachedGradient(w, h int, top, bottom color.RGBA) *CachedGradient {
	if w <= 0 || h <= 0 {
		return &CachedGradient{img: ebiten.NewImage(1, 1)}
	}

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

// Draw renders the cached gradient onto screen at position (x, y).
func (cg *CachedGradient) Draw(screen *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(cg.img, op)
}

// Image returns the underlying pre-rendered gradient image.
func (cg *CachedGradient) Image() *ebiten.Image {
	return cg.img
}

// lerp8 linearly interpolates between two uint8 values.
func lerp8(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t)
}
