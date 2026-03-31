// strategy_random.go — 随机模糊策略。
// 随机建塔/升级/卖塔/开波，用于广覆盖和发现 panic。
package autoplay

import "math/rand"

// RandomStrategy 随机策略。
type RandomStrategy struct {
	rng            *rand.Rand
	buildChance    float64 // 每帧建塔概率
	upgradeChance  float64 // 每帧升级概率
	sellChance     float64 // 每帧卖塔概率
	waveDelay      int     // 波结束后等待帧数
	framesSinceEnd int     // 自上次波结束后的帧数
	wardenKeys     []string
}

// NewRandomStrategy 创建随机策略。
func NewRandomStrategy(seed int64) *RandomStrategy {
	return &RandomStrategy{
		rng:           rand.New(rand.NewSource(seed)),
		buildChance:   0.02,
		upgradeChance: 0.005,
		sellChance:    0.001,
		waveDelay:     30,
		wardenKeys:    []string{"prince", "core", "chain", "skystrike", "envoy"},
	}
}

func (s *RandomStrategy) Name() string { return "random" }

func (s *RandomStrategy) Init(_ *GameState) {}

func (s *RandomStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		key := s.wardenKeys[s.rng.Intn(len(s.wardenKeys))]
		return []Action{{Type: ActionSelectWarden, WardenKey: key}}
	}

	// 事件选择（交互模式 6 = modeEvent）— 随机选择以覆盖所有事件
	if state.InteractMode == 6 {
		return []Action{{Type: ActionChooseEvent, EventIndex: s.rng.Intn(4)}}
	}

	// 开波
	if !state.WaveActive && state.Wave < state.MaxWaves {
		s.framesSinceEnd++
		if s.framesSinceEnd > s.waveDelay {
			actions = append(actions, Action{Type: ActionStartWave})
			s.framesSinceEnd = 0
		}
	}

	// 随机建塔
	if s.rng.Float64() < s.buildChance && len(state.BuildCells) > 0 && len(state.TowerDefs) > 0 {
		def := state.TowerDefs[s.rng.Intn(len(state.TowerDefs))]
		if state.Gold >= def.Cost {
			cell := state.BuildCells[s.rng.Intn(len(state.BuildCells))]
			actions = append(actions, Action{
				Type:     ActionBuild,
				TowerKey: def.Key,
				Cell:     cell,
			})
		}
	}

	// 随机升级
	if s.rng.Float64() < s.upgradeChance && len(state.Towers) > 0 && state.Gold >= 10 {
		t := state.Towers[s.rng.Intn(len(state.Towers))]
		actions = append(actions, Action{
			Type: ActionUpgrade,
			Row:  t.Row,
			Col:  t.Col,
		})
	}

	// 随机卖塔
	if s.rng.Float64() < s.sellChance && len(state.Towers) > 0 {
		t := state.Towers[s.rng.Intn(len(state.Towers))]
		actions = append(actions, Action{
			Type: ActionSell,
			Row:  t.Row,
			Col:  t.Col,
		})
	}

	return actions
}
