// handler_spinaoe.go — 旋转范围伤害攻击方式（自管理）。
// 无弹射物，范围内全体受伤，伤害比例由 param (damageRatio) 配置。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

// SpinAoEHandler 旋转 AoE。
type SpinAoEHandler struct{}

func (h *SpinAoEHandler) SelfManaged() bool { return true }

func (h *SpinAoEHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {
	// spin_aoe 不通过 Fire 触发
}

func (h *SpinAoEHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// 旋转角度持续更新
	// 旋转速度随攻速缩放：基准 3 rad/s 对应攻速 0.3，攻速越快旋转越快
	spinSpeed := 3.0 * (t.AttackSpeed / 0.3)
	t.SpinAngle += dt * spinSpeed
	if t.SpinAngle > 2*math.Pi {
		t.SpinAngle -= 2 * math.Pi
	}

	// 旋转激活计时衰减
	if t.SpinActive > 0 {
		t.SpinActive -= dt
	}

	// 冷却
	t.FireTimer -= dt
	if t.FireTimer > 0 {
		return
	}

	// 从能力配置读取 damageRatio (param 字段)
	damageRatio := 0.5
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilitySpinAoe]; def != nil {
			if def.Param > 0 {
				damageRatio = def.Param
			}
		}
	}

	hasTarget := false
	r := t.Range

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		dist := math.Hypot(e.X-t.X, e.Y-t.Y)
		if dist > r {
			return
		}
		hasTarget = true
		dmg := t.Damage * damageRatio
		ApplyHit(HitInput{
			Tower: t, Target: e, BaseDamage: dmg, Style: ctx.Style,
			Enemies: ctx.Enemies, Projectiles: ctx.Projectiles, OnCC: ctx.OnCC,
			OnSplashVFX: ctx.OnSplashVFX,
		}, ctx.OnHit)
	})

	if hasTarget {
		t.SpinActive = 0.3
		t.FireAnim = 0.4
		t.FireTimer = 1.0 / t.AttackSpeed
		if ctx.OnFire != nil {
			ctx.OnFire(t, ctx.Style)
		}
	}
}
