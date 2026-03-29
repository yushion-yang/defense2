// handler_scatter.go — 锥形散射攻击方式（即时命中 + 视觉弹丸）。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// scatter 默认参数
const (
	scatterPellets   = 3                        // 弹丸数
	scatterHalfSpread = 30 * math.Pi / 180      // 半角30度（弧度）
	scatterHitRadius = 15.0                     // 每颗弹丸命中判定半径
	scatterDmgRatio  = 0.7                      // 每颗弹丸伤害比例
)

// ScatterHandler 锥形散射。
type ScatterHandler struct{}

func (h *ScatterHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	baseAngle := math.Atan2(target.Y-t.Y, target.X-t.X)
	damagePerPellet := t.Damage * scatterDmgRatio

	for i := 0; i < scatterPellets; i++ {
		frac := float64(i) / float64(scatterPellets-1)
		angle := baseAngle - scatterHalfSpread + frac*2*scatterHalfSpread

		dirX := math.Cos(angle)
		dirY := math.Sin(angle)

		// 射线检查：沿方向到射程距离内的敌人
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			ex := e.X - t.X
			ey := e.Y - t.Y
			proj := ex*dirX + ey*dirY
			if proj < 0 || proj > t.Range {
				return
			}
			perpDist := math.Abs(ex*(-dirY) + ey*dirX)
			if perpDist > scatterHitRadius+e.Radius {
				return
			}
			e.HP -= damagePerPellet
			if ctx.OnHit != nil {
				ctx.OnHit(e, damagePerPellet, e.HP <= 0, ctx.Style)
			}
		})

		// 视觉弹丸
		speed := t.ProjectileSpeed
		if speed <= 0 {
			speed = 400
		}
		ctx.Projectiles.FireScatter(t.X, t.Y, angle, t.Range, speed, t.Key)
	}
}
