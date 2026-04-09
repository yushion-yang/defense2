// handler_scatter.go — 锥形散射攻击方式。
// 发射 N 颗独立穿透弹，各自沿直线飞行并穿透敌人（与环射相同机制）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

const (
	scatterBasePellets   = 3  // 基础弹丸数
	scatterDefaultSpread = 60 // 默认扇形角度（度）
)

// ScatterHandler 锥形散射。
type ScatterHandler struct{}

func (h *ScatterHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
	baseAngle := math.Atan2(target.Y-t.Y, target.X-t.X)
	speed := t.ProjectileSpeed
	if speed <= 0 {
		speed = 400
	}

	// 从能力配置读取 extraPellets 和 spreadAngle
	pellets := scatterBasePellets
	spreadDeg := float64(scatterDefaultSpread)
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable["scatter"]; def != nil {
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			extra := int(def.CalcScale(str))
			pellets += extra
			if def.Param > 0 {
				spreadDeg = def.Param
			}
		}
	}
	halfSpread := spreadDeg / 2 * math.Pi / 180
	shotRange := t.Range

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
