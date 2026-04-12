package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DashedLine draws a dashed line from (x1,y1) to (x2,y2).
// Each visible segment has length dashLen, followed by a gap of gapLen.
func DashedLine(screen *ebiten.Image, x1, y1, x2, y2, width, dashLen, gapLen float32, clr color.Color) {
	x1, y1, x2, y2 = S32(x1), S32(y1), S32(x2), S32(y2)
	width, dashLen, gapLen = S32(width), S32(dashLen), S32(gapLen)
	dx := x2 - x1
	dy := y2 - y1
	totalLen := float32(math.Hypot(float64(dx), float64(dy)))
	if totalLen < 0.001 {
		return
	}

	// Unit direction vector.
	ux := dx / totalLen
	uy := dy / totalLen

	var dist float32
	drawing := true // Start with a dash.

	for dist < totalLen {
		if drawing {
			segEnd := dist + dashLen
			if segEnd > totalLen {
				segEnd = totalLen
			}
			sx := x1 + ux*dist
			sy := y1 + uy*dist
			ex := x1 + ux*segEnd
			ey := y1 + uy*segEnd
			vector.StrokeLine(screen, sx, sy, ex, ey, width, clr, true)
			dist = segEnd
		} else {
			dist += gapLen
		}
		drawing = !drawing
	}
}

// DashedCircle draws a dashed circle outline centered at (cx, cy) with radius r.
// dashLen and gapLen are arc lengths along the circumference.
func DashedCircle(screen *ebiten.Image, cx, cy, r, width, dashLen, gapLen float32, clr color.Color) {
	cx, cy, r = S32(cx), S32(cy), S32(r)
	width, dashLen, gapLen = S32(width), S32(dashLen), S32(gapLen)
	if r <= 0 {
		return
	}

	circumference := 2 * math.Pi * float64(r)
	if circumference < 0.001 {
		return
	}

	var dist float64
	drawing := true
	draws := 0
	const maxDashes = 64 // 防止极大半径导致数千次 StrokeLine

	for dist < circumference && draws < maxDashes {
		if drawing {
			segEnd := dist + float64(dashLen)
			if segEnd > circumference {
				segEnd = circumference
			}

			startAngle := dist / float64(r)
			endAngle := segEnd / float64(r)

			sx := float32(float64(cx) + float64(r)*math.Cos(startAngle))
			sy := float32(float64(cy) + float64(r)*math.Sin(startAngle))
			ex := float32(float64(cx) + float64(r)*math.Cos(endAngle))
			ey := float32(float64(cy) + float64(r)*math.Sin(endAngle))

			vector.StrokeLine(screen, sx, sy, ex, ey, width, clr, true)
			draws++
			dist = segEnd
		} else {
			dist += float64(gapLen)
		}
		drawing = !drawing
	}
}
