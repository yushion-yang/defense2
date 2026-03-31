// strategy_greedy.go — 贪心启发策略。
// 模拟正常玩家行为：优先路径拐角建塔，按性价比选塔，存钱升级。
package autoplay

import (
	"math"
	"sort"
)

// GreedyStrategy 贪心策略。
type GreedyStrategy struct {
	wardenKey string
	phase     int // 0=early(build), 1=mid(upgrade), 2=late(optimize)
	lastBuild int // 上次建塔的 tick
}

// NewGreedyStrategy 创建贪心策略。
func NewGreedyStrategy() *GreedyStrategy {
	return &GreedyStrategy{
		wardenKey: "prince",
	}
}

func (s *GreedyStrategy) Name() string { return "greedy" }

func (s *GreedyStrategy) Init(_ *GameState) {}

func (s *GreedyStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: s.wardenKey}}
	}

	// 阶段判定
	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}
	switch {
	case progress < 0.33:
		s.phase = 0
	case progress < 0.66:
		s.phase = 1
	default:
		s.phase = 2
	}

	// 开波：非活跃时立即开
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	switch s.phase {
	case 0: // 早期：建塔
		actions = append(actions, s.decideBuild(state)...)
	case 1: // 中期：升级为主，偶尔建塔
		actions = append(actions, s.decideUpgrade(state)...)
		if state.Tick-s.lastBuild > 300 { // 5 秒没建塔则补建
			actions = append(actions, s.decideBuild(state)...)
		}
	case 2: // 后期：全力升级
		actions = append(actions, s.decideUpgrade(state)...)
	}

	return actions
}

// decideBuild 选最优性价比塔+最佳位置建造。
func (s *GreedyStrategy) decideBuild(state *GameState) []Action {
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return nil
	}

	// 按性价比排序塔定义
	type scored struct {
		def   TowerDefInfo
		score float64
	}
	var defs []scored
	for _, d := range state.TowerDefs {
		if d.Cost <= state.Gold {
			efficiency := (d.Damage * d.Range) / float64(d.Cost)
			defs = append(defs, scored{def: d, score: efficiency})
		}
	}
	if len(defs) == 0 {
		return nil
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].score > defs[j].score })
	best := defs[0].def

	// 选最佳位置（优先靠近地图中心，因为路径通常经过中心）
	bestCell := state.BuildCells[0]
	centerX := 600.0 // map center X
	centerY := 270.0 // map center Y
	bestDist := math.MaxFloat64
	for _, c := range state.BuildCells {
		dist := math.Hypot(c.X-centerX, c.Y-centerY)
		if dist < bestDist {
			bestDist = dist
			bestCell = c
		}
	}

	s.lastBuild = state.Tick
	return []Action{{
		Type:     ActionBuild,
		TowerKey: best.Key,
		Cell:     bestCell,
	}}
}

// decideUpgrade 升级已有塔中伤害最高的。
func (s *GreedyStrategy) decideUpgrade(state *GameState) []Action {
	if len(state.Towers) == 0 || state.Gold < 10 {
		return nil
	}

	// 找伤害最高的塔
	var best TowerInfo
	bestDmg := -1.0
	for _, t := range state.Towers {
		if t.Damage > bestDmg {
			bestDmg = t.Damage
			best = t
		}
	}

	return []Action{{
		Type: ActionUpgrade,
		Row:  best.Row,
		Col:  best.Col,
	}}
}
