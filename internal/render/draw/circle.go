package draw

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const circleSegments = 48

// CircleOutline draws a circle outline using a single StrokeCircle call.
func CircleOutline(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	if r <= 0 {
		return
	}
	vector.StrokeCircle(screen, S32(cx), S32(cy), S32(r), S32(width), clr, AA())
}

// FilledCircle draws a filled circle with anti-aliasing enabled.
func FilledCircle(screen *ebiten.Image, cx, cy, r float32, clr color.Color) {
	vector.DrawFilledCircle(screen, S32(cx), S32(cy), S32(r), clr, AA())
}

// Glow draws a soft glow effect: an outer transparent ring fading into an
// inner solid circle. When a glow pass is active (BeginGlowPass), draws are
// redirected to the offscreen glow buffer for additive compositing.
func Glow(screen *ebiten.Image, cx, cy, innerR, outerR float32, clr color.RGBA) {
	if outerR <= 0 {
		return
	}

	// Redirect to glow buffer when a glow pass is active.
	target := screen
	if GlowPassActive() {
		target = GlowTarget()
	}

	scx, scy := S32(cx), S32(cy)
	outerClr := color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: clr.A / 4}
	vector.DrawFilledCircle(target, scx, scy, S32(outerR), outerClr, AA())

	if innerR > 0 {
		vector.DrawFilledCircle(target, scx, scy, S32(innerR), clr, AA())
	}
}

// Diamond draws a diamond (rotated square) outline centered at (cx, cy).
func Diamond(screen *ebiten.Image, cx, cy, r, width float32, clr color.Color) {
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	top := [2]float32{scx, scy - sr}
	right := [2]float32{scx + sr, scy}
	bottom := [2]float32{scx, scy + sr}
	left := [2]float32{scx - sr, scy}

	aa := AA()
	vector.StrokeLine(screen, top[0], top[1], right[0], right[1], sw, clr, aa)
	vector.StrokeLine(screen, right[0], right[1], bottom[0], bottom[1], sw, clr, aa)
	vector.StrokeLine(screen, bottom[0], bottom[1], left[0], left[1], sw, clr, aa)
	vector.StrokeLine(screen, left[0], left[1], top[0], top[1], sw, clr, aa)
}

// DiamondRotated draws a diamond outline rotated by angle (radians) around its center.
func DiamondRotated(screen *ebiten.Image, cx, cy, r, width float32, angle float64, clr color.Color) {
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	cos, sin := float32(math.Cos(angle)), float32(math.Sin(angle))

	// Diamond vertices relative to center, then rotate
	offsets := [4][2]float32{
		{0, -sr},  // top
		{sr, 0},   // right
		{0, sr},   // bottom
		{-sr, 0},  // left
	}
	var pts [4][2]float32
	for i, o := range offsets {
		pts[i][0] = scx + o[0]*cos - o[1]*sin
		pts[i][1] = scy + o[0]*sin + o[1]*cos
	}

	aa := AA()
	vector.StrokeLine(screen, pts[0][0], pts[0][1], pts[1][0], pts[1][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[1][0], pts[1][1], pts[2][0], pts[2][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[2][0], pts[2][1], pts[3][0], pts[3][1], sw, clr, aa)
	vector.StrokeLine(screen, pts[3][0], pts[3][1], pts[0][0], pts[0][1], sw, clr, aa)
}

// ThickLine draws a thick line with round caps from (x1,y1) to (x2,y2).
func ThickLine(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color) {
	var path vector.Path
	path.MoveTo(S32(x1), S32(y1))
	path.LineTo(S32(x2), S32(y2))

	vector.StrokePath(screen, &path, &vector.StrokeOptions{
		Width:   S32(width),
		LineCap: vector.LineCapRound,
	}, &vector.DrawPathOptions{
		AntiAlias:  AA(),
		ColorScale: colorScale(clr),
	})
}

// Line draws a StrokeLine with auto-scaling.
func Line(screen *ebiten.Image, x1, y1, x2, y2, width float32, clr color.Color, aa bool) {
	vector.StrokeLine(screen, S32(x1), S32(y1), S32(x2), S32(y2), S32(width), clr, aa)
}

// FilledRect draws a filled rectangle with auto-scaling.
func FilledRect(screen *ebiten.Image, x, y, w, h float32, clr color.Color, aa bool) {
	vector.DrawFilledRect(screen, S32(x), S32(y), S32(w), S32(h), clr, aa)
}

// Sprite draws an image centered at logical (cx, cy) with the given logical display size.
// Handles HiDPI scaling automatically. No manual GeoM scaling needed.
func Sprite(screen, img *ebiten.Image, cx, cy, displaySize float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := displaySize / w * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	screen.DrawImage(img, &op)
}

// SpriteRotated draws an image centered at logical (cx, cy) with a display size and rotation.
// rotation is in radians. offsetY is a logical vertical offset (e.g. walk wobble).
func SpriteRotated(screen, img *ebiten.Image, cx, cy, displaySize, rotation, offsetY float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := displaySize / w * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, (cy+offsetY)*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaled draws an image centered at logical (cx, cy) with a pre-computed logical scale.
// Useful when you need extra transforms (e.g. fire pulse animation).
func SpriteScaled(screen, img *ebiten.Image, cx, cy, logicalScale float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaledRotated draws an image centered at logical (cx, cy) with scale and rotation.
// rotation is in radians (0 = original orientation, positive = clockwise).
func SpriteScaledRotated(screen, img *ebiten.Image, cx, cy, logicalScale, rotation float64) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	screen.DrawImage(img, &op)
}

// SpriteScaledRotatedAlpha draws an image centered at logical (cx, cy) with scale, rotation, and alpha.
// alpha is 0.0 (transparent) to 1.0 (opaque).
func SpriteScaledRotatedAlpha(screen, img *ebiten.Image, cx, cy, logicalScale, rotation, alpha float64) {
	if img == nil || alpha <= 0 {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	s := logicalScale * Scale
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(rotation)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx*Scale, cy*Scale)
	if alpha < 1.0 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	screen.DrawImage(img, &op)
}

// Arc draws a stroked arc (partial circle outline) from startAngle to endAngle (radians).
func Arc(screen *ebiten.Image, cx, cy, r, startAngle, endAngle, width float32, clr color.Color) {
	if r <= 0 {
		return
	}
	scx, scy, sr, sw := S32(cx), S32(cy), S32(r), S32(width)
	span := endAngle - startAngle
	segments := int(float64(circleSegments) * float64(span) / (2 * math.Pi))
	if segments < 4 {
		segments = 4
	}
	step := float64(span) / float64(segments)
	var prevX, prevY float32
	for i := 0; i <= segments; i++ {
		angle := float64(startAngle) + float64(i)*step
		x := scx + sr*float32(math.Cos(angle))
		y := scy + sr*float32(math.Sin(angle))
		if i > 0 {
			vector.StrokeLine(screen, prevX, prevY, x, y, sw, clr, AA())
		}
		prevX = x
		prevY = y
	}
}
