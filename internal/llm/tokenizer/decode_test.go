package tokenizer

import (
	"testing"
)

// tokenIDs converts token names to IDs using the vocab.
func tokenIDs(v *Vocab, tokens ...string) []int {
	out := make([]int, len(tokens))
	for i, tok := range tokens {
		out[i] = v.TokenToID(tok)
	}
	return out
}

func TestDecodeActions_Build(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_BUILD", "k_laser", "R1C4"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.Type != ActBuild {
		t.Errorf("Type = %d, want ActBuild(%d)", a.Type, ActBuild)
	}
	if a.TowerKey != "laser" {
		t.Errorf("TowerKey = %q, want %q", a.TowerKey, "laser")
	}
	if a.Row != 1 || a.Col != 4 {
		t.Errorf("Row,Col = %d,%d, want 1,4", a.Row, a.Col)
	}
}

func TestDecodeActions_Upgrade(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_UPGRADE", "R2C5"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.Type != ActUpgrade {
		t.Errorf("Type = %d, want ActUpgrade(%d)", a.Type, ActUpgrade)
	}
	if a.Row != 2 || a.Col != 5 {
		t.Errorf("Row,Col = %d,%d, want 2,5", a.Row, a.Col)
	}
}

func TestDecodeActions_Sell(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_SELL", "R3C6"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.Type != ActSell {
		t.Errorf("Type = %d, want ActSell(%d)", a.Type, ActSell)
	}
	if a.Row != 3 || a.Col != 6 {
		t.Errorf("Row,Col = %d,%d, want 3,6", a.Row, a.Col)
	}
}

func TestDecodeActions_Wave(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_WAVE"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Type != ActWave {
		t.Errorf("Type = %d, want ActWave(%d)", actions[0].Type, ActWave)
	}
}

func TestDecodeActions_Warden(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_WARDEN", "w_prince"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	a := actions[0]
	if a.Type != ActWarden {
		t.Errorf("Type = %d, want ActWarden(%d)", a.Type, ActWarden)
	}
	if a.WardenKey != "prince" {
		t.Errorf("WardenKey = %q, want %q", a.WardenKey, "prince")
	}
}


func TestDecodeActions_Wait(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_WAIT"))

	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1", len(actions))
	}
	if actions[0].Type != ActWait {
		t.Errorf("Type = %d, want ActWait(%d)", actions[0].Type, ActWait)
	}
}

func TestDecodeActions_Multiple(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, tokenIDs(v, "ACT_BUILD", "k_freeze", "R0C0", "ACT_WAVE"))

	if len(actions) != 2 {
		t.Fatalf("got %d actions, want 2", len(actions))
	}

	a0 := actions[0]
	if a0.Type != ActBuild {
		t.Errorf("actions[0].Type = %d, want ActBuild(%d)", a0.Type, ActBuild)
	}
	if a0.TowerKey != "freeze" {
		t.Errorf("actions[0].TowerKey = %q, want %q", a0.TowerKey, "freeze")
	}
	if a0.Row != 0 || a0.Col != 0 {
		t.Errorf("actions[0].Row,Col = %d,%d, want 0,0", a0.Row, a0.Col)
	}

	a1 := actions[1]
	if a1.Type != ActWave {
		t.Errorf("actions[1].Type = %d, want ActWave(%d)", a1.Type, ActWave)
	}
}

func TestDecodeActions_Incomplete(t *testing.T) {
	v := loadTestVocab(t)
	// ACT_BUILD needs k_type + grid, but only k_laser follows (no grid)
	actions := DecodeActions(v, tokenIDs(v, "ACT_BUILD", "k_laser"))

	if len(actions) != 0 {
		t.Errorf("got %d actions, want 0 (incomplete sequence)", len(actions))
	}
}

func TestDecodeActions_Empty(t *testing.T) {
	v := loadTestVocab(t)
	actions := DecodeActions(v, []int{})

	if len(actions) != 0 {
		t.Errorf("got %d actions, want 0", len(actions))
	}
}

func TestParseGridToken(t *testing.T) {
	tests := []struct {
		token   string
		wantRow int
		wantCol int
		wantOK  bool
	}{
		{"R0C0", 0, 0, true},
		{"R2C5", 2, 5, true},
		{"R5C11", 5, 11, true},
		{"R1C4", 1, 4, true},
		{"", 0, 0, false},
		{"k_laser", 0, 0, false},
		{"RC", 0, 0, false},
		{"R-1C0", 0, 0, false},
		{"RxCy", 0, 0, false},
	}
	for _, tt := range tests {
		row, col, ok := parseGridToken(tt.token)
		if ok != tt.wantOK {
			t.Errorf("parseGridToken(%q): ok = %v, want %v", tt.token, ok, tt.wantOK)
			continue
		}
		if ok && (row != tt.wantRow || col != tt.wantCol) {
			t.Errorf("parseGridToken(%q) = (%d, %d), want (%d, %d)",
				tt.token, row, col, tt.wantRow, tt.wantCol)
		}
	}
}
