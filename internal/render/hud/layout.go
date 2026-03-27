package hud

import "defense2/internal/core/game"

// Layout defines all UI region positions (centralized).
var Layout = struct {
	TopBarY    float32
	TopBarH    float32
	BuildMenuY float32
	BuildMenuH float32
	InfoPanelX float32
	InfoPanelW float32
	WaveInfoX  float32
}{
	TopBarY:    4,
	TopBarH:    24,
	BuildMenuY: float32(game.ScreenHeight) - 80,
	BuildMenuH: 76,
	InfoPanelX: 4,
	InfoPanelW: 200,
	WaveInfoX:  float32(game.ScreenWidth) - 160,
}
