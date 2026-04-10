package tokenizer

import (
	"os"
	"testing"
)

func loadTestVocab(t *testing.T) *Vocab {
	t.Helper()
	f, err := os.Open("../../../config/llm/vocab.json")
	if err != nil {
		t.Fatalf("open vocab: %v", err)
	}
	defer f.Close()
	v, err := LoadVocab(f)
	if err != nil {
		t.Fatalf("load vocab: %v", err)
	}
	return v
}

// helper: convert token IDs back to token strings for readable assertions.
func idsToTokens(v *Vocab, ids []int) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		tok := v.IDToToken(id)
		if tok == "" {
			tok = "???"
		}
		out[i] = tok
	}
	return out
}

func containsToken(tokens []string, target string) bool {
	for _, t := range tokens {
		if t == target {
			return true
		}
	}
	return false
}

func TestEncode_EmptyState(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold:       100,
		Lives:      20,
		MaxLives:   20,
		Wave:       1,
		WaveActive: false,
		GameSpeed:  1,
		MapPixelW:  1200,
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	// Must start with BOS, SEP
	if tokens[0] != "BOS" {
		t.Errorf("first token = %q, want BOS", tokens[0])
	}
	if tokens[1] != "SEP" {
		t.Errorf("second token = %q, want SEP", tokens[1])
	}
	// Must end with EOS
	if tokens[len(tokens)-1] != "EOS" {
		t.Errorf("last token = %q, want EOS", tokens[len(tokens)-1])
	}

	// Header section: gold=G2, lives=L9, W1, WIDLE, SPD1, SEP
	if !containsToken(tokens, "G2") {
		t.Errorf("expected G2 for gold=100, tokens=%v", tokens)
	}
	if !containsToken(tokens, "L9") {
		t.Errorf("expected L9 for lives=20/20, tokens=%v", tokens)
	}
	if !containsToken(tokens, "W1") {
		t.Errorf("expected W1, tokens=%v", tokens)
	}
	if !containsToken(tokens, "WIDLE") {
		t.Errorf("expected WIDLE, tokens=%v", tokens)
	}
	if !containsToken(tokens, "SPD1") {
		t.Errorf("expected SPD1, tokens=%v", tokens)
	}

	// No ENM, TWR, CELL, WDN tokens
	for _, tok := range tokens {
		if tok == "ENM" || tok == "TWR" || tok == "CELL" || tok == "WDN" {
			t.Errorf("unexpected token %q in empty state", tok)
		}
	}
}

func TestEncode_GoldBuckets(t *testing.T) {
	v := loadTestVocab(t)

	tests := []struct {
		gold int
		want string
	}{
		{0, "G0"},
		{49, "G0"},
		{50, "G1"},
		{99, "G1"},
		{100, "G2"},
		{199, "G2"},
		{200, "G3"},
		{399, "G3"},
		{400, "G4"},
		{799, "G4"},
		{800, "G5"},
		{10000, "G5"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := goldBucket(tt.gold)
			if got != tt.want {
				t.Errorf("goldBucket(%d) = %q, want %q", tt.gold, got, tt.want)
			}
			// Also verify it exists in vocab
			if v.TokenToID(got) == -1 {
				t.Errorf("token %q not in vocab", got)
			}
		})
	}
}

func TestEncode_LivesBuckets(t *testing.T) {
	v := loadTestVocab(t)

	tests := []struct {
		lives, maxLives int
		want            string
	}{
		{20, 20, "L9"}, // 100% → int(1.0*10)=10, capped to 9 → L9
		{10, 20, "L5"}, // 50% → int(0.5*10)=5 → L5
		{1, 20, "L0"},  // 5% → int(0.05*10)=0 → L0
		{0, 20, "L0"},  // 0% → L0
		{19, 20, "L9"}, // 95% → int(0.95*10)=9 → L9
		{18, 20, "L9"}, // 90% → int(0.9*10)=9 → L9
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := livesBucket(tt.lives, tt.maxLives)
			if got != tt.want {
				t.Errorf("livesBucket(%d, %d) = %q, want %q", tt.lives, tt.maxLives, got, tt.want)
			}
			if v.TokenToID(got) == -1 {
				t.Errorf("token %q not in vocab", got)
			}
		})
	}
}

