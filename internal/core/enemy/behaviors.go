// behaviors.go — 敌人行为系统。
// 实现狂暴、治疗光环、自然回血、隐身、分裂、旗手光环等敌人主动行为。
package enemy

import (
	"math"

	"defense2/internal/config"
	"defense2/internal/core/buff"
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

// BehaviorEvents 一帧内行为系统产生的事件（供外部播放音效/VFX）。
type BehaviorEvents struct {
	Heals     []HealEvent   // 治疗事件
	Reveals   []RevealEvent // 隐身破解事件
	Regens    int           // 本帧有回血的敌人数
	Berserks  int           // 本帧刚触发狂暴的敌人数
	HasBuffer bool          // 本帧是否有活跃的旗手 buffer
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

	// Phase 1: 清除上一帧的 SpeedBuff（每帧由 buffer 重新写入）
	pool.Each(func(e *Enemy) {
		e.SpeedBuff = 0
	})

	// Phase 2: 执行各行为
	pool.Each(func(e *Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}

		// 狂暴检查（所有敌人，不限于特定 Behavior）
		if UpdateBerserk(e) {
			events.Berserks++
		}

		// 自然回血（所有配置了 RegenPerSec 的敌人，包括 regenerator 行为）
		if e.RegenPerSec > 0 {
			if UpdateRegeneration(e, dt) > 0 {
				events.Regens++
			}
		}

		switch e.Behavior {
		case "healer":
			tickHealer(e, pool, dt, &events)
		case "stealth":
			tickStealth(e, dt, &events)
		case "buffer":
			tickBuffer(e, pool)
			if e.BuffRadius > 0 && !e.AbilitySilenced {
				events.HasBuffer = true
			}
		}
		// splitter/deathSpawn 在死亡时触发，不在此处 tick

		// ── 能力系统 tick ──

		// 受击冲刺计时器衰减
		if e.DashActiveT > 0 {
			e.DashActiveT -= dt
			if e.DashActiveT <= 0 {
				e.DashActiveT = 0
			}
		}
		if e.DashCooldownT > 0 {
			e.DashCooldownT -= dt
		}

		// 相位偏移
		if e.PhaseCooldown > 0 && !e.AbilitySilenced {
			e.PhaseTimer -= dt
			if e.PhaseTimer <= 0 && !e.PhaseActive {
				// 进入免伤相位（可被选中/命中，但免疫伤害）
				e.PhaseActive = true
				e.IsDamageImmune = true
				e.PhaseTimer = e.PhaseDuration
			} else if e.PhaseActive && e.PhaseTimer <= 0 {
				// 相位结束
				e.PhaseActive = false
				e.IsDamageImmune = false
				e.PhaseTimer = e.PhaseCooldown
			}
		}

		// 削强：维护连接计时（实际找塔+施加/移除在 stage.go 中执行）
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

		// 净化
		if e.PurgeInterval > 0 {
			e.PurgeTimer -= dt
			if e.PurgeTimer <= 0 {
				e.PurgeTimer = e.PurgeInterval
				// 清除所有负面效果（CC/DoT/Debuff）via BuffList
				e.Buffs.ClearByCategory(buff.CatCC, buff.CatDoT, buff.CatDebuff)
				e.Speed = e.BaseSpeed // 清除减速后恢复速度
				e.ZoneDmgAccum = 0    // 区域伤害不在 BuffList 中
				e.PurgeFlash = 0.4    // 触发净化脉冲视觉
				// 净化后短暂免疫
				if e.PurgeImmuneDur > 0 {
					e.Buffs.Add(buff.Buff{
						ID:        "controlImmune",
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
func tickHealer(e *Enemy, pool *Pool, dt float64, events *BehaviorEvents) {
	if e.HealPower <= 0 || e.AbilitySilenced {
		return
	}

	e.HealCooldown -= dt
	if e.HealCooldown > 0 {
		return
	}
	e.HealCooldown = e.HealInterval

	r2 := e.HealRadius * e.HealRadius
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
		healAmount := other.MaxHP * e.HealPower
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
		e.Buffs.RemoveByID("stealth")
		events.Reveals = append(events.Reveals, RevealEvent{X: e.X, Y: e.Y})
		e.Behavior = ""
	}
}

// tickBuffer 旗手行为：每帧对范围内友军施加移速加成。
func tickBuffer(e *Enemy, pool *Pool) {
	if e.BuffRadius <= 0 || e.AbilitySilenced {
		return
	}
	r2 := e.BuffRadius * e.BuffRadius
	pool.Each(func(other *Enemy) {
		if other == e || !other.Active || other.IsDying() || other.IsSpawning() {
			return
		}
		dx := other.X - e.X
		dy := other.Y - e.Y
		if dx*dx+dy*dy <= r2 {
			// 取最高加成（多个 buffer 不叠加，取最大值）
			if e.BuffAmount > other.SpeedBuff {
				other.SpeedBuff = e.BuffAmount
			}
		}
	})
}

// HandleSplitterDeath 处理分裂体死亡：在死亡位置生成子体。
// 返回成功生成的子体列表。子体继承父体的路径和 PathIndex，
// 血量为 MaxHP * SplitScale，速度按 balance 配置缩放。
func HandleSplitterDeath(e *Enemy, pool *Pool) []*Enemy {
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
		children = append(children, child)
	}
	return children
}

// UpdateBerserk 检查并触发狂暴状态。
// 当血量比例降到阈值以下时，永久提升移动速度。
// 返回 true 表示本次刚触发狂暴。
func UpdateBerserk(e *Enemy) bool {
	// 未配置狂暴或已触发过
	if e.BerserkThreshold <= 0 || e.BerserkTriggered {
		return false
	}

	// 最大血量为零则跳过
	if e.MaxHP <= 0 {
		return false
	}

	// 检查血量比例是否低于阈值
	ratio := e.HP / e.MaxHP
	if ratio > e.BerserkThreshold {
		return false
	}

	// 触发狂暴：永久提升基础速度
	e.BerserkTriggered = true
	e.Buffs.Add(buff.Buff{ID: "berserk", Category: buff.CatBehavior, Source: "archetype", Duration: -1, Remaining: -1})
	e.BaseSpeed *= e.BerserkSpeedScale

	// 如果当前未被减速，同步更新当前速度
	if !e.IsSlowed() {
		e.Speed = e.BaseSpeed
	} else {
		// 被减速中：按当前减速倍率重新计算
		e.Speed = e.BaseSpeed * e.GetSlowFactor()
	}
	return true
}

// UpdateRegeneration 处理敌人自然回血。
// 返回本帧实际回复的血量。
func UpdateRegeneration(e *Enemy, dt float64) float64 {
	if e.RegenPerSec <= 0 || e.HP >= e.MaxHP {
		return 0
	}

	// 计算实际回复量（不超过最大血量）
	healed := math.Min(e.RegenPerSec*dt, e.MaxHP-e.HP)
	e.HP += healed
	return healed
}

// UpdateTeleport 处理传送兵定时跳跃。
// 返回 true 表示本帧发生了传送。
func UpdateTeleport(e *Enemy, dt float64) bool {
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
