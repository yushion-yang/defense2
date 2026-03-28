// draw_warden.go — 战灵渲染。
// 将战灵绘制为半透明发光圆形，附身时显示在塔上方。
package render

import (
	"image/color"

	"defense2/internal/core/warden"
	wardenTypes "defense2/internal/core/warden/types"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawWarden 渲染单个战灵。
func DrawWarden(screen *ebiten.Image, w *warden.Warden) {
	if w == nil || !w.Active {
		return
	}

	switch s := w.State.(type) {
	case *wardenTypes.EnvoyState:
		drawEnvoy(screen, s, w)
	}
}

// drawEnvoy 渲染使者型战灵。
func drawEnvoy(screen *ebiten.Image, s *wardenTypes.EnvoyState, w *warden.Warden) {
	cx := float32(s.X)
	cy := float32(s.Y)
	if cx == 0 && cy == 0 {
		return // 尚未定位
	}

	r := float32(8)

	// 光环（半透明紫色）
	vector.DrawFilledCircle(screen, cx, cy, r+4,
		color.RGBA{R: 180, G: 120, B: 255, A: 40}, false)

	// 本体（紫色小圆）
	bodyClr := color.RGBA{R: 180, G: 120, B: 255, A: 200}
	if s.Phase == "possessing" {
		bodyClr = color.RGBA{R: 255, G: 200, B: 100, A: 230} // 附身时金色
	}
	vector.DrawFilledCircle(screen, cx, cy, r, bodyClr, false)
}
