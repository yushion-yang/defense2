package autoplay

import "testing"

func TestLLMStrategy_Name(t *testing.T) {
	s := &LLMStrategy{}
	if s.Name() != "llm" {
		t.Errorf("expected 'llm', got %q", s.Name())
	}
}

func TestLLMStrategy_ShouldDecide_WaveChange(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)
	if s.shouldDecide(state) {
		t.Error("should not decide when nothing changed")
	}
	state.WaveActive = true
	if !s.shouldDecide(state) {
		t.Error("should decide when wave state changes")
	}
}

func TestLLMStrategy_ShouldDecide_GoldCross(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 40
	s.snapshot(state)
	state.Gold = 60
	if !s.shouldDecide(state) {
		t.Error("should decide when gold crosses bucket")
	}
}

func TestLLMStrategy_ShouldDecide_Interval(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)
	// Simulate Decide() incrementing tickSinceDecide before each shouldDecide call.
	// shouldDecide only resets tick counter when triggered, so it accumulates.
	for range 59 {
		s.tickSinceDecide++
		if s.shouldDecide(state) {
			t.Fatal("should not decide before 60 ticks")
		}
	}
	s.tickSinceDecide++
	if !s.shouldDecide(state) {
		t.Error("should decide after 60 ticks")
	}
}

func TestFilterValid_RejectsInsuffGold(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 10
	actions := []Action{{Type: ActionBuild, TowerKey: "laser", Cell: state.BuildCells[0], Row: state.BuildCells[0].Row, Col: state.BuildCells[0].Col}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject build when gold insufficient")
	}
}

func TestFilterValid_AcceptsValidBuild(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 200
	actions := []Action{{Type: ActionBuild, TowerKey: "laser", Cell: state.BuildCells[0], Row: state.BuildCells[0].Row, Col: state.BuildCells[0].Col}}
	valid := s.filterValid(actions, state)
	if len(valid) != 1 {
		t.Error("should accept valid build")
	}
}

func TestFilterValid_RejectsOccupiedCell(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Gold = 200
	// Build on row=5, col=5 which is NOT in BuildCells
	actions := []Action{{Type: ActionBuild, TowerKey: "laser", Row: 5, Col: 5}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject build on non-available cell")
	}
}

func TestFilterValid_WaveNotActive(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.WaveActive = true
	actions := []Action{{Type: ActionStartWave}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject start wave when already active")
	}
}

func TestGoldToBucket(t *testing.T) {
	tests := []struct{ gold, bucket int }{
		{0, 0}, {49, 0}, {50, 1}, {99, 1}, {100, 2}, {199, 2},
		{200, 3}, {399, 3}, {400, 4}, {799, 4}, {800, 5}, {9999, 5},
	}
	for _, tc := range tests {
		if got := goldToBucket(tc.gold); got != tc.bucket {
			t.Errorf("goldToBucket(%d) = %d, want %d", tc.gold, got, tc.bucket)
		}
	}
}

func TestLLMStrategy_ShouldDecide_LivesDrop(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)
	state.Lives = state.Lives - 1
	if !s.shouldDecide(state) {
		t.Error("should decide when lives drop")
	}
}

func TestLLMStrategy_ShouldDecide_TowersChanged(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	s.snapshot(state)
	state.Towers = append(state.Towers, TowerInfo{Key: "laser", Row: 1, Col: 2})
	if !s.shouldDecide(state) {
		t.Error("should decide when tower count changes")
	}
}

func TestFilterValid_UpgradeExistingTower(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Towers = []TowerInfo{{Key: "laser", Row: 1, Col: 2}}
	actions := []Action{{Type: ActionUpgrade, Row: 1, Col: 2}}
	valid := s.filterValid(actions, state)
	if len(valid) != 1 {
		t.Error("should accept upgrade on existing tower")
	}
}

func TestFilterValid_UpgradeNonExistent(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	actions := []Action{{Type: ActionUpgrade, Row: 9, Col: 9}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject upgrade on non-existent tower")
	}
}

func TestFilterValid_SellExistingTower(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Towers = []TowerInfo{{Key: "laser", Row: 1, Col: 2}}
	actions := []Action{{Type: ActionSell, Row: 1, Col: 2}}
	valid := s.filterValid(actions, state)
	if len(valid) != 1 {
		t.Error("should accept sell on existing tower")
	}
}

func TestFilterValid_WaveAtMaxWaves(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	state.Wave = state.MaxWaves
	actions := []Action{{Type: ActionStartWave}}
	valid := s.filterValid(actions, state)
	if len(valid) != 0 {
		t.Error("should reject start wave when at max waves")
	}
}

func TestFilterValid_NoopAlwaysValid(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	actions := []Action{{Type: ActionNoop}}
	valid := s.filterValid(actions, state)
	if len(valid) != 1 {
		t.Error("noop should always be valid")
	}
}

func TestConvertActions_Build(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	decoded := []decodedAction{
		{typ: actBuild, towerKey: "laser", row: 1, col: 2},
	}
	actions := s.convertActions(decoded, state)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	a := actions[0]
	if a.Type != ActionBuild || a.TowerKey != "laser" || a.Row != 1 || a.Col != 2 {
		t.Errorf("unexpected build action: %+v", a)
	}
	// Cell should be filled from BuildCells
	if a.Cell.Row != 1 || a.Cell.Col != 2 {
		t.Errorf("expected cell (1,2), got (%d,%d)", a.Cell.Row, a.Cell.Col)
	}
}

func TestConvertActions_Mixed(t *testing.T) {
	s := &LLMStrategy{}
	state := mockState()
	decoded := []decodedAction{
		{typ: actWave},
		{typ: actWait},
		{typ: actUpgrade, row: 3, col: 4},
	}
	actions := s.convertActions(decoded, state)
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
	if actions[0].Type != ActionStartWave {
		t.Errorf("expected ActionStartWave, got %v", actions[0].Type)
	}
	if actions[1].Type != ActionNoop {
		t.Errorf("expected ActionNoop, got %v", actions[1].Type)
	}
	if actions[2].Type != ActionUpgrade || actions[2].Row != 3 || actions[2].Col != 4 {
		t.Errorf("unexpected upgrade action: %+v", actions[2])
	}
}
