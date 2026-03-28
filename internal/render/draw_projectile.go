// draw_projectile.go — 弹射物渲染。
// 将存活弹射物绘制为黄色小圆点。
package render

import (
	"image/color"

	"defense2/internal/core/projectile"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawProjectiles 渲染所有存活弹射物。
func DrawProjectiles(screen *ebiten.Image, pool *projectile.Pool) {
	clr := color.RGBA{R: 255, G: 220, B: 100, A: 255} // 黄色
	pool.Each(func(p *projectile.Projectile) {
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(p.Radius), clr, false)
	})
}
