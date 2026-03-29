// chain.go — 聚能战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 被动增强所有塔的战力，同时主动攻击敌人。
package types

import (
	"fmt"

	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&ChainBehavior{})
}

// ChainState 聚能战灵的内部状态。
type ChainState struct {
	warden.WardenState                       // 嵌入公共基座
	TowerBonus         float64               // 每座塔的战力加成
	BoostedSet         map[*tower.Tower]bool // 已加成的塔集合
}

// Base 实现 Stateful 接口。
func (s *ChainState) Base() *warden.WardenState { return &s.WardenState }

// ChainBehavior 聚能战灵行为实现。
type ChainBehavior struct{}

func (b *ChainBehavior) Type() string { return "chain" }

func (b *ChainBehavior) Init(w *warden.Warden) interface{} {
	return &ChainState{
		WardenState: warden.WardenState{
			Damage:         15,
			AttackInterval: 2.0,
			Range:          150,
			MoveSpeed:      300,
		},
		TowerBonus: 10,
		BoostedSet: make(map[*tower.Tower]bool),
	}
}

const chainOrbitDist = 100.0

func (b *ChainBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*ChainState)
	if !ok {
		return
	}
	dt := ctx.DT

	// 1. 塔增强：为新建的塔添加战力临时加成
	ctx.Towers.Each(func(t *tower.Tower) {
		if !s.BoostedSet[t] {
			ensureStrength(t)
			t.Strength.SetTemp(fmt.Sprintf("chain_warden_%d", w.ID), s.TowerBonus)
			s.BoostedSet[t] = true
		}
	})

	// 2. 轨道运动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, chainOrbitDist, dt)
	}

	// 3. 攻击最近敌人
	s.AttackTimer -= dt
	if s.AttackTimer <= 0 {
		s.AttackTimer += s.AttackInterval
		s.BasicAttack(ctx)
	}

	// 4. 射击线衰减
	s.DecayShootTimer(dt)
}

// ensureStrength 确保塔有 StrengthData（懒初始化）。
func ensureStrength(t *tower.Tower) {
	if t.Strength == nil {
		t.Strength = strength.NewStrengthData()
	}
}
