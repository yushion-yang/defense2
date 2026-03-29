// core_mech.go — 机甲战灵。
// Core Mech 是移动型战灵，围绕敌群轨道运动并射击。
// 行为循环：定位敌群中心 → 轨道运动 → 攻击最近敌人（支持单体/AoE）。
package types

import (
	"fmt"
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
			Range:          200,
			MoveSpeed:      360,
			AoERadius:      60, // 范围攻击半径（4+敌人时触发）
		},
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *CoreState) DescParams(w *warden.Warden) map[string]string {
	return map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", s.Damage),
		"aoeThreshold":   fmt.Sprintf("%d", coreAoeThreshold),
		"execHpPct":      fmt.Sprintf("%.0f", coreExecHpPct*100),
		"execDmg":        fmt.Sprintf("%.0f", s.Damage*coreExecMulti),
	}
}

const (
	coreOrbitDist    = 110.0
	coreAoeThreshold = 4   // 射程内敌人数 ≥ 此值时切换范围攻击
	coreExecHpPct    = 0.3 // 目标血量 < 30% 触发斩杀
	coreExecMulti    = 1.5 // 斩杀伤害倍率
)

func (b *coreBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*CoreState)
	if !ok {
		return
	}
	dt := ctx.DT
	s.ApplyStrength(w)

	// 1. 移动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, coreOrbitDist, dt)
	} else {
		s.Wander(dt)
	}

	// 2. 攻击（仅有敌人时计时，少于4目标时攻速翻倍）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			interval := s.AttackInterval
			inRange := countInRange(s, ctx)
			if inRange < coreAoeThreshold {
				interval *= 0.5 // 单体模式攻速翻倍
			}
			s.AttackTimer += interval
			coreAttack(s, ctx)
		}
	}

	// 3. 射击线衰减
	s.DecayShootTimer(dt)
}

// countInRange 统计射程内敌人数量。
func countInRange(s *CoreState, ctx *warden.TickContext) int {
	n := 0
	ctx.Enemies.Each(func(e *enemy.Enemy) {
		if math.Hypot(e.X-s.X, e.Y-s.Y) <= s.Range {
			n++
		}
	})
	return n
}

// coreAttack 机甲战灵攻击：斩杀加成 + 智能 AoE 切换。
// 单体模式发射弹射物，AoE 模式对范围内所有敌人各发射一颗弹射物。
func coreAttack(s *CoreState, ctx *warden.TickContext) {
	nearest := s.FindNearest(ctx.Enemies)
	if nearest == nil {
		return
	}

	speed := s.ProjectileSpeed
	if speed <= 0 {
		speed = 400
	}

	inRange := countInRange(s, ctx)

	if inRange >= coreAoeThreshold && s.AoERadius > 0 {
		// AoE 模式：对范围内每个敌人发射弹射物
		tx, ty := nearest.X, nearest.Y
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-tx, e.Y-ty) <= s.AoERadius {
				dmg := s.Damage
				if e.HP < e.MaxHP*coreExecHpPct {
					dmg *= coreExecMulti
				}
				if ctx.Projectiles != nil {
					ctx.Projectiles.Fire(s.X, s.Y, e.X, e.Y, dmg, speed, 4, e, "warden")
				}
			}
		})
	} else {
		// 单体模式：发射弹射物（斩杀加成）
		dmg := s.Damage
		if nearest.HP < nearest.MaxHP*coreExecHpPct {
			dmg *= coreExecMulti
		}
		if ctx.Projectiles != nil {
			ctx.Projectiles.Fire(s.X, s.Y, nearest.X, nearest.Y, dmg, speed, 4, nearest, "warden")
		}
	}

	if ctx.OnFire != nil {
		ctx.OnFire()
	}

	s.LastTargetX = nearest.X
	s.LastTargetY = nearest.Y
	s.ShootTimer = 0.15
}
