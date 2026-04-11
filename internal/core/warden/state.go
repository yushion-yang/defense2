// state.go — 战灵公共状态基座。
// WardenState 嵌入到每种战灵类型的状态中，提供位置/移动/战斗/渲染等共享字段和辅助方法。
package warden

import (
	"math"
	"math/rand"

	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
)

// WardenState 战灵公共状态，嵌入到所有类型的 State 中。
type WardenState struct {
	// --- 位置与移动 ---
	X, Y        float64 // 当前像素位置
	MoveSpeed   float64 // 移动速度 px/s（0 = 不移动/瞬移）
	MapWidth    float64 // 地图像素宽度（clamp 上限，0 则回退 1200）
	MapHeight   float64 // 地图像素高度（clamp 上限，0 则回退 540）
	OrbitAngle  float64 // 轨道运动角度（弧度）
	FacingAngle float64 // 朝向角度（弧度，根据移动方向更新）

	// --- 轨道中心平滑 ---
	smoothCX, smoothCY float64 // 平滑后的轨道中心
	smoothInit         bool    // 是否已初始化平滑中心

	// --- 空闲游荡 ---
	wanderX, wanderY float64 // 游荡目标点
	wanderTimer      float64 // 游荡目标切换倒计时

	// --- 战斗（可选，0 表示无攻击能力）---
	BaseDamage      float64 // 基础伤害（Init 设定，不变）
	Damage          float64 // 当前帧有效伤害（= BaseDamage * 强度倍率）
	AttackInterval  float64 // 攻击间隔秒数（0 = 不攻击）
	AttackTimer     float64 // 攻击冷却倒计时
	Range           float64 // 攻击范围像素
	AoERadius       float64 // AoE 半径（0 = 单体）
	ProjectileSpeed float64 // 弹射物速度 px/s（0 = 使用默认 350）

	// --- 阶段机（可选，"" 表示常驻模式）---
	Phase string  // 当前阶段
	Timer float64 // 当前阶段倒计时

	// --- 渲染 ---
	LastTargetX float64 // 上次射击目标 X
	LastTargetY float64 // 上次射击目标 Y
	ShootTimer  float64 // 射击线视觉计时器

	// --- 拖尾 ---
	TrailHistory  [8][2]float64 // 位置历史环形缓冲（最多 8 帧）
	TrailCursor   int           // 当前写入位置
	TrailRecordCD float64       // 记录间隔倒计时
}

// Stateful 由所有战灵 State 结构体实现，用于获取公共基座。
type Stateful interface {
	Base() *WardenState
}

// DescProvider 由支持 HUD 占位符替换的战灵状态实现。
type DescProvider interface {
	DescParams(w *Warden) map[string]string
}

// ── 技能 Owner 接口实现 ──

func (s *WardenState) GetX() float64      { return s.X }
func (s *WardenState) GetY() float64      { return s.Y }
func (s *WardenState) GetRange() float64  { return s.Range }
func (s *WardenState) GetDamage() float64 { return s.Damage }

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

// ApplyStrength 根据战灵感知强度更新有效伤害。
// 公式：Damage = BaseDamage * PerceivedStrength / 100
// 强度 100 = 基准（伤害 = BaseDamage），200 = 双倍。
func (s *WardenState) ApplyStrength(w *Warden) {
	if s.BaseDamage == 0 {
		s.BaseDamage = s.Damage
	}
	str := w.PerceivedStrength
	if str < 1 {
		str = 1 // 避免零伤害
	}
	s.Damage = s.BaseDamage * str / 100
}

// ── 移动 ────────────────────────────────────────

// smoothSpeed 平滑追踪速度（像素/秒）。越大追踪越快。
// 600px/s 确保战灵在 2400px 宽地图上 ~4 秒内跟上敌群。
const smoothSpeed = 600.0

// smoothSnapDist 超过此距离时直接跳到目标中心（防止长距离追踪延迟）。
const smoothSnapDist = 800.0

func (s *WardenState) mapW() float64 {
	if s.MapWidth > 0 {
		return s.MapWidth
	}
	return 1200
}
func (s *WardenState) mapH() float64 {
	if s.MapHeight > 0 {
		return s.MapHeight
	}
	return 540
}

