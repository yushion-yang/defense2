// core_mech.go — 机甲战灵。
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

// CoreState 机甲战灵的内部状态。
type CoreState struct {
	warden.WardenState // 嵌入公共基座
}

// Base 实现 Stateful 接口。
func (s *CoreState) Base() *warden.WardenState { return &s.WardenState }

// coreBehavior 机甲战灵行为实现。
type coreBehavior struct{}

func (b *coreBehavior) Type() string { return "core" }

func (b *coreBehavior) Init(w *warden.Warden) interface{} {
	return &CoreState{
		WardenState: warden.WardenState{
			Damage:         25,
			AttackInterval: 1.2,
			Range:          160,
			MoveSpeed:      360,
			AoERadius:      0,
		},
	}
}

const coreOrbitDist = 110.0

func (b *coreBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*CoreState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 轨道运动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, coreOrbitDist, dt)
	}

	// 2. 攻击
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		coreAttack(s, ctx)
	}

	// 3. 射击线衰减
	s.DecayShootTimer(dt)
}

// coreAttack 机甲战灵攻击：斩杀加成 + 智能 AoE 切换。
func coreAttack(s *CoreState, ctx *warden.TickContext) {
	nearest := s.FindNearest(ctx.Enemies)
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
		// AoE 攻击
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

	s.LastTargetX = nearest.X
	s.LastTargetY = nearest.Y
	s.ShootTimer = 0.15
}
