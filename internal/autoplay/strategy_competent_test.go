//go:build unittest

package autoplay

import (
	"math"
	"testing"
)

// ── 路径拐角检测 ──

func TestFindCorners_LShape(t *testing.T) {
	// L 形路径：3 个点，中间有 90° 拐角
	wps := []PathPoint{
		{0, 0}, {100, 0}, {100, 100},
	}
	corners := findCorners(wps)
	if !corners[1] {
		t.Error("expected corner at index 1 (L-turn)")
	}
}

func TestFindCorners_Straight(t *testing.T) {
	// 直线：无拐角
	wps := []PathPoint{
		{0, 0}, {50, 0}, {100, 0}, {150, 0},
	}
	corners := findCorners(wps)
	for i, c := range corners {
		if c {
			t.Errorf("unexpected corner at index %d on straight line", i)
		}
	}
}

func TestFindCorners_SCurve(t *testing.T) {
	// S 形：两个 90° 拐角
	wps := []PathPoint{
		{0, 0}, {100, 0}, {100, 100}, {200, 100},
	}
	corners := findCorners(wps)
	if !corners[1] {
		t.Error("expected corner at index 1")
	}
	if !corners[2] {
		t.Error("expected corner at index 2")
	}
}

func TestFindCorners_TooFewPoints(t *testing.T) {
	corners := findCorners([]PathPoint{{0, 0}, {100, 0}})
	if len(corners) != 2 {
		t.Errorf("expected 2 elements, got %d", len(corners))
	}
	for _, c := range corners {
		if c {
			t.Error("should have no corners with < 3 points")
		}
	}
}

// ── 格子评分 ──

func TestScoreCell_CornerHigherThanStraight(t *testing.T) {
	// L 形路径
	path := []PathPoint{{0, 0}, {120, 0}, {120, 120}}
	corners := findCorners(path)

	// 靠近拐角的格子
	cornerCell := Cell{Row: 0, Col: 0, X: 150, Y: 30}
	// 靠近直线段的格子
	straightCell := Cell{Row: 0, Col: 0, X: 60, Y: 30}

	cornerScore := scoreCell(cornerCell, [][]PathPoint{path}, [][]bool{corners})
	straightScore := scoreCell(straightCell, [][]PathPoint{path}, [][]bool{corners})

	if cornerScore <= straightScore {
		t.Errorf("corner cell score (%.3f) should be > straight cell score (%.3f)",
			cornerScore, straightScore)
	}
}

func TestScoreCell_FarCellLowScore(t *testing.T) {
	path := []PathPoint{{0, 0}, {60, 0}, {120, 0}}
	corners := findCorners(path)

	nearCell := Cell{X: 60, Y: 30}
	farCell := Cell{X: 60, Y: 300}

	nearScore := scoreCell(nearCell, [][]PathPoint{path}, [][]bool{corners})
	farScore := scoreCell(farCell, [][]PathPoint{path}, [][]bool{corners})

	if nearScore <= farScore {
		t.Errorf("near cell (%.3f) should score higher than far cell (%.3f)", nearScore, farScore)
	}
}

func TestScoreCell_MultiPathBonus(t *testing.T) {
	// 两条路径在同一点交汇
	path1 := []PathPoint{{0, 0}, {100, 50}, {200, 0}}
	path2 := []PathPoint{{0, 100}, {100, 50}, {200, 100}}
	corners1 := findCorners(path1)
	corners2 := findCorners(path2)

	// 交汇点附近的格子
	cell := Cell{X: 100, Y: 50}

	multiScore := scoreCell(cell, [][]PathPoint{path1, path2}, [][]bool{corners1, corners2})
	singleScore := scoreCell(cell, [][]PathPoint{path1}, [][]bool{corners1})

	if multiScore <= singleScore {
		t.Errorf("multi-path score (%.3f) should be > single-path score (%.3f)",
			multiScore, singleScore)
	}
}

// ── 策略决策 ──

func TestCompetentStrategy_SelectsWarden(t *testing.T) {
	s := NewCompetentStrategy(WithCompetentWarden("sage"))
	state := &GameState{
		Lives:    20,
		Gold:     100,
		MaxWaves: 12,
	}
	s.Init(state)
	actions := s.Decide(state)
	if len(actions) != 1 || actions[0].Type != ActionSelectWarden || actions[0].WardenKey != "sage" {
		t.Error("first action should be selecting warden 'sage'")
	}
}