// MoveOrbit 围绕 (cx, cy) 以 idealDist 为理想距离进行轨道运动。
// 当距离目标中心过远时，先直线追赶到轨道半径附近再开始绕转。
func (s *WardenState) MoveOrbit(cx, cy, idealDist, dt float64) {
	// dt-aware 平滑轨道中心
	if !s.smoothInit {
		s.smoothCX = cx
		s.smoothCY = cy
		s.smoothInit = true
	} else {
		dx := cx - s.smoothCX
		dy := cy - s.smoothCY
		dist := math.Hypot(dx, dy)
		if dist > smoothSnapDist {
			s.smoothCX = cx
			s.smoothCY = cy
		} else {
			maxMove := smoothSpeed * dt
			if dist > maxMove {
				s.smoothCX += (dx / dist) * maxMove
				s.smoothCY += (dy / dist) * maxMove
			} else {
				s.smoothCX = cx
				s.smoothCY = cy
			}
		}
	}
	scx, scy := s.smoothCX, s.smoothCY

	if idealDist <= 0 {
		return
	}

	prevX, prevY := s.X, s.Y

	// 首次定位：直接 teleport
	if s.X == 0 && s.Y == 0 {
		s.X = clampF(scx+idealDist, 0, s.mapW())
		s.Y = clampF(scy, 0, s.mapH())
		return
	}

	// 计算当前距离中心的距离
	distToCenter := math.Hypot(s.X-scx, s.Y-scy)
	engageThresh := idealDist * 2.5 // 超过此距离 → 追赶模式

	if distToCenter > engageThresh {
		// ── 追赶模式：直线高速向中心移动 ──
		dx := scx - s.X
		dy := scy - s.Y
		speed := s.MoveSpeed * 1.8 // 追赶时 1.8 倍速
		maxMove := speed * dt
		if distToCenter > maxMove {
			s.X += (dx / distToCenter) * maxMove
			s.Y += (dy / distToCenter) * maxMove
		} else {
			s.X = scx
			s.Y = scy
		}
	} else {
		// ── 轨道模式：正常绕转 ──
		angularSpeed := s.MoveSpeed / idealDist
		s.OrbitAngle += angularSpeed * dt
		targetX := scx + idealDist*math.Cos(s.OrbitAngle)
		targetY := scy + idealDist*math.Sin(s.OrbitAngle)

		dx := targetX - s.X
		dy := targetY - s.Y
		dist := math.Hypot(dx, dy)
		maxMove := s.MoveSpeed * dt
		if dist > maxMove {
			s.X += (dx / dist) * maxMove
			s.Y += (dy / dist) * maxMove
		} else {
			s.X = targetX
			s.Y = targetY
		}
	}

	s.X = clampF(s.X, 0, s.mapW())
	s.Y = clampF(s.Y, 0, s.mapH())
	s.updateFacing(prevX, prevY)
	s.RecordTrail(dt)
}

// Wander 无敌人时缓慢游荡。每 3-5 秒换一个随机目标点，缓慢漂移过去。
func (s *WardenState) Wander(dt float64) {
	// 首次定位：teleport 到地图中心（与 MoveOrbit 一致，防止卡在原点）
	if s.X == 0 && s.Y == 0 {
		s.X = s.mapW() / 2
		s.Y = s.mapH() / 2
	}

	prevX, prevY := s.X, s.Y

	// 首次或到期：选新游荡点
	s.wanderTimer -= dt
	if s.wanderTimer <= 0 || (s.wanderX == 0 && s.wanderY == 0) {
		mw, mh := s.mapW(), s.mapH()
		s.wanderX = 200 + rand.Float64()*(mw-400) // 避免边缘
		s.wanderY = 100 + rand.Float64()*(mh-200)
		s.wanderTimer = 3 + rand.Float64()*2
	}

	// 缓慢移向目标点（1/3 速度）
	dx := s.wanderX - s.X
	dy := s.wanderY - s.Y
	dist := math.Hypot(dx, dy)
	speed := s.MoveSpeed * 0.3
	maxMove := speed * dt
	if dist > maxMove {
		s.X += (dx / dist) * maxMove
		s.Y += (dy / dist) * maxMove
	} else {
		s.X = s.wanderX
		s.Y = s.wanderY
	}

	s.X = clampF(s.X, 0, s.mapW())
	s.Y = clampF(s.Y, 0, s.mapH())
	s.updateFacing(prevX, prevY)
	s.RecordTrail(dt)
}

// updateFacing 根据移动方向平滑更新朝向角度。
func (s *WardenState) updateFacing(prevX, prevY float64) {
	dx := s.X - prevX
	dy := s.Y - prevY
	if math.Hypot(dx, dy) < 0.1 {
		return
	}
	target := math.Atan2(dy, dx)
	// 角度平滑插值（避免突变，每帧趋近 15%）
	diff := target - s.FacingAngle
	// 归一化到 [-π, π]
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	s.FacingAngle += diff * 0.15
}

// RecordTrail 按间隔记录位置到拖尾缓冲。
func (s *WardenState) RecordTrail(dt float64) {
	const trailInterval = 0.04 // 每 40ms 记录一次（~25Hz）
	s.TrailRecordCD -= dt
	if s.TrailRecordCD > 0 {
		return
	}
	s.TrailRecordCD = trailInterval
	// Skip recording zero position (warden not yet placed)
	if s.X == 0 && s.Y == 0 {
		return
	}
	s.TrailHistory[s.TrailCursor] = [2]float64{s.X, s.Y}
	s.TrailCursor = (s.TrailCursor + 1) % len(s.TrailHistory)
}

