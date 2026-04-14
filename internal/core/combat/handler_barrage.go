// handler_barrage.go — 连射攻击方式（自管理）。
// 标准冷却触发后进入连射状态，每 burstDelay 秒发射一颗追踪弹，
// 每弹造成 damageRatio 倍伤害。每弹独立走 ApplyHit 管线，有效对抗 damageCap。
//
// 这是 6 种攻击方式中最复杂的一种，实现了两级时序控制：
//   - 外层冷却（FireTimer）：控制连射轮次间隔，由 AttackSpeed 决定
//   - 内层连射（BarrageTimer）：控制单轮内每颗弹的发射间隔（burstDelay=0.08s）
//
// 状态机：
//
//	冷却中（FireTimer > 0）→ 索敌成功 → 进入连射（BarrageBurst > 0）
//	→ 每帧消耗 BarrageTimer → timer<=0 时发射一弹 → BarrageBurst-- → 连射结束
//	→ 重新进入冷却（FireTimer = 1/AttackSpeed）
//
// 对抗 damageCap 的核心设计：
//
//	每颗弹射物命中时独立走 ApplyHit → ApplyDamage 管线，
//	各自受 damageCap 限制。如果 cap=50，3 弹连射总伤害 = 50×3 = 150。
//	这让 barrage 成为对抗高 damageCap 精英怪/Boss 的首选攻击方式。
//
// 超高攻速优化：
//
//	当 shotInterval < dt 时（攻速极快，一帧内应触发多轮连射），
//	直接批量发射所有弹射物，避免因帧率限制丢失攻击次数。
//
// 对应精灵：gatling（加特林枪管造型）。
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
	barrageMaxBurstsPerFrame = 10   // 单帧最大连射轮数（防止极端攻速卡死）
)

// BarrageHandler 连射攻击方式。
type BarrageHandler struct{}

func (h *BarrageHandler) SelfManaged() bool { return true }

func (h *BarrageHandler) Fire(_ *tower.Tower, _ *enemy.Enemy, _ *AttackContext) {
	// barrage 不通过 Fire 触发，由 Tick 自管理
}

// Tick 每帧更新：先处理连射状态（BarrageBurst>0），再处理冷却状态。
// 两阶段互斥——同一帧只会在其中一个分支执行。
func (h *BarrageHandler) Tick(t *tower.Tower, ctx *AttackContext) {
	dt := ctx.DT

	// ── 连射进行中（内层时序：逐弹发射）──
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
		// 无目标时钳制 FireTimer 为 0，防止负值无限累积。
		// 否则长时间无目标后首次攻击会触发"超高攻速追赶"批量发射。
		t.FireTimer = 0
		return
	}

	// 计算弹丸数：从 abilities.json 读取，由 CalcScale(str) 根据力量动态计算。
	// 力量越高弹数越多（str=100→2弹，str=200→3弹...），这是 barrage 的核心成长曲线。
	bullets := barrageDefaultBullets
	if def := getAbilityDef(tower.AbilityBarrage); def != nil {
		str := t.EffectiveStrength()
		bullets = int(math.Floor(def.CalcScale(str)))
		if bullets < 1 {
			bullets = 1
		}
	}

	// 超高攻速追赶：当 FireTimer 欠债超过一个 shotInterval 时，
	// 说明这一帧内应该触发多轮 burst（攻速极快导致冷却在帧间多次归零）。
	// 此时跳过逐帧展开的连射动画，直接批量发射所有弹射物。
	// barrageMaxBurstsPerFrame=10 防止极端攻速卡死主循环。
	shotInterval := 1.0 / t.AttackSpeed
	if shotInterval > 0 && -t.FireTimer > shotInterval {
		extraBursts := int(-t.FireTimer / shotInterval)
		if extraBursts > barrageMaxBurstsPerFrame-1 {
			extraBursts = barrageMaxBurstsPerFrame - 1
		}
		totalBursts := 1 + extraBursts
		damageRatio, _ := barrageParams()
		dmg := t.Damage * damageRatio
		bal := config.GlobalBalance().Combat
		speed := t.ProjectileSpeed
		if speed <= 0 {
			speed = bal.DefaultProjectileSpeed
		}
		t.Angle = math.Atan2(target.Y-t.Y, target.X-t.X)
		for b := 0; b < totalBursts; b++ {
			for i := 0; i < bullets; i++ {
				ctx.Projectiles.Fire(t.X, t.Y, target.X, target.Y, dmg, speed, bal.DefaultProjectileRadius, target, t.InstanceKey)
			}
		}
		t.FireTimer += float64(totalBursts) * shotInterval
		t.FireAnim = barrageFireAnimDuration
		if ctx.OnFire != nil {
			ctx.OnFire(t, ctx.Style)
		}
		return
	}

	// 正常攻速：进入逐帧展开的连射状态。
	// BarrageTimer=0 使得下一帧 Tick 时 timer<=0 条件成立，立即发射首弹。
	// 这比 BarrageTimer=burstDelay 更流畅——玩家看到索敌后立刻开火，无延迟。
	_, burstDelay := barrageParams()
	t.BarrageBurst = bullets
	t.BarrageTarget = target
	t.BarrageTimer = 0

	_ = burstDelay
}

// barrageParams 从能力配置读取 damageRatio 和 burstDelay。
// Param = damageRatio（每弹伤害 = 塔伤害 × ratio），Param2 = burstDelay（弹间间隔秒数）。
// 这两个参数决定了 barrage 的 DPS 特性：低 ratio 高弹数 = 对抗 damageCap 更优。
func barrageParams() (damageRatio, burstDelay float64) {
	damageRatio = barrageDefaultDmgRatio
	burstDelay = barrageDefaultBurstDelay
	if def := getAbilityDef(tower.AbilityBarrage); def != nil {
		if def.Param > 0 {
			damageRatio = def.Param
		}
		if def.Param2 > 0 {
			burstDelay = def.Param2
		}
	}
	return
}
