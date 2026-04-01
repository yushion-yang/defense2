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
	// 从能力配置读取参数
	baseShots := 3
	extraShots := 0
	rangeMult := 1.2
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def, ok := abTable["radial"]; ok {
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			extraShots = int(math.Floor(def.CalcScale(str)))
			if def.Param > 0 {
				rangeMult = def.Param
			}
		}
	}

	totalShots := baseShots + extraShots
	shotRange := t.Range * rangeMult
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = 350
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
