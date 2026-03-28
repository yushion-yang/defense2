// handler_scatter.go — 锥形散射攻击方式（即时命中 + 视觉弹丸）。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

const scatterHitRadius = 15.0 // 每颗弹丸的命中判定半径

// ScatterHandler 锥形散射。
type ScatterHandler struct{}

func (h *ScatterHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	pellets := t.ScatterPellets
	if pellets <= 0 {
		pellets = 3
	}
	halfSpread := t.ScatterSpread
	if halfSpread <= 0 {
		halfSpread = 30 * math.Pi / 180 // 默认 30 度半角
	}

	baseAngle := math.Atan2(target.Y-t.Y, target.X-t.X)
	damagePerPellet := t.Damage * 0.7 // 每颗弹丸 70% 伤害

	for i := 0; i < pellets; i++ {
		// 均匀分布在扇形内
		var angle float64
		if pellets == 1 {
			angle = baseAngle
		} else {
			frac := float64(i) / float64(pellets-1) // 0 to 1
			angle = baseAngle - halfSpread + frac*2*halfSpread
		}

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
				ctx.OnHit(e, damagePerPellet, e.HP <= 0)
			}
		})

		// 视觉弹丸（非追踪，纯飞行效果）
		speed := t.ProjectileSpeed
		if speed <= 0 {
			speed = 400
		}
		ctx.Projectiles.FireScatter(t.X, t.Y, angle, t.Range, speed, t.Key)
	}
}
