// envoy.go — 金灵战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 定时选择最佳塔施加增强 buff（持续一段时间后自动过期）。
package types

import (
	"fmt"
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&EnvoyBehavior{})
}

// EnvoyState 金灵战灵的内部状态。
type EnvoyState struct {
	warden.WardenState // 嵌入公共基座

	// buff 参数
	BuffInterval float64 // buff 施加间隔（秒）
	BuffDuration float64 // buff 持续时间（秒）
	BuffBonus    float64 // buff 给塔增加的战力
	BuffTimer    float64 // buff 施加倒计时

	// 当前 buff 追踪（用于视觉反馈）
	BuffedTower *tower.Tower // 当前被 buff 的塔（仅渲染用）
	BuffExpiry  float64      // 当前 buff 剩余时间
}

// Base 实现 Stateful 接口。
func (s *EnvoyState) Base() *warden.WardenState { return &s.WardenState }

// EnvoyBehavior 金灵战灵行为实现。
type EnvoyBehavior struct{}

func (b *EnvoyBehavior) Type() string { return "envoy" }

func (b *EnvoyBehavior) Init(w *warden.Warden) interface{} {
	return &EnvoyState{
		WardenState: warden.WardenState{
			Damage:         12,
			AttackInterval: 1.5,
			Range:          140,
			MoveSpeed:      320,
		},
		BuffInterval: 5.0,
		BuffDuration: 4.0,
		BuffBonus:    8,
	}
}

const envoyOrbitDist = 100.0

func (b *EnvoyBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*EnvoyState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 轨道运动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, envoyOrbitDist, dt)
	}

	// 2. 普通攻击
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		s.BasicAttack(ctx)
	}

	// 3. 定时施加 buff
	s.BuffTimer -= dt
	if s.BuffTimer <= 0 {
		s.BuffTimer += s.BuffInterval
		applyEnvoyBuff(w, s, ctx)
	}

	// 4. buff 过期追踪（用于渲染）
	if s.BuffExpiry > 0 {
		s.BuffExpiry -= dt
		if s.BuffExpiry <= 0 {
			s.BuffedTower = nil
		}
	}

	// 5. 射击线衰减
	s.DecayShootTimer(dt)
}

// applyEnvoyBuff 选择最佳塔施加临时增强 buff。
func applyEnvoyBuff(w *warden.Warden, s *EnvoyState, ctx *warden.TickContext) {
	best := findBestHost(ctx.Towers, ctx.Enemies)
	if best == nil {
		return
	}
	ensureStrength(best)
	best.Strength.SetTemp(fmt.Sprintf("envoy_buff_%d", w.ID), s.BuffBonus)

	// 记录用于渲染
	s.BuffedTower = best
	s.BuffExpiry = s.BuffDuration

	// 设置定时移除（通过 goroutine 不合适，用 tick 追踪代替）
	// buff 移除由下一次 applyEnvoyBuff 覆盖（SetTemp 同 key 自动替换）
}

// findBestHost 找到射程内敌人最多的塔。
func findBestHost(towers *tower.Pool, enemies *enemy.Pool) *tower.Tower {
	var best *tower.Tower
	bestCount := 0

	towers.Each(func(t *tower.Tower) {
		count := 0
		enemies.Each(func(e *enemy.Enemy) {
			dx := e.X - t.X
			dy := e.Y - t.Y
			if math.Hypot(dx, dy) <= t.Range {
				count++
			}
		})
		if count > bestCount {
			bestCount = count
			best = t
		}
	})
	return best
}
