// handler_scatter.go — 锥形散射攻击方式。
// 发射 N 颗真实弹丸，各自独立飞行+碰撞检测。
// 同一敌人被多颗命中时合并为一次伤害（由 TickProjectileHits 处理）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// scatter 默认参数
const (
	scatterBasePellets = 3                  // 基础弹丸数
	scatterDefaultSpread = 60               // 默认扇形角度
	scatterPelletR     = 5.0                // 弹丸碰撞半径
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

	// 同一次散射共享 groupID，TickProjectileHits 据此合并命中
	groupID := ctx.Projectiles.NextScatterGroup()

	for i := 0; i < pellets; i++ {
		frac := float64(i) / float64(pellets-1)
		angle := baseAngle - halfSpread + frac*2*halfSpread
		ctx.Projectiles.FireScatterPellet(
			t.X, t.Y, angle, t.Damage, t.Range, speed,
			scatterPelletR, t.InstanceKey, groupID,
		)
	}
}
