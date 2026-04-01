// handler_laser.go — 即时光束攻击方式。
// 无弹射物飞行时间，即时对目标造成伤害并创建 Beam 视觉。
package combat

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// laser 默认视觉参数
const (
	laserDuration = 0.12            // 光束显示时长（秒）
	laserWidth    = 3.0             // 光束宽度（像素）
)
var laserColor = [3]uint8{244, 63, 94} // #f43f5e 玫红

// LaserHandler 即时光束。
type LaserHandler struct{}

func (h *LaserHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	// 统一命中处理（能力触发 + 扣血 + 击杀）
	ApplyHit(HitInput{
		Tower: t, Target: target, BaseDamage: t.Damage, Style: ctx.Style,
		Enemies: ctx.Enemies, Projectiles: ctx.Projectiles, OnCC: ctx.OnCC,
	}, ctx.OnHit)

	// 创建 beam 视觉
	ctx.Beams.Add(Beam{
		X1: t.X, Y1: t.Y,
		X2: target.X, Y2: target.Y,
		Width:   laserWidth,
		Color:   laserColor,
		Life:    laserDuration,
		MaxLife: laserDuration,
	})
}
