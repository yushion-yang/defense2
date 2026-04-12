// handler_barrage.go — 连射攻击方式（自管理）。
// 标准冷却触发后进入连射状态，每 burstDelay 秒发射一颗追踪弹，
// 每弹造成 damageRatio 倍伤害。每弹独立走 ApplyHit 管线，有效对抗 damageCap。
package combat

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
)

const (
	barrageDefaultBullets    = 2    // 默认弹丸数（强度100）
	barrageDefaultDmgRatio   = 0.5  // 默认每弹伤害比例
	barrageDefaultBurstDelay = 0.08 // 默认每弹间隔（秒）
	barrageFireAnimDuration  = 0.15 // 每弹开火动画时长
)

// BarrageHandler 连射攻击方式。
type BarrageHandler struct{}

func (h *BarrageHandler) SelfManaged() bool { return true }

func (h *BarrageHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {
	// barrage 不通过 Fire 触发，由 Tick 自管理
}

func (h *BarrageHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// ── 连射进行中 ──
	if t.BarrageBurst > 0 {
		t.BarrageTimer -= dt
		if t.BarrageTimer > 0 {
			return
		}

		// 检查连射目标是否仍有效
		target := t.BarrageTarget
		if target == nil || !target.Active || target.IsDying() || target.IsSpawning() {
			// 目标死亡/无效，切换最近敌人
			target = tower.FindNearestEnemy(t, ctx.Enemies)
			t.BarrageTarget = target
		}
		if target == nil {
			// 无可用目标，中止连射
			t.BarrageBurst = 0
			t.BarrageTarget = nil
			t.FireTimer = 1.0 / t.AttackSpeed
			return
		}

		// 更新朝向
		t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)

		// 发射一颗追踪弹
		damageRatio, burstDelay := barrageParams()
		dmg := t.Damage * damageRatio
		bal := config.GlobalBalance().Combat
		speed := t.ProjectileSpeed
		if speed <= 0 {
			speed = bal.DefaultProjectileSpeed
		}
		ctx.Projectiles.Fire(t.X, t.Y, target.X, target.Y, dmg, speed, bal.DefaultProjectileRadius, target, t.InstanceKey)

		t.BarrageBurst--
		t.FireAnim = barrageFireAnimDuration
		if ctx.OnFire != nil {
			ctx.OnFire(t, ctx.Style)
		}

		if t.BarrageBurst > 0 {
			t.BarrageTimer = burstDelay
		} else {
			// 连射结束，进入冷却
			t.BarrageTarget = nil
			t.FireTimer = 1.0 / t.AttackSpeed
		}
		return
	}

	// ── 冷却阶段 ──
	t.FireTimer -= dt
	if t.FireTimer > 0 {
		return
	}

	// 索敌
	target := tower.AcquireTarget(t, ctx.Enemies)
	if target == nil {
		return
	}

	// 计算弹丸数
	bullets := barrageDefaultBullets
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilityBarrage]; def != nil {
			str := 100.0
			if t.Strength != nil {
				str = t.Strength.Effective()
			}
			bullets = int(math.Floor(def.CalcScale(str)))
			if bullets < 1 {
				bullets = 1
			}
		}
	}

	// 进入连射状态（首弹立即发射）
	_, burstDelay := barrageParams()
	t.BarrageBurst = bullets
	t.BarrageTarget = target
	t.BarrageTimer = 0 // 首弹立即触发（下次 Tick 时 timer<=0 会立即发射）

	// 为避免首帧跳过，设一个极小值让下一帧触发
	_ = burstDelay
}

// barrageParams 从能力配置读取 damageRatio 和 burstDelay。
func barrageParams() (damageRatio, burstDelay float64) {
	damageRatio = barrageDefaultDmgRatio
	burstDelay = barrageDefaultBurstDelay
	if abTable := config.GlobalAbilityTable(); abTable != nil {
		if def := abTable[tower.AbilityBarrage]; def != nil {
			if def.Param > 0 {
				damageRatio = def.Param
			}
			if def.Param2 > 0 {
				burstDelay = def.Param2
			}
		}
	}
	return
}
