// handler_widebeam.go — 宽光束攻击方式（贯穿路径所有敌人）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// wideBeam 默认视觉参数
const (
	wideBeamDuration = 0.15 // 光束显示时长（秒）
	wideBeamWidth    = 6.0  // 光束宽度（像素）
)

var wideBeamColor = [3]uint8{147, 197, 253} // #93c5fd 蓝

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

	// 射线延伸 N 倍射程
	beamLen := t.Range * config.GlobalBalance().Combat.WideBeamRangeMult
	endX := t.X + dirX*beamLen
	endY := t.Y + dirY*beamLen
	halfW := wideBeamWidth / 2

	// 对路径上所有敌人造成伤害（垂直距离检查）
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() {
			return
		}
		ex := e.X - t.X
		ey := e.Y - t.Y
		proj := ex*dirX + ey*dirY
		if proj < 0 || proj > beamLen {
			return
		}
		perpDist := math.Abs(ex*(-dirY) + ey*dirX)
		if perpDist > halfW+e.Radius {
			return
		}
		ApplyHit(HitInput{
			Tower: t, Target: e, BaseDamage: t.Damage, Style: ctx.Style,
			Enemies: ctx.Enemies, Projectiles: ctx.Projectiles, OnCC: ctx.OnCC,
		}, ctx.OnHit)
	})

	// Beam 视觉
	ctx.Beams.Add(Beam{
		X1: t.X, Y1: t.Y,
		X2: endX, Y2: endY,
		Width:   wideBeamWidth,
		Color:   wideBeamColor,
		Life:    wideBeamDuration,
		MaxLife: wideBeamDuration,
		Wide:    true,
	})
}
