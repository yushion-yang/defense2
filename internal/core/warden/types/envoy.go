// envoy.go — 使者型战灵。
// 使者是纯增强型战灵，不攻击敌人。
// 行为循环：空闲 → 附身（选择周围敌人最多的塔，增强其伤害）→ 冷却 → 空闲。
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

// EnvoyState 使者的内部状态。
type EnvoyState struct {
	warden.WardenState              // 嵌入公共基座
	PossessedTower     *tower.Tower // 正在附身的塔（nil 表示空闲）
	PossessDuration    float64      // 附身持续时间（秒）
	CooldownDuration   float64      // 冷却持续时间（秒）
	DamageBonus        float64      // 附身时给塔增加的战力
}

// Base 实现 Stateful 接口。
func (s *EnvoyState) Base() *warden.WardenState { return &s.WardenState }

// EnvoyBehavior 使者行为实现。
type EnvoyBehavior struct{}

func (b *EnvoyBehavior) Type() string { return "envoy" }

func (b *EnvoyBehavior) Init(w *warden.Warden) interface{} {
	return &EnvoyState{
		WardenState: warden.WardenState{
			Phase: "idle",
		},
		PossessDuration:  4.0,
		CooldownDuration: 3.0,
		DamageBonus:      5,
	}
}

func (b *EnvoyBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*EnvoyState)
	if !ok {
		return
	}

	switch s.Phase {
	case "idle":
		best := findBestHost(ctx.Towers, ctx.Enemies)
		if best != nil {
			s.PossessedTower = best
			s.Phase = "possessing"
			s.Timer = s.PossessDuration
			s.X = best.X
			s.Y = best.Y
			ensureStrength(best)
			best.Strength.SetTemp(fmt.Sprintf("envoy_warden_%d", w.ID), s.DamageBonus)
		}

	case "possessing":
		s.Timer -= ctx.DT
		if s.PossessedTower != nil {
			s.X = s.PossessedTower.X
			s.Y = s.PossessedTower.Y - 20
		}
		if s.Timer <= 0 {
			if s.PossessedTower != nil && s.PossessedTower.Strength != nil {
				s.PossessedTower.Strength.RemoveTemp(fmt.Sprintf("envoy_warden_%d", w.ID))
			}
			s.PossessedTower = nil
			s.Phase = "idle"
			s.Timer = s.CooldownDuration
		}
	}

	// 冷却倒计时
	if s.Phase == "idle" && s.Timer > 0 {
		s.Timer -= ctx.DT
	}
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
