// layout.go — HUD layout helpers.
// All layout constants live in the theme package; this file provides
// computed convenience values that depend on multiple theme constants.
package hud

import "defense2/internal/render/theme"

// TopBar pill position (centered horizontally).
var topBarX = float32((theme.CanvasW - theme.TopBarW) / 2)
