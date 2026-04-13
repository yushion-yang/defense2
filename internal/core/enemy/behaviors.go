// behaviors.go — 敌人行为系统：每帧驱动的主动行为与能力计时器。
//
// 本文件实现敌人的 8 种主动行为和能力 tick，由 TickBehaviors() 统一入口调用。
// TickBehaviors 在 pipeline 中的位置：movement 之后、tower 索敌射击之前。
//
// ═══════════════════════════════════════════════════════════════════
// 行为执行顺序及其原因
// ═══════════════════════════════════════════════════════════════════
//
//  1. Berserk（狂暴）  — 最先检查，因为狂暴会修改 BaseSpeed，影响后续移速计算
//  2. Regen（自然回血）— 在治疗光环前，确保自回血和光环治疗独立结算
//  3. HealAura（治疗光环）— 需要遍历范围内友军，开销较大，放在 regen 后
//  4. BufferAura（旗手加速光环）— 每帧给范围内友军施加短时 speedUp buff
//  5. Stealth（隐身）— 检查是否被攻击破隐，或自然到期
//  6. 能力计时器（dashOnHit/phaseShift/strDrain/purge）— 独立于行为，所有敌人都检查
//
// 注意：
//   - splitter/deathSpawn 在死亡时触发（pool.go Kill()），不在此处 tick
//   - AbilitySilenced 标记会禁用 healAura/bufferAura/phaseShift/strDrain 的主动效果
//   - 行为 buff 参数存储在 BuffList 中（healAura→Value=power,Value2=radius 等）
//   - HealInterval/HealCooldown 是 Enemy 上的运行时字段（非 buff）
//
// 关联文件：
//   - enemy.go: Enemy struct 定义及 BuffList 查询 helpers
//   - pool.go: Kill() 中触发 splitter/deathSpawn
//   - movement.go: 移速受 berserk/speedUp/dash 影响
//   - tick_ability.go: AbilitySilenced 每帧重置+设置
package enemy

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/buff"
)

const (
	purgeFlashDuration     = 0.4 // 净化闪光时长(秒)
	speedBuffShortDuration = 0.1 // buffer 光环速度buff持续(秒)
)

// HealEvent 治疗事件记录（用于渲染治疗特效）。
type HealEvent struct {
	HealerID int     // 治疗者 ID
	TargetID int     // 被治疗者 ID
	Restored float64 // 实际回复量
	TargetX  float64 // 被治疗者 X 坐标
	TargetY  float64 // 被治疗者 Y 坐标
	Target   *Enemy  // 被治疗者指针（避免外部按 ID 遍历查找）
}

// RegenEvent 回血事件记录（用于渲染回血浮字）。
type RegenEvent struct {
	X, Y     float64 // 敌人坐标
	Restored float64 // 本帧实际回复量
}

// BehaviorEvents 一帧内行为系统产生的所有事件（供 stage.go 播放音效/VFX）。
// 每帧创建一个新实例，TickBehaviors 填充后返回，调用方消费后丢弃。
type BehaviorEvents struct {
	Heals     []HealEvent   // 治疗事件（含治疗者/目标/回复量/坐标）
	Reveals   []RevealEvent // 隐身破解事件（含坐标，用于播放破隐特效）
	Regens    []RegenEvent  // 回血事件（含坐标和回复量，用于浮字）
	Berserks  int           // 本帧刚触发狂暴的敌人数（用于播放狂暴音效）
	HasBuffer bool          // 本帧是否有活跃的旗手 buffer（用于渲染光环 VFX）
}

// RevealEvent 隐身破解事件。
type RevealEvent struct {
	X, Y float64
}

