// core_mech.go — 战斗机甲型战灵。
// Core Mech 是移动型战灵，围绕敌群轨道运动并射击。
// 行为循环：定位敌群中心 → 轨道运动 → 攻击最近敌人（支持单体/AoE）。
package types

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&coreBehavior{})
}

// CoreState 战斗机甲的内部状态。
type CoreState struct {
	X, Y           float64 // 当前像素位置
	OrbitAngle     float64 // 弧度，当前轨道角度
	AttackTimer    float64 // 攻击冷却计时器
	Damage         float64 // 基础伤害
	AttackInterval float64 // 攻击间隔（秒）
	Range          float64 // 攻击范围（像素）
	MoveSpeed      float64 // 移动速度（像素/秒）
	AoERadius      float64 // AoE 半径（0 表示无 AoE）
	// 渲染用字段
	LastTargetX float64 // 上次射击目标 X
	LastTargetY float64 // 上次射击目标 Y
	ShootTimer  float64 // 射击线渲染计时器
}

// coreBehavior 战斗机甲行为实现。
type coreBehavior struct{}

func (b *coreBehavior) Type() string { return "core" }

func (b *coreBehavior) Init(w *warden.Warden) interface{} {
	return &CoreState{
		X:              600,
		Y:              270,
		Damage:         25,
		AttackInterval: 1.2,
		Range:          160,
		MoveSpeed:      360,
		AoERadius:      0,
	}
}

func (b *coreBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*CoreState)
	if !ok {
		return
	}
	dt := ctx.DT

	// --- 1. 计算敌群中心 ---
	clusterX, clusterY, enemyCount := computeClusterCenter(ctx.Enemies)

	// --- 2. 轨道运动 ---
	if enemyCount > 0 {
		moveOrbit(s, clusterX, clusterY, dt)
	}

	// --- 3. 攻击 ---
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		performAttack(s, ctx)
	}

	// --- 4. 射击线计时器衰减 ---
	if s.ShootTimer > 0 {
		s.ShootTimer -= dt
	}
}

// computeClusterCenter 计算所有存活敌人的质心。
func computeClusterCenter(enemies *enemy.Pool) (cx, cy float64, count int) {
	var sumX, sumY float64
	enemies.Each(func(e *enemy.Enemy) {
		sumX += e.X
		sumY += e.Y
		count++
	})
	if count == 0 {
		return 0, 0, 0
	}
	cx = sumX / float64(count)
	cy = sumY / float64(count)
	return cx, cy, count
}

// moveOrbit 维持与敌群中心的理想距离并进行轨道运动。
const idealOrbitDist = 110.0

func moveOrbit(s *CoreState, cx, cy, dt float64) {
	dx := s.X - cx
	dy := s.Y - cy
	dist := math.Hypot(dx, dy)
	step := s.MoveSpeed * dt

	if dist < 1 {
		// 几乎重合，先偏移到理想距离
		s.X = cx + idealOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + idealOrbitDist*math.Sin(s.OrbitAngle)
	} else if math.Abs(dist-idealOrbitDist) > step {
		// 距离偏差大于一步，向理想距离移动
		dir := 1.0
		if dist > idealOrbitDist {
			dir = -1.0
		}
		nx := dx / dist
		ny := dy / dist
		s.X += nx * dir * step
		s.Y += ny * dir * step
	} else {
		// 在理想距离附近，进行轨道旋转
		s.OrbitAngle += step / idealOrbitDist
		s.X = cx + idealOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + idealOrbitDist*math.Sin(s.OrbitAngle)
	}

	// 限制在屏幕范围内
	s.X = clampF(s.X, 0, 1200)
	s.Y = clampF(s.Y, 0, 540)
}

// performAttack 寻找最近敌人并攻击。
func performAttack(s *CoreState, ctx *warden.TickContext) {
	var nearest *enemy.Enemy
	nearestDist := math.MaxFloat64

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		d := math.Hypot(e.X-s.X, e.Y-s.Y)
		if d <= s.Range && d < nearestDist {
			nearestDist = d
			nearest = e
		}
	})

	if nearest == nil {
		return
	}

	// 统计射程内敌人数量
	inRange := 0
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if math.Hypot(e.X-s.X, e.Y-s.Y) <= s.Range {
			inRange++
		}
	})

	dmg := s.Damage
	// 斩杀加成：目标血量低于 30% 时伤害 ×1.5
	if nearest.HP < nearest.MaxHP*0.3 {
		dmg *= 1.5
	}

	if inRange >= 4 && s.AoERadius > 0 {
		// AoE 攻击：以目标为中心，对 AoE 半径内所有敌人造成伤害
		tx, ty := nearest.X, nearest.Y
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-tx, e.Y-ty) <= s.AoERadius {
				aoeDmg := s.Damage
				if e.HP < e.MaxHP*0.3 {
					aoeDmg *= 1.5
				}
				e.HP -= aoeDmg
				if e.HP <= 0 && e.Active {
					e.Active = false
					if ctx.OnKill != nil {
						ctx.OnKill()
					}
				}
			}
		})
	} else {
		// 单体攻击
		nearest.HP -= dmg
		if nearest.HP <= 0 && nearest.Active {
			nearest.Active = false
			if ctx.OnKill != nil {
				ctx.OnKill()
			}
		}
	}

	// 设置渲染用射击线数据
	s.LastTargetX = nearest.X
	s.LastTargetY = nearest.Y
	s.ShootTimer = 0.15
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
