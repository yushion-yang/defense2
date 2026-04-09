// handler_projectile.go — 标准追踪弹攻击方式。
package combat

import (
	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ProjectileHandler 标准追踪弹。
type ProjectileHandler struct{}

func (h *ProjectileHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	bal := config.GlobalBalance()
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = bal.Combat.DefaultProjectileSpeed
	}
	ctx.Projectiles.Fire(t.X, t.Y, target.X, target.Y, t.Damage, speed, bal.Combat.DefaultProjectileRadius, target, t.InstanceKey)
}