// TickBehaviors 每帧统一执行所有敌人的行为逻辑。
// 在敌人移动之后、塔索敌射击之前调用。
// 返回本帧产生的行为事件（供 stage.go 播放音效/VFX）。
func TickBehaviors(pool *Pool, dt float64) BehaviorEvents {
	var events BehaviorEvents

	// 遍历所有存活敌人，逐个执行行为逻辑。
	// 注意：用 Each 而非 EachActive，因为行为系统需要已在 dying/spawning 中的敌人也被检查跳过。
	pool.Each(func(e *Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}

		// 狂暴检查（所有敌人，不限于特定 Behavior）
		if TickBerserk(e) {
			events.Berserks++
		}

		// 自然回血（从 BuffList 读取 regen Value）
		if b, ok := e.Buffs.Get(buff.IDRegen); ok && b.Value > 0 {
			if healed := TickRegeneration(e, b.Value, dt); healed > 0 {
				events.Regens = append(events.Regens, RegenEvent{X: e.X, Y: e.Y, Restored: healed})
			}
		}

		// 治疗光环（原型 healer 或波次 buff healAura 均可触发）
		if e.HasHealAura() {
			tickHealer(e, pool, dt, &events)
		}

		// 加速光环（原型 buffer 或波次 buff speedAura 均可触发）
		if e.HasBufferAura() {
			tickBuffer(e, pool)
			if !e.AbilitySilenced {
				events.HasBuffer = true
			}
		}

		// 隐身（仅原型 stealth，不通过波次 buff 分配）
		if e.Behavior == BehaviorStealth {
			tickStealth(e, dt, &events)
		}
		// splitter/deathSpawn 在死亡时触发，不在此处 tick

		// ── 能力系统 tick（所有敌人，不限于特定 Behavior 类型）──

		// 受击冲刺计时器衰减（触发在 apply_hit.go 中，此处仅做倒计时）
		if e.DashActiveT > 0 {
			e.DashActiveT -= dt
			if e.DashActiveT <= 0 {
				e.DashActiveT = 0
			}
		}
		if e.DashCooldownT > 0 {
			e.DashCooldownT -= dt
		}

		// 相位偏移状态机（参数从 BuffList 读取：Value=免伤时长, Value2=冷却时间）
		// 冷却→免伤→冷却 循环。免伤期间 IsDamageImmune=true，可被选中但不受伤。
		// 沉默时强制结束免伤并重置冷却。
		if pb, ok := e.Buffs.Get(buff.IDPhaseShift); ok {
			if e.AbilitySilenced {
				// 沉默时强制结束免伤相位
				if e.PhaseActive {
					e.PhaseActive = false
					e.IsDamageImmune = false
					e.PhaseTimer = pb.Value2 // 重置为冷却时间
				}
			} else {
				e.PhaseTimer -= dt
				if e.PhaseTimer <= 0 && !e.PhaseActive {
					// 进入免伤相位（可被选中/命中，但免疫伤害）
					e.PhaseActive = true
					e.IsDamageImmune = true
					e.PhaseTimer = pb.Value // phaseDuration
				} else if e.PhaseActive && e.PhaseTimer <= 0 {
					// 相位结束
					e.PhaseActive = false
					e.IsDamageImmune = false
					e.PhaseTimer = pb.Value2 // phaseCooldown
				}
			}
		}

		// 削强（strengthDrain）：本文件只维护连接计时器状态机，
		// 实际找最近塔 + 施加 debuff + 渲染连接线在 stage.go tick_strDrain 中执行。
		// 状态机：冷却→标记需连接(StrDrainActiveT>0)→stage 找塔→连接中→到期→冷却
		if e.StrDrainRatio > 0 {
			if e.AbilitySilenced {
				// 被沉默时断开连接
				if e.StrDrainActiveT > 0 {
					e.StrDrainActiveT = 0
					e.StrDrainTargetRC = [2]int{}
				}
			} else if e.StrDrainActiveT > 0 {
				// 连接中：衰减持续时间
				e.StrDrainActiveT -= dt
				if e.StrDrainActiveT <= 0 {
					e.StrDrainActiveT = 0
					e.StrDrainTargetRC = [2]int{}
					e.StrDrainTimer = e.StrDrainInterval // 重新进入冷却
				}
			} else {
				// 冷却中
				e.StrDrainTimer -= dt
				if e.StrDrainTimer <= 0 {
					e.StrDrainActiveT = e.StrDrainDuration // 标记需要连接
				}
			}
		}

		// 净化（purge）：周期性清除所有负面效果并短暂免疫。
		// Boss 必带净化（spawner.go 中强制注入）。这是塔防中对抗 CC 堆叠的核心机制。
		if e.PurgeInterval > 0 {
			e.PurgeTimer -= dt
			if e.PurgeTimer <= 0 {
				e.PurgeTimer = e.PurgeInterval
				// 清除所有负面效果（CC/DoT/Debuff）via BuffList
				e.Buffs.ClearByCategory(buff.CatCC, buff.CatDoT, buff.CatDebuff)
				e.Speed = e.BaseSpeed             // 清除减速后恢复速度
				e.ZoneDmgAccum = 0                // 区域伤害不在 BuffList 中
				e.PurgeFlash = purgeFlashDuration // 触发净化脉冲视觉
				// 净化后短暂免疫
				if e.PurgeImmuneDur > 0 {
					e.Buffs.Add(buff.Buff{
						ID:        buff.IDControlImmune,
						Category:  buff.CatDefense,
						Source:    "purge",
						Duration:  e.PurgeImmuneDur,
						Remaining: e.PurgeImmuneDur,
					})
					e.IsControlImmune = true
					e.IsStunImmune = true
					e.IsSlowImmune = true
				}
			}
		}
	})

	return events
}

