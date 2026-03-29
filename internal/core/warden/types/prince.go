// prince.go — 小王子型战灵。
// 小王子是移动型战灵，冲刺穿越敌群造成伤害并留下火焰痕迹。
// 行为循环：空闲（等待攻击间隔）→ 冲刺（向敌群中心冲锋）→ 冷却 → 空闲。
package types

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&princeBehavior{})
}

// PrinceState 小王子的内部状态。
type PrinceState struct {
	warden.WardenState // 嵌入公共基座

	// 冲刺专属
	DashEndX     float64               // 冲刺终点 X
	DashEndY     float64               // 冲刺终点 Y
	DashProgress float64               // 冲刺进度 0-1
	DashStartX   float64               // 冲刺起点 X
	DashStartY   float64               // 冲刺起点 Y
	HitSet       map[*enemy.Enemy]bool // 本次冲刺中已命中的敌人

	// 火焰痕迹
	Trails         []FireTrail // 活跃的火焰痕迹列表
	EffectDPS      float64     // 火焰痕迹每秒伤害
	EffectDuration float64     // 火焰痕迹持续时间（秒）
}

// Base 实现 Stateful 接口。
func (s *PrinceState) Base() *warden.WardenState { return &s.WardenState }

// FireTrail 冲刺结束时留下的火焰痕迹，持续灼烧范围内敌人。
type FireTrail struct {
	X, Y    float64 // 痕迹中心位置
	Life    float64 // 剩余存活时间（秒）
	MaxLife float64 // 最大存活时间（秒）
	Radius  float64 // 灼烧判定半径
	DPS     float64 // 每秒灼烧伤害
}

// princeBehavior 小王子行为实现。
type princeBehavior struct{}

func (b *princeBehavior) Type() string { return "prince" }

func (b *princeBehavior) Init(w *warden.Warden) interface{} {
	return &PrinceState{
		WardenState: warden.WardenState{
			X:              600,
			Y:              270,
			Phase:          "idle",
			Damage:         30,
			AttackInterval: 3,
			MoveSpeed:      600,
			AoERadius:      20,
		},
		EffectDPS:      10,
		EffectDuration: 2,
	}
}

func (b *princeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*PrinceState)
	if !ok {
		return
	}

	// 更新火焰痕迹（所有阶段都需要 tick）
	tickTrails(s, ctx)

	switch s.Phase {
	case "idle":
		s.Timer -= ctx.DT
		if s.Timer <= 0 {
			target := warden.FindClusterCenter(ctx.Enemies, 80.0)
			if target != nil {
				s.DashStartX = s.X
				s.DashStartY = s.Y
				s.DashEndX = target.X
				s.DashEndY = target.Y
				s.DashProgress = 0
				s.HitSet = make(map[*enemy.Enemy]bool)
				s.Phase = "dashing"
			}
		}

	case "dashing":
		tickDash(s, ctx)

	case "cooldown":
		s.Timer -= ctx.DT
		if s.Timer <= 0 {
			s.Phase = "idle"
			s.Timer = 0
		}
	}
}

// tickDash 处理冲刺阶段：插值移动、碰撞检测、击杀回调。
func tickDash(s *PrinceState, ctx *warden.TickContext) {
	dx := s.DashEndX - s.DashStartX
	dy := s.DashEndY - s.DashStartY
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		finishDash(s)
		return
	}

	step := (s.MoveSpeed * ctx.DT) / dist
	s.DashProgress += step

	if s.DashProgress >= 1.0 {
		s.DashProgress = 1.0
	}
	s.X = s.DashStartX + dx*s.DashProgress
	s.Y = s.DashStartY + dy*s.DashProgress

	// 碰撞检测
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if s.HitSet[e] {
			return
		}
		if math.Hypot(e.X-s.X, e.Y-s.Y) < s.AoERadius {
			e.HP -= s.Damage
			s.HitSet[e] = true
			if e.HP <= 0 && ctx.OnKill != nil {
				ctx.OnKill()
			}
		}
	})

	if s.DashProgress >= 1.0 {
		finishDash(s)
	}
}

// finishDash 冲刺结束：在终点留下火焰痕迹，进入冷却。
func finishDash(s *PrinceState) {
	s.Trails = append(s.Trails, FireTrail{
		X:       s.DashEndX,
		Y:       s.DashEndY,
		Life:    s.EffectDuration,
		MaxLife: s.EffectDuration,
		Radius:  s.AoERadius,
		DPS:     s.EffectDPS,
	})
	s.HitSet = nil
	s.Phase = "cooldown"
	s.Timer = 1.0
}

// tickTrails 更新所有火焰痕迹：对范围内敌人造成灼烧伤害，移除过期痕迹。
func tickTrails(s *PrinceState, ctx *warden.TickContext) {
	alive := s.Trails[:0]
	for i := range s.Trails {
		t := &s.Trails[i]
		t.Life -= ctx.DT
		if t.Life <= 0 {
			continue
		}
		dmg := t.DPS * ctx.DT
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-t.X, e.Y-t.Y) < t.Radius {
				e.HP -= dmg
				if e.HP <= 0 && ctx.OnKill != nil {
					ctx.OnKill()
				}
			}
		})
		alive = append(alive, *t)
	}
	s.Trails = alive
}
