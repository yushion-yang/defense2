// behaviors.go — 敌人行为系统。
// 实现狂暴、治疗光环、自然回血等敌人主动行为。
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
