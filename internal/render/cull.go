// cull.go — viewport culling for large-map camera rendering.
// When the map exceeds the screen size, entities outside the visible viewport
// (plus a margin) can be skipped to reduce draw calls.
//
// Usage: call SetViewport once per frame before rendering world entities.
// Individual draw functions call IsInView to skip off-screen entities.
package render

import "defense2/internal/core/game"

// viewport holds the current camera state for culling checks.
var viewport struct {
	camX, camY float64
	active     bool // true when the map needs camera scrolling
}

// ResetViewport 清除视口裁剪状态（局间清理用）。
func ResetViewport() { SetViewport(0, 0, false) }

// SetViewport updates the current camera position for viewport culling.
// Call this once per frame before drawing world entities.
// When camActive is false (map fits on screen), all entities are considered in-view.
func SetViewport(camX, camY float64, camActive bool) {
	viewport.camX = camX
	viewport.camY = camY
	viewport.active = camActive
}

// IsInView returns true if the point (x, y) in world coordinates falls within
// the visible viewport expanded by a generous margin. When the camera is
// inactive (small maps), every point is in-view.
func IsInView(x, y float64) bool {
	if !viewport.active {
		return true
	}
	const margin = 120 // logical pixels — covers large sprites + trails
	return x >= viewport.camX-margin &&
		x <= viewport.camX+float64(game.ScreenWidth)+margin &&
		y >= viewport.camY-margin &&
		y <= viewport.camY+float64(game.ScreenHeight)+margin
}
