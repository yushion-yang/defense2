// scale.go — HiDPI 缩放核心，draw 包的基石。
//
// 本项目所有游戏逻辑和渲染代码统一使用逻辑坐标(1200×540)，与具体设备分辨率无关。
// 本文件提供全局缩放因子 Scale 和坐标转换工具函数，使上层代码完全不需要关心
// 物理像素与逻辑像素的映射——类似浏览器 Canvas 的 devicePixelRatio 机制。
//
// 关键设计：AA() 在 HiDPI 设备(Scale>1)时返回 false，因为 Retina 2x 物理像素
// 足够小，肉眼看不出锯齿，但 vector AA 的性能开销约 2 倍。这是 30fps→90fps 的关键优化。
package draw

import "github.com/hajimehoshi/ebiten/v2"

// Scale 是设备像素比（非 Retina 为 1.0，Retina 为 2.0）。
// 由 game.LayoutF 在启动时设置。draw 包所有绘图函数内部自动乘以此值。
var Scale float64 = 1.0

// S 将 float64 逻辑坐标值乘以设备缩放因子，转换为物理像素值。
func S(v float64) float64 { return v * Scale }

// S32 将 float32 逻辑坐标值乘以设备缩放因子（float32 版本，用于 vector API）。
func S32(v float32) float32 { return v * float32(Scale) }

// CursorPos 返回鼠标光标的逻辑坐标(1200×540 空间)。
// Ebitengine 的 CursorPosition() 返回物理像素，这里除以 Scale 转回逻辑坐标。
// 所有游戏代码应使用此函数而非直接调用 ebiten.CursorPosition()。
func CursorPos() (float64, float64) {
	x, y := ebiten.CursorPosition()
	return float64(x) / Scale, float64(y) / Scale
}

// TouchPos 返回触摸点的逻辑坐标（同理，将物理像素转回逻辑空间）。
// 所有游戏代码应使用此函数而非直接调用 ebiten.TouchPosition()。
func TouchPos(id ebiten.TouchID) (float64, float64) {
	x, y := ebiten.TouchPosition(id)
	return float64(x) / Scale, float64(y) / Scale
}

// AA 返回是否应启用 vector 抗锯齿。
// HiDPI 设备(Scale > 1)物理像素极小，AA 效果不可见，但性能开销翻倍(~2x)。
// 关闭 AA 是全局最大的单项性能优化。
func AA() bool { return Scale <= 1.0 }
