// handler_auradot.go — 持续范围毒伤攻击方式（自管理）。
// 无弹射物，定时对范围内所有敌人造成伤害。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// AuraDotHandler 范围持续伤害。
type AuraDotHandler struct{}

func (h *AuraDotHandler) SelfManaged() bool { return true }

func (h *AuraDotHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {}

func (h *AuraDotHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// 脉冲动画
	if t.AuraPulse > 0 {
		t.AuraPulse -= dt
	}

	// 冷却
	t.FireTimer -= dt
	if t.FireTimer > 0 {
		return
	}

	// 对范围内敌人造成伤害
	r := t.Range
	hit := false
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		if dist > r+e.Radius {
			return
		}
		hit = true
		ApplyHit(HitInput{
			Tower: t, Target: e, BaseDamage: t.Damage, Style: ctx.Style,
			Enemies: ctx.Enemies, Projectiles: ctx.Projectiles,
		}, ctx.OnHit)
	})

	if hit {
		t.AuraPulse = 0.3
	}
	t.FireTimer = 1.0 / t.AttackSpeed
}
