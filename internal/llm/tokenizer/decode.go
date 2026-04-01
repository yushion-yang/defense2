package tokenizer

import (
	"strconv"
	"strings"
)

// ActionType mirrors autoplay.ActionType to avoid circular imports.
type ActionType int

const (
	ActBuild   ActionType = iota
	ActUpgrade
	ActSell
	ActWave
	ActWarden
	ActWait
)

// DecodedAction is the tokenizer's representation of an action.
type DecodedAction struct {
	Type      ActionType
	TowerKey  string // Build: tower type (without "k_" prefix)
	Row, Col  int    // Build/Upgrade/Sell: grid position
	WardenKey string // Warden: type (without "w_" prefix)
}

// DecodeActions parses a sequence of token IDs into structured actions.
//
// Each action starts with an ACT_* token:
//   - ACT_BUILD <k_type> <RxCy> → 3 tokens
//   - ACT_UPGRADE <RxCy> → 2 tokens
//   - ACT_SELL <RxCy> → 2 tokens
//   - ACT_WAVE → 1 token
//   - ACT_WARDEN <w_type> → 2 tokens
//   - ACT_WAIT → 1 token
//
// Incomplete action sequences and unknown tokens are skipped.
func DecodeActions(v *Vocab, tokenIDs []int) []DecodedAction {
	var actions []DecodedAction
	i := 0
	n := len(tokenIDs)

	for i < n {
		tok := v.IDToToken(tokenIDs[i])
		switch tok {
		case "ACT_BUILD":
			// Need 2 more tokens: k_type and RxCy
			if i+2 >= n {
				return actions // incomplete
			}
			towerTok := v.IDToToken(tokenIDs[i+1])
			gridTok := v.IDToToken(tokenIDs[i+2])
			if !strings.HasPrefix(towerTok, "k_") {
				i++
				continue
			}
			row, col, ok := parseGridToken(gridTok)
			if !ok {
				i++
				continue
			}
			actions = append(actions, DecodedAction{
				Type:     ActBuild,
				TowerKey: strings.TrimPrefix(towerTok, "k_"),
				Row:      row,
				Col:      col,
			})
			i += 3

		case "ACT_UPGRADE":
			if i+1 >= n {
				return actions
			}
			gridTok := v.IDToToken(tokenIDs[i+1])
			row, col, ok := parseGridToken(gridTok)
			if !ok {
				i++
				continue
			}
			actions = append(actions, DecodedAction{
				Type: ActUpgrade,
				Row:  row,
				Col:  col,
			})
			i += 2

		case "ACT_SELL":
			if i+1 >= n {
				return actions
			}
			gridTok := v.IDToToken(tokenIDs[i+1])
			row, col, ok := parseGridToken(gridTok)
			if !ok {
				i++
				continue
			}
			actions = append(actions, DecodedAction{
				Type: ActSell,
				Row:  row,
				Col:  col,
			})
			i += 2

		case "ACT_WAVE":
			actions = append(actions, DecodedAction{Type: ActWave})
			i++

		case "ACT_WARDEN":
			if i+1 >= n {
				return actions
			}
			wardenTok := v.IDToToken(tokenIDs[i+1])
			if !strings.HasPrefix(wardenTok, "w_") {
				i++
				continue
			}
			actions = append(actions, DecodedAction{
				Type:      ActWarden,
				WardenKey: strings.TrimPrefix(wardenTok, "w_"),
			})
			i += 2

		case "ACT_WAIT":
			actions = append(actions, DecodedAction{Type: ActWait})
			i++

		default:
			// Unknown token, skip
			i++
		}
	}

	return actions
}

// parseGridToken parses a grid position token like "R2C5" into (row, col).
// Returns (0, 0, false) if the token does not match the expected format.
func parseGridToken(token string) (row, col int, ok bool) {
	if len(token) < 4 || token[0] != 'R' {
		return 0, 0, false
	}

	cIdx := strings.IndexByte(token, 'C')
	if cIdx < 2 {
		return 0, 0, false
	}

	r, err := strconv.Atoi(token[1:cIdx])
	if err != nil || r < 0 {
		return 0, 0, false
	}

	c, err := strconv.Atoi(token[cIdx+1:])
	if err != nil || c < 0 {
		return 0, 0, false
	}

	return r, c, true
}