// tickHealer 治疗兵行为：周期性治疗范围内友军。
// 参数来源：power/radius 从 BuffList healAura buff 读取；HealInterval/HealCooldown 是 Enemy 运行时字段。
// HealPower 作为比例（如 0.05=5%），按目标 MaxHP 计算治疗量，确保治疗量与目标血量成正比。
// 被沉默(AbilitySilenced)时跳过。
func tickHealer(e *Enemy, pool *Pool, dt float64, events *BehaviorEvents) {
	b, ok := e.Buffs.Get(buff.IDHealAura)
	if !ok || e.AbilitySilenced {
		return
	}
	healPower := b.Value
	healRadius := b.Value2
	if healPower <= 0 {
		return
	}

	e.HealCooldown -= dt
	if e.HealCooldown > 0 {
		return
	}
	e.HealCooldown = e.HealInterval

	r2 := healRadius * healRadius
	pool.Each(func(other *Enemy) {
		if !other.Active || other.IsDying() || other.IsSpawning() {
			return
		}
		if other.HP >= other.MaxHP {
			return
		}
		dx := other.X - e.X
		dy := other.Y - e.Y
		if dx*dx+dy*dy > r2 {
			return
		}
		// HealPower 作为比例（如0.05=5%），按目标MaxHP计算治疗量
		healAmount := other.MaxHP * healPower
		restored := math.Min(healAmount, other.MaxHP-other.HP)
		other.HP += restored
		events.Heals = append(events.Heals, HealEvent{
			HealerID: e.ID,
			TargetID: other.ID,
			Restored: restored,
			TargetX:  other.X,
			TargetY:  other.Y,
			Target:   other,
		})
	})
}

// tickStealth 隐身兵行为：被击中或自然到期时破隐。
// BuffList.Tick() in the pipeline handles timer decrement and expiry.
func tickStealth(e *Enemy, dt float64, events *BehaviorEvents) {
	if !e.IsStealthed() {
		// Stealth expired (naturally via BuffList.Tick or hit-break on previous frame).
		// Emit reveal once, then clear Behavior to stop dispatching here.
		events.Reveals = append(events.Reveals, RevealEvent{X: e.X, Y: e.Y})
		e.Behavior = ""
		return
	}
	// Break stealth on hit (HitFlash is set by apply_hit when enemy takes damage)
	if e.HitFlash > 0 {
		e.Buffs.RemoveByID(buff.IDStealth)
		events.Reveals = append(events.Reveals, RevealEvent{X: e.X, Y: e.Y})
		e.Behavior = ""
	}
}

// tickBuffer 旗手行为：每帧对范围内友军施加短时移速 buff。
// 参数来源：radius/amount 从 BuffList bufferAura buff 读取。
// 实现方式：每帧给范围内友军 Add 一个 0.1s 的 speedUp buff（Strongest 模式保留最大值）。
// 如果旗手死亡或被沉默，speedUp 自然过期（0.1s 后消失）。
func tickBuffer(e *Enemy, pool *Pool) {
	b, ok := e.Buffs.Get(buff.IDBufferAura)
	if !ok || e.AbilitySilenced {
		return
	}
	amount := b.Value  // speed up ratio
	radius := b.Value2 // aura radius
	if radius <= 0 {
		return
	}
	r2 := radius * radius
	pool.Each(func(other *Enemy) {
		if other == e || !other.Active || other.IsDying() || other.IsSpawning() {
			return
		}
		dx := other.X - e.X
		dy := other.Y - e.Y
		if dx*dx+dy*dy <= r2 {
			// Short-lived speedUp buff, refreshed each frame. Strongest mode keeps highest value.
			other.Buffs.Add(buff.Buff{
				ID: buff.IDSpeedUp, Category: buff.CatBehavior, Source: "bufferAura",
				Value: amount, Duration: speedBuffShortDuration, Remaining: speedBuffShortDuration,
			})
		}
	})
}

