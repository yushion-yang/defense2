package autoplay

import "testing"

func TestGenerateTestPlan_Count(t *testing.T) {
	cases := GenerateTestPlan()
	// 应生成合理数量的测试用例（>40）
	if len(cases) < 40 {
		t.Errorf("expected at least 40 test cases, got %d", len(cases))
	}
}

func TestGenerateTestPlan_UniqueIDs(t *testing.T) {
	cases := GenerateTestPlan()
	seen := make(map[string]bool)
	for _, tc := range cases {
		if seen[tc.ID] {
			t.Errorf("duplicate test case ID: %s", tc.ID)
		}
		seen[tc.ID] = true
	}
}

func TestGenerateTestPlan_AllTowersCovered(t *testing.T) {
	cases := GenerateTestPlan()
	towers := make(map[string]bool)
	for _, tc := range cases {
		if f, ok := tc.Strategy.(*FocusStrategy); ok {
			towers[f.towerKey] = true
		}
	}
	for _, key := range TowerKeys {
		if !towers[key] {
			t.Errorf("tower %q not covered by any focus test", key)
		}
	}
}

func TestParseCLICases(t *testing.T) {
	cases := ParseCLICases(2, "random,greedy", "map_01", "normal", "prince")
	// 2 runs × 2 strategies = 4
	if len(cases) != 4 {
		t.Errorf("expected 4 cases, got %d", len(cases))
	}
}

func TestParseCLICases_Focus(t *testing.T) {
	cases := ParseCLICases(1, "focus", "map_01", "normal", "prince")
	// 1 run × 8 towers = 8
	if len(cases) != 8 {
		t.Errorf("expected 8 focus cases, got %d", len(cases))
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a,b,c", 3},
		{"single", 1},
		{"", 0},
		{"a,,b", 2},
	}
	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitCSV(%q) = %d items, want %d", tt.input, len(got), tt.want)
		}
	}
}
