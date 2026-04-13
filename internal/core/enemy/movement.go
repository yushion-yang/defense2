// movement.go — 敌人沿路径点列表移动。
//
// 纯函数，无渲染/DOM 依赖。由 pipeline orchestrator 每帧调用。
//
// 路径跟随逻辑：
//  1. 确定行进路径（优先 e.Path，回退到 fallbackWaypoints）
//  2. 检查移动阻止条件：IsDummy / 眩晕(stun) / 定身(root)
//  3. 计算有效速度：base * slowFactor * (1+speedUp) * (1+dashBoost)
//  4. 向当前目标路径点移动，到达后切换下一个
//  5. 走完所有路径点 → 返回 true（敌人抵达基地，stage 层扣生命值）
//
// 速度修正优先级：
//   - BaseSpeed：原始速度（可被 berserk 永久修改）
//   - Speed：= BaseSpeed * slowFactor（由 TickStatusEffects 维护）
//   - SpeedUp：BufferAura 的短时加速（叠加乘算）
//   - DashActiveT：受击冲刺的临时加速（叠加乘算）
//
// 关联文件：
//   - enemy.go: Speed/BaseSpeed 字段，GetSpeedUp() 方法
//   - behaviors.go: berserk 修改 BaseSpeed，dash 设置 DashActiveT
//   - gamemap/gamemap.go: Point 和路径定义
package enemy

import (
	"math"

	"defense2/internal/core/gamemap"
)

// MoveAlongPath 驱动敌人向下一个路径点移动。
// 到达路径终点时返回 true（表示该敌人抵达基地，stage 层应扣除生命值并 KillImmediate）。
// fallbackWaypoints 是默认路径（单路径地图使用），多路径地图时 e.Path 已在 Spawn 后赋值。
func MoveAlongPath(e *Enemy, fallbackWaypoints []gamemap.Point, dt float64) bool {
	// 木桩怪/静止敌人：不移动、不到达终点
	// BuffList.Tick handles stun/root countdown; no manual decrement needed.
	if e.IsDummy || (e.BaseSpeed == 0 && e.Speed == 0) {
		return false
	}

	// 确定行进路径（优先使用敌人自带路径）
	waypoints := e.Path
	if len(waypoints) == 0 {
		waypoints = fallbackWaypoints
	}

	if e.PathIndex >= len(waypoints) {
		e.ReachedEnd = true
		return true
	}

	// 眩晕中：不移动（BuffList.Tick handles countdown）
	if e.IsStunned() {
		return false
	}

	// 定身中：不移动（BuffList.Tick handles countdown）
	if e.IsRooted() {
		return false
	}

	// 计算有效速度：e.Speed 已包含 slow factor，再叠加 speedUp 和 dash 加成
	speed := e.Speed
	if su := e.GetSpeedUp(); su > 0 {
		speed *= (1 + su)
	}
	if e.DashActiveT > 0 {
		speed *= (1 + e.DashSpeedBoost)
	}

	target := waypoints[e.PathIndex]
	dx := target.X - e.X
	dy := target.Y - e.Y
	dist := math.Hypot(dx, dy)
	step := speed * dt

	if dist <= step {
		// 足够接近，吸附到路径点并切换下一个
		e.X = target.X
		e.Y = target.Y
		e.PathIndex++
		if e.PathIndex >= len(waypoints) {
			e.ReachedEnd = true
			return true
		}
	} else {
		// 沿方向向量前进一步
		e.X += (dx / dist) * step
		e.Y += (dy / dist) * step
	}
	return false
}
