// handler_charge.go — 蓄力攻击方式（自管理冷却）。
// 持续蓄力，满蓄时发射高伤弹射物。伤害 = 塔伤害 × chargeShot 倍率。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ChargeHandler 蓄力攻击。
type ChargeHandler struct{}

func (h *ChargeHandler) SelfManaged() bool { return true }

func (h *ChargeHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	mult := chargeMultiplier(t)
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

// chargeMultiplier 从 chargeShot 能力配置读取当前强度下的伤害倍率。
func chargeMultiplier(t *tower.Tower) float64 {
	abTable := config.GlobalAbilityTable()
	if abTable == nil {
		return 2.0
	}
	def, ok := abTable["chargeShot"]
	if !ok {
		return 2.0
	}
	str := 100.0
	if t.Strength != nil {
		str = t.Strength.Effective()
	}
	mult := def.CalcScale(str) // base + potential * (strength/100)
	if mult < 1 {
		mult = 1
	}
	return mult
}
