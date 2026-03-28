// handler_pierce.go — 穿刺弹攻击方式。
// 命中后继续飞向下一个未命中的敌人，伤害衰减。
package combat

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// PierceHandler 穿刺弹。
type PierceHandler struct{}

func (h *PierceHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = 350
	}
	targets := t.PierceTargets
	if targets <= 0 {
		targets = 2
	}
	decay := t.PierceDecay
	if decay <= 0 {
		decay = 0.8
	}
	ctx.Projectiles.FirePierce(t.X, t.Y, target.X, target.Y, t.Damage, speed, target, t.Key, targets, decay)
}
