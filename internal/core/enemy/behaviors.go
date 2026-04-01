// behaviors.go — 敌人行为系统。
// 实现狂暴、治疗光环、自然回血、隐身、分裂、旗手光环等敌人主动行为。
package enemy

import "math"

// HealEvent 治疗事件记录（用于渲染治疗特效）。
type HealEvent struct {
	HealerID int     // 治疗者 ID
	TargetID int     // 被治疗者 ID
	Restored float64 // 实际回复量
	TargetX  float64 // 被治疗者 X 坐标
	TargetY  float64 // 被治疗者 Y 坐标
}

// BehaviorEvents 一帧内行为系统产生的事件（供外部播放音效/VFX）。
type BehaviorEvents struct {
	Heals   []HealEvent // 治疗事件（位置信息用于音效/特效）
	Reveals []RevealEvent // 隐身破解事件
	Regens  int         // 本帧有回血的敌人数（用于判断是否播放 regen 音效）
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
		if e.IsDying() {
			return
		}

		// 狂暴检查（所有敌人，不限于特定 Behavior）
		UpdateBerserk(e)

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
		}
		// splitter 在死亡时触发，不在此处 tick
	})

	return events
}

// tickHealer 治疗兵行为：周期性治疗范围内友军。
func tickHealer(e *Enemy, pool *Pool, dt float64, events *BehaviorEvents) {
	if e.HealPower <= 0 {
		return
	}

	e.HealCooldown -= dt
	if e.HealCooldown > 0 {
		return
	}
	e.HealCooldown = e.HealInterval

	r2 := e.HealRadius * e.HealRadius
	pool.Each(func(other *Enemy) {
		if other == e || !other.Active || other.IsDying() {
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
		restored := math.Min(e.HealPower, other.MaxHP-other.HP)
		other.HP += restored
		events.Heals = append(events.Heals, HealEvent{
			HealerID: e.ID,
			TargetID: other.ID,
			Restored: restored,
			TargetX:  other.X,
			TargetY:  other.Y,
		})
	})
}

// tickStealth 隐身兵行为：计时到期或被击中时破隐。
func tickStealth(e *Enemy, dt float64, events *BehaviorEvents) {
	if !e.Stealthed {
		return
	}
	e.StealthTimer -= dt
	if e.StealthTimer <= 0 || e.HitFlash > 0 {
		e.Stealthed = false
		e.StealthTimer = 0
		events.Reveals = append(events.Reveals, RevealEvent{X: e.X, Y: e.Y})
	}
}

// tickBuffer 旗手行为：每帧对范围内友军施加移速加成。
func tickBuffer(e *Enemy, pool *Pool) {
	if e.BuffRadius <= 0 {
		return
	}
	r2 := e.BuffRadius * e.BuffRadius
	pool.Each(func(other *Enemy) {
		if other == e || !other.Active || other.IsDying() {
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
// 返回成功生成的子体数量。子体继承父体的路径和 PathIndex，
// 血量为 MaxHP * SplitScale，速度 ×1.4。
func HandleSplitterDeath(e *Enemy, pool *Pool) int {
	if e.SplitCount <= 0 {
		return 0
	}

	spawned := 0
	childHP := e.MaxHP * e.SplitScale
	if childHP < 1 {
		childHP = 1
	}
	childSpeed := e.BaseSpeed * 1.4

	for i := 0; i < e.SplitCount; i++ {
		// 子体在父体位置略微偏移
		offsetX := float64(i-e.SplitCount/2) * 6
		child := pool.Spawn(e.X+offsetX, e.Y, childHP, childSpeed, e.PathIndex, e.Archetype, &SpawnConfig{
			HpScale:    1, // 已经计算好绝对值
			SpeedScale: 1, // 已经计算好绝对值
			Radius:     e.Radius * 0.7,
		})
		if child != nil {
			child.Path = e.Path
			spawned++
		}
	}
	return spawned
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
	e.BaseSpeed *= e.BerserkSpeedScale

	// 如果当前未被减速，同步更新当前速度
	if e.SlowTimer <= 0 {
		e.Speed = e.BaseSpeed
	} else {
		// 被减速中：按当前减速倍率重新计算
		e.Speed = e.BaseSpeed * e.SlowFactor
	}
	return true
}

// UpdateHealing 处理治疗光环行为。
// 遍历所有存活敌人中的治疗者，对范围内受伤友军施加治疗。
// 返回本帧发生的所有治疗事件。
func UpdateHealing(enemies []*Enemy, dt float64) []HealEvent {
	var events []HealEvent

	for _, healer := range enemies {
		// 跳过非治疗者或已死亡的
		if healer.HealPower <= 0 || !healer.Active {
			continue
		}

		// 冷却中
		healer.HealCooldown -= dt
		if healer.HealCooldown > 0 {
			continue
		}

		// 重置冷却
		healer.HealCooldown = healer.HealInterval

		// 遍历范围内的友军
		radiusSq := healer.HealRadius * healer.HealRadius
		for _, target := range enemies {
			if !target.Active || target.ID == healer.ID {
				continue
			}

			// 满血不治疗
			if target.HP >= target.MaxHP {
				continue
			}

			// 距离检查
			dx := target.X - healer.X
			dy := target.Y - healer.Y
			distSq := dx*dx + dy*dy
			if distSq > radiusSq {
				continue
			}

			// 施加治疗（不超过最大血量）
			restored := math.Min(healer.HealPower, target.MaxHP-target.HP)
			target.HP += restored

			events = append(events, HealEvent{
				HealerID: healer.ID,
				TargetID: target.ID,
				Restored: restored,
				TargetX:  target.X,
				TargetY:  target.Y,
			})
		}
	}
	return events
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
