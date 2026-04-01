package tokenizer

import (
	"encoding/json"
	"fmt"
	"io"
)

// Vocab maps between token strings and integer IDs.
type Vocab struct {
	tokenToID map[string]int
	idToToken map[int]string
}

type vocabJSON struct {
	Version int            `json:"version"`
	Tokens  map[string]int `json:"tokens"`
}

// LoadVocab reads a vocab.json from r.
func LoadVocab(r io.Reader) (*Vocab, error) {
	var raw vocabJSON
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode vocab: %w", err)
	}
	v := &Vocab{
		tokenToID: raw.Tokens,
		idToToken: make(map[int]string, len(raw.Tokens)),
	}
	for tok, id := range raw.Tokens {
		v.idToToken[id] = tok
	}
	return v, nil
}

// Size returns the number of tokens in the vocabulary.
func (v *Vocab) Size() int { return len(v.tokenToID) }

// TokenToID returns the integer ID for a token string, or -1 if not found.
func (v *Vocab) TokenToID(token string) int {
	id, ok := v.tokenToID[token]
	if !ok {
		return -1
	}
	return id
}

// IDToToken returns the token string for an integer ID, or "" if not found.
func (v *Vocab) IDToToken(id int) string {
	return v.idToToken[id]
}

// PAD returns the ID of the PAD special token.
func (v *Vocab) PAD() int { return v.tokenToID["PAD"] }

// BOS returns the ID of the BOS (begin of sequence) special token.
func (v *Vocab) BOS() int { return v.tokenToID["BOS"] }

// EOS returns the ID of the EOS (end of sequence) special token.
func (v *Vocab) EOS() int { return v.tokenToID["EOS"] }

// SEP returns the ID of the SEP (separator) special token.
func (v *Vocab) SEP() int { return v.tokenToID["SEP"] }
