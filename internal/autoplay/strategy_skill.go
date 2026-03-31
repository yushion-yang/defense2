// strategy_skill.go — 技能测试策略。
// 建塔后立即装指定技能，同时给战灵装技能。验证技能分配和触发。
package autoplay

// SkillTestStrategy 技能测试策略。
type SkillTestStrategy struct {
	skillName    string
	towerBuilt   bool
	skillAssigned bool
	wardenSkilled bool
	inner        *GreedyStrategy // 复用贪心策略做基础决策
}

// NewSkillTestStrategy 创建技能测试策略。
func NewSkillTestStrategy(skillName string) *SkillTestStrategy {
	return &SkillTestStrategy{
		skillName: skillName,
		inner:     NewGreedyStrategy(),
	}
}

func (s *SkillTestStrategy) Name() string      { return "skill_" + s.skillName }
func (s *SkillTestStrategy) SkillName() string { return s.skillName }

func (s *SkillTestStrategy) Init(state *GameState) {
	s.inner.Init(state)
}

func (s *SkillTestStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	// 基础决策
	actions := s.inner.Decide(state)

	// 有塔后装技能
	if !s.skillAssigned && len(state.Towers) > 0 {
		t := state.Towers[0]
		if t.SkillName == "" {
			actions = append(actions, Action{
				Type:      ActionAssignSkill,
				Row:       t.Row,
				Col:       t.Col,
				SkillName: s.skillName,
			})
			s.skillAssigned = true
		}
	}

	// 战灵也装技能（用不同的技能）
	if !s.wardenSkilled && state.WardenReady {
		// 给战灵装一个不同于塔的技能
		wardenSkill := s.skillName
		if len(SkillNames) > 1 {
			for _, sk := range SkillNames {
				if sk != s.skillName {
					wardenSkill = sk
					break
				}
			}
		}
		actions = append(actions, Action{
			Type:          ActionAssignSkill,
			SkillName:     wardenSkill,
			SkillToWarden: true,
		})
		s.wardenSkilled = true
	}

	return actions
}
