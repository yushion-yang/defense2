// handler_scatter.go — 锥形散射攻击方式。
// 发射 N 颗真实弹丸，各自独立飞行+碰撞检测。
// 同一敌人被多颗命中时合并为一次伤害（由 TickProjectileHits 处理）。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// scatter 默认参数
const (
	scatterPellets    = 3                  // 弹丸数
	scatterHalfSpread = 30 * math.Pi / 180 // 半角30度（弧度）
	scatterPelletR    = 5.0                // 弹丸碰撞半径
)

// ScatterHandler 锥形散射。
type ScatterHandler struct{}

func (h *ScatterHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	baseAngle := math.Atan2(target.Y-t.Y, target.X-t.X)
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = 400
	}

	// 同一次散射共享 groupID，TickProjectileHits 据此合并命中
	groupID := ctx.Projectiles.NextScatterGroup()

	for i := 0; i < scatterPellets; i++ {
		frac := float64(i) / float64(scatterPellets-1)
		angle := baseAngle - scatterHalfSpread + frac*2*scatterHalfSpread
		ctx.Projectiles.FireScatterPellet(
			t.X, t.Y, angle, t.Damage, t.Range, speed,
			scatterPelletR, t.InstanceKey, groupID,
		)
	}
}
