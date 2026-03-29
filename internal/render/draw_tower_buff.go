// draw_tower_buff.go — 炮塔金灵 buff 强化特效。
// 仅对金灵战灵施加的临时 buff（key 含 "envoy_buff"）显示旋转五角星芒阵。
package render

import (
	"image/color"
	"math"
	"strings"

	"defense2/internal/core/tower"
	"defense2/internal/render/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawTowerBuffEffect 在被金灵 buff 的塔上绘制五角星芒阵。
// 只对 key 含 "envoy_buff" 的 buff 生效。
func DrawTowerBuffEffect(screen *ebiten.Image, t *tower.Tower, animTime float64) {
	// 检查是否有金灵 buff
	var envoyBuff *tower.TowerBuff
	for i := range t.Buffs {
		if strings.Contains(t.Buffs[i].Key, "envoy_buff") {
			envoyBuff = &t.Buffs[i]
			break
		}
	}
	if envoyBuff == nil {
		return
	}

	cx, cy := float32(t.X), float32(t.Y)

	// buff 剩余比例（用于渐隐）
	p := float32(1.0)
	if envoyBuff.Duration > 0 && envoyBuff.Remaining > 0 {
		p = float32(envoyBuff.Remaining / envoyBuff.Duration)
		if p > 1 {
			p = 1
		}
	}

	alpha := uint8(float32(100) * p)
	starR := float32(28) // 足够大覆盖炮塔（炮塔精灵约 24px）
	rotation := float32(animTime * 0.6)
	pulse := float32(1.0 + 0.1*math.Sin(animTime*2.5))

	// 外层金色光环（脉冲）
	draw.CircleOutline(screen, cx, cy, starR*pulse, 1,
		color.RGBA{R: 255, G: 220, B: 80, A: alpha / 2})

	// 五角星芒阵（旋转）
	drawPentagram(screen, cx, cy, starR*0.85, rotation, alpha)

	// 内层光晕
	draw.FilledCircle(screen, cx, cy, 8*pulse,
		color.RGBA{R: 255, G: 230, B: 120, A: alpha / 3})
}

// drawPentagram 绘制五角星芒阵。
func drawPentagram(screen *ebiten.Image, cx, cy, r float32, rotation float32, alpha uint8) {
	clr := color.RGBA{R: 255, G: 210, B: 80, A: alpha}

	var pts [5][2]float32
	for i := 0; i < 5; i++ {
		angle := float64(rotation) + float64(i)*2*math.Pi/5 - math.Pi/2
		pts[i] = [2]float32{
			cx + r*float32(math.Cos(angle)),
			cy + r*float32(math.Sin(angle)),
		}
	}

	// 连线画星（0→2→4→1→3→0）
	order := [6]int{0, 2, 4, 1, 3, 0}
	for i := 0; i < 5; i++ {
		a, b := order[i], order[i+1]
		draw.Line(screen, pts[a][0], pts[a][1], pts[b][0], pts[b][1], 1.2, clr, true)
	}

	// 五角顶点光点
	for _, pt := range pts {
		draw.FilledCircle(screen, pt[0], pt[1], 2,
			color.RGBA{R: 255, G: 240, B: 150, A: alpha})
	}
}
