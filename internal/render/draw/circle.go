package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const circleSegments = 48

// CircleOutline draws a circle outline using a 48-segment line approximation.
func CircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	if r <= 0 {
		return
	}

	step := 2 * math.Pi / circleSegments
	var prevX, prevY float32

	for i := 0; i <= circleSegments; i++ {
		angle := float64(i) * step
		x := cx + r*float32(math.Cos(angle))
		y := cy + r*float32(math.Sin(angle))
		if i > 0 {
			vector.StrokeLine(screen, prevX, prevY, x, y, width, clr, true)
		}
		prevX = x
		prevY = y
	}
}

// FilledCircle draws a filled circle with anti-aliasing enabled.
func FilledCircle(screen *ebiten.Image, cx, cy, r float32, clr color.Color) {
	vector.DrawFilledCircle(screen, cx, cy, r, clr, true)
}

// Glow draws a soft glow effect: an outer transparent ring fading into an
// inner solid circle. Uses two concentric filled circles with different
// alpha values to approximate radial falloff.
func Glow(screen *ebiten.Image, cx, cy, innerR, outerR float32, clr color.RGBA) {
	if outerR <= 0 {
		return
	}

	// Outer glow: low alpha version of the color.
	outerClr := color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 4}
	vector.DrawFilledCircle(screen, cx, cy, outerR, outerClr, true)

	// Inner solid circle.
	if innerR > 0 {
		vector.DrawFilledCircle(screen, cx, cy, innerR, clr, true)
	}
}

// Diamond draws a diamond (rotated square) outline centered at (cx, cy).
// The parameter r is the distance from center to each vertex.
func Diamond(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	top := [2]float32{cx, cy - r}
	right := [2]float32{cx + r, cy}
	bottom := [2]float32{cx, cy + r}
	left := [2]float32{cx - r, cy}

	vector.StrokeLine(screen, top[0], top[1], right[0], right[1], width, clr, true)
	vector.StrokeLine(screen, right[0], right[1], bottom[0], bottom[1], width, clr, true)
	vector.StrokeLine(screen, bottom[0], bottom[1], left[0], left[1], width, clr, true)
	vector.StrokeLine(screen, left[0], left[1], top[0], top[1], width, clr, true)
}

// ThickLine draws a thick line with round caps from (x1,y1) to (x2,y2)
// using a filled path for smooth rendering.
func ThickLine(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color) {
	var path vector.Path
	path.MoveTo(x1, y1)
	path.LineTo(x2, y2)

	vector.StrokePath(screen, &path, &vector.StrokeOptions{
		Width:   width,
		LineCap: vector.LineCapRound,
	}, &vector.DrawPathOptions{
		AntiAlias:  true,
		ColorScale: colorScale(clr),
	})
}
