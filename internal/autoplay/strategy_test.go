package autoplay

import "testing"

// mockState 创建一个用于测试的基础 GameState。
func mockState() *GameState {
	return &GameState{
		Tick:     100,
		Gold:     200,
		Lives:    20,
		Wave:     3,
		MaxWaves: 25,
		BuildCells: []Cell{
			{Row: 1, Col: 2, X: 100, Y: 100},
			{Row: 2, Col: 3, X: 200, Y: 200},
			{Row: 3, Col: 4, X: 300, Y: 300},
		},
		TowerDefs: []TowerDefInfo{
			{Key: "laser", Cost: 50, Range: 120, Damage: 10, Index: 0},
			{Key: "freeze", Cost: 40, Range: 100, Damage: 5, Index: 1},
		},
		Towers:      nil,
		WardenReady: true,
	}
}

func TestRandomStrategy_Name(t *testing.T) {
	s := NewRandomStrategy(42)
	if s.Name() != "random" {
		t.Errorf("expected 'random', got %q", s.Name())
	}
}

func TestRandomStrategy_SelectsWarden(t *testing.T) {
	s := NewRandomStrategy(42)
	state := mockState()
	state.WardenReady = false

	actions := s.Decide(state)
	if len(actions) == 0 {
		t.Fatal("expected at least 1 action for warden selection")
	}
	if actions[0].Type != ActionSelectWarden {
		t.Errorf("expected ActionSelectWarden, got %v", actions[0].Type)
	}
	if actions[0].WardenKey == "" {
		t.Error("expected non-empty warden key")
	}
}

func TestRandomStrategy_NoActionsOnGameOver(t *testing.T) {
	s := NewRandomStrategy(42)
	state := mockState()
	state.GameOver = true

	actions := s.Decide(state)
	if len(actions) != 0 {
		t.Errorf("expected no actions on game over, got %d", len(actions))
	}
}

func TestGreedyStrategy_Name(t *testing.T) {
	s := NewGreedyStrategy()
	if s.Name() != "greedy" {
		t.Errorf("expected 'greedy', got %q", s.Name())
	}
}

func TestGreedyStrategy_BuildsWhenGoldAvailable(t *testing.T) {
	s := NewGreedyStrategy()
	state := mockState()

	actions := s.Decide(state)
	hasBuild := false
	for _, a := range actions {
		if a.Type == ActionBuild {
			hasBuild = true
			break
		}
	}
	if !hasBuild {
		t.Error("expected greedy to build a tower when gold available")
	}
}

func TestFocusStrategy_Name(t *testing.T) {
	s := NewFocusStrategy("laser")
	if s.Name() != "focus_laser" {
		t.Errorf("expected 'focus_laser', got %q", s.Name())
	}
}

func TestFocusStrategy_BuildsOnlyTargetType(t *testing.T) {
	s := NewFocusStrategy("laser")
	state := mockState()

	actions := s.Decide(state)
	for _, a := range actions {
		if a.Type == ActionBuild && a.TowerKey != "laser" {
			t.Errorf("focus_laser built %q instead of laser", a.TowerKey)
		}
	}
}

func TestFocusStrategy_UpgradesAfterAllBuilt(t *testing.T) {
	s := NewFocusStrategy("laser")
	state := mockState()
	state.BuildCells = nil // 没有可建位置
	state.Towers = []TowerInfo{{Key: "laser", Row: 1, Col: 2, Damage: 10, Cost: 50}}
	state.Gold = 100

	actions := s.Decide(state)
	hasUpgrade := false
	for _, a := range actions {
		if a.Type == ActionUpgrade {
			hasUpgrade = true
			break
		}
	}
	if !hasUpgrade {
		t.Error("expected focus to upgrade after all cells filled")
	}
}

func TestScenarioStrategy_Advances(t *testing.T) {
	steps := []ScenarioStep{
		{
			WaitUntil: nil,
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionStartWave}}
			},
		},
		{
			WaitUntil: func(s *GameState) bool { return s.Wave > 1 },
			Actions: func(_ *GameState) []Action {
				return []Action{{Type: ActionStartWave}}
			},
		},
	}
	s := NewScenarioStrategy("test", steps)
	state := mockState()

	// 第一步立即执行
	actions := s.Decide(state)
	if len(actions) != 1 || actions[0].Type != ActionStartWave {
		t.Error("expected first step to execute immediately")
	}

	// 第二步需要 Wave > 1（当前 Wave=3，满足）
	actions = s.Decide(state)
	if len(actions) != 1 {
		t.Error("expected second step to execute (condition met)")
	}

	// 步骤用完
	actions = s.Decide(state)
	if len(actions) != 0 {
		t.Error("expected no actions after all steps done")
	}
}
