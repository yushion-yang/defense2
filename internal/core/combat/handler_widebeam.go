// handler_widebeam.go — 宽光束攻击方式（贯穿路径所有敌人）。
//
// wideBeam 的设计定位是"直线贯穿群体伤害"：从塔向目标方向发射一道宽光束，
// 射线路径上的所有敌人受到全额伤害（不衰减），射程 = 塔射程 × rangeMult。
//
// 与 scatter（散射）的区别：
//   - wideBeam 是单条直线，所有命中敌人受相同伤害
//   - scatter 是扇形多弹，每弹独立命中
//
// 碰撞检测使用向量投影+垂直距离：
//   - proj = 敌人到塔的向量在射线方向的投影长度（判断是否在射线前方且在射程内）
//   - perpDist = 敌人到射线的垂直距离（判断是否在光束宽度内）
//   - 加上敌人半径(e.Radius)的容差，让大型敌人更容易被命中
//
// 视觉效果通过 BeamPool 管理，光束有 Life 衰减（淡出效果）。
//
// 对应精灵：prism（棱镜造型）。
package combat

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// wideBeam 默认参数
const (
	wideBeamDuration        = 0.15 // 光束显示时长（秒）
	wideBeamWidth           = 6.0  // 光束宽度（像素）
	wideBeamDefaultRangeMul = 3.0  // 射程倍率 fallback（abilities.json 未加载时使用）
)

var wideBeamColor = [3]uint8{147, 197, 253} // #93c5fd 蓝

// WideBeamHandler 宽光束。
type WideBeamHandler struct{}

// Fire 向目标方向发射一道宽光束，对路径上所有敌人造成即时伤害。
// 流程：计算射线方向 → 遍历敌人做向量投影碰撞检测 → 命中者走 ApplyHit → 生成光束视觉
func (h *WideBeamHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	// 计算射线方向（从塔指向目标的单位向量）
	dx := target.X - t.X
	dy := target.Y - t.Y
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		return
	}
	dirX := dx / dist
	dirY := dy / dist

	// 从 abilities.json 读取射程倍率（唯一真相源）。
	// rangeMult 让光束射程远于塔的索敌范围，体现"贯穿"特性——
	// 塔只需看到第一个敌人，光束就能打到身后更远处的敌人。
	rangeMult := wideBeamDefaultRangeMul
	if def := getAbilityDef(tower.AbilityWideBeam); def != nil && def.Param > 0 {
		rangeMult = def.Param
	}
	beamLen := t.Range * rangeMult
	endX := t.X + dirX*beamLen
	endY := t.Y + dirY*beamLen
	halfW := wideBeamWidth / 2

	// 对路径上所有敌人造成伤害。
	// 碰撞检测：proj 是沿射线方向的投影（判断前后），perpDist 是垂直距离（判断宽度）。
	// 两个条件同时满足 = 敌人在光束矩形区域内。
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
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
			OnSplashVFX: ctx.OnSplashVFX,
		}, ctx.OnHit)
	})

	// 生成光束视觉效果：加入 BeamPool，由渲染层 draw_beam.go 绘制，
	// Life 从 MaxLife 衰减到 0 实现淡出。Wide=true 触发宽光束专用渲染路径。
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
