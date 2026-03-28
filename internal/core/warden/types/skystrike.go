// skystrike.go — 天降型战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 定期在敌人最密集处释放 AoE 打击作为特殊能力。
package types

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&SkystrikeBehavior{})
}

// SkystrikeState 天降的内部状态。
type SkystrikeState struct {
	X, Y           float64 // 当前像素位置
	OrbitAngle     float64 // 当前轨道角度（弧度）
	MoveSpeed      float64 // 移动速度（像素/秒）

	// 普通攻击
	AttackTimer    float64 // 普攻冷却计时器
	Damage         float64 // 普攻伤害
	AttackInterval float64 // 普攻间隔（秒）
	Range          float64 // 普攻范围（像素）

	// AoE 天降打击（特殊能力）
	AoEDamage      float64 // AoE 伤害
	AoEInterval    float64 // AoE 间隔（秒）
	AoERadius      float64 // AoE 半径（像素）
	AoETimer       float64 // AoE 冷却计时器

	// 渲染用字段
	LastTargetX float64 // 上次射击目标 X
	LastTargetY float64 // 上次射击目标 Y
	ShootTimer  float64 // 射击线渲染计时器
	StrikeX     float64 // 上次 AoE 打击位置 X
	StrikeY     float64 // 上次 AoE 打击位置 Y
	StrikeTimer float64 // AoE 视觉效果倒计时
}

// SkystrikeBehavior 天降行为实现。
type SkystrikeBehavior struct{}

func (b *SkystrikeBehavior) Type() string { return "skystrike" }

func (b *SkystrikeBehavior) Init(w *warden.Warden) interface{} {
	return &SkystrikeState{
		X:              600,
		Y:              270,
		MoveSpeed:      320,
		Damage:         18,
		AttackInterval: 1.5,
		Range:          140,
		AoEDamage:      40,
		AoEInterval:    5.0,
		AoERadius:      60,
	}
}

func (b *SkystrikeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*SkystrikeState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 轨道运动
	clusterX, clusterY, enemyCount := computeClusterCenter(ctx.Enemies)
	if enemyCount > 0 {
		skystrikeMoveOrbit(s, clusterX, clusterY, dt)
	}

	// 2. 普通攻击（最近敌人）
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		skystrikeAttack(s, ctx)
	}

	// 3. AoE 天降打击（特殊能力）
	if s.StrikeTimer > 0 {
		s.StrikeTimer -= dt
	}
	s.AoETimer -= dt
	if s.AoETimer <= 0 {
		s.AoETimer += s.AoEInterval
		skystrikeAoE(s, ctx)
	}

	// 4. 射击线计时器衰减
	if s.ShootTimer > 0 {
		s.ShootTimer -= dt
	}
}

// skystrikeMoveOrbit 天降战灵的轨道运动。
const skystrikeOrbitDist = 120.0

func skystrikeMoveOrbit(s *SkystrikeState, cx, cy, dt float64) {
	dx := s.X - cx
	dy := s.Y - cy
	dist := math.Hypot(dx, dy)
	step := s.MoveSpeed * dt

	if dist < 1 {
		s.X = cx + skystrikeOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + skystrikeOrbitDist*math.Sin(s.OrbitAngle)
	} else if math.Abs(dist-skystrikeOrbitDist) > step {
		dir := 1.0
		if dist > skystrikeOrbitDist {
			dir = -1.0
		}
		nx := dx / dist
		ny := dy / dist
		s.X += nx * dir * step
		s.Y += ny * dir * step
	} else {
		s.OrbitAngle += step / skystrikeOrbitDist
		s.X = cx + skystrikeOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + skystrikeOrbitDist*math.Sin(s.OrbitAngle)
	}

	s.X = clampF(s.X, 0, 1200)
	s.Y = clampF(s.Y, 0, 540)
}

// skystrikeAttack 普通攻击最近敌人。
func skystrikeAttack(s *SkystrikeState, ctx *warden.TickContext) {
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

	nearest.HP -= s.Damage
	if nearest.HP <= 0 && nearest.Active {
		nearest.Active = false
		if ctx.OnKill != nil {
			ctx.OnKill()
		}
	}

	s.LastTargetX = nearest.X
	s.LastTargetY = nearest.Y
	s.ShootTimer = 0.15
}

// skystrikeAoE AoE 天降打击（在最密集敌群处释放范围伤害）。
func skystrikeAoE(s *SkystrikeState, ctx *warden.TickContext) {
	center := findDensestEnemy(ctx.Enemies, s.AoERadius)
	if center == nil {
		return
	}

	cx, cy := center.X, center.Y
	s.StrikeX = cx
	s.StrikeY = cy
	s.StrikeTimer = 0.5

	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if math.Hypot(e.X-cx, e.Y-cy) <= s.AoERadius {
			e.HP -= s.AoEDamage
			if e.HP <= 0 && e.Active {
				e.Active = false
				if ctx.OnKill != nil {
					ctx.OnKill()
				}
			}
		}
	})
}

// findDensestEnemy 找到邻居最多的敌人作为 AoE 中心。
func findDensestEnemy(enemies *enemy.Pool, radius float64) *enemy.Enemy {
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
