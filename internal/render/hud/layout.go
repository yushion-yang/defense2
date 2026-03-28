// layout.go — HUD 布局坐标集中管理。
// 所有 UI 区域的位置、尺寸在此统一定义，渲染代码只读取 Layout 变量。
package hud

import "defense2/internal/core/game"

// Layout 集中定义所有 HUD 区域坐标。
var Layout = struct {
	// 顶部信息栏
	TopBarY float32
	TopBarH float32

	// 底部建塔菜单
	BuildMenuX float32
	BuildMenuY float32
	BuildMenuW float32
	BuildMenuH float32
	SlotW      float32 // 单个塔槽宽度
	SlotH      float32
	SlotGap    float32 // 槽间距

	// 左下塔信息面板
	InfoPanelX float32
	InfoPanelY float32
	InfoPanelW float32
	InfoPanelH float32

	// 右上波次信息
	WaveInfoX float32
	WaveInfoY float32
}{
	TopBarY: 0,
	TopBarH: 28,

	BuildMenuX: float32(game.ScreenWidth)/2 - 250,
	BuildMenuY: float32(game.ScreenHeight) - 70,
	BuildMenuW: 500,
	BuildMenuH: 66,
	SlotW:      110,
	SlotH:      58,
	SlotGap:    8,

	InfoPanelX: 4,
	InfoPanelY: float32(game.ScreenHeight) - 140,
	InfoPanelW: 180,
	InfoPanelH: 64,

	WaveInfoX: float32(game.ScreenWidth) - 180,
	WaveInfoY: 0,
}
