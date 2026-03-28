// handler_widebeam.go — 宽光束攻击方式（贯穿路径所有敌人）。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// WideBeamHandler 宽光束。
type WideBeamHandler struct{}

func (h *WideBeamHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	// 计算射线方向
	dx := target.X - t.X
	dy := target.Y - t.Y
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		return
	}
	dirX := dx / dist
	dirY := dy / dist

	// 射线延伸 3 倍射程
	beamLen := t.Range * 3
	endX := t.X + dirX*beamLen
	endY := t.Y + dirY*beamLen

	w := t.BeamWidth
	if w <= 0 {
		w = 8
	}
	halfW := w / 2

	// 对路径上所有敌人造成伤害（垂直距离检查）
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		// 投影到射线
		ex := e.X - t.X
		ey := e.Y - t.Y
		proj := ex*dirX + ey*dirY
		if proj < 0 || proj > beamLen {
			return
		}
		// 垂直距离
		perpDist := math.Abs(ex*(-dirY) + ey*dirX)
		hitR := halfW + e.Radius
		if perpDist > hitR {
			return
		}
		e.HP -= t.Damage
		if ctx.OnHit != nil {
			ctx.OnHit(e, t.Damage, e.HP <= 0)
		}
	})

	// Beam 视觉
	dur := t.BeamDuration
	if dur <= 0 {
		dur = 0.15
	}
	clr := t.BeamColor
	if clr == [3]uint8{} {
		clr = [3]uint8{147, 197, 253}
	}
	ctx.Beams.Add(Beam{
		X1: t.X, Y1: t.Y,
		X2: endX, Y2: endY,
		Width:   w,
		Color:   clr,
		Life:    dur,
		MaxLife: dur,
		Wide:    true,
	})
}
