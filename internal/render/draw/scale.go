package draw

import "github.com/hajimehoshi/ebiten/v2"

// Scale is the device pixel ratio (1.0 on non-Retina, 2.0 on Retina).
// Set by game.LayoutF at startup. All drawing functions scale internally.
var Scale float64 = 1.0

// S scales a float64 value by the device scale factor.
func S(v float64) float64 { return v * Scale }

// S32 scales a float32 value by the device scale factor.
func S32(v float32) float32 { return v * float32(Scale) }

// CursorPos returns the mouse cursor position in logical coordinates (1200×540 space).
// Equivalent to Canvas's automatic event coordinate scaling.
func CursorPos() (float64, float64) {
	x, y := ebiten.CursorPosition()
	return float64(x) / Scale, float64(y) / Scale
}

// TouchPos returns a touch position in logical coordinates.
func TouchPos(id ebiten.TouchID) (float64, float64) {
	x, y := ebiten.TouchPosition(id)
	return float64(x) / Scale, float64(y) / Scale
}
