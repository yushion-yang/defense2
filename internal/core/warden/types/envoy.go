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
	X, Y           float64 // 当前像素位置
	Phase          string  // "idle" 或 "possessing"
	Timer          float64 // 当前阶段剩余时间（秒）
	PossessedTower *tower.Tower // 正在附身的塔（nil 表示空闲）
	PossessDuration float64     // 附身持续时间（秒）
	CooldownDuration float64    // 冷却持续时间（秒）
	DamageBonus    float64      // 附身时给塔增加的伤害
}

// EnvoyBehavior 使者行为实现。
type EnvoyBehavior struct{}

func (b *EnvoyBehavior) Type() string { return "envoy" }

func (b *EnvoyBehavior) Init(w *warden.Warden) interface{} {
	return &EnvoyState{
		Phase:            "idle",
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
		// 寻找最佳附身目标：周围敌人最多的塔
		best := findBestHost(ctx.Towers, ctx.Enemies)
		if best != nil {
			s.PossessedTower = best
			s.Phase = "possessing"
			s.Timer = s.PossessDuration
			s.X = best.X
			s.Y = best.Y
			// 增强塔战力
			ensureStrength(best)
			best.Strength.SetTemp(fmt.Sprintf("envoy_warden_%d", w.ID), s.DamageBonus)
		}

	case "possessing":
		s.Timer -= ctx.DT
		if s.PossessedTower != nil {
			s.X = s.PossessedTower.X
			s.Y = s.PossessedTower.Y - 20 // 悬浮在塔上方
		}
		if s.Timer <= 0 {
			// 附身结束，移除战力加成，进入冷却
			if s.PossessedTower != nil && s.PossessedTower.Strength != nil {
				s.PossessedTower.Strength.RemoveTemp(fmt.Sprintf("envoy_warden_%d", w.ID))
			}
			s.PossessedTower = nil
			s.Phase = "idle"
			s.Timer = s.CooldownDuration
		}
	}

	// 冷却倒计时（idle 状态复用 Timer）
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
