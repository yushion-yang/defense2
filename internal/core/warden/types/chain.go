// chain.go — 聚能战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 被动：将 150px 内的炮塔串联，串联体内每座塔获得 (塔数×10) 强度。
package types

import (
	"fmt"
	"math"

	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/warden"
)

func init() {
	warden.RegisterBehavior(&ChainBehavior{})
}

// ChainLink 一对串联塔的位置（用于渲染能量连线）。
type ChainLink struct {
	X1, Y1, X2, Y2 float64
}

// ChainState 聚能战灵的内部状态。
type ChainState struct {
	warden.WardenState                          // 嵌入公共基座
	ChainRange         float64                  // 串联范围（px）
	BonusPerTower      float64                  // 每座串联塔贡献的强度
	lastBonuses        map[*tower.Tower]float64 // 上一帧各塔设置的加成值
	ChainLinks         []ChainLink              // 当前帧的串联连线（渲染用）
}

// Base 实现 Stateful 接口。
func (s *ChainState) Base() *warden.WardenState { return &s.WardenState }

// ChainBehavior 聚能战灵行为实现。
type ChainBehavior struct{}

func (b *ChainBehavior) Type() string { return "chain" }

// 注意：以下硬编码值应与 config/wardens/wardens.json 保持一致
func (b *ChainBehavior) Init(w *warden.Warden) interface{} {
	return &ChainState{
		WardenState: warden.WardenState{
			Damage:         12,
			AttackInterval: 1.2,
			Range:          150,
			MoveSpeed:      300,
		},
		ChainRange:    150,
		BonusPerTower: 10,
		lastBonuses:   make(map[*tower.Tower]float64),
	}
}

// DescParams 返回 HUD 占位符参数。
func (s *ChainState) DescParams(w *warden.Warden) map[string]string {
	return map[string]string{
		"attackInterval": fmt.Sprintf("%.1f", s.AttackInterval),
		"damage":         fmt.Sprintf("%.0f", s.Damage),
		"chainRange":     fmt.Sprintf("%.0f", s.ChainRange),
		"bonusPerTower":  fmt.Sprintf("%.0f", s.BonusPerTower),
	}
}

const chainOrbitDist = 100.0

func (b *ChainBehavior) Tick(w *warden.Warden, ctx *warden.TickContext) {
	s, ok := w.State.(*ChainState)
	if !ok {
		return
	}
	dt := ctx.DT
	s.ApplyStrength(w)

	// 1. 串联体：按距离分组，每座塔获得 groupSize × BonusPerTower 强度
	chainTowerBuff(w, s, ctx)

	// 2. 移动
	cx, cy, count := warden.ComputeClusterCenter(ctx.Enemies)
	if count > 0 {
		s.MoveOrbit(cx, cy, chainOrbitDist, dt)
	} else {
		s.Wander(dt)
	}

	// 3. 攻击（仅有敌人时计时）
	if count > 0 {
		s.AttackTimer -= dt
		if s.AttackTimer <= 0 {
			s.AttackTimer += s.AttackInterval
			s.BasicAttack(ctx)
		}
	}

	// 4. 射击线衰减
	s.DecayShootTimer(dt)
}

// chainTowerBuff 按距离对塔分组（Union-Find），每座塔获得 groupSize × bonus 临时强度。
func chainTowerBuff(w *warden.Warden, s *ChainState, ctx *warden.TickContext) {
	var towers []*tower.Tower
	ctx.Towers.Each(func(t *tower.Tower) {
		towers = append(towers, t)
	})
	if len(towers) == 0 {
		return
	}

	// Union-Find
	parent := make([]int, len(towers))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// 串联 ChainRange 内的塔，同时记录连接对（供渲染用）
	s.ChainLinks = s.ChainLinks[:0]
	for i := 0; i < len(towers); i++ {
		for j := i + 1; j < len(towers); j++ {
			if math.Hypot(towers[i].X-towers[j].X, towers[i].Y-towers[j].Y) <= s.ChainRange {
				union(i, j)
				s.ChainLinks = append(s.ChainLinks, ChainLink{
					X1: towers[i].X, Y1: towers[i].Y,
					X2: towers[j].X, Y2: towers[j].Y,
				})
			}
		}
	}

	// 统计各组大小
	groups := make(map[int]int)
	for i := range towers {
		groups[find(i)]++
	}

	// 应用加成
	key := fmt.Sprintf("chain_warden_%d", w.ID)
	newBonuses := make(map[*tower.Tower]float64, len(towers))
	for i, t := range towers {
		groupSize := groups[find(i)]
		bonus := float64(groupSize) * s.BonusPerTower
		ensureStrength(t)
		t.ApplyBuff(tower.TowerBuff{
			Key:       key,
			Source:    "聚能战灵",
			Desc:      fmt.Sprintf("+%.0f 强度 (%d塔串联)", bonus, groupSize),
			Value:     bonus,
			Duration:  -1,
			Remaining: -1, // 永久，每帧刷新
		})
		newBonuses[t] = bonus
	}

	// 清除已不在场的塔的旧加成
	for t := range s.lastBonuses {
		if _, exists := newBonuses[t]; !exists {
			t.RemoveBuff(key)
		}
	}
	s.lastBonuses = newBonuses
}

// ensureStrength 确保塔有 StrengthData（懒初始化）。
func ensureStrength(t *tower.Tower) {
	if t.Strength == nil {
		t.Strength = strength.NewStrengthData()
	}
}
