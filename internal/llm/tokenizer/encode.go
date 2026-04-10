package tokenizer

import "fmt"

// EncodeInput is a mirror of autoplay.GameState fields needed for encoding.
// Avoids importing autoplay (which imports Ebitengine).
type EncodeInput struct {
	Gold       int
	Lives      int
	MaxLives   int // for bucket calculation
	Wave       int
	WaveActive bool
	GameSpeed  int     // 1, 2, or 3
	MapPixelW  float64 // map width in pixels, for path progress

	Enemies []EncodeEnemy
	Towers  []EncodeTower
	Cells   []EncodeCell

	WardenReady bool
	WardenType  string // "prince", "core", etc.
}

// EncodeEnemy holds the subset of enemy state needed for tokenization.
type EncodeEnemy struct {
	X, Y                                       float64
	HP, MaxHP                                  float64
	Archetype                                  string
	IsSlowed, IsStunned, IsBurning, IsBleeding bool
}

// EncodeTower holds the subset of tower state needed for tokenization.
type EncodeTower struct {
	Key         string
	Row, Col    int
	Strength    int // 0-100
	AttackStyle string
	HasTarget   bool
}

// EncodeCell represents a buildable grid cell.
type EncodeCell struct {
	Row, Col int
}

// Encode converts a game state snapshot into a sequence of token IDs.
//
// Sequence format:
//
//	BOS SEP
//	<gold> <lives> W<wave> WACT|WIDLE SPD<speed> SEP
//	[ENM <archetype> <hp> <path> [statuses...] SEP] x enemies
//	[TWR <key> R<row>C<col> <str> <style> TGTYES|TGTNO SEP] x towers
//	[CELL R<row>C<col>] x cells SEP
//	WDN <warden_type> SEP  (only if WardenReady)
//	EOS
func Encode(v *Vocab, input *EncodeInput) []int {
	// Pre-allocate: header(8) + per-enemy(~8) + per-tower(~8) + cells(~3) + warden(3) + bookends(3)
	est := 8 + len(input.Enemies)*8 + len(input.Towers)*8 + len(input.Cells)*2 + 6
	seq := make([]int, 0, est)

	emit := func(token string) {
		id := v.TokenToID(token)
		seq = append(seq, id)
	}

	// BOS SEP
	emit("BOS")
	emit("SEP")

	// Header: gold, lives, wave, wave-status, speed, SEP
	emit(goldBucket(input.Gold))
	emit(livesBucket(input.Lives, input.MaxLives))
	emit(waveToken(input.Wave))
	if input.WaveActive {
		emit("WACT")
	} else {
		emit("WIDLE")
	}
	emit(speedToken(input.GameSpeed))
	emit("SEP")

	// Enemies
	for i := range input.Enemies {
		e := &input.Enemies[i]
		emit("ENM")
		emit("a_" + e.Archetype)
		emit(hpBucket(e.HP, e.MaxHP))
		emit(pathProgress(e.X, input.MapPixelW))
		// Status effects (only present ones)
		if e.IsSlowed {
			emit("s_slow")
		}
		if e.IsStunned {
			emit("s_stun")
		}
		if e.IsBurning {
			emit("s_burn")
		}
		if e.IsBleeding {
			emit("s_bleed")
		}
		emit("SEP")
	}

	// Towers
	for i := range input.Towers {
		tw := &input.Towers[i]
		emit("TWR")
		emit("k_" + tw.Key)
		emit(gridToken(tw.Row, tw.Col))
		emit(strengthBucket(tw.Strength))
		emit("as_" + tw.AttackStyle)
		if tw.HasTarget {
			emit("TGTYES")
		} else {
			emit("TGTNO")
		}
		emit("SEP")
	}

	// Cells
	for i := range input.Cells {
		c := &input.Cells[i]
		emit("CELL")
		emit(gridToken(c.Row, c.Col))
	}
	if len(input.Cells) > 0 {
		emit("SEP")
	}

	// Warden (only if ready)
	if input.WardenReady {
		emit("WDN")
		emit("w_" + input.WardenType)
		emit("SEP")
	}

	// EOS
	emit("EOS")

	return seq
}

// goldBucket maps gold amount to a bucket token.
//
//	0-49→G0, 50-99→G1, 100-199→G2, 200-399→G3, 400-799→G4, 800+→G5
func goldBucket(gold int) string {
	switch {
	case gold < 50:
		return "G0"
	case gold < 100:
		return "G1"
	case gold < 200:
		return "G2"
	case gold < 400:
		return "G3"
	case gold < 800:
		return "G4"
	default:
		return "G5"
	}
}

// livesBucket maps lives/maxLives ratio to a 10% bucket token (L0-L9).
func livesBucket(lives, maxLives int) string {
	if maxLives <= 0 {
		return "L0"
	}
	ratio := float64(lives) / float64(maxLives)
	bucket := int(ratio * 10)
	if bucket > 9 {
		bucket = 9
	}
	if bucket < 0 {
		bucket = 0
	}
	return fmt.Sprintf("L%d", bucket)
}

// hpBucket maps hp/maxHP ratio to a bucket token (h0-h9).
func hpBucket(hp, maxHP float64) string {
	if maxHP <= 0 {
		return "h0"
	}
	ratio := hp / maxHP
	bucket := int(ratio * 10)
	if bucket > 9 {
		bucket = 9
	}
	if bucket < 0 {
		bucket = 0
	}
	return fmt.Sprintf("h%d", bucket)
}

// pathProgress maps x position / map width to a bucket token (p0-p9).
func pathProgress(x, mapW float64) string {
	if mapW <= 0 {
		return "p0"
	}
	ratio := x / mapW
	bucket := int(ratio * 10)
	if bucket > 9 {
		bucket = 9
	}
	if bucket < 0 {
		bucket = 0
	}
	return fmt.Sprintf("p%d", bucket)
}

// strengthBucket maps tower strength (0-100) to a bucket token (str0-str9).
func strengthBucket(str int) string {
	bucket := str / 10
	if bucket > 9 {
		bucket = 9
	}
	if bucket < 0 {
		bucket = 0
	}
	return fmt.Sprintf("str%d", bucket)
}

// gridToken returns the grid position token R<row>C<col>.
func gridToken(row, col int) string {
	return fmt.Sprintf("R%dC%d", row, col)
}

// waveToken returns the wave number token W<n>, clamped to 1-30.
func waveToken(wave int) string {
	if wave < 1 {
		wave = 1
	}
	if wave > 30 {
		wave = 30
	}
	return fmt.Sprintf("W%d", wave)
}

// speedToken returns the game speed token SPD<n>, clamped to 1-3.
func speedToken(speed int) string {
	if speed < 1 {
		speed = 1
	}
	if speed > 3 {
		speed = 3
	}
	return fmt.Sprintf("SPD%d", speed)
}
