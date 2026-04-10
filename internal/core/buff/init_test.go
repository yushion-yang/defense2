package buff

import (
	"sync"
	"testing"
)

// resetGlobalRules resets the singleton for test isolation.
func resetGlobalRules() {
	globalRules = nil
	globalRulesOnce = sync.Once{}
}

func TestInitGlobalRules(t *testing.T) {
	resetGlobalRules()
	defer resetGlobalRules()

	jsonData := []byte(`{
		"modes": {},
		"rules": {
			"slow":    {"mode": "strongest", "cap": 0.8},
			"stun":    {"mode": "override"},
			"dot":     {"mode": "independentPerSource"},
			"shield":  {"mode": "independent"}
		}
	}`)

	if err := InitGlobalRules(jsonData); err != nil {
		t.Fatalf("InitGlobalRules: %v", err)
	}

	rules := GlobalRules()
	if rules == nil {
		t.Fatal("GlobalRules() returned nil after init")
	}
	if len(rules) != 4 {
		t.Fatalf("want 4 rules, got %d", len(rules))
	}
	if rules["slow"].Mode != Strongest {
		t.Errorf("slow mode: want Strongest, got %d", rules["slow"].Mode)
	}
	if rules["slow"].Cap != 0.8 {
		t.Errorf("slow cap: want 0.8, got %f", rules["slow"].Cap)
	}
}

func TestInitGlobalRules_OnceSemantics(t *testing.T) {
	resetGlobalRules()
	defer resetGlobalRules()

	first := []byte(`{"rules": {"slow": {"mode": "strongest"}}}`)
	second := []byte(`{"rules": {"stun": {"mode": "override"}}}`)

	if err := InitGlobalRules(first); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call should be a no-op.
	if err := InitGlobalRules(second); err != nil {
		t.Fatalf("second call: %v", err)
	}

	rules := GlobalRules()
	if _, ok := rules["slow"]; !ok {
		t.Error("expected 'slow' from first call")
	}
	if _, ok := rules["stun"]; ok {
		t.Error("'stun' should not exist — second call should be no-op")
	}
}

func TestInitGlobalRules_InvalidJSON(t *testing.T) {
	resetGlobalRules()
	defer resetGlobalRules()

	err := InitGlobalRules([]byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if GlobalRules() != nil {
		t.Error("GlobalRules should be nil after failed init")
	}
}

func TestNewDefaultBuffList_WithRules(t *testing.T) {
	resetGlobalRules()
	defer resetGlobalRules()

	jsonData := []byte(`{"rules": {"slow": {"mode": "strongest", "cap": 0.8}}}`)
	if err := InitGlobalRules(jsonData); err != nil {
		t.Fatalf("InitGlobalRules: %v", err)
	}

	bl := NewDefaultBuffList()
	if bl == nil {
		t.Fatal("NewDefaultBuffList returned nil")
	}

	// Verify it uses global rules — add a slow buff and check cap.
	bl.Add(Buff{ID: "slow", Value: 0.95, Duration: 5, Remaining: 5})
	got, ok := bl.Get("slow")
	if !ok {
		t.Fatal("slow buff not found")
	}
	if got.Value != 0.8 {
		t.Errorf("slow value should be capped at 0.8, got %f", got.Value)
	}
}

func TestNewDefaultBuffList_WithoutInit(t *testing.T) {
	resetGlobalRules()
	defer resetGlobalRules()

	// Without InitGlobalRules, should still work (empty rules → Override fallback).
	bl := NewDefaultBuffList()
	if bl == nil {
		t.Fatal("NewDefaultBuffList returned nil without init")
	}

	bl.Add(Buff{ID: "slow", Value: 0.5, Duration: 3, Remaining: 3})
	if !bl.Has("slow") {
		t.Error("buff should exist with fallback rules")
	}
}
