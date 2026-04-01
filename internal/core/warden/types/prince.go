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

// Fireball 火球，从最佳打击位置出现，沿直线穿透尽可能多的敌人。
type Fireball struct {
	X, Y     float64               // 当前位置
	StartX   float64               // 起点 X
	StartY   float64               // 起点 Y
	EndX     float64               // 终点 X
	EndY     float64               // 终点 Y
	Progress float64               // 飞行进度 0-1
	Speed    float64               // 飞行速度 px/s
	Damage   float64               // 穿透伤害（攻击力部分）
	HpPctDmg float64               // 额外最大生命值百分比伤害（0.05 = 5%）
	Radius   float64               // 碰撞半径
	HitSet   map[*enemy.Enemy]bool // 已命中的敌人
}

const fireballHpPct = 0.05    // 5% 最大生命值额外伤害
const fireballLineLen = 400.0 // 火球穿透飞行距离

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

// 注意：以下硬编码值应与 config/wardens/wardens.json 保持一致
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
		"fireballHpPct":    fmt.Sprintf("%.0f", fireballHpPct*100),
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

// spawnFireball 在敌群最密集处召唤火球，沿最优方向穿透尽可能多的敌人。
func spawnFireball(s *PrinceState, ctx *warden.TickContext) {
	// 1. 找到最佳打击中心（敌群最密集的敌人位置）
	center := warden.FindDensestEnemy(ctx.Enemies, 80)
	if center == nil {
		return
	}

	// 2. 从该中心扫描 36 方向，找到穿透最多敌人的角度
	bestAngle := 0.0
	bestCount := 0
	var enemies []*enemy.Enemy
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		enemies = append(enemies, e)
	})
	for i := 0; i < 36; i++ {
		a := float64(i) * math.Pi / 18
		count := countOnLine(enemies, center.X, center.Y, a, fireballLineLen, s.FireballRadius)
		if count > bestCount {
			bestCount = count
			bestAngle = a
		}
	}

	// 3. 起点 = 密集中心沿反方向偏移，终点 = 沿方向延伸
	cosA, sinA := math.Cos(bestAngle), math.Sin(bestAngle)
	startX := center.X - cosA*50 // 从中心后方 50px 出现
	startY := center.Y - sinA*50
	endX := center.X + cosA*fireballLineLen
	endY := center.Y + sinA*fireballLineLen

	s.Fireballs = append(s.Fireballs, Fireball{
		X: startX, Y: startY,
		StartX: startX, StartY: startY,
		EndX: endX, EndY: endY,
		Speed:    s.FireballSpeed,
		Damage:   s.Damage * s.FireballDmgRatio,
		HpPctDmg: fireballHpPct,
		Radius:   s.FireballRadius,
		HitSet:   make(map[*enemy.Enemy]bool),
	})
}

// countOnLine 统计沿 (cx,cy) + angle 方向、长 length、宽 radius 的矩形内的敌人数。
func countOnLine(enemies []*enemy.Enemy, cx, cy, angle, length, width float64) int {
	cosA, sinA := math.Cos(angle), math.Sin(angle)
	count := 0
	for _, e := range enemies {
		if !e.Active || e.HP <= 0 {
			continue
		}
		dx, dy := e.X-cx, e.Y-cy
		along := dx*cosA + dy*sinA
		perp := math.Abs(-dx*sinA + dy*cosA)
		if along >= -50 && along <= length && perp <= width {
			count++
		}
	}
	return count
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

		// 穿透伤害 = 攻击力伤害 + 5% MaxHP（boss 免疫百分比部分）
		// 注意：此处手动检查 e.Boss 跳过百分比伤害，而非设置 DamageInput.IsPercentHP，
		// 因为 warden.ApplyDamage 固定 IsPercentHP=false 且火球混合了固定+百分比两部分伤害。
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if fb.HitSet[e] {
				return
			}
			if math.Hypot(e.X-fb.X, e.Y-fb.Y) < fb.Radius {
				dmg := fb.Damage
				if fb.HpPctDmg > 0 && !e.Boss {
					dmg += e.MaxHP * fb.HpPctDmg
				}
				warden.ApplyDamage(ctx, e, dmg, false)
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
