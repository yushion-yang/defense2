// handler_radial.go — 360度环射直线穿透攻击方式。
// 同时向 3+x 个方向发射直线穿透弹，射程 = 塔射程 × rangeMult。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// RadialHandler 环射。
type RadialHandler struct{}

func (h *RadialHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	bal := config.GlobalBalance()
	// 从能力配置读取总发射数和射程倍率（与 bounce 同模式：CalcScale = 总数）
	totalShots := bal.Combat.RadialBaseShots // fallback
	rangeMult := bal.Combat.RadialRangeMult
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

	// 以目标方向为基准角，等角间隔发射直线穿透弹
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
