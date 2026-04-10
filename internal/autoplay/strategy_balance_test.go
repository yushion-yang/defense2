//go:build unittest

package autoplay

import "testing"

func TestBalanceGreedyStrategy_Name(t *testing.T) {
	s := NewBalanceGreedyStrategy()
	if s.Name() != "balance_greedy" {
		t.Errorf("got %q, want balance_greedy", s.Name())
	}
}

func TestBalanceGreedyStrategy_SelectsWarden(t *testing.T) {
	s := NewBalanceGreedyStrategy(WithWardenKey("chain"))
	state := &GameState{WardenReady: false}
	actions := s.Decide(state)
	if len(actions) != 1 || actions[0].Type != ActionSelectWarden || actions[0].WardenKey != "chain" {
		t.Errorf("expected warden select chain, got %+v", actions)
	}
}

func TestBalanceGreedyStrategy_NoActionsOnGameOver(t *testing.T) {
	s := NewBalanceGreedyStrategy()
	state := &GameState{GameOver: true, WardenReady: true}
	actions := s.Decide(state)
	if len(actions) != 0 {
		t.Errorf("expected no actions on game over, got %d", len(actions))
	}
}

func TestBalanceGreedyStrategy_MaxTowersLimit(t *testing.T) {
	s := NewBalanceGreedyStrategy(WithMaxTowers(1))
	state := &GameState{
		WardenReady: true,
		Gold:        200,
		Wave:        1,
		MaxWaves:    20,
		BuildCells:  []Cell{{Row: 0, Col: 0}, {Row: 1, Col: 1}},
		TowerDefs:   []TowerDefInfo{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
	}

	// First build should work
	actions := s.Decide(state)
	hasBuild := false
	for _, a := range actions {
		if a.Type == ActionBuild {
			hasBuild = true
		}
	}
	if !hasBuild {
		t.Error("expected build action on first decide")
	}

	// Second decide should not build (maxTowers=1)
	actions = s.Decide(state)
	for _, a := range actions {
		if a.Type == ActionBuild {
			t.Error("expected no build when maxTowers reached")
		}
	}
}

func TestBalanceGreedyStrategy_NoTowersMode(t *testing.T) {
	s := NewBalanceGreedyStrategy(WithNoTowers(true))
	state := &GameState{
		WardenReady: true,
		Gold:        200,
		Wave:        0,
		MaxWaves:    20,
		BuildCells:  []Cell{{Row: 0, Col: 0}},
		TowerDefs:   []TowerDefInfo{{Key: "basic", Cost: 50, Damage: 10, Range: 100}},
	}

	actions := s.Decide(state)
	for _, a := range actions {
		if a.Type == ActionBuild {
			t.Error("noTowers mode should never build")
		}
	}
}

func TestAllBalanceScenarios_Count(t *testing.T) {
	scenarios := AllBalanceScenarios()
	if len(scenarios) < 20 {
		t.Errorf("expected at least 20 balance scenarios, got %d", len(scenarios))
	}
}

func TestAllBalanceScenarios_UniqueIDs(t *testing.T) {
	scenarios := AllBalanceScenarios()
	seen := make(map[string]bool)
	for _, s := range scenarios {
		if seen[s.ID] {
			t.Errorf("duplicate scenario ID: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestAllBalanceScenarios_HaveAssertions(t *testing.T) {
	for _, s := range AllBalanceScenarios() {
		if len(s.Assertions) == 0 {
			t.Errorf("scenario %s has no assertions", s.ID)
		}
	}
}

func TestBalanceAssertionsMap_MatchesScenarios(t *testing.T) {
	m := BalanceAssertionsMap()
	for _, s := range AllBalanceScenarios() {
		if _, ok := m[s.ID]; !ok {
			t.Errorf("missing assertions for scenario %s in BalanceAssertionsMap", s.ID)
		}
	}
}

func TestFinalCheckAssertion_Victory(t *testing.T) {
	a := &Assertion{Name: "test_victory", Type: "victory"}
	fs := &FinalStats{Victory: true}
	if !checkFinalAssertion(a, fs) {
		t.Error("victory assertion should pass when Victory=true")
	}
	fs.Victory = false
	if checkFinalAssertion(a, fs) {
		t.Error("victory assertion should fail when Victory=false")
	}
}

func TestFinalCheckAssertion_WavesSurvived(t *testing.T) {
	a := &Assertion{Name: "test", Type: "waves_survived_gte", Param: 10}
	if !checkFinalAssertion(a, &FinalStats{WavesSurvived: 15}) {
		t.Error("waves_survived_gte 10 should pass for 15 waves")
	}
	if checkFinalAssertion(a, &FinalStats{WavesSurvived: 5}) {
		t.Error("waves_survived_gte 10 should fail for 5 waves")
	}
}

func TestFinalCheckAssertion_LeakRate(t *testing.T) {
	a := &Assertion{Name: "test", Type: "leak_rate_lte", Expected: 0.1}
	fs := &FinalStats{TotalKills: 100, TotalLeaked: 5}
	if !checkFinalAssertion(a, fs) {
		t.Error("leak rate 0.05 should pass for threshold 0.1")
	}
	fs.TotalLeaked = 20
	if checkFinalAssertion(a, fs) {
		t.Error("leak rate 0.2 should fail for threshold 0.1")
	}
}

func TestFinalCheckAssertion_DPSGrowth(t *testing.T) {
	a := &Assertion{Name: "test", Type: "dps_growth"}
	// Growing DPS
	fs := &FinalStats{DPSSnapshots: []float64{10, 12, 20, 25}}
	if !checkFinalAssertion(a, fs) {
		t.Error("dps_growth should pass when second half > first half")
	}
	// Declining DPS
	fs.DPSSnapshots = []float64{25, 20, 10, 5}
	if checkFinalAssertion(a, fs) {
		t.Error("dps_growth should fail when second half < first half")
	}
}

func TestFinalCheckAssertion_BossAlive(t *testing.T) {
	a := &Assertion{Name: "test", Type: "boss_alive_gte", Param: 5}
	fs := &FinalStats{BossStats: []BossStat{{AliveSeconds: 10}}}
	if !checkFinalAssertion(a, fs) {
		t.Error("boss_alive_gte 5 should pass for 10s")
	}
	fs.BossStats[0].AliveSeconds = 3
	if checkFinalAssertion(a, fs) {
		t.Error("boss_alive_gte 5 should fail for 3s")
	}
}

func TestFinalCheckAssertion_PaceNotBoring(t *testing.T) {
	a := &Assertion{Name: "test", Type: "pace_not_boring"}
	if !checkFinalAssertion(a, &FinalStats{PaceStats: &PaceStat{IdleCombatRatio: 1.5}}) {
		t.Error("pace 1.5 should not be boring")
	}
	if checkFinalAssertion(a, &FinalStats{PaceStats: &PaceStat{IdleCombatRatio: 4.0}}) {
		t.Error("pace 4.0 should be boring")
	}
}
