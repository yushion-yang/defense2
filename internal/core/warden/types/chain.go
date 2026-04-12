// chain.go — 聚能战灵。
// 移动型战灵，围绕敌群轨道运动并射击。
// 被动：将 150px 内的炮塔串联，串联体内每座塔获得 (塔数×10) 强度。
package types

import (
	"fmt"
	"math"
	"sort"

	"defense2/internal/core/buff"
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
	OrbitDist          float64                  // 围绕敌群的轨道距离
	lastBonuses        map[*tower.Tower]float64 // 上一帧各塔设置的加成值
	ChainLinks         []ChainLink              // 当前帧的串联连线（渲染用）
	buffKey            string                   // 缓存的 buff ID，避免逐帧 Sprintf
}

// Base 实现 Stateful 接口。
func (s *ChainState) Base() *warden.WardenState { return &s.WardenState }

// ChainBehavior 聚能战灵行为实现。
type ChainBehavior struct{}

func (b *ChainBehavior) Type() string { return "chain" }

// Init initializes chain warden behavior.
// NOTE: Stats are currently hardcoded. See config/wardens/wardens.json for planned externalization.
// Hardcoded: damage=12, attackInterval=1.2, range=150, moveSpeed=300, chainRange=150, bonusPerTower=10
func (b *ChainBehavior) Init(w *warden.Warden) interface{} {
	p := w.Params
	return &ChainState{
		WardenState: warden.WardenState{
			Damage:         12,
			AttackInterval: 1.2,
			Range:          150,
			MoveSpeed:      300,
		},
		ChainRange:    warden.ParamOr(p, "chainRange", 150),
		BonusPerTower: warden.ParamOr(p, "bonusPerTower", 10),
		OrbitDist:     warden.ParamOr(p, "orbitDist", 100.0),
		lastBonuses:   make(map[*tower.Tower]float64),
		buffKey:       fmt.Sprintf("chain_warden_%d", w.ID),
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
		s.MoveOrbit(cx, cy, s.OrbitDist, dt)
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

// towerEdge 表示两塔之间的候选边（Kruskal MST 用）。
type towerEdge struct {
	i, j int
	dist float64
}

// chainTowerBuff 按距离对塔分组（Union-Find），每座塔获得 groupSize × bonus 临时强度。
// 渲染用的 ChainLinks 以 MST（最小生成树）形式生成，避免全连接导致的视觉杂乱。
func chainTowerBuff(w *warden.Warden, s *ChainState, ctx *warden.TickContext) {
	var towers []*tower.Tower
	ctx.Towers.Each(func(t *tower.Tower) {
		towers = append(towers, t)
	})
	if len(towers) == 0 {
		return
	}

	// 收集 ChainRange 内的所有候选边，按距离排序（Kruskal）
	var edges []towerEdge
	for i := 0; i < len(towers); i++ {
		for j := i + 1; j < len(towers); j++ {
			d := math.Hypot(towers[i].X-towers[j].X, towers[i].Y-towers[j].Y)
			if d <= s.ChainRange {
				edges = append(edges, towerEdge{i, j, d})
			}
		}
	}
	sort.Slice(edges, func(a, b int) bool {
		return edges[a].dist < edges[b].dist
	})

	// Kruskal MST：按距离从小到大合并，只保留合并成功的边（即 MST 边）
	parent := make([]int, len(towers))
	for i := range parent {
		parent[i] = i
	}

	s.ChainLinks = s.ChainLinks[:0]
	for _, e := range edges {
		if strength.UFFind(parent, e.i) == strength.UFFind(parent, e.j) {
			continue // 已在同组，跳过（避免环）
		}
		strength.UFUnion(parent, nil, e.i, e.j)
		s.ChainLinks = append(s.ChainLinks, ChainLink{
			X1: towers[e.i].X, Y1: towers[e.i].Y,
			X2: towers[e.j].X, Y2: towers[e.j].Y,
		})
	}

	// 统计各组大小
	groups := make(map[int]int)
	for i := range towers {
		groups[strength.UFFind(parent, i)]++
	}

	// 应用加成
	key := s.buffKey
	newBonuses := make(map[*tower.Tower]float64, len(towers))
	for i, t := range towers {
		groupSize := groups[strength.UFFind(parent, i)]
		bonus := float64(groupSize) * s.BonusPerTower
		ensureStrength(t)
		t.Buffs.Add(buff.Buff{
			ID:        key,
			Category:  buff.CatAura,
			Source:    "chain_warden",
			Value:     bonus,
			Duration:  -1,
			Remaining: -1, // 永久，每帧刷新
		})
		// 注意：不再调用 SetTemp()。链加成的强度由 pipeline.TickTowerAbilities
		// 中的 RebuildChainNetwork → SetTemp("chain", bonus) 统一负责。
		// 此处的 Buff 仅供 HUD 显示链加成信息。
		newBonuses[t] = bonus
	}

	// 清除已不在场的塔的旧加成（含已售出的塔）
	for t := range s.lastBonuses {
		if !t.Active {
			delete(s.lastBonuses, t)
			continue
		}
		if _, exists := newBonuses[t]; !exists {
			t.Buffs.RemoveByID(key)
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
