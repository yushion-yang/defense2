// strategy_visual.go — 视觉目录策略。
// 系统性构建所有视觉内容：建造全部塔类型、装载技能、等待攻击、
// 观察状态效果。目标是单局尽可能多地触发视觉内容。
package autoplay

// VisualCatalogStrategy 视觉目录策略。
// 按顺序建造每种塔（每种一座），然后给部分塔装技能，
// 等待战斗自然产生攻击方式视觉和状态效果视觉。
type VisualCatalogStrategy struct {
	towerIdx  int    // 下一个要建的塔类型索引
	skillIdx  int    // 下一个要装的技能索引
	phase     int    // 0=建塔, 1=装技能, 2=等待观察
	waitTicks int    // 观察等待帧数
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

	// 事件轮换选择
	if state.InteractMode == 6 {
		return []Action{{Type: ActionChooseEvent, EventIndex: state.Wave % 3}}
	}

	// 开波（总是开波以产生敌人）
	if !state.WaveActive && state.Wave < state.MaxWaves {
		actions = append(actions, Action{Type: ActionStartWave})
	}

	switch s.phase {
	case 0: // 建塔阶段：每种塔建一座
		actions = append(actions, s.decideBuildAll(state)...)
	case 1: // 装技能阶段
		actions = append(actions, s.decideAssignSkills(state)...)
	case 2: // 观察阶段：等待视觉内容自然出现
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
		s.phase = 1 // 所有塔都建完了
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

// decideAssignSkills 给已建的塔轮流装技能。
func (s *VisualCatalogStrategy) decideAssignSkills(state *GameState) []Action {
	if s.skillIdx >= len(SkillNames) || s.skillIdx >= len(state.Towers) {
		s.phase = 2 // 技能装完了
		return nil
	}

	t := state.Towers[s.skillIdx]
	sk := SkillNames[s.skillIdx]
	s.skillIdx++

	var actions []Action
	actions = append(actions, Action{
		Type:      ActionAssignSkill,
		Row:       t.Row,
		Col:       t.Col,
		SkillName: sk,
	})

	// 同时给战灵装一个技能
	if s.skillIdx == 1 {
		wsk := SkillNames[len(SkillNames)-1] // 用最后一个技能
		actions = append(actions, Action{
			Type:          ActionAssignSkill,
			SkillName:     wsk,
			SkillToWarden: true,
		})
	}

	return actions
}
