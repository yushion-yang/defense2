// chain.go — 能量串联型战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 被动增强所有塔的战力，同时主动攻击敌人。
package types

import (
	"fmt"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&ChainBehavior{})
}

// ChainState 能量串联的内部状态。
type ChainState struct {
	X, Y           float64             // 当前像素位置
	OrbitAngle     float64             // 当前轨道角度（弧度）
	AttackTimer    float64             // 攻击冷却计时器
	Damage         float64             // 单次伤害
	AttackInterval float64             // 攻击间隔（秒）
	Range          float64             // 攻击范围（像素）
	MoveSpeed      float64             // 移动速度（像素/秒）
	TowerBonus     float64             // 每座塔的战力加成
	BoostedSet     map[*tower.Tower]bool // 已加成的塔集合
	// 渲染用字段
	LastTargetX float64 // 上次射击目标 X
	LastTargetY float64 // 上次射击目标 Y
	ShootTimer  float64 // 射击线渲染计时器
}

// ChainBehavior 能量串联行为实现。
type ChainBehavior struct{}

func (b *ChainBehavior) Type() string { return "chain" }

func (b *ChainBehavior) Init(w *warden.Warden) interface{} {
	return &ChainState{
		X:              600,
		Y:              270,
		Damage:         15,
		AttackInterval: 2.0,
		Range:          150,
		MoveSpeed:      300,
		TowerBonus:     10,
		BoostedSet:     make(map[*tower.Tower]bool),
	}
}

func (b *ChainBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*ChainState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 塔增强：为新建的塔添加战力临时加成
	ctx.Towers.Each(func(t *tower.Tower) {
		if !s.BoostedSet[t] {
			ensureStrength(t)
			if sd, ok := t.Strength.(*strength.StrengthData); ok {
				sd.SetTemp(fmt.Sprintf("chain_warden_%d", w.ID), s.TowerBonus)
			}
			s.BoostedSet[t] = true
		}
	})

	// 2. 轨道运动（围绕敌群中心）
	clusterX, clusterY, enemyCount := computeClusterCenter(ctx.Enemies)
	if enemyCount > 0 {
		chainMoveOrbit(s, clusterX, clusterY, dt)
	}

	// 3. 攻击最近敌人
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		chainAttack(s, ctx)
	}

	// 4. 射击线计时器衰减
	if s.ShootTimer > 0 {
		s.ShootTimer -= dt
	}
}

// chainMoveOrbit 链战灵的轨道运动（与 core_mech 相同逻辑）。
const chainOrbitDist = 100.0

func chainMoveOrbit(s *ChainState, cx, cy, dt float64) {
	dx := s.X - cx
	dy := s.Y - cy
	dist := math.Hypot(dx, dy)
	step := s.MoveSpeed * dt

	if dist < 1 {
		s.X = cx + chainOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + chainOrbitDist*math.Sin(s.OrbitAngle)
	} else if math.Abs(dist-chainOrbitDist) > step {
		dir := 1.0
		if dist > chainOrbitDist {
			dir = -1.0
		}
		nx := dx / dist
		ny := dy / dist
		s.X += nx * dir * step
		s.Y += ny * dir * step
	} else {
		s.OrbitAngle += step / chainOrbitDist
		s.X = cx + chainOrbitDist*math.Cos(s.OrbitAngle)
		s.Y = cy + chainOrbitDist*math.Sin(s.OrbitAngle)
	}

	s.X = clampF(s.X, 0, 1200)
	s.Y = clampF(s.Y, 0, 540)
}

// chainAttack 链战灵攻击最近敌人。
func chainAttack(s *ChainState, ctx *warden.TickContext) {
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

// ensureStrength 确保塔有 StrengthData（懒初始化）。
func ensureStrength(t *tower.Tower) {
	if t.Strength == nil {
		t.Strength = strength.NewStrengthData()
	}
}
