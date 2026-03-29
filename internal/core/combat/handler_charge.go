// handler_charge.go — 蓄力攻击方式（自管理冷却）。
// 持续蓄力，满蓄时发射高伤弹射物。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ChargeHandler 蓄力攻击。
type ChargeHandler struct{}

func (h *ChargeHandler) SelfManaged() bool { return true }

// charge 默认蓄力倍率（由 chargeShot 能力的 CalcScale 在运行时覆盖）
const chargeDefaultMult = 3.0

func (h *ChargeHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	mult := chargeDefaultMult
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = 600
	}
	ctx.Projectiles.FireCharge(t.X, t.Y, target.X, target.Y, t.Damage*mult, speed, target, t.Key)

	t.ChargeProgress = 0
	t.ChargeReady = false
	t.FireAnim = 0.15
	if ctx.OnFire != nil {
		ctx.OnFire(ctx.Style)
	}
}

func (h *ChargeHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// 射击动画衰减
	if t.FireAnim > 0 {
		t.FireAnim -= dt
	}

	// 蓄力进度
	chargeTime := 1.0 / t.AttackSpeed // 用 fireRate 作为蓄力时间
	if chargeTime <= 0 {
		chargeTime = 2.5
	}
	if !t.ChargeReady {
		t.ChargeProgress += dt / chargeTime
		if t.ChargeProgress >= 1.0 {
			t.ChargeProgress = 1.0
			t.ChargeReady = true
		}
	}

	// 有目标且满蓄 → 发射
	if t.ChargeReady {
		target := tower.AcquireTarget(t, ctx.Enemies)
		if target != nil {
			t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)
			h.Fire(t, target, ctx)
		}
	}
}
