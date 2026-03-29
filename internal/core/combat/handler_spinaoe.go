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
	// 旋转速度随攻速缩放：基准 3 rad/s 对应攻速 0.3，攻速越快旋转越快
	spinSpeed := 3.0 * (t.AttackSpeed / 0.3)
	t.SpinAngle += dt * spinSpeed
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

	// spin_aoe 默认参数
	const innerRatio = 0.5 // 内圈半径比例
	const innerBonus = 1.5 // 内圈伤害倍率

	hasTarget := false
	r := t.Range
	innerR := r * innerRatio

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		if dist > r+e.Radius {
			return
		}
		hasTarget = true
		dmg := t.Damage
		if dist <= innerR {
			dmg *= innerBonus
		}
		e.HP -= dmg
		if ctx.OnHit != nil {
			ctx.OnHit(e, dmg, e.HP <= 0, ctx.Style)
		}
	})

	if hasTarget {
		t.SpinActive = 0.3
		t.FireTimer = 1.0 / t.AttackSpeed
		if ctx.OnFire != nil {
			ctx.OnFire(ctx.Style)
		}
	}
}
