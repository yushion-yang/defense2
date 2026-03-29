// state.go — 战灵公共状态基座。
// WardenState 嵌入到每种战灵类型的状态中，提供位置/移动/战斗/渲染等共享字段和辅助方法。
package warden

import (
	"math"

	"defense2/internal/core/enemy"
)

// WardenState 战灵公共状态，嵌入到所有类型的 State 中。
type WardenState struct {
	// --- 位置与移动 ---
	X, Y       float64 // 当前像素位置
	MoveSpeed  float64 // 移动速度 px/s（0 = 不移动/瞬移）
	OrbitAngle float64 // 轨道运动角度（弧度）

	// --- 轨道中心平滑 ---
	smoothCX, smoothCY float64 // 平滑后的轨道中心
	smoothInit         bool    // 是否已初始化平滑中心

	// --- 战斗（可选，0 表示无攻击能力）---
	Damage         float64 // 单次伤害
	AttackInterval float64 // 攻击间隔秒数（0 = 不攻击）
	AttackTimer    float64 // 攻击冷却倒计时
	Range          float64 // 攻击范围像素
	AoERadius      float64 // AoE 半径（0 = 单体）

	// --- 阶段机（可选，"" 表示常驻模式）---
	Phase string  // 当前阶段
	Timer float64 // 当前阶段倒计时

	// --- 渲染 ---
	LastTargetX float64 // 上次射击目标 X
	LastTargetY float64 // 上次射击目标 Y
	ShootTimer  float64 // 射击线视觉计时器
}

// Stateful 由所有战灵 State 结构体实现，用于获取公共基座。
type Stateful interface {
	Base() *WardenState
}

// CanAttack 返回该战灵是否具有攻击能力。
func (s *WardenState) CanAttack() bool {
	return s.AttackInterval > 0
}

// DecayShootTimer 按 dt 递减射击线计时器。
func (s *WardenState) DecayShootTimer(dt float64) {
	if s.ShootTimer > 0 {
		s.ShootTimer -= dt
	}
}

// smoothLerp 平滑插值系数（越小越平滑，0.05 = 每帧追踪 5%）。
const smoothLerp = 0.05

// MoveOrbit 围绕 (cx, cy) 以 idealDist 为理想距离进行轨道运动。
// 对轨道中心做指数平滑，避免敌群质心跳动导致战灵闪现。
// 首次调用（位置为 0,0）时直接 teleport 到轨道位置。
func (s *WardenState) MoveOrbit(cx, cy, idealDist, dt float64) {
	// 平滑轨道中心
	if !s.smoothInit {
		s.smoothCX = cx
		s.smoothCY = cy
		s.smoothInit = true
	} else {
		s.smoothCX += (cx - s.smoothCX) * smoothLerp
		s.smoothCY += (cy - s.smoothCY) * smoothLerp
	}
	scx, scy := s.smoothCX, s.smoothCY

	// 首次定位：直接 teleport 到目标轨道
	if s.X == 0 && s.Y == 0 {
		s.X = scx + idealDist*math.Cos(s.OrbitAngle)
		s.Y = scy + idealDist*math.Sin(s.OrbitAngle)
		s.X = clampF(s.X, 0, 1200)
		s.Y = clampF(s.Y, 0, 540)
		return
	}

	dx := s.X - scx
	dy := s.Y - scy
	dist := math.Hypot(dx, dy)
	step := s.MoveSpeed * dt

	if dist < 1 {
		s.X = scx + idealDist*math.Cos(s.OrbitAngle)
		s.Y = scy + idealDist*math.Sin(s.OrbitAngle)
	} else if math.Abs(dist-idealDist) > step {
		dir := 1.0
		if dist > idealDist {
			dir = -1.0
		}
		nx := dx / dist
		ny := dy / dist
		s.X += nx * dir * step
		s.Y += ny * dir * step
	} else {
		s.OrbitAngle += step / idealDist
		s.X = scx + idealDist*math.Cos(s.OrbitAngle)
		s.Y = scy + idealDist*math.Sin(s.OrbitAngle)
	}

	s.X = clampF(s.X, 0, 1200)
	s.Y = clampF(s.Y, 0, 540)
}

// FindNearest 查找 Range 范围内最近的敌人。
func (s *WardenState) FindNearest(enemies *enemy.Pool) *enemy.Enemy {
	var nearest *enemy.Enemy
	nearestDist := math.MaxFloat64

	enemies.Each(func(e *enemy.Enemy) {
		d := math.Hypot(e.X-s.X, e.Y-s.Y)
		if d <= s.Range && d < nearestDist {
			nearestDist = d
			nearest = e
		}
	})
	return nearest
}

// BasicAttack 对最近敌人造成 Damage 伤害。
// 返回命中目标（nil 表示未命中）。调用方负责击杀回调。
func (s *WardenState) BasicAttack(ctx *TickContext) *enemy.Enemy {
	target := s.FindNearest(ctx.Enemies)
	if target == nil {
		return nil
	}

	target.HP -= s.Damage
	if target.HP <= 0 && target.Active {
		target.Active = false
		if ctx.OnKill != nil {
			ctx.OnKill()
		}
	}

	s.LastTargetX = target.X
	s.LastTargetY = target.Y
	s.ShootTimer = 0.15
	return target
}

// ComputeClusterCenter 计算所有存活敌人的质心。
func ComputeClusterCenter(enemies *enemy.Pool) (cx, cy float64, count int) {
	var sumX, sumY float64
	enemies.Each(func(e *enemy.Enemy) {
		sumX += e.X
		sumY += e.Y
		count++
	})
	if count == 0 {
		return 0, 0, 0
	}
	return sumX / float64(count), sumY / float64(count), count
}

// FindClusterCenter 寻找周围敌人最多的敌人（敌群中心）。
func FindClusterCenter(enemies *enemy.Pool, clusterRange float64) *enemy.Enemy {
	var best *enemy.Enemy
	bestCount := 0

	enemies.Each(func(e *enemy.Enemy) {
		count := 0
		enemies.Each(func(other *enemy.Enemy) {
			if math.Hypot(other.X-e.X, other.Y-e.Y) <= clusterRange {
				count++
			}
		})
		if count > bestCount {
			bestCount = count
			best = e
		}
	})
	return best
}

// FindDensestEnemy 找到在给定半径内邻居最多的敌人，用于 AoE 中心选择。
func FindDensestEnemy(enemies *enemy.Pool, radius float64) *enemy.Enemy {
	var alive []*enemy.Enemy
	enemies.Each(func(e *enemy.Enemy) {
		alive = append(alive, e)
	})
	if len(alive) == 0 {
		return nil
	}

	var best *enemy.Enemy
	bestCount := -1

	for _, candidate := range alive {
		count := 0
		for _, other := range alive {
			if math.Hypot(other.X-candidate.X, other.Y-candidate.Y) <= radius {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			best = candidate
		}
	}
	return best
}

// clampF 将值限制在 [lo, hi] 范围内。
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
