// core_mech.go — 机甲战灵。
// Core Mech 是移动型战灵，围绕敌群轨道运动并射击。
// 行为循环：定位敌群中心 → 轨道运动 → 攻击最近敌人（支持单体/AoE）。
package types

import (
	"fmt"
	"math"

	"defense2/internal/config"
	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&coreBehavior{})
}

// CoreState 机甲战灵的内部状态。
type CoreState struct {
	warden.WardenState // 嵌入公共基座

	// 类型特定参数（从配置读取）
	OrbitDist    float64 // 围绕敌群的轨道距离
	AoeThreshold int     // 射程内敌人数 ≥ 此值时切换范围攻击
	ExecHpPct    float64 // 目标血量 < 此百分比时触发秒杀
}

// Base 实现 Stateful 接口。
func (s *CoreState) Base() *warden.WardenState { return &s.WardenState }

// coreBehavior 机甲战灵行为实现。
type coreBehavior struct{}

func (b *coreBehavior) Type() string { return "core" }

// Init 初始化机甲战灵。
// 硬编码值作为 fallback，与 wardens.json 中 core 的配置保持一致。
// JSON 覆盖链：Init 设默认 → NewWarden 用 JSON 覆盖基础属性 → ParamOr 读配置参数。
func (b *coreBehavior) Init(w *warden.Warden) any {
	p := w.Params
	return &CoreState{
		WardenState: warden.WardenState{
			Damage:         15,
			AttackInterval: 1.0,
			Range:          160,
			MoveSpeed:      360,
			AoERadius:      warden.ParamOr(p, "aoeRadius", 60), // 范围攻击半径（4+敌人时触发）
		},
		OrbitDist:    warden.ParamOr(p, "orbitDist", 110.0),
		AoeThreshold: warden.ParamOrInt(p, "aoeThreshold", 4),
		ExecHpPct:    warden.ParamOr(p, "execHpPct", 0.15),
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *CoreState) DescParams(w *warden.Warden) map[string]string {
	return map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", s.Damage),
		"aoeThreshold":   fmt.Sprintf("%d", s.AoeThreshold),
		"execHpPct":      fmt.Sprintf("%.0f", s.ExecHpPct*100),
	}
}

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
		s.MoveOrbit(cx, cy, s.OrbitDist, dt)
	} else {
		s.Wander(dt)
	}

	// 2. 攻击（仅有敌人时计时，少于4目标时攻速翻倍）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			interval := s.AttackInterval
			inRange := countInRange(s, ctx)
			if inRange < s.AoeThreshold {
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

// coreExecDmg 已废弃：斩杀判定移至弹射物命中时（tick_combat.go processHit）。
// 保留此函数签名以便编译通过，但不再使用发射时计算的伤害值。
// 返回基础伤害，斩杀判定由弹射物的 ExecuteHpPct 字段在命中时触发。
func coreExecDmg(_ *enemy.Enemy, baseDmg, _ float64) float64 {
	return baseDmg
}

// coreAttack 机甲战灵攻击：秒杀（对 Boss 无效）+ 智能 AoE 切换。
// 单体模式发射弹射物，AoE 模式对射程内所有敌人各发射一颗弹射物。
// 斩杀判定在命中时触发（通过 ExecuteHpPct 字段），而非发射时计算。
func coreAttack(s *CoreState, ctx *warden.TickContext) {
	nearest := s.FindNearest(ctx.Enemies)
	if nearest == nil {
		return
	}

	speed := s.ProjectileSpeed
	if speed <= 0 {
		speed = config.GlobalBalance().Combat.WardenMechProjectileSpeed
	}
	if speed <= 0 {
		speed = 400 // ultimate fallback
	}

	inRange := countInRange(s, ctx)

	if inRange >= s.AoeThreshold {
		// AoE 模式：对射程内每个敌人发射带斩杀的弹射物
		ctx.Enemies.Each(func(e *enemy.Enemy) {
			if math.Hypot(e.X-s.X, e.Y-s.Y) <= s.Range {
				if ctx.Projectiles != nil {
					ctx.Projectiles.FireWithExecute(s.X, s.Y, e.X, e.Y, s.Damage, speed, 4, e, "warden", s.ExecHpPct)
				}
			}
		})
		if ctx.OnSpecial != nil {
			ctx.OnSpecial()
		}
	} else {
		// 单体模式：发射带斩杀的弹射物
		if ctx.Projectiles != nil {
			ctx.Projectiles.FireWithExecute(s.X, s.Y, nearest.X, nearest.Y, s.Damage, speed, 4, nearest, "warden", s.ExecHpPct)
		}
	}

	if ctx.OnFire != nil {
		ctx.OnFire()
	}

	s.LastTargetX = nearest.X
	s.LastTargetY = nearest.Y
	s.ShootTimer = 0.15
}