// OnSplitterDeath 处理分裂体死亡：在死亡位置生成子体。
// 返回成功生成的子体列表。子体继承父体的路径和 PathIndex，
// 血量为 MaxHP * SplitScale，速度按 balance 配置缩放。
func OnSplitterDeath(e *Enemy, pool *Pool) []*Enemy {
	if e.SplitCount <= 0 {
		return nil
	}

	bal := config.GlobalBalance()
	childHP := e.MaxHP * e.SplitScale
	if childHP < 1 {
		childHP = 1
	}
	childSpeed := e.BaseSpeed * bal.Split.SpeedScale

	var children []*Enemy
	for i := 0; i < e.SplitCount; i++ {
		// 子体在父体位置略微偏移
		offsetX := float64(i-e.SplitCount/2) * bal.Split.ChildOffset
		child := pool.Spawn(e.X+offsetX, e.Y, childHP, childSpeed, e.PathIndex, e.Archetype, &SpawnConfig{
			HpScale:    1, // 已经计算好绝对值
			SpeedScale: 1, // 已经计算好绝对值
			Radius:     e.Radius * bal.Split.RadiusRatio,
		})
		if child == nil {
			break // 池满
		}
		child.Path = e.Path
		// 分裂子体奖励削减
		if bal.Split.RewardScale > 0 {
			child.RewardScale *= bal.Split.RewardScale
		}
		children = append(children, child)
	}
	return children
}

// TickBerserk 检查并触发狂暴状态（一次性，不可逆）。
// 当血量比例降到阈值以下时，永久提升 BaseSpeed。
// 参数从 BuffList 读取：Value=speedScale（如 1.5=加速 50%），Value2=threshold（如 0.5=50% HP）。
// BerserkTriggered 标记确保只触发一次。
// 触发后如果当前被减速，按 slow factor 重新计算 Speed。
func TickBerserk(e *Enemy) bool {
	// 已触发过
	if e.BerserkTriggered {
		return false
	}

	// 从 BuffList 读取狂暴参数
	b, ok := e.Buffs.Get(buff.IDBerserk)
	if !ok {
		return false
	}
	threshold := b.Value2
	speedScale := b.Value

	// 最大血量为零则跳过
	if e.MaxHP <= 0 {
		return false
	}

	// 检查血量比例是否低于阈值
	ratio := e.HP / e.MaxHP
	if ratio > threshold {
		return false
	}

	// 触发狂暴：永久提升基础速度
	e.BerserkTriggered = true
	e.BaseSpeed *= speedScale

	// 如果当前未被减速，同步更新当前速度
	if !e.IsSlowed() {
		e.Speed = e.BaseSpeed
	} else {
		// 被减速中：按当前减速倍率重新计算
		e.Speed = e.BaseSpeed * e.GetSlowFactor()
	}
	return true
}

// TickRegeneration 处理敌人自然回血。
// regenPerSec is the healing rate from the regen buff's Value.
// 返回本帧实际回复的血量。
func TickRegeneration(e *Enemy, regenPerSec, dt float64) float64 {
	if regenPerSec <= 0 || e.HP >= e.MaxHP {
		return 0
	}

	// 计算实际回复量（不超过最大血量）
	healed := math.Min(regenPerSec*dt, e.MaxHP-e.HP)
	e.HP += healed
	return healed
}

// TickTeleport 处理传送兵定时跳跃：每隔 TeleportInterval 秒跳过 TeleportSkip 个路径段。
// 传送后直接更新 X/Y 到新路径点（瞬移，无过渡动画）。
// 返回 true 表示本帧发生了传送（供 stage 播放传送特效）。
func TickTeleport(e *Enemy, dt float64) bool {
	if e.TeleportInterval <= 0 || e.Path == nil {
		return false
	}
	e.TeleportTimer -= dt
	if e.TeleportTimer > 0 {
		return false
	}
	e.TeleportTimer = e.TeleportInterval

	// 跳过 N 个路径段
	skip := e.TeleportSkip
	if skip <= 0 {
		skip = 1
	}
	newIdx := e.PathIndex + skip
	if newIdx >= len(e.Path) {
		newIdx = len(e.Path) - 1
	}
	e.PathIndex = newIdx
	e.X = e.Path[newIdx].X
	e.Y = e.Path[newIdx].Y
	return true
}