// ── 战斗 ────────────────────────────────────────

// FindNearest 查找攻击范围内最近的敌人。
// 搜索半径 = max(Range, 250)，确保轨道运动时仍能找到目标。
func (s *WardenState) FindNearest(enemies *enemy.Pool) *enemy.Enemy {
	searchRange := s.Range
	if searchRange < 250 {
		searchRange = 250
	}

	var nearest *enemy.Enemy
	nearestDist := math.MaxFloat64

	enemies.Each(func(e *enemy.Enemy) {
		if e.IsDying() || e.IsSpawning() {
			return
		}
		d := math.Hypot(e.X-s.X, e.Y-s.Y)
		if d <= searchRange && d < nearestDist {
			nearestDist = d
			nearest = e
		}
	})
	return nearest
}

// BasicAttack 对最近敌人发射弹射物。
// 通过弹射物系统处理伤害和视觉，不再直接扣 HP。
// 返回目标敌人（nil 表示无目标）。
func (s *WardenState) BasicAttack(ctx *TickContext) *enemy.Enemy {
	target := s.FindNearest(ctx.Enemies)
	if target == nil {
		return nil
	}

	speed := s.ProjectileSpeed
	if speed <= 0 {
		speed = 350
	}

	// 通过弹射物池发射（伤害/碰撞/渲染由弹射物系统处理）
	if ctx.Projectiles != nil {
		ctx.Projectiles.Fire(s.X, s.Y, target.X, target.Y, s.Damage, speed, 4, target, "warden")
	}

	s.LastTargetX = target.X
	s.LastTargetY = target.Y
	s.ShootTimer = 0.15

	// 音效回调
	if ctx.OnFire != nil {
		ctx.OnFire()
	}

	return target
}

// ApplyDamage 对敌人造成伤害的统一入口。
// 处理：扣 HP → 浮字回调 → 击杀回调。所有战灵伤害都应走此方法。
func ApplyDamage(ctx *TickContext, e *enemy.Enemy, dmg float64, crit bool) {
	if e == nil || !e.Active || e.IsDying() || e.IsSpawning() || dmg <= 0 {
		return
	}
	// 走伤害管线（免疫/减免/阈值/遥测统一处理）
	r := combat.ProcessDamage(combat.DamageInput{
		Target:      e,
		RawDamage:   dmg,
		DamageType:  combat.DmgPhysical,
		SourceLabel: "warden",
	})
	finalDmg := r.FinalDamage
	if r.Blocked {
		finalDmg = 0
	}
	if ctx.OnDamage != nil {
		ctx.OnDamage(e.X, e.Y-10, finalDmg, crit)
	}
	// 不直接设 e.Active=false，由 TickEnemyStatusEffects 安全网调 Pool.Kill()
	// 确保 Pool.Count 正确递减，否则 CheckVictory 永远不触发。
	if r.Killed && e.Active {
		if ctx.OnKill != nil {
			ctx.OnKill(e)
		}
	}
}

// ── 敌群分析 ────────────────────────────────────

// ComputeClusterCenter 计算敌群作战中心。
// 策略：找到邻居最多的敌人（密集点），以它为中心。
// 当敌人分散时，质心可能在空旷处导致战灵绕空气转。
// 用密集点代替质心，确保战灵总是朝最多敌人的地方移动。
func ComputeClusterCenter(enemies *enemy.Pool) (cx, cy float64, count int) {
	const clusterRadius = 200.0 // 密集判定半径

	var alive []*enemy.Enemy
	enemies.Each(func(e *enemy.Enemy) {
		if !e.IsDying() {
			alive = append(alive, e)
		}
	})
	count = len(alive)
	if count == 0 {
		return 0, 0, 0
	}
	if count <= 3 {
		// 少量敌人直接用质心（密集算法开销不值得）
		var sx, sy float64
		for _, e := range alive {
			sx += e.X
			sy += e.Y
		}
		return sx / float64(count), sy / float64(count), count
	}

	// 找邻居最多的敌人
	best := alive[0]
	bestNeighbors := 0
	for _, e := range alive {
		n := 0
		for _, o := range alive {
			if math.Hypot(o.X-e.X, o.Y-e.Y) <= clusterRadius {
				n++
			}
		}
		if n > bestNeighbors {
			bestNeighbors = n
			best = e
		}
	}
	return best.X, best.Y, count
}

// FindDensestEnemy 找到在给定半径内邻居最多的敌人。
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