func TestEncode_OneEnemy(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold: 100, Lives: 20, MaxLives: 20,
		Wave: 5, WaveActive: true, GameSpeed: 2,
		MapPixelW: 1200,
		Enemies: []EncodeEnemy{
			{
				X: 600, Y: 270,
				HP: 50, MaxHP: 100,
				Archetype: "normal",
			},
		},
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	if !containsToken(tokens, "ENM") {
		t.Fatalf("expected ENM token, tokens=%v", tokens)
	}
	if !containsToken(tokens, "a_normal") {
		t.Errorf("expected a_normal, tokens=%v", tokens)
	}
	// HP 50/100 = 0.5 → h5 (bucket 5, since int(0.5*10)=5, capped to 9)
	if !containsToken(tokens, "h5") {
		t.Errorf("expected h5 for HP=50/100, tokens=%v", tokens)
	}
	// path: x=600, mapW=1200 → 0.5 → p5
	if !containsToken(tokens, "p5") {
		t.Errorf("expected p5 for x=600/1200, tokens=%v", tokens)
	}
	// No status effects
	for _, tok := range []string{"s_slow", "s_stun", "s_burn", "s_bleed", "s_shield"} {
		if containsToken(tokens, tok) {
			t.Errorf("unexpected status token %q", tok)
		}
	}
}

func TestEncode_EnemyWithStatus(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold: 50, Lives: 15, MaxLives: 20,
		Wave: 3, WaveActive: true, GameSpeed: 1,
		MapPixelW: 1200,
		Enemies: []EncodeEnemy{
			{
				X: 300, Y: 100,
				HP: 80, MaxHP: 100,
				Archetype: "elite",
				IsSlowed:  true, IsBurning: true,
			},
		},
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	if !containsToken(tokens, "a_elite") {
		t.Errorf("expected a_elite, tokens=%v", tokens)
	}
	if !containsToken(tokens, "s_slow") {
		t.Errorf("expected s_slow for slowed enemy, tokens=%v", tokens)
	}
	if !containsToken(tokens, "s_burn") {
		t.Errorf("expected s_burn for burning enemy, tokens=%v", tokens)
	}
	// Should NOT have other statuses
	if containsToken(tokens, "s_stun") {
		t.Errorf("unexpected s_stun")
	}
	if containsToken(tokens, "s_bleed") {
		t.Errorf("unexpected s_bleed")
	}
}

func TestEncode_OneTower(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold: 200, Lives: 20, MaxLives: 20,
		Wave: 10, WaveActive: false, GameSpeed: 1,
		MapPixelW: 1200,
		Towers: []EncodeTower{
			{
				Key: "basic", Row: 2, Col: 5,
				Strength: 35, AttackStyle: "projectile",
				HasTarget: true,
			},
		},
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	if !containsToken(tokens, "TWR") {
		t.Fatalf("expected TWR token, tokens=%v", tokens)
	}
	if !containsToken(tokens, "k_basic") {
		t.Errorf("expected k_laser, tokens=%v", tokens)
	}
	if !containsToken(tokens, "R2C5") {
		t.Errorf("expected R2C5, tokens=%v", tokens)
	}
	// strength 35/10=3 → str3
	if !containsToken(tokens, "str3") {
		t.Errorf("expected str3 for strength=35, tokens=%v", tokens)
	}
	if !containsToken(tokens, "as_projectile") {
		t.Errorf("expected as_projectile, tokens=%v", tokens)
	}
	if !containsToken(tokens, "TGTYES") {
		t.Errorf("expected TGTYES, tokens=%v", tokens)
	}
}

func TestEncode_Cells(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold: 100, Lives: 20, MaxLives: 20,
		Wave: 1, WaveActive: false, GameSpeed: 1,
		MapPixelW: 1200,
		Cells: []EncodeCell{
			{Row: 1, Col: 3},
			{Row: 4, Col: 7},
		},
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	if !containsToken(tokens, "CELL") {
		t.Fatalf("expected CELL token, tokens=%v", tokens)
	}
	if !containsToken(tokens, "R1C3") {
		t.Errorf("expected R1C3, tokens=%v", tokens)
	}
	if !containsToken(tokens, "R4C7") {
		t.Errorf("expected R4C7, tokens=%v", tokens)
	}
}

