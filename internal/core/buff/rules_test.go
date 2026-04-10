package buff

import (
	"testing"
)

func TestLoadRules(t *testing.T) {
	jsonData := []byte(`{
		"modes": {},
		"rules": {
			"slow": {"mode": "strongest", "cap": 0.8},
			"stun": {"mode": "override"},
			"bleed": {"mode": "independentPerSource"},
			"weaken": {"mode": "strongest", "cap": 0.5},
			"damageDown": {"mode": "multiplicative", "floor": 0.2},
			"controlImmune": {"mode": "override", "priority": 80}
		}
	}`)
	rules, err := LoadRules(jsonData)
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}
	if len(rules) != 6 {
		t.Fatalf("want 6 rules, got %d", len(rules))
	}

	tests := []struct {
		id       string
		wantMode StackMode
		wantCap  float64
		wantFlr  float64
		wantPri  int
	}{
		{"slow", Strongest, 0.8, 0, 0},
		{"stun", Override, 0, 0, 0},
		{"bleed", IndependentPerSource, 0, 0, 0},
		{"weaken", Strongest, 0.5, 0, 0},
		{"damageDown", Multiplicative, 0, 0.2, 0},
		{"controlImmune", Override, 0, 0, 80},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			r, ok := rules[tc.id]
			if !ok {
				t.Fatalf("missing rule for %q", tc.id)
			}
			if r.Mode != tc.wantMode {
				t.Errorf("mode: want %d, got %d", tc.wantMode, r.Mode)
			}
			if r.Cap != tc.wantCap {
				t.Errorf("cap: want %f, got %f", tc.wantCap, r.Cap)
			}
			if r.Floor != tc.wantFlr {
				t.Errorf("floor: want %f, got %f", tc.wantFlr, r.Floor)
			}
			if r.Priority != tc.wantPri {
				t.Errorf("priority: want %d, got %d", tc.wantPri, r.Priority)
			}
		})
	}
}

func TestLoadRules_InvalidJSON(t *testing.T) {
	_, err := LoadRules([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadRules_UnknownMode(t *testing.T) {
	jsonData := []byte(`{"rules": {"test": {"mode": "unknownMode"}}}`)
	_, err := LoadRules(jsonData)
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestLoadRules_EmptyRules(t *testing.T) {
	jsonData := []byte(`{"rules": {}}`)
	rules, err := LoadRules(jsonData)
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}
}
