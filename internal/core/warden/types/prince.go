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
	X, Y           float64            // 当前像素位置
	Phase          string             // "idle", "dashing", "cooldown"
	Timer          float64            // 当前阶段剩余/倒计时（秒）
	DashEndX       float64            // 冲刺终点 X
	DashEndY       float64            // 冲刺终点 Y
	DashProgress   float64            // 冲刺进度 0-1
	DashStartX     float64            // 冲刺起点 X
	DashStartY     float64            // 冲刺起点 Y
	HitSet         map[*enemy.Enemy]bool // 本次冲刺中已命中的敌人
	Trails         []FireTrail        // 活跃的火焰痕迹列表
	Damage         float64            // 冲刺命中伤害
	MoveSpeed      float64            // 冲刺速度（像素/秒）
	AoERadius      float64            // 冲刺命中判定半径
	EffectDPS      float64            // 火焰痕迹每秒伤害
	EffectDuration float64            // 火焰痕迹持续时间（秒）
	AttackInterval float64            // 攻击间隔（秒）
}

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
		X:              600,
		Y:              270,
		Phase:          "idle",
		Timer:          0,
		Damage:         30,
		AttackInterval: 3,
		MoveSpeed:      600,
		AoERadius:      20,
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
			// 寻找敌群中心（周围敌人最多的敌人）
			target := findClusterCenter(ctx.Enemies)
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
		// 目标太近，直接结束
		finishDash(s)
		return
	}

	// 按速度推进进度
	step := (s.MoveSpeed * ctx.DT) / dist
	s.DashProgress += step

	// 插值当前位置
	if s.DashProgress >= 1.0 {
		s.DashProgress = 1.0
	}
	s.X = s.DashStartX + dx*s.DashProgress
	s.Y = s.DashStartY + dy*s.DashProgress

	// 碰撞检测：命中范围内未击中的敌人
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

	// 冲刺完成
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
		// 范围内敌人受到灼烧伤害
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

// findClusterCenter 寻找周围敌人最多的敌人（敌群中心）。
func findClusterCenter(enemies *enemy.Pool) *enemy.Enemy {
	const clusterRange = 80.0

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
