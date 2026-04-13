// handler_radial.go — 360度环射直线穿透攻击方式。
// 同时向 3+x 个方向发射直线穿透弹，射程 = 塔射程 × rangeMult。
//
// radial 的设计定位是"全向穿透防御"：以塔为中心，360 度等角间隔
// 同时发射穿透弹，每颗弹沿直线飞行并穿透所有命中敌人。
//
// 与 scatter（散射）的区别：
//   - radial 是 360 度全方向（无死角），scatter 是前方扇形（有方向性）
//   - radial 以目标方向为基准角对齐，确保至少有一弹直接指向目标
//
// 与 spin_aoe 的区别：
//   - radial 是瞬发弹射物（有飞行时间），spin_aoe 是即时伤害
//   - radial 弹射物穿透飞行（可打到远处），spin_aoe 只影响射程内
//
// 弹数由 CalcScale(str) 动态计算（最少 3 颗），力量越高方向越密。
// rangeMult 让弹射物飞行距离超过索敌范围，增加覆盖面积。
//
// 对应精灵：nova（新星爆发造型）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// RadialHandler 环射。
type RadialHandler struct{}

// Fire 以塔为中心向 360 度发射穿透弹。
// 流程：读取弹数和射程倍率 → 计算基准角（朝向目标）→ 等角间隔发射 FirePenetrate
func (h *RadialHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	// 从 abilities.json 读取发射数和射程倍率（唯一真相源）
	totalShots := 4
	rangeMult := 1.2
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def, ok := abTable[tower.AbilityRadial]; ok {
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			totalShots = int(math.Floor(def.CalcScale(str)))
			if totalShots < 3 {
				totalShots = 3
			}
			if def.Param > 0 {
				rangeMult = def.Param
			}
		}
	}
	shotRange := t.Range * rangeMult
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = config.GlobalBalance().Combat.DefaultProjectileSpeed
	}

	// 以目标方向为基准角，等角间隔发射直线穿透弹。
	// 基准角对齐目标确保至少一弹精确命中当前目标，
	// 其余弹均匀覆盖其他方向。target 为 nil 时退化为从 0 度开始。
	baseAngle := 0.0
	if target != nil {
		baseAngle = math.Atan2(target.Y-t.Y, target.X-t.X)
	}
	angleStep := 2 * math.Pi / float64(totalShots)
	for i := 0; i < totalShots; i++ {
		angle := baseAngle + float64(i)*angleStep
		endX := t.X + math.Cos(angle)*shotRange
		endY := t.Y + math.Sin(angle)*shotRange

		ctx.Projectiles.FirePenetrate(
			t.X, t.Y, endX, endY,
			t.Damage, speed, t.InstanceKey,
		)
	}
}
