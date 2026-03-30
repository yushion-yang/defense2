// strategy_scenario.go — 脚本化场景策略。
// 按预定步骤执行操作，覆盖 FSM 状态转换和边界情况。
package autoplay

// ScenarioStep 场景步骤。
type ScenarioStep struct {
	// WaitUntil 等待条件满足后执行 Actions。nil 表示立即执行。
	WaitUntil func(state *GameState) bool
	// Actions 返回本步骤要执行的操作。
	Actions func(state *GameState) []Action
}

// ScenarioStrategy 脚本化策略。
type ScenarioStrategy struct {
	name    string
	steps   []ScenarioStep
	stepIdx int
	done    bool
}

// NewScenarioStrategy 创建脚本化策略。
func NewScenarioStrategy(name string, steps []ScenarioStep) *ScenarioStrategy {
	return &ScenarioStrategy{
		name:  name,
		steps: steps,
	}
}

func (s *ScenarioStrategy) Name() string { return "scenario_" + s.name }

func (s *ScenarioStrategy) Init(_ *GameState) {}

func (s *ScenarioStrategy) Decide(state *GameState) []Action {
	if s.done || s.stepIdx >= len(s.steps) {
		s.done = true
		return nil
	}

	step := s.steps[s.stepIdx]

	// 检查等待条件
	if step.WaitUntil != nil && !step.WaitUntil(state) {
		return nil
	}

	// 执行操作
	actions := step.Actions(state)
	s.stepIdx++
	return actions
}

// ─── 预定义场景 ───

// BuildFlowScenario 建塔流程：选战灵 → 开波 → 建塔。
func BuildFlowScenario() *ScenarioStrategy {
	return NewScenarioStrategy("build-flow", []ScenarioStep{
		// 选战灵
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions: func(s *GameState) []Action {
				return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
			},
		},
		// 等有金币
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 30 },
			Actions: func(s *GameState) []Action {
				if len(s.BuildCells) == 0 || len(s.TowerDefs) == 0 {
					return nil
				}
				return []Action{{
					Type:     ActionBuild,
					TowerKey: s.TowerDefs[0].Key,
					Cell:     s.BuildCells[0],
				}}
			},
		},
		// 开波
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionStartWave}}
			},
		},
	})
}

// TowerLifecycleScenario 塔生命周期：建 → 升级 → 卖。
func TowerLifecycleScenario() *ScenarioStrategy {
	var builtRow, builtCol int
	return NewScenarioStrategy("tower-lifecycle", []ScenarioStep{
		// 选战灵
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions: func(s *GameState) []Action {
				return []Action{{Type: ActionSelectWarden, WardenKey: "envoy"}}
			},
		},
		// 建塔
		{
			WaitUntil: func(s *GameState) bool {
				return s.Gold >= 30 && len(s.BuildCells) > 0
			},
			Actions: func(s *GameState) []Action {
				cell := s.BuildCells[0]
				builtRow = cell.Row
				builtCol = cell.Col
				return []Action{{
					Type:     ActionBuild,
					TowerKey: s.TowerDefs[0].Key,
					Cell:     cell,
				}}
			},
		},
		// 升级
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 10 },
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionUpgrade, Row: builtRow, Col: builtCol}}
			},
		},
		// 卖塔
		{
			WaitUntil: func(s *GameState) bool { return s.Tick > 120 },
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionSell, Row: builtRow, Col: builtCol}}
			},
		},
	})
}

// ZeroGoldBuildScenario 零金币尝试建塔（边界测试）。
func ZeroGoldBuildScenario() *ScenarioStrategy {
	return NewScenarioStrategy("zero-gold-build", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions: func(s *GameState) []Action {
				return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
			},
		},
		// 尝试在金币不足时建塔（应失败但不 panic）
		{
			WaitUntil: func(s *GameState) bool { return s.Gold < 30 && len(s.BuildCells) > 0 },
			Actions: func(s *GameState) []Action {
				// 找最贵的塔
				most := s.TowerDefs[0]
				for _, d := range s.TowerDefs {
					if d.Cost > most.Cost {
						most = d
					}
				}
				return []Action{{
					Type:     ActionBuild,
					TowerKey: most.Key,
					Cell:     s.BuildCells[0],
				}}
			},
		},
	})
}

// RapidActionScenario 快速操作序列（压力测试）。
func RapidActionScenario() *ScenarioStrategy {
	return NewScenarioStrategy("rapid-actions", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions: func(s *GameState) []Action {
				return []Action{{Type: ActionSelectWarden, WardenKey: "chain"}}
			},
		},
		// 同帧多操作：建多塔 + 开波
		{
			WaitUntil: func(s *GameState) bool {
				return s.Gold >= 100 && len(s.BuildCells) >= 3
			},
			Actions: func(s *GameState) []Action {
				var actions []Action
				for i := 0; i < 3 && i < len(s.BuildCells) && i < len(s.TowerDefs); i++ {
					actions = append(actions, Action{
						Type:     ActionBuild,
						TowerKey: s.TowerDefs[i%len(s.TowerDefs)].Key,
						Cell:     s.BuildCells[i],
					})
				}
				actions = append(actions, Action{Type: ActionStartWave})
				return actions
			},
		},
	})
}

// AllScenariosMap 返回所有预定义场景的名称→构造函数映射。
func AllScenariosMap() map[string]func() Strategy {
	return map[string]func() Strategy{
		"build-flow":       func() Strategy { return BuildFlowScenario() },
		"tower-lifecycle":  func() Strategy { return TowerLifecycleScenario() },
		"zero-gold-build":  func() Strategy { return ZeroGoldBuildScenario() },
		"rapid-actions":    func() Strategy { return RapidActionScenario() },
	}
}
