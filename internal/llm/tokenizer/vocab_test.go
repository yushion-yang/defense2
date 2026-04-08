package tokenizer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testVocabJSON = `{
  "version": 1,
  "tokens": {
    "PAD": 0, "BOS": 1, "EOS": 2, "SEP": 3,
    "G0": 4, "G1": 5,
    "ACT_BUILD": 10, "ACT_WAIT": 11,
    "k_laser": 20, "R0C0": 30
  }
}`

func TestLoadVocab(t *testing.T) {
	v, err := LoadVocab(strings.NewReader(testVocabJSON))
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}

	if got := v.Size(); got != 10 {
		t.Errorf("Size() = %d, want 10", got)
	}

	tests := []struct {
		token string
		id    int
	}{
		{"PAD", 0},
		{"BOS", 1},
		{"EOS", 2},
		{"SEP", 3},
		{"G0", 4},
		{"G1", 5},
		{"ACT_BUILD", 10},
		{"ACT_WAIT", 11},
		{"k_laser", 20},
		{"R0C0", 30},
	}
	for _, tt := range tests {
		if got := v.TokenToID(tt.token); got != tt.id {
			t.Errorf("TokenToID(%q) = %d, want %d", tt.token, got, tt.id)
		}
		if got := v.IDToToken(tt.id); got != tt.token {
			t.Errorf("IDToToken(%d) = %q, want %q", tt.id, got, tt.token)
		}
	}
}

func TestVocab_SpecialTokens(t *testing.T) {
	v, err := LoadVocab(strings.NewReader(testVocabJSON))
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}

	if got := v.PAD(); got != 0 {
		t.Errorf("PAD() = %d, want 0", got)
	}
	if got := v.BOS(); got != 1 {
		t.Errorf("BOS() = %d, want 1", got)
	}
	if got := v.EOS(); got != 2 {
		t.Errorf("EOS() = %d, want 2", got)
	}
	if got := v.SEP(); got != 3 {
		t.Errorf("SEP() = %d, want 3", got)
	}
}

func TestVocab_UnknownToken(t *testing.T) {
	v, err := LoadVocab(strings.NewReader(testVocabJSON))
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}

	if got := v.TokenToID("NONEXISTENT"); got != -1 {
		t.Errorf("TokenToID(NONEXISTENT) = %d, want -1", got)
	}
	if got := v.IDToToken(999); got != "" {
		t.Errorf("IDToToken(999) = %q, want empty", got)
	}
}

func TestLoadVocab_RealFile(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	vocabPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "config", "llm", "vocab.json")

	f, err := os.Open(vocabPath)
	if err != nil {
		t.Fatalf("open vocab.json: %v", err)
	}
	defer f.Close()

	v, err := LoadVocab(f)
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}

	if got := v.Size(); got != 208 {
		t.Errorf("Size() = %d, want 208", got)
	}

	keyTokens := []string{
		"PAD", "BOS", "EOS", "SEP",
		"G0", "ENM", "TWR",
		"ACT_BUILD", "ACT_WAIT", "ACT_WAVE",
		"k_laser", "R0C0",
		"WDN", "w_prince",
	}
	for _, tok := range keyTokens {
		if id := v.TokenToID(tok); id == -1 {
			t.Errorf("key token %q not found in real vocab", tok)
		}
	}
}

func TestVocab_Bidirectional(t *testing.T) {
	v, err := LoadVocab(strings.NewReader(testVocabJSON))
	if err != nil {
		t.Fatalf("LoadVocab: %v", err)
	}

	for token, id := range v.tokenToID {
		gotID := v.TokenToID(token)
		if gotID != id {
			t.Errorf("TokenToID(%q) = %d, want %d", token, gotID, id)
		}
		gotToken := v.IDToToken(gotID)
		if gotToken != token {
			t.Errorf("IDToToken(%d) = %q, want %q", gotID, gotToken, token)
		}
	}
}
