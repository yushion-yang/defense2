// skystrike.go — 水灵战灵。
// 移动型战灵，围绕敌群轨道运动。
// 被动：每隔一段时间轮流施展三种攻击之一（多目标/连击/百分比）。
package types

import (
	"fmt"
	"math/rand"

	"defense2/internal/core/enemy"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&SkystrikeBehavior{})
}

// SkystrikeState 水灵战灵的内部状态。
type SkystrikeState struct {
	warden.WardenState // 嵌入公共基座

	// 三模式攻击参数
	SpecialInterval float64 // 特殊攻击间隔（秒）
	SpecialTimer    float64 // 特殊攻击倒计时

	// 模式 1：多目标伤害
	MultiTargets  int     // 目标数
	MultiDmgRatio float64 // 每目标伤害 = 攻击力 × 此比例

	// 模式 2：单目标连击
	BurstHits     int     // 连击段数
	BurstDmgRatio float64 // 每段伤害 = 攻击力 × 此比例

	// 模式 3：百分比最大生命值
	HpTargets int     // 目标数
	HpPercent float64 // 每目标伤害 = MaxHP × 此百分比

	// 轨道距离
	OrbitDist float64 // 围绕敌群的轨道距离

	// 渲染用字段
	Strikes  []StrikeVFX // 活跃的天降打击特效列表
	LastMode int         // 上次使用的模式（1/2/3，渲染用）

	// 连击延时状态（模式 2）
	BurstTarget   *enemy.Enemy // 连击目标
	BurstDmgValue float64      // 每段伤害值
	BurstPending  int          // 剩余连击段数
	BurstDelay    float64      // 段间倒计时
}

// StrikeVFX 单个天降打击的视觉效果状态。
type StrikeVFX struct {
	X, Y   float64      // 打击位置（初始快照）
	Target *enemy.Enemy // 跟踪目标（存活时跟随其 X 坐标）
	Timer  float64      // 剩余显示时间（0.8s）
	Mode   int          // 攻击模式（1/2/3）
}

// Base 实现 Stateful 接口。
func (s *SkystrikeState) Base() *warden.WardenState { return &s.WardenState }

// SkystrikeBehavior 水灵战灵行为实现。
type SkystrikeBehavior struct{}

func (b *SkystrikeBehavior) Type() string { return "skystrike" }

// Init initializes skystrike warden behavior.
// NOTE: Stats are currently hardcoded. See config/wardens/wardens.json for planned externalization.
// Hardcoded: damage=10, attackInterval=1.5, range=140, moveSpeed=320,
//
//	specialInterval=1, multiTargets=3, multiDmgRatio=2.0,
//	burstHits=5, burstDmgRatio=1.0, hpTargets=3, hpPercent=0.10
func (b *SkystrikeBehavior) Init(w *warden.Warden) interface{} {
	p := w.Params
	return &SkystrikeState{
		WardenState: warden.WardenState{
			Damage:         10,
			AttackInterval: 1.5,
			Range:          140,
			MoveSpeed:      320,
		},
		SpecialInterval: warden.ParamOr(p, "specialInterval", 1.0),
		MultiTargets:    warden.ParamOrInt(p, "multiTargets", 3),
		MultiDmgRatio:   warden.ParamOr(p, "multiDmgRatio", 2.0), // 200% 攻击力
		BurstHits:       warden.ParamOrInt(p, "burstHits", 5),
		BurstDmgRatio:   warden.ParamOr(p, "burstDmgRatio", 1.0), // 100% 攻击力 x 5 段
		HpTargets:       warden.ParamOrInt(p, "hpTargets", 3),
		HpPercent:       warden.ParamOr(p, "hpPercent", 0.10), // 10% 最大生命值
		OrbitDist:       warden.ParamOr(p, "orbitDist", 120.0),
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *SkystrikeState) DescParams(w *warden.Warden) map[string]string {
	return map[string]string{
		"attackInterval":  fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":          fmt.Sprintf("%.0f", s.Damage),
		"specialInterval": fmt.Sprintf("%.0f", s.SpecialInterval),
		"multiTargets":    fmt.Sprintf("%d", s.MultiTargets),
		"multiDmg":        fmt.Sprintf("%.0f", s.Damage*s.MultiDmgRatio),
		"burstHits":       fmt.Sprintf("%d", s.BurstHits),
		"burstDmg":        fmt.Sprintf("%.0f", s.Damage*s.BurstDmgRatio),
		"hpTargets":       fmt.Sprintf("%d", s.HpTargets),
		"hpPct":           fmt.Sprintf("%.0f", s.HpPercent*100),
	}
}

const skystrikeOrbitDist = 120.0

func (b *SkystrikeBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*SkystrikeState)
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

	// 特效递减
	tickStrikes(s, dt)

	// 连击延时处理（模式 2 分段出伤）
	if s.BurstPending > 0 {
		s.BurstDelay -= dt
		if s.BurstDelay <= 0 {
			s.BurstDelay += 0.15
			if s.BurstTarget != nil && s.BurstTarget.Active {
				applyDmg(s.BurstTarget, s.BurstDmgValue, ctx)
				addStrike(s, s.BurstTarget, 2)
			}
			s.BurstPending--
		}
	}
	// 普攻（需要射程内有敌人）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			s.AttackTimer += s.AttackInterval
			s.BasicAttack(ctx)
		}
	}

	// 被动：水灵秘术（场上有敌人即可，不需要进入攻击范围）
	if count > 0 {
		s.SpecialTimer -= dt
		if s.SpecialTimer <= 0 {
			s.SpecialTimer += s.SpecialInterval
			skystrikeSpecial(s, ctx)
			if ctx.OnSpecial != nil {
				ctx.OnSpecial()
			}
		}
	}

	// 4. 射击线衰减
	s.DecayShootTimer(dt)
}

