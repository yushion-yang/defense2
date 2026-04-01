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

// ─── 经济验证 §2.1 ───

// EconSellRefundScenario 建塔→卖塔→验证金币回收。
func EconSellRefundScenario() *ScenarioStrategy {
	var builtRow, builtCol int
	return NewScenarioStrategy("econ-sell-refund", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		// 建塔
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 50 && len(s.BuildCells) > 0 },
			Actions: func(s *GameState) []Action {
				cell := s.BuildCells[0]
				builtRow, builtCol = cell.Row, cell.Col
				return []Action{{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: cell}}
			},
		},
		// 等 30 帧后卖塔
		{
			WaitUntil: func(s *GameState) bool { return s.Tick > 120 },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSell, Row: builtRow, Col: builtCol}} },
		},
		// 开波让游戏正常结束
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// EconRapidBuildScenario 快速连建多塔（不应负金币）。
func EconRapidBuildScenario() *ScenarioStrategy {
	return NewScenarioStrategy("econ-rapid-build", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 100 && len(s.BuildCells) >= 4 },
			Actions: func(s *GameState) []Action {
				var actions []Action
				for i := 0; i < 4 && i < len(s.BuildCells); i++ {
					actions = append(actions, Action{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: s.BuildCells[i]})
				}
				return actions
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// ─── 战斗伤害 §2.2 ───

// CombatSingleTowerScenario 单塔 DPS 验证。
func CombatSingleTowerScenario() *ScenarioStrategy {
	return NewScenarioStrategy("combat-single-tower", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 50 && len(s.BuildCells) > 0 },
			Actions: func(s *GameState) []Action {
				return []Action{{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: s.BuildCells[0]}}
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// CombatMultiTowerScenario 多塔叠加伤害验证。
func CombatMultiTowerScenario() *ScenarioStrategy {
	return NewScenarioStrategy("combat-multi-tower", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 150 && len(s.BuildCells) >= 3 },
			Actions: func(s *GameState) []Action {
				var actions []Action
				for i := 0; i < 3; i++ {
					actions = append(actions, Action{Type: ActionBuild, TowerKey: s.TowerDefs[0].Key, Cell: s.BuildCells[i]})
				}
				return actions
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// CombatAllTowerTypesScenario 逐一建所有塔类型。
func CombatAllTowerTypesScenario() *ScenarioStrategy {
	builtCount := 0
	return NewScenarioStrategy("combat-all-towers", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		// 持续建塔直到所有类型都建了
		{
			WaitUntil: func(s *GameState) bool {
				return builtCount >= len(s.TowerDefs) || len(s.BuildCells) == 0
			},
			Actions: func(s *GameState) []Action {
				var actions []Action
				for builtCount < len(s.TowerDefs) && builtCount < len(s.BuildCells) {
					if s.Gold >= s.TowerDefs[builtCount].Cost {
						actions = append(actions, Action{
							Type: ActionBuild, TowerKey: s.TowerDefs[builtCount].Key, Cell: s.BuildCells[builtCount],
						})
						builtCount++
					} else {
						break
					}
				}
				if len(actions) == 0 {
					// 金币不够，开波赚钱
					return []Action{{Type: ActionStartWave}}
				}
				return actions
			},
		},
		// 开波验证所有塔都能攻击
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// ─── 状态效果 §2.4 ───

// CCSlowStackScenario 减速叠加上限验证。
func CCSlowStackScenario() *ScenarioStrategy {
	return NewScenarioStrategy("cc-slow-stack", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		// 建 3 座 freeze 塔
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 180 && len(s.BuildCells) >= 3 },
			Actions: func(s *GameState) []Action {
				freezeKey := "freeze"
				var actions []Action
				for i := 0; i < 3 && i < len(s.BuildCells); i++ {
					actions = append(actions, Action{Type: ActionBuild, TowerKey: freezeKey, Cell: s.BuildCells[i]})
				}
				return actions
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// CCStunStopScenario 眩晕停止移动验证。
func CCStunStopScenario() *ScenarioStrategy {
	return NewScenarioStrategy("cc-stun-stop", []ScenarioStep{
		{
			WaitUntil: func(s *GameState) bool { return !s.WardenReady },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}} },
		},
		// 建 electric 塔（有 stun 能力）
		{
			WaitUntil: func(s *GameState) bool { return s.Gold >= 80 && len(s.BuildCells) >= 2 },
			Actions: func(s *GameState) []Action {
				return []Action{
					{Type: ActionBuild, TowerKey: "electric", Cell: s.BuildCells[0]},
					{Type: ActionBuild, TowerKey: "electric", Cell: s.BuildCells[1]},
				}
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return !s.WaveActive },
			Actions:   func(_ *GameState) []Action { return []Action{{Type: ActionStartWave}} },
		},
	})
}

// ─── 攻击方式 × 能力覆盖 §2.2 + §2.4 ───

// ─── 攻击方式 × 能力覆盖（自定义策略，非 ScenarioStep）───

// AbilityCoverageStrategy 一局中建多座塔，赋予不同攻击方式 + 能力，然后打波验证。
type AbilityCoverageStrategy struct {
	phase      int  // 0=warden, 1=build, 2=addAbility, 3=play
	builtCount int
	abilDone   bool
	// 塔位置记录
	towerPositions [][2]int // [row, col]
}

// 攻击方式能力列表（每座塔 1 种）
var atkAbils = []string{"scatter", "wideBeam", "spinAoe", "pierce", "bounce"}
var ccAbils2 = []string{"slowPower", "stunChance", "slowDuration", "stunDuration", "slowPower"}
var dmgAbils2 = []string{"crit", "splash", "flatDamage", "deathMark", "distanceDamage"}
var dotAbils2 = []string{"burn", "bleedDot", "poison", "weaken", "burn"}
var buffAbils2 = []string{"damageUpAura", "attackSpeedAura", "rangeAura", "critAura", "soloBoost"}
var zoneAbils2 = []string{"poisonZone", "silenceZone", "curseZone", "weakenZone", "poisonZone"}

func AttackStyleCoverageScenario() Strategy {
	return &AbilityCoverageStrategy{}
}

func (s *AbilityCoverageStrategy) Name() string    { return "scenario_attack-style-coverage" }
func (s *AbilityCoverageStrategy) Init(_ *GameState) {}

func (s *AbilityCoverageStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	switch s.phase {
	case 0: // 选战灵
		if !state.WardenReady {
			return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
		}
		s.phase = 1

	case 1: // 建塔（每帧尝试建 1 座）
		target := len(atkAbils)
		if s.builtCount >= target {
			s.phase = 2
			return s.Decide(state) // 立即进入下一阶段
		}
		if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
			s.phase = 2
			return s.Decide(state)
		}
		if state.Gold >= state.TowerDefs[0].Cost {
			cell := state.BuildCells[0]
			s.towerPositions = append(s.towerPositions, [2]int{cell.Row, cell.Col})
			s.builtCount++
			return []Action{{Type: ActionBuild, TowerKey: state.TowerDefs[0].Key, Cell: cell}}
		}
		// 金币不够，开波赚钱
		if !state.WaveActive {
			return []Action{{Type: ActionStartWave}}
		}

	case 2: // 给每座塔添加能力
		if !s.abilDone {
			s.abilDone = true
			var actions []Action
			for i, pos := range s.towerPositions {
				if i >= len(atkAbils) {
					break
				}
				r, c := pos[0], pos[1]
				actions = append(actions,
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: atkAbils[i]},
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: ccAbils2[i]},
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: dmgAbils2[i]},
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: dotAbils2[i]},
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: buffAbils2[i]},
					Action{Type: ActionAddAbility, Row: r, Col: c, AbilityName: zoneAbils2[i]},
				)
			}
			s.phase = 3
			return actions
		}
		s.phase = 3

	case 3: // 持续开波
		if !state.WaveActive && state.Wave < state.MaxWaves {
			return []Action{{Type: ActionStartWave}}
		}
	}
	return nil
}

// AllScenariosMap 返回所有预定义场景的名称→构造函数映射。
func AllScenariosMap() map[string]func() Strategy {
	return map[string]func() Strategy{
		// 原有
		"build-flow":       func() Strategy { return BuildFlowScenario() },
		"tower-lifecycle":  func() Strategy { return TowerLifecycleScenario() },
		"zero-gold-build":  func() Strategy { return ZeroGoldBuildScenario() },
		"rapid-actions":    func() Strategy { return RapidActionScenario() },
		// 经济验证
		"econ-sell-refund": func() Strategy { return EconSellRefundScenario() },
		"econ-rapid-build": func() Strategy { return EconRapidBuildScenario() },
		// 战斗伤害
		"combat-single-tower": func() Strategy { return CombatSingleTowerScenario() },
		"combat-multi-tower":  func() Strategy { return CombatMultiTowerScenario() },
		"combat-all-towers":   func() Strategy { return CombatAllTowerTypesScenario() },
		// 状态效果
		"cc-slow-stack": func() Strategy { return CCSlowStackScenario() },
		"cc-stun-stop":  func() Strategy { return CCStunStopScenario() },
		// 攻击方式 × 能力覆盖
		"attack-style-coverage": func() Strategy { return AttackStyleCoverageScenario() },
	}
}
