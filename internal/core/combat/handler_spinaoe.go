// handler_spinaoe.go — 旋转范围伤害攻击方式（自管理）。
// 无弹射物，范围内全体受伤，伤害比例由 param (damageRatio) 配置。
//
// spin_aoe 的设计定位是"持续群体伤害"：塔不发射弹射物，而是像旋风一样
// 持续旋转，每次 tick 对射程内所有敌人造成即时伤害。
//
// 与 barrage 的区别：
//   - barrage 是瞬发多弹（对抗 damageCap），spin_aoe 是持续 AoE（对抗数量）
//   - barrage 有追踪弹射物（可弹射/溅射），spin_aoe 是即时伤害（无弹射物）
//   - barrage 锁定单目标连射，spin_aoe 无差别打击范围内所有敌人
//
// 旋转速度 = spinBaseSpeed × (AttackSpeed / spinRefAttackSpeed)，
// 攻速越快旋转越快，既影响视觉效果也影响实际 tick 频率。
//
// 对应精灵：cyclone（旋风造型）。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

const (
	spinBaseSpeed        = 3.0 // 旋转基准速度(rad/s)
	spinRefAttackSpeed   = 0.3 // 旋转速度参考攻速
	spinDefaultDmgRatio  = 0.5 // spin AoE 默认伤害比例
	spinActiveDuration   = 0.3 // 旋转激活指示时长(秒)
	spinFireAnimDuration = 0.4 // 开火动画时长(秒)
	spinMaxShotsPerFrame = 20  // 单帧最大 tick 次数（防止极端攻速卡死）
)

// SpinAoEHandler 旋转 AoE。
type SpinAoEHandler struct{}

func (h *SpinAoEHandler) SelfManaged() bool { return true }

func (h *SpinAoEHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {
	// spin_aoe 不通过 Fire 触发
}

// Tick 每帧更新：持续旋转 + 周期性 AoE 伤害。
// 旋转是纯视觉效果（SpinAngle），伤害判定由 FireTimer 周期控制。
// 两者独立——即使无敌人，塔仍持续旋转（视觉反馈：塔始终活跃）。
func (h *SpinAoEHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// 旋转角度持续更新（纯视觉，不影响伤害判定）。
	// 旋转速度随攻速缩放：基准 3 rad/s 对应攻速 0.3，攻速越快旋转越快。
	spinSpeed := spinBaseSpeed * (t.AttackSpeed / spinRefAttackSpeed)
	t.SpinAngle += dt * spinSpeed
	if t.SpinAngle > 2*math.Pi {
		t.SpinAngle -= 2 * math.Pi
	}

	// 旋转激活计时衰减
	if t.SpinActive > 0 {
		t.SpinActive -= dt
	}

	// 冷却：循环消耗 timer，允许超高攻速每帧多次 tick。
	// 与 barrage 的超高攻速追赶类似，for 循环确保不丢失攻击次数。
	t.FireTimer -= dt
	if t.FireTimer > 0 {
		return
	}

	// 从能力配置读取 damageRatio (param 字段)。
	// damageRatio 控制每次 tick 的伤害 = 塔伤害 × ratio。
	// 因为 spin_aoe 每秒 tick 多次，ratio 通常 < 1 以平衡总 DPS。
	damageRatio := spinDefaultDmgRatio
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilitySpinAoe]; def != nil {
			if def.Param > 0 {
				damageRatio = def.Param
			}
		}
	}

	// for 循环：消耗 FireTimer 欠债，每次循环对范围内所有敌人造成一次伤害。
	// spinMaxShotsPerFrame=20 防止极端攻速卡死。
	// hasTarget 标记：如果范围内无敌人，提前跳出循环节省算力。
	shotInterval := 1.0 / t.AttackSpeed
	fired := false
	for shots := 0; t.FireTimer <= 0 && shots < spinMaxShotsPerFrame; shots++ {
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

		if !hasTarget {
			break
		}
		t.FireTimer += shotInterval
		fired = true
	}

	if fired {
		t.SpinActive = spinActiveDuration
		t.FireAnim = spinFireAnimDuration
		if ctx.OnFire != nil {
			ctx.OnFire(t, ctx.Style)
		}
	}
}
