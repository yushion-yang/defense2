// hover.go — Global long-press hover tracker for touch devices.
// On desktop (mouse), HoverPos() returns CursorPos() as-is.
// On touch, holding a finger for ~500ms without moving triggers "hover mode",
// making the finger position act as a mouse cursor for tooltips/highlights.
//
// Call TickHover() once per frame at the top of Game.Update().
package draw

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Long-press thresholds.
const (
	longPressFrames    = 30   // ~500ms at 60fps
	longPressMoveLimit = 10.0 // logical pixels — cancel if finger moves beyond this
)

type hoverPhase int

const (
	hvIdle    hoverPhase = iota // no touch or desktop mouse
	hvWaiting                   // touch down, counting frames
	hvActive                    // long press triggered — finger = hover cursor
)

// Global hover state (package-level, single-threaded Ebitengine model).
var (
	hvPhase    hoverPhase
	hvStartX   float64 // press-down position (logical)
	hvStartY   float64
	hvX, hvY   float64 // current hover position (logical)
	hvFrames   int     // frames held without exceeding move limit
	hvConsumed bool    // true for 1 frame after long-press release
	hvTouchID  ebiten.TouchID
	hvTracking bool // whether hvTouchID is valid
)

// TickHover updates the long-press tracker. Must be called once per frame
// at the top of Game.Update(), BEFORE any scene Update.
func TickHover() {
	// Clear consumed flag from previous frame.
	hvConsumed = false

	// Detect active touches.
	ids := ebiten.AppendTouchIDs(nil)
	touching := len(ids) > 0

	switch hvPhase {
	case hvIdle:
		if !touching {
			return
		}
		// New touch — start tracking.
		justPressed := inpututil.AppendJustPressedTouchIDs(nil)
		if len(justPressed) == 0 {
			return
		}
		hvTouchID = justPressed[0]
		hvTracking = true
		tx, ty := ebiten.TouchPosition(hvTouchID)
		hvStartX = float64(tx) / Scale
		hvStartY = float64(ty) / Scale
		hvX, hvY = hvStartX, hvStartY
		hvFrames = 0
		hvPhase = hvWaiting

	case hvWaiting:
		if !hvTouchActive(ids) {
			// Released before threshold — normal tap, no hover.
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		tx, ty := ebiten.TouchPosition(hvTouchID)
		lx, ly := float64(tx)/Scale, float64(ty)/Scale
		dist := math.Hypot(lx-hvStartX, ly-hvStartY)
		if dist > longPressMoveLimit {
			// Moved too far — cancel, let Gesture handle as drag/tap.
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		hvFrames++
		if hvFrames >= longPressFrames {
			hvPhase = hvActive
			hvX, hvY = lx, ly
		}

	case hvActive:
		if !hvTouchActive(ids) {
			// Released — consume this release so it doesn't become a tap.
			hvConsumed = true
			hvPhase = hvIdle
			hvTracking = false
			return
		}
		// Update hover position as finger moves.
		tx, ty := ebiten.TouchPosition(hvTouchID)
		hvX, hvY = float64(tx)/Scale, float64(ty)/Scale
	}
}

// HoverPos returns the effective hover position for UI highlights/tooltips.
//
// Desktop (mouse): always returns (CursorPos(), true).
// Touch: returns the long-press position while active, otherwise (-1, -1, false).
func HoverPos() (float64, float64, bool) {
	// If any touch is active, we're in touch mode — use long-press tracker.
	if hvTracking || hvConsumed {
		if hvPhase == hvActive {
			return hvX, hvY, true
		}
		return -1, -1, false
	}
	// Desktop mode — mouse cursor is always a valid hover source.
	x, y := CursorPos()
	return x, y, true
}

// LongPressConsumed returns true for exactly 1 frame after a long-press
// hover release. Callers should check this to suppress the tap.
func LongPressConsumed() bool {
	return hvConsumed
}

// hvTouchActive checks if the tracked touch ID is still active.
func hvTouchActive(ids []ebiten.TouchID) bool {
	if !hvTracking {
		return false
	}
	for _, id := range ids {
		if id == hvTouchID {
			return true
		}
	}
	return false
}
