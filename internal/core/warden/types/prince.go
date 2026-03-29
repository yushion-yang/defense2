// prince.go — 火灵战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 被动：定时召唤火球从虚空冲向敌群密集处，穿透敌人并留下火焰痕迹。
package types

import (
	"fmt"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&princeBehavior{})
}

// PrinceState 火灵战灵的内部状态。
type PrinceState struct {
	warden.WardenState // 嵌入公共基座

	// 火球召唤参数
	FireballInterval float64 // 火球召唤间隔（秒）
	FireballTimer    float64 // 火球召唤倒计时
	FireballDmgRatio float64 // 火球伤害 = 攻击力 × 此比例
	FireballSpeed    float64 // 火球飞行速度 px/s
	FireballRadius   float64 // 火球碰撞/痕迹半径

	// 火焰痕迹
	Trails        []FireTrail // 活跃的火焰痕迹列表
	TrailDpsRatio float64     // 火焰痕迹 DPS = 攻击力 × 此比例
	TrailDuration float64     // 火焰痕迹持续时间（秒）

	// 活跃火球
	Fireballs []Fireball
}

// Base 实现 Stateful 接口。
func (s *PrinceState) Base() *warden.WardenState { return &s.WardenState }

// Fireball 虚空召唤的火球，从屏幕外飞向敌群密集处，穿透路径上的敌人。
type Fireball struct {
	X, Y     float64               // 当前位置
	StartX   float64               // 起点 X
	StartY   float64               // 起点 Y
	EndX     float64               // 终点 X
	EndY     float64               // 终点 Y
	Progress float64               // 飞行进度 0-1
	Speed    float64               // 飞行速度 px/s
	Damage   float64               // 穿透伤害
	Radius   float64               // 碰撞半径
	HitSet   map[*enemy.Enemy]bool // 已命中的敌人
}

// trailTickInterval 火焰痕迹伤害判定周期（秒）。
const trailTickInterval = 0.5

// FireTrail 火球到达后留下的火焰痕迹，持续灼烧范围内敌人。
type FireTrail struct {
	X, Y      float64 // 痕迹中心位置
	Life      float64 // 剩余存活时间（秒）
	MaxLife   float64 // 最大存活时间（秒）
	Radius    float64 // 灼烧判定半径
	DPS       float64 // 每秒灼烧伤害
	TickTimer float64 // 伤害判定倒计时
}

// princeBehavior 火灵战灵行为实现。
type princeBehavior struct{}

func (b *princeBehavior) Type() string { return "prince" }

func (b *princeBehavior) Init(w *warden.Warden) interface{} {
	return &PrinceState{
		WardenState: warden.WardenState{
			Damage:         12,
			AttackInterval: 1.2,
			Range:          140,
			MoveSpeed:      350,
		},
		FireballInterval: 4.0,
		FireballDmgRatio: 2.0, // 火球伤害 = 200% 攻击力
		FireballSpeed:    500,
		FireballRadius:   20,
		TrailDpsRatio:    0.5, // 痕迹 DPS = 50% 攻击力
		TrailDuration:    2.0,
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *PrinceState) DescParams(w *warden.Warden) map[string]string {
	return map[string]string{
		"attackInterval":   fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":           fmt.Sprintf("%.0f", s.Damage),
		"fireballInterval": fmt.Sprintf("%.0f", s.FireballInterval),
		"fireballDmg":      fmt.Sprintf("%.0f", s.Damage*s.FireballDmgRatio),
		"trailDuration":    fmt.Sprintf("%.0f", s.TrailDuration),
		"trailDps":         fmt.Sprintf("%.0f", s.Damage*s.TrailDpsRatio),
	}
}

const princeOrbitDist = 100.0

func (b *princeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*PrinceState)
	if !ok {
		return
	}
	dt := ctx.DT
	s.ApplyStrength(w)

	// 1. 移动（有敌人轨道运动，无敌人游荡）
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, princeOrbitDist, dt)
	} else {
		s.Wander(dt)
	}

	// 2. 普攻（需要射程内有敌人）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			s.AttackTimer += s.AttackInterval
			s.BasicAttack(ctx)
		}
	}

	// 3. 被动：虚空火球（场上有敌人即可，不需要进入攻击范围）
	if count > 0 {
		s.FireballTimer -= dt
		if s.FireballTimer <= 0 {
			s.FireballTimer += s.FireballInterval
			spawnFireball(s, ctx)
			if ctx.OnSpecial != nil {
				ctx.OnSpecial()
			}
		}
	}

	// 4. 更新活跃火球
	tickFireballs(s, ctx)

	// 5. 更新火焰痕迹
	tickTrails(s, ctx)

	// 6. 射击线衰减
	s.DecayShootTimer(dt)
}

// spawnFireball 从战灵身后方向召唤一颗火球飞向敌群密集处。
func spawnFireball(s *PrinceState, ctx *warden.TickContext) {
	target := warden.FindClusterCenter(ctx.Enemies, 80.0)
	if target == nil {
		return
	}

	// 起点：从战灵反方向 200px 处（虚空召唤效果）
	dx := target.X - s.X
	dy := target.Y - s.Y
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}
	startX := s.X - (dx/dist)*200
	startY := s.Y - (dy/dist)*200

	s.Fireballs = append(s.Fireballs, Fireball{
		X:      startX,
		Y:      startY,
		StartX: startX,
		StartY: startY,
		EndX:   target.X,
		EndY:   target.Y,
		Speed:  s.FireballSpeed,
		Damage: s.Damage * s.FireballDmgRatio,
		Radius: s.FireballRadius,
		HitSet: make(map[*enemy.Enemy]bool),
	})
}

// tickFireballs 更新所有活跃火球：移动、穿透伤害、到达后留下痕迹。
func tickFireballs(s *PrinceState, ctx *warden.TickContext) {
	alive := s.Fireballs[:0]
	for i := range s.Fireballs {
		fb := &s.Fireballs[i]
		dx := fb.EndX - fb.StartX
		dy := fb.EndY - fb.StartY
		dist := math.Hypot(dx, dy)
		if dist < 1 {
			continue
		}

		step := (fb.Speed * ctx.DT) / dist
		fb.Progress += step
		if fb.Progress > 1.0 {
			fb.Progress = 1.0
		}
		fb.X = fb.StartX + dx*fb.Progress
		fb.Y = fb.StartY + dy*fb.Progress

		// 穿透伤害
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if fb.HitSet[e] {
				return
			}
			if math.Hypot(e.X-fb.X, e.Y-fb.Y) < fb.Radius {
				warden.ApplyDamage(ctx, e, fb.Damage, false)
				fb.HitSet[e] = true
			}
		})

		if fb.Progress >= 1.0 {
			// 到达终点：留下火焰痕迹
			s.Trails = append(s.Trails, FireTrail{
				X:       fb.EndX,
				Y:       fb.EndY,
				Life:    s.TrailDuration,
				MaxLife: s.TrailDuration,
				Radius:  fb.Radius,
				DPS:     s.Damage * s.TrailDpsRatio,
			})
			continue // 不保留已到达的火球
		}
		alive = append(alive, *fb)
	}
	s.Fireballs = alive
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
		t.TickTimer -= ctx.DT
		if t.TickTimer <= 0 {
			t.TickTimer += trailTickInterval
			dmg := t.DPS * trailTickInterval
			ctx.Enemies.Each(func(e *enemy.Enemy) {
				if math.Hypot(e.X-t.X, e.Y-t.Y) < t.Radius {
					warden.ApplyDamage(ctx, e, dmg, false)
				}
			})
		}
		alive = append(alive, *t)
	}
	s.Trails = alive
}
