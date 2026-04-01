// strategy_visual.go — 视觉目录策略。
// 系统性构建所有视觉内容：建造全部塔类型、等待攻击、
// 观察状态效果。目标是单局尽可能多地触发视觉内容。
package autoplay

// VisualCatalogStrategy 视觉目录策略。
// 按顺序建造每种塔（每种一座），然后
// 等待战斗自然产生攻击方式视觉和状态效果视觉。
type VisualCatalogStrategy struct {
	towerIdx  int // 下一个要建的塔类型索引
	phase     int // 0=建塔, 1=等待观察
	waitTicks int // 观察等待帧数
}

// NewVisualCatalogStrategy 创建视觉目录策略。
func NewVisualCatalogStrategy() *VisualCatalogStrategy {
	return &VisualCatalogStrategy{}
}

func (s *VisualCatalogStrategy) Name() string { return "visual_catalog" }
func (s *VisualCatalogStrategy) Init(_ *GameState) {}

func (s *VisualCatalogStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
	}

	// 开波（总是开波以产生敌人）
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	switch s.phase {
	case 0: // 建塔阶段：每种塔建一座
		actions = append(actions, s.decideBuildAll(state)...)
	case 1: // 观察阶段：等待视觉内容自然出现
		s.waitTicks++
		// 持续升级以保持战斗活跃
		if s.waitTicks%30 == 0 && len(state.Towers) > 0 && state.Gold >= 10 {
			idx := (s.waitTicks / 30) % len(state.Towers)
			t := state.Towers[idx]
			actions = append(actions, Action{Type: ActionUpgrade, Row: t.Row, Col: t.Col})
		}
	}

	return actions
}

// decideBuildAll 每种塔建一座。
func (s *VisualCatalogStrategy) decideBuildAll(state *GameState) []Action {
	if s.towerIdx >= len(TowerKeys) || len(state.BuildCells) == 0 {
		s.phase = 1 // 所有塔都建完了，进入观察阶段
		return nil
	}

	key := TowerKeys[s.towerIdx]

	// 找到对应的塔定义
	var cost int
	found := false
	for _, d := range state.TowerDefs {
		if d.Key == key {
			cost = d.Cost
			found = true
			break
		}
	}
	if !found {
		s.towerIdx++
		return nil
	}

	if state.Gold < cost {
		return nil // 等钱
	}

	cell := state.BuildCells[0]
	s.towerIdx++
	return []Action{{Type: ActionBuild, TowerKey: key, Cell: cell}}
}

