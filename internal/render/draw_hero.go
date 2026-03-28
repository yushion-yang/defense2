// draw_hero.go — 英雄渲染。
// 将英雄绘制为绿色三角形（指向移动方向），头顶显示等级和经验条。
package render

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/core/hero"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawHero 渲染英雄。
func DrawHero(screen *ebiten.Image, h *hero.Hero) {
	if h == nil || !h.Active {
		return
	}
	cx := float32(h.X)
	cy := float32(h.Y)
	r := float32(h.BodyRadius)

	// 本体（绿色圆形）
	bodyClr := color.RGBA{R: 60, G: 220, B: 80, A: 255}
	if h.State == hero.StateEngage {
		bodyClr = color.RGBA{R: 220, G: 200, B: 60, A: 255} // 交战时变黄
	}
	vector.DrawFilledCircle(screen, cx, cy, r, bodyClr, false)

	// 攻击范围（仅交战时显示）
	if h.State == hero.StateEngage {
		drawCircleOutline(screen, cx, cy, float32(h.AttackRange), 1,
			color.RGBA{R: 60, G: 220, B: 80, A: 30})
	}

	// 牵引范围（淡蓝虚线效果）
	drawCircleOutline(screen, float32(h.BaseX), float32(h.BaseY), float32(h.LeashRadius), 0.5,
		color.RGBA{R: 100, G: 150, B: 200, A: 20})

	// 头顶等级标签
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Lv%d", h.Level),
		int(cx)-8, int(cy-r-14))

	// 经验条
	if h.Level < 10 {
		barW := r * 2.5
		barH := float32(2)
		barX := cx - barW/2
		barY := cy - r - 4

		// 背景
		vector.DrawFilledRect(screen, barX, barY, barW, barH,
			color.RGBA{R: 40, G: 40, B: 40, A: 180}, false)
		// 填充
		ratio := float32(h.XP) / float32(h.XPToNext)
		vector.DrawFilledRect(screen, barX, barY, barW*ratio, barH,
			color.RGBA{R: 100, G: 200, B: 255, A: 255}, false)
	}
}

// drawCircleOutline 仅在本文件中使用时复用 draw_tower.go 中的同名函数会冲突，
// 这里利用包内可见性直接调用——实际上 Go 同包同名函数不允许，
// 所以这个函数在 draw_tower.go 中已定义，此处直接引用即可。
// 注意：Go 同包函数可跨文件直接调用，无需重复定义。
var _ = math.Pi // 避免 unused import
