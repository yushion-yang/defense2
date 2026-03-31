// strategy_focus.go — 单塔极限策略。
// 只建一种塔，填满所有位置后全力升级。测试单塔上限和平衡性。
package autoplay

// FocusStrategy 单塔策略。
type FocusStrategy struct {
	towerKey  string
	wardenKey string
	allBuilt  bool
	upgradeIdx int // 轮询升级的塔索引
}

// NewFocusStrategy 创建单塔策略。
func NewFocusStrategy(towerKey string) *FocusStrategy {
	return &FocusStrategy{
		towerKey:  towerKey,
		wardenKey: "prince",
	}
}

func (s *FocusStrategy) Name() string    { return "focus_" + s.towerKey }
func (s *FocusStrategy) TowerKey() string { return s.towerKey }

func (s *FocusStrategy) Init(_ *GameState) {}

func (s *FocusStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: s.wardenKey}}
	}

	// 开波
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	// 找到目标塔定义
	var targetDef *TowerDefInfo
	for _, d := range state.TowerDefs {
		if d.Key == s.towerKey {
			def := d
			targetDef = &def
			break
		}
	}
	if targetDef == nil {
		return actions
	}

	// 阶段1：建塔（填满所有位置）
	if !s.allBuilt && len(state.BuildCells) > 0 {
		if state.Gold >= targetDef.Cost {
			cell := state.BuildCells[0]
			actions = append(actions, Action{
				Type:     ActionBuild,
				TowerKey: s.towerKey,
				Cell:     cell,
			})
		}
		return actions
	}

	if len(state.BuildCells) == 0 {
		s.allBuilt = true
	}

	// 阶段2：轮询升级
	if len(state.Towers) > 0 && state.Gold >= 10 {
		idx := s.upgradeIdx % len(state.Towers)
		t := state.Towers[idx]
		actions = append(actions, Action{
			Type: ActionUpgrade,
			Row:  t.Row,
			Col:  t.Col,
		})
		s.upgradeIdx++
	}

	return actions
}
