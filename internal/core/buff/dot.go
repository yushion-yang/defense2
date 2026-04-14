// dot.go — 持续伤害（DoT）计时与伤害计算。
//
// DoT（Damage over Time）buff 不像普通 buff 那样每帧生效，
// 而是按固定间隔（dotInterval，如 0.5 秒）触发一跳伤害。
// buff 的 Value 字段存储的是 DPS（每秒伤害），每跳实际伤害 = DPS * dotInterval。
//
// 核心不变量：TickDoT 必须在 Tick 之前调用。
// 原因：Tick 会移除 Remaining<=0 的过期 buff。如果先 Tick，
// 一个恰好本帧过期的 DoT 会被移除，TickDoT 就统计不到它的最后一跳。
// 正确顺序保证"过期的 DoT 仍然造出最后一跳伤害，然后才被清除"。
package buff

// TickDoT 处理 DoT 伤害的固定间隔累积。
//
// 参数：
//   - dt:          本帧经过的时间（秒）
//   - dotInterval: DoT 跳伤间隔（秒），如 burn=0.5, poison=1.0（来自 balance.json combat.dotTickInterval）
//
// 返回值：
//   - 本帧应造成的总 DoT 伤害（0 表示本帧未触发跳伤）
//
// 流程：
//  1. 先检查是否有任何 CatDoT buff 存在，没有则重置计时器并返回 0（避免无 DoT 时空转计时器）
//  2. 累加帧时间到 dotTimer，不足一个 interval 则跳过
//  3. 触发时消耗一个 interval（保留余数），遍历所有 DoT buff 求和
func (bl *BuffList) TickDoT(dt, dotInterval float64) float64 {
	// 快速检查：是否存在任何 DoT buff。没有则重置计时器避免"幽灵累积"
	// （即敌人曾有 DoT 但已清除，下次再中 DoT 时不应立即触发一跳）
	hasDoT := false
	for i := range bl.active {
		if bl.active[i].Category == CatDoT {
			hasDoT = true
			break
		}
	}
	if !hasDoT {
		bl.dotTimer = 0
		return 0
	}

	// 累积帧间隔时间，未达到触发阈值则本帧不造伤害
	bl.dotTimer += dt
	if bl.dotTimer < dotInterval {
		return 0
	}

	// 触发一跳：消耗一个 interval，保留余数供下次累积
	// （不用取模，因为正常帧率下 dt 远小于 interval，不会一帧跨多跳）
	bl.dotTimer -= dotInterval

	// 遍历所有 DoT buff，累加本跳总伤害
	// 多个 DoT（如同时流血+灼烧+中毒）各自独立贡献伤害
	totalDmg := 0.0
	for i := range bl.active {
		b := &bl.active[i]
		if b.Category == CatDoT && b.Value > 0 {
			totalDmg += b.Value * dotInterval // DPS × 间隔 = 单跳伤害
		}
	}
	return totalDmg
}
