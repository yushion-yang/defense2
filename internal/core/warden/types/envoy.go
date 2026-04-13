// envoy.go — 金灵战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 定时选择最佳塔施加增强 buff（持续一段时间后自动过期）。
package types

import (
	"fmt"
	"math"

	"defense2/internal/core/buff"
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
	BuffInterval  float64 // buff 施加间隔（秒）
	BuffDuration  float64 // buff 持续时间（秒）
	BuffThreshold float64 // 临时 buff = max(0, 强度 - 此阈值)
	PermGrant     float64 // 每次触发永久赋予塔的强度
	BuffTimer     float64 // buff 施加倒计时

	// 轨道距离
	OrbitDist float64 // 围绕敌群的轨道距离

	// 当前 buff 追踪（用于视觉反馈）
	BuffedTower *tower.Tower // 当前被 buff 的塔（仅渲染用）
	BuffExpiry  float64      // 当前 buff 剩余时间
}

// Base 实现 Stateful 接口。
func (s *EnvoyState) Base() *warden.WardenState { return &s.WardenState }

// EnvoyBehavior 金灵战灵行为实现。
type EnvoyBehavior struct{}

func (b *EnvoyBehavior) Type() string { return "envoy" }

// Init initializes envoy warden behavior.
// NOTE: Stats are currently hardcoded. See config/wardens/wardens.json for planned externalization.
// Hardcoded: damage=12, attackInterval=1.2, range=140, moveSpeed=320,
//
//	buffInterval=10, buffDuration=6, buffThreshold=100, permGrant=5
func (b *EnvoyBehavior) Init(w *warden.Warden) interface{} {
	p := w.Params
	return &EnvoyState{
		WardenState: warden.WardenState{
			Damage:         12,
			AttackInterval: 1.2,
			Range:          140,
			MoveSpeed:      320,
		},
		BuffInterval:  warden.ParamOr(p, "buffInterval", 10.0),
		BuffDuration:  warden.ParamOr(p, "buffDuration", 6.0),  // 比 interval 短 1s，确保 buff 会到期
		BuffThreshold: warden.ParamOr(p, "buffThreshold", 100), // 临时 buff = 强度 - 100
		PermGrant:     warden.ParamOr(p, "permGrant", 5),       // 每次永久 +5 强度
		OrbitDist:     warden.ParamOr(p, "orbitDist", 100.0),
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *EnvoyState) DescParams(w *warden.Warden) map[string]string {
	bonus := w.PerceivedStrength - s.BuffThreshold
	if bonus < 0 {
		bonus = 0
	}
	return map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", s.Damage),
		"buffInterval":   fmt.Sprintf("%.0f", s.BuffInterval),
		"buffDuration":   fmt.Sprintf("%.0f", s.BuffDuration),
		"buffThreshold":  fmt.Sprintf("%.0f", s.BuffThreshold),
		"buffBonus":      fmt.Sprintf("%.0f", bonus),
		"permGrant":      fmt.Sprintf("%.0f", s.PermGrant),
	}
}

func (b *EnvoyBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*EnvoyState)
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

	// 2. 普通攻击（仅有敌人时计时）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			s.AttackTimer += s.AttackInterval
			s.BasicAttack(ctx)
		}
	}

	// 3. 定时施加 buff
	s.BuffTimer -= dt
	if s.BuffTimer <= 0 {
		s.BuffTimer += s.BuffInterval
		applyEnvoyBuff(w, s, ctx)
		if ctx.OnSpecial != nil {
			ctx.OnSpecial()
		}
	}

	// 4. If buffed tower was sold/removed, clear the reference
	if s.BuffedTower != nil && !s.BuffedTower.Active {
		key := fmt.Sprintf("envoy_buff_%d", w.ID)
		s.BuffedTower.Buffs.RemoveByID(key)
		if s.BuffedTower.Strength != nil {
			s.BuffedTower.Strength.RemoveTemp(key)
		}
		s.BuffedTower = nil
		s.BuffExpiry = 0
	}

	// 5. 每帧重写临时强度（ClearTransient 每帧清空 Temp，需要重建）
	if s.BuffExpiry > 0 {
		s.BuffExpiry -= dt
		if s.BuffExpiry <= 0 {
			// buff 到期：清引用，不再重写 Temp
			s.BuffedTower = nil
		} else if s.BuffedTower != nil && s.BuffedTower.Active && s.BuffedTower.Strength != nil {
			// buff 存活期间：每帧重写 SetTemp
			tempBonus := w.PerceivedStrength - s.BuffThreshold
			if tempBonus > 0 {
				key := fmt.Sprintf("envoy_buff_%d", w.ID)
				s.BuffedTower.Strength.SetTemp(key, tempBonus)
				s.BuffedTower.StatsDirty = true
			}
		}
	}

	// 6. 射击线衰减
	s.DecayShootTimer(dt)
}

// applyEnvoyBuff 选择最佳塔施加 buff。
// 每次触发：永久 +PermGrant 强度 + 临时 max(0, 强度-100) 强度。
// 只要有塔就触发，无需敌人。
func applyEnvoyBuff(w *warden.Warden, s *EnvoyState, ctx *warden.TickContext) {
	best := findBestHostOrAny(ctx.Towers, ctx.Enemies)
	if best == nil {
		return
	}

	key := fmt.Sprintf("envoy_buff_%d", w.ID)

	// 切换目标时，主动移除旧塔的临时 buff
	if s.BuffedTower != nil && s.BuffedTower != best {
		s.BuffedTower.Buffs.RemoveByID(key)
		if s.BuffedTower.Strength != nil {
			s.BuffedTower.Strength.RemoveTemp(key)
		}
	}

	ensureStrength(best)

	// 永久增加强度
	if s.PermGrant > 0 {
		best.Strength.AddPermanent(s.PermGrant)
	}

	// 临时 buff = max(0, 强度 - 阈值)
	tempBonus := w.PerceivedStrength - s.BuffThreshold
	if tempBonus < 0 {
		tempBonus = 0
	}
	best.Buffs.Add(buff.Buff{
		ID:        key,
		Category:  buff.CatAura,
		Source:    "envoy_warden",
		Value:     tempBonus,
		Duration:  s.BuffDuration,
		Remaining: s.BuffDuration,
	})
	if tempBonus > 0 {
		best.Strength.SetTemp(key, tempBonus)
	}
	best.StatsDirty = true

	// 记录用于渲染
	s.BuffedTower = best
	s.BuffExpiry = s.BuffDuration
}

// findBestHostOrAny 有敌人时选射程内敌人最多的塔，无敌人时选任意塔。
func findBestHostOrAny(towers *tower.Pool, enemies *enemy.Pool) *tower.Tower {
	best := findBestHost(towers, enemies)
	if best != nil {
		return best
	}
	// 无敌人或无塔有敌人在射程内：选任意一座塔
	var any *tower.Tower
	towers.Each(func(t *tower.Tower) {
		if any == nil {
			any = t
		}
	})
	return any
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
