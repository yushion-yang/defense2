// handler_laser.go — 即时光束攻击方式。
// 无弹射物飞行时间，即时对目标造成伤害并创建 Beam 视觉。
package combat

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// LaserHandler 即时光束。
type LaserHandler struct{}

func (h *LaserHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	// 即时伤害
	target.HP -= t.Damage
	if target.HP <= 0 {
		if ctx.OnHit != nil {
			ctx.OnHit(target, t.Damage, true, ctx.Style)
		}
	} else if ctx.OnHit != nil {
		ctx.OnHit(target, t.Damage, false, ctx.Style)
	}

	// 创建 beam 视觉
	dur := t.BeamDuration
	if dur <= 0 {
		dur = 0.15
	}
	w := t.BeamWidth
	if w <= 0 {
		w = 3
	}
	clr := t.BeamColor
	if clr == [3]uint8{} {
		clr = [3]uint8{147, 197, 253} // #93c5fd
	}
	ctx.Beams.Add(Beam{
		X1: t.X, Y1: t.Y,
		X2: target.X, Y2: target.Y,
		Width:   w,
		Color:   clr,
		Life:    dur,
		MaxLife: dur,
	})
}