// skystrikeSpecial 轮流施展三种攻击模式（1->2->3->1->...）。
func skystrikeSpecial(s *SkystrikeState, ctx *warden.TickContext) {
	alive := collectAlive(ctx.Enemies)
	if len(alive) == 0 {
		return
	}

	// 轮流：上次模式 +1，循环 1→2→3→1
	mode := s.LastMode + 1
	if mode > 3 {
		mode = 1
	}
	s.LastMode = mode

	switch mode {
	case 1:
		skystrikeMulti(s, ctx, alive)
	case 2:
		skystrikeBurst(s, ctx, alive)
	case 3:
		skystrikeHpPercent(s, ctx, alive)
	}
}

// 模式 1：随机 N 目标，各受 ratio% 攻击力伤害。
func skystrikeMulti(s *SkystrikeState, ctx *warden.TickContext, alive []*enemy.Enemy) {
	targets := pickRandom(alive, s.MultiTargets)
	dmg := s.Damage * s.MultiDmgRatio
	for _, e := range targets {
		applyDmg(e, dmg, ctx)
		addStrike(s, e, 1)
	}
}

// 模式 2：随机单目标，N 段延时连击（每段间隔 0.15s）。
func skystrikeBurst(s *SkystrikeState, ctx *warden.TickContext, alive []*enemy.Enemy) {
	target := alive[rand.Intn(len(alive))]
	dmg := s.Damage * s.BurstDmgRatio
	// 第一段立即出伤
	applyDmg(target, dmg, ctx)
	addStrike(s, target, 2)
	// 剩余段数进入延时队列
	s.BurstTarget = target
	s.BurstDmgValue = dmg
	s.BurstPending = s.BurstHits - 1
	s.BurstDelay = 0.15
}

// 模式 3：随机 N 目标，各受 X% 最大生命值伤害（Boss 免疫）。
func skystrikeHpPercent(s *SkystrikeState, ctx *warden.TickContext, alive []*enemy.Enemy) {
	// 过滤掉 Boss（Boss 免疫百分比伤害）
	var nonBoss []*enemy.Enemy
	for _, e := range alive {
		if !e.Boss {
			nonBoss = append(nonBoss, e)
		}
	}
	if len(nonBoss) == 0 {
		return
	}
	targets := pickRandom(nonBoss, s.HpTargets)
	for _, e := range targets {
		dmg := e.MaxHP * s.HpPercent
		applyDmg(e, dmg, ctx)
		addStrike(s, e, 3)
	}
}

// applyDmg 对敌人施加伤害（统一走 ApplyDamage）。
func applyDmg(e *enemy.Enemy, dmg float64, ctx *warden.TickContext) {
	warden.ApplyDamage(ctx, e, dmg, false)
}

// collectAlive 收集所有存活敌人。
func collectAlive(pool *enemy.Pool) []*enemy.Enemy {
	var result []*enemy.Enemy
	pool.Each(func(e *enemy.Enemy) {
		result = append(result, e)
	})
	return result
}

// pickRandom 从列表中随机选 n 个（不重复，不足则全选）。
func pickRandom(list []*enemy.Enemy, n int) []*enemy.Enemy {
	if n >= len(list) {
		return list
	}
	perm := rand.Perm(len(list))
	result := make([]*enemy.Enemy, n)
	for i := 0; i < n; i++ {
		result[i] = list[perm[i]]
	}
	return result
}

// addStrike 在指定敌人位置添加一个天降打击特效。
func addStrike(s *SkystrikeState, e *enemy.Enemy, mode int) {
	s.Strikes = append(s.Strikes, StrikeVFX{
		X: e.X, Y: e.Y, Target: e, Timer: 0.8, Mode: mode,
	})
	s.LastMode = mode
}

// tickStrikes 递减计时器，跟随存活敌人的 X 坐标，移除过期的。
func tickStrikes(s *SkystrikeState, dt float64) {
	n := 0
	for i := range s.Strikes {
		st := &s.Strikes[i]
		st.Timer -= dt
		if st.Timer <= 0 {
			continue
		}
		// 跟随存活敌人的横坐标
		if st.Target != nil && st.Target.Active {
			st.X = st.Target.X
			st.Y = st.Target.Y
		}
		s.Strikes[n] = s.Strikes[i]
		n++
	}
	s.Strikes = s.Strikes[:n]
}
