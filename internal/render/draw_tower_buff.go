// draw_tower_buff.go — 炮塔金灵 buff 强化特效。
// 仅对金灵战灵施加的临时 buff（ID 含 "envoy_buff"）显示旋转五角星芒阵。
package render

import (
	"strings"

	"defense2/internal/core/buff"
	"defense2/internal/core/tower"
	"defense2/internal/render/vfx"

	"github.com/hajimehoshi/ebiten/v2"
)

// DrawTowerBuffEffect 在被金灵 buff 的塔上绘制五角星芒阵。
// 只对 ID 含 "envoy_buff" 的 buff 生效。
func DrawTowerBuffEffect(screen *ebiten.Image, t *tower.Tower, animTime float64) {
	var envoyBuff *buff.Buff
	for _, b := range t.Buffs.Active() {
		if strings.Contains(b.ID, "envoy_buff") {
			envoyBuff = &b
			break
		}
	}
	if envoyBuff == nil {
		return
	}

	cx, cy := float32(t.X), float32(t.Y)
	remainRatio := 1.0
	if envoyBuff.Duration > 0 && envoyBuff.Remaining > 0 {
		remainRatio = envoyBuff.Remaining / envoyBuff.Duration
		if remainRatio > 1 {
			remainRatio = 1
		}
	}
	vfx.DrawPentagram(screen, cx, cy, remainRatio, animTime)
}
