// handler_scatter.go — 锥形散射攻击方式。
// 发射 N 颗独立穿透弹，各自沿直线飞行并穿透敌人（与环射相同机制）。
//
// scatter 的设计定位是"扇形穿透覆盖"：以目标方向为中心轴，
// 在 spreadDeg 度扇形内均匀发射 N 颗穿透弹。每颗弹独立飞行，
// 可穿透路径上的所有敌人（FirePenetrate）。
//
// 与 radial（环射）的区别：
//   - scatter 是前方扇形（有方向性），适合打一条路线上的密集敌群
//   - radial 是 360 度全方向，适合被包围时的全向防御
//
// 弹丸数由 CalcScale(str) 动态计算——力量越高弹数越多，覆盖面积越大。
// 散布角度（spreadDeg）从 abilities.json 的 Param 字段读取。
//
// 对应精灵：shotgun（霰弹枪造型）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// ScatterHandler 锥形散射。
type ScatterHandler struct{}

// Fire 向目标方向发射扇形穿透弹群。
// 流程：计算基准角度 → 读取弹数和散布角 → 等角间隔生成每颗弹的飞行方向 → FirePenetrate
func (h *ScatterHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	baseAngle := math.Atan2(target.Y-t.Y, target.X-t.X)
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = config.GlobalBalance().Combat.DefaultProjectileSpeed
	}

	// 从 abilities.json 读取弹丸数和散布角度（唯一真相源）
	pellets := 3 // safety fallback
	spreadDeg := 60.0
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilityScatter]; def != nil {
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			pellets = int(math.Floor(def.CalcScale(str)))
			if pellets < 2 {
				pellets = 2
			}
			if def.Param > 0 {
				spreadDeg = def.Param
			}
		}
	}
	halfSpread := spreadDeg / 2 * math.Pi / 180
	shotRange := t.Range

	// 等角间隔发射：第一颗弹在扇形左边缘，最后一颗在右边缘，中间均匀分布。
	// frac 从 0 到 1 线性变化，angle 从 baseAngle-halfSpread 到 baseAngle+halfSpread。
	for i := 0; i < pellets; i++ {
		frac := float64(i) / float64(pellets-1)
		angle := baseAngle - halfSpread + frac*2*halfSpread
		endX := t.X + math.Cos(angle)*shotRange
		endY := t.Y + math.Sin(angle)*shotRange

		ctx.Projectiles.FirePenetrate(
			t.X, t.Y, endX, endY,
			t.Damage, speed, t.InstanceKey,
		)
	}
}
