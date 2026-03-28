// handler_spinaoe.go — 旋转范围伤害攻击方式（自管理）。
// 无弹射物，范围内全体受伤，内圈加伤。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// SpinAoEHandler 旋转 AoE。
type SpinAoEHandler struct{}

func (h *SpinAoEHandler) SelfManaged() bool { return true }

func (h *SpinAoEHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {
	// spin_aoe 不通过 Fire 触发
}

func (h *SpinAoEHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// 旋转角度持续更新
	t.SpinAngle += dt * 3.0 // ~3 rad/s
	if t.SpinAngle > 2*math.Pi {
		t.SpinAngle -= 2 * math.Pi
	}

	// 旋转激活计时衰减
	if t.SpinActive > 0 {
		t.SpinActive -= dt
	}

	// 冷却
	t.FireTimer -= dt
	if t.FireTimer > 0 {
		return
	}

	// 检查范围内是否有敌人
	hasTarget := false
	r := t.Range
	innerR := r * t.InnerRatioR
	bonus := t.InnerDmgBonus
	if bonus <= 0 {
		bonus = 1.5
	}

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		if dist > r+e.Radius {
			return
		}
		hasTarget = true
		dmg := t.Damage
		if dist <= innerR {
			dmg *= bonus
		}
		e.HP -= dmg
		if ctx.OnHit != nil {
			ctx.OnHit(e, dmg, e.HP <= 0)
		}
	})

	if hasTarget {
		t.SpinActive = 0.3
		t.FireTimer = 1.0 / t.AttackSpeed
		if ctx.OnFire != nil {
			ctx.OnFire()
		}
	}
}
