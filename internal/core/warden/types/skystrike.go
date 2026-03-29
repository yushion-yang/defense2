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
	warden.WardenState // 嵌入公共基座

	// AoE 天降打击（特殊能力，区别于基座的普攻）
	AoEDamage   float64 // AoE 伤害
	AoEInterval float64 // AoE 间隔（秒）
	AoERadius   float64 // AoE 半径（像素）
	AoETimer    float64 // AoE 冷却计时器

	// 渲染用字段（AoE 视觉效果）
	StrikeX     float64 // 上次 AoE 打击位置 X
	StrikeY     float64 // 上次 AoE 打击位置 Y
	StrikeTimer float64 // AoE 视觉效果倒计时
}

// Base 实现 Stateful 接口。
func (s *SkystrikeState) Base() *warden.WardenState { return &s.WardenState }

// SkystrikeBehavior 天降行为实现。
type SkystrikeBehavior struct{}

func (b *SkystrikeBehavior) Type() string { return "skystrike" }

func (b *SkystrikeBehavior) Init(w *warden.Warden) interface{} {
	return &SkystrikeState{
		WardenState: warden.WardenState{
			X:              600,
			Y:              270,
			MoveSpeed:      320,
			Damage:         18,
			AttackInterval: 1.5,
			Range:          140,
		},
		AoEDamage:   40,
		AoEInterval: 5.0,
		AoERadius:   60,
	}
}

const skystrikeOrbitDist = 120.0

func (b *SkystrikeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*SkystrikeState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 轨道运动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, skystrikeOrbitDist, dt)
	}

	// 2. 普通攻击
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		s.BasicAttack(ctx)
	}

	// 3. AoE 天降打击
	if s.StrikeTimer > 0 {
		s.StrikeTimer -= dt
	}
	s.AoETimer -= dt
	if s.AoETimer <= 0 {
		s.AoETimer += s.AoEInterval
		skystrikeAoE(s, ctx)
	}

	// 4. 射击线衰减
	s.DecayShootTimer(dt)
}

// skystrikeAoE AoE 天降打击（在最密集敌群处释放范围伤害）。
func skystrikeAoE(s *SkystrikeState, ctx *warden.TickContext) {
	center := warden.FindDensestEnemy(ctx.Enemies, s.AoERadius)
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
