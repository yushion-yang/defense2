package draw

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	whitePixelOnce  sync.Once
	whitePixelImage *ebiten.Image
)

// whitePixel returns a cached 3x3 white image used as the source for
// DrawTriangles-based fills. Sampling at (1,1) avoids edge artifacts.
func whitePixel() *ebiten.Image {
	whitePixelOnce.Do(func() {
		whitePixelImage = ebiten.NewImage(3, 3)
		pix := make([]byte, 3*3*4)
		for i := range pix {
			pix[i] = 0xff
		}
		whitePixelImage.WritePixels(pix)
	})
	return whitePixelImage
}

// FillPath renders a vector.Path as a solid fill using DrawTriangles with a
// white pixel source. This gives us full control over vertex colors.
func FillPath(screen *ebiten.Image, path *vector.Path, clr color.Color) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 {
		return
	}

	r, g, b, a := clr.RGBA()
	if a == 0 {
		return
	}

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = rf
		vs[i].ColorG = gf
		vs[i].ColorB = bf
		vs[i].ColorA = af
	}

	screen.DrawTriangles(vs, is, whitePixel(), &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
		FillRule:  ebiten.FillRuleNonZero,
	})
}

// roundRectPath builds a vector.Path for a rounded rectangle.
func roundRectPath(x, y, w, h, radius float32) vector.Path {
	if radius < 0 {
		radius = 0
	}
	maxR := min32(w, h) / 2
	if radius > maxR {
		radius = maxR
	}

	var p vector.Path

	// Start at top-left, after the radius.
	p.MoveTo(x+radius, y)

	// Top edge -> top-right corner.
	p.LineTo(x+w-radius, y)
	p.ArcTo(x+w, y, x+w, y+radius, radius)

	// Right edge -> bottom-right corner.
	p.LineTo(x+w, y+h-radius)
	p.ArcTo(x+w, y+h, x+w-radius, y+h, radius)

	// Bottom edge -> bottom-left corner.
	p.LineTo(x+radius, y+h)
	p.ArcTo(x, y+h, x, y+h-radius, radius)

	// Left edge -> top-left corner.
	p.LineTo(x, y+radius)
	p.ArcTo(x, y, x+radius, y, radius)

	p.Close()
	return p
}

// RoundRect draws a filled rounded rectangle.
func RoundRect(screen *ebiten.Image, x, y, w, h, radius float32, clr color.Color) {
	path := roundRectPath(S32(x), S32(y), S32(w), S32(h), S32(radius))
	FillPath(screen, &path, clr)
}

// StrokeRoundRect draws an outlined rounded rectangle.
func StrokeRoundRect(screen *ebiten.Image, x, y, w, h, radius, strokeWidth float32, clr color.Color) {
	path := roundRectPath(S32(x), S32(y), S32(w), S32(h), S32(radius))
	vector.StrokePath(screen, &path, &vector.StrokeOptions{
		Width:    S32(strokeWidth),
		LineJoin: vector.LineJoinRound,
	}, &vector.DrawPathOptions{
		AntiAlias:  true,
		ColorScale: colorScale(clr),
	})
}

// StrokeRect draws a simple rectangle outline using StrokeLine.
func StrokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.Color) {
	sx, sy, sw, sh, swidth := S32(x), S32(y), S32(w), S32(h), S32(width)
	vector.StrokeLine(screen, sx, sy, sx+sw, sy, swidth, clr, true)
	vector.StrokeLine(screen, sx+sw, sy, sx+sw, sy+sh, swidth, clr, true)
	vector.StrokeLine(screen, sx+sw, sy+sh, sx, sy+sh, swidth, clr, true)
	vector.StrokeLine(screen, sx, sy+sh, sx, sy, swidth, clr, true)
}

// Pill draws a pill shape (a rounded rectangle where radius = h/2).
func Pill(screen *ebiten.Image, x, y, w, h float32, clr color.Color) {
	// Don't double-scale: RoundRect already scales internally.
	path := roundRectPath(S32(x), S32(y), S32(w), S32(h), S32(h)/2)
	FillPath(screen, &path, clr)
}

// colorScale converts a color.Color to ebiten.ColorScale for DrawPathOptions.
func colorScale(clr color.Color) ebiten.ColorScale {
	r, g, b, a := clr.RGBA()
	var cs ebiten.ColorScale
	if a == 0 {
		cs.Scale(0, 0, 0, 0)
		return cs
	}
	cs.SetR(float32(r) / 0xffff)
	cs.SetG(float32(g) / 0xffff)
	cs.SetB(float32(b) / 0xffff)
	cs.SetA(float32(a) / 0xffff)
	return cs
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