func TestEncode_Warden(t *testing.T) {
	v := loadTestVocab(t)

	// With warden ready
	input := &EncodeInput{
		Gold: 100, Lives: 20, MaxLives: 20,
		Wave: 1, WaveActive: false, GameSpeed: 1,
		MapPixelW:   1200,
		WardenReady: true,
		WardenType:  "prince",
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	if !containsToken(tokens, "WDN") {
		t.Errorf("expected WDN when warden ready, tokens=%v", tokens)
	}
	if !containsToken(tokens, "w_prince") {
		t.Errorf("expected w_prince, tokens=%v", tokens)
	}

	// Without warden
	input2 := &EncodeInput{
		Gold: 100, Lives: 20, MaxLives: 20,
		Wave: 1, WaveActive: false, GameSpeed: 1,
		MapPixelW:   1200,
		WardenReady: false,
	}
	ids2 := Encode(v, input2)
	tokens2 := idsToTokens(v, ids2)

	if containsToken(tokens2, "WDN") {
		t.Errorf("unexpected WDN when warden not ready, tokens=%v", tokens2)
	}
}

func TestEncode_FullState(t *testing.T) {
	v := loadTestVocab(t)
	input := &EncodeInput{
		Gold: 350, Lives: 14, MaxLives: 20,
		Wave: 12, WaveActive: true, GameSpeed: 2,
		MapPixelW: 1200,
		Enemies: []EncodeEnemy{
			{
				X: 240, Y: 200,
				HP: 90, MaxHP: 100,
				Archetype: "normal",
				IsSlowed:  true,
			},
			{
				X: 960, Y: 300,
				HP: 20, MaxHP: 200,
				Archetype: "boss",
			},
		},
		Towers: []EncodeTower{
			{
				Key: "electric", Row: 3, Col: 8,
				Strength: 72, AttackStyle: "scatter",
				HasTarget: true,
			},
		},
		Cells: []EncodeCell{
			{Row: 0, Col: 2},
			{Row: 5, Col: 11},
		},
		WardenReady: true,
		WardenType:  "chain",
	}
	ids := Encode(v, input)
	tokens := idsToTokens(v, ids)

	// Verify structure: BOS SEP ... EOS
	if tokens[0] != "BOS" || tokens[1] != "SEP" {
		t.Errorf("expected BOS SEP start, got %v", tokens[:2])
	}
	if tokens[len(tokens)-1] != "EOS" {
		t.Errorf("expected EOS end, got %v", tokens[len(tokens)-1])
	}

	// Header: G3 (200-399), L7 (70%), W12, WACT, SPD2
	if !containsToken(tokens, "G3") {
		t.Errorf("expected G3 for gold=350, tokens=%v", tokens)
	}
	if !containsToken(tokens, "L7") {
		t.Errorf("expected L7 for lives=14/20 (70%%), tokens=%v", tokens)
	}
	if !containsToken(tokens, "W12") {
		t.Errorf("expected W12, tokens=%v", tokens)
	}
	if !containsToken(tokens, "WACT") {
		t.Errorf("expected WACT, tokens=%v", tokens)
	}
	if !containsToken(tokens, "SPD2") {
		t.Errorf("expected SPD2, tokens=%v", tokens)
	}

	// Enemy 1: normal, h9 (90%), p2 (240/1200=20%), s_slow
	if !containsToken(tokens, "a_normal") {
		t.Errorf("expected a_normal, tokens=%v", tokens)
	}
	if !containsToken(tokens, "s_slow") {
		t.Errorf("expected s_slow, tokens=%v", tokens)
	}

	// Enemy 2: boss, h1 (10%), p8 (80%)
	if !containsToken(tokens, "a_boss") {
		t.Errorf("expected a_boss, tokens=%v", tokens)
	}

	// Tower: electric, R3C8, str7, scatter, TGTYES
	if !containsToken(tokens, "k_electric") {
		t.Errorf("expected k_electric, tokens=%v", tokens)
	}
	if !containsToken(tokens, "R3C8") {
		t.Errorf("expected R3C8, tokens=%v", tokens)
	}
	if !containsToken(tokens, "str7") {
		t.Errorf("expected str7 for strength=72, tokens=%v", tokens)
	}
	if !containsToken(tokens, "as_scatter") {
		t.Errorf("expected as_scatter, tokens=%v", tokens)
	}

	// Cells
	if !containsToken(tokens, "R0C2") {
		t.Errorf("expected R0C2, tokens=%v", tokens)
	}
	if !containsToken(tokens, "R5C11") {
		t.Errorf("expected R5C11, tokens=%v", tokens)
	}

	// Warden
	if !containsToken(tokens, "WDN") {
		t.Errorf("expected WDN, tokens=%v", tokens)
	}
	if !containsToken(tokens, "w_chain") {
		t.Errorf("expected w_chain, tokens=%v", tokens)
	}

	// Verify all IDs are valid (no -1)
	for i, id := range ids {
		if id == -1 {
			t.Errorf("token at index %d has ID -1 (unknown token)", i)
		}
	}

	t.Logf("Full state tokens (%d): %v", len(tokens), tokens)
}