func TestCompetentStrategy_BuildsBeforeStartingWave(t *testing.T) {
	s := NewCompetentStrategy(WithCompetentSeed(42))
	state := &GameState{
		Lives:       20,
		Gold:        100,
		MaxWaves:    12,
		WardenReady: true,
		BuildCells: []Cell{
			{Row: 3, Col: 5, X: 330, Y: 210},
			{Row: 5, Col: 8, X: 510, Y: 330},
		},
		TowerDefs: []TowerDefInfo{
			{Key: "basic", Cost: 30, Range: 100, Damage: 10, Index: 0},
		},
		MapInfo: &MapInfo{
			Waypoints: []PathPoint{{300, 200}, {400, 200}, {400, 300}},
			CellSize:  60,
			Rows:      9,
			Cols:      20,
		},
	}
	s.Init(state)
	actions := s.Decide(state)

	// In setup phase, should build (not start wave)
	hasBuild := false
	hasWave := false
	for _, a := range actions {
		if a.Type == ActionBuild {
			hasBuild = true
		}
		if a.Type == ActionStartWave {
			hasWave = true
		}
	}
	if !hasBuild {
		t.Error("should build during setup phase")
	}
	if hasWave {
		t.Error("should not start wave during setup phase")
	}
}

func TestCompetentStrategy_StartsWaveAfterSetup(t *testing.T) {
	s := NewCompetentStrategy(WithCompetentSeed(42))
	state := &GameState{
		Lives:       20,
		Gold:        5, // 买不起塔
		MaxWaves:    12,
		WardenReady: true,
		BuildCells:  []Cell{{Row: 3, Col: 5, X: 330, Y: 210}},
		TowerDefs:   []TowerDefInfo{{Key: "basic", Cost: 30, Range: 100, Damage: 10}},
		Towers: []TowerInfo{
			{Key: "basic", Row: 1, Col: 2, X: 150, Y: 90, Damage: 10, Range: 100, AttackSpeed: 1.0},
		},
		MapInfo: &MapInfo{
			Waypoints: []PathPoint{{300, 200}},
			CellSize:  60, Rows: 9, Cols: 20,
		},
	}
	s.Init(state)
	// 模拟已经建过塔（触发 setup → playing 转换）
	s.builtCount = 2
	actions := s.Decide(state)

	// 有塔但买不起新塔 → 应开波
	hasWave := false
	for _, a := range actions {
		if a.Type == ActionStartWave {
			hasWave = true
		}
	}
	if !hasWave {
		t.Error("should start wave when can't afford to build but has towers")
	}
}

func TestCompetentStrategy_NoWaveWithoutTowers(t *testing.T) {
	s := NewCompetentStrategy(WithCompetentSeed(42))
	state := &GameState{
		Lives:       20,
		Gold:        5,
		MaxWaves:    12,
		WardenReady: true,
		BuildCells:  []Cell{{Row: 3, Col: 5, X: 330, Y: 210}},
		TowerDefs:   []TowerDefInfo{{Key: "basic", Cost: 30, Range: 100, Damage: 10}},
		MapInfo: &MapInfo{
			Waypoints: []PathPoint{{300, 200}},
			CellSize:  60, Rows: 9, Cols: 20,
		},
	}
	s.Init(state)
	actions := s.Decide(state)

	// 没有塔 → 不应该开波
	for _, a := range actions {
		if a.Type == ActionStartWave {
			t.Error("should NOT start wave when no towers exist")
		}
	}
}

func TestCompetentStrategy_Name(t *testing.T) {
	s := NewCompetentStrategy()
	if s.Name() != "competent" {
		t.Errorf("expected 'competent', got %q", s.Name())
	}
}

// ── 场景矩阵 ──

func TestAllSimScenarios_Count(t *testing.T) {
	scenarios := AllSimScenarios()
	if len(scenarios) < 60 {
		t.Errorf("expected at least 60 scenarios, got %d", len(scenarios))
	}
}

func TestAllSimScenarios_UniqueIDs(t *testing.T) {
	scenarios := AllSimScenarios()
	seen := make(map[string]bool)
	for _, s := range scenarios {
		if seen[s.ID] {
			t.Errorf("duplicate scenario ID: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestAllSimScenarios_AllHaveAssertions(t *testing.T) {
	for _, s := range AllSimScenarios() {
		if len(s.Assertions) == 0 {
			t.Errorf("scenario %s has no assertions", s.ID)
		}
	}
}

func TestAllSimScenarios_ToTestCase(t *testing.T) {
	scenarios := AllSimScenarios()
	tc := scenarios[0].ToTestCase()
	if tc.ModeID != "simulation" {
		t.Errorf("expected ModeID='simulation', got %q", tc.ModeID)
	}
}

func TestStableHash_Deterministic(t *testing.T) {
	h1 := stableHash("sim_map_01_easy")
	h2 := stableHash("sim_map_01_easy")
	if h1 != h2 {
		t.Error("stableHash should be deterministic")
	}
	h3 := stableHash("sim_map_02_easy")
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

// ── 辅助 ──

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

var _ = almostEqual // suppress unused
