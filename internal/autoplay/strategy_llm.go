package autoplay

import (
	"fmt"
	"os"

	"defense2/internal/llm/engine"
	"defense2/internal/llm/tokenizer"
	"defense2/internal/llm/weights"
)

// decodedAction is an internal mirror of tokenizer.DecodedAction
// to keep convertActions testable without exposing tokenizer types.
type decodedAction struct {
	typ       int
	towerKey  string
	row, col  int
	wardenKey string
	eventIdx  int
	skillName string
}

// Action type constants mirroring tokenizer.Act* to avoid exporting tokenizer types.
const (
	actBuild   = int(tokenizer.ActBuild)
	actUpgrade = int(tokenizer.ActUpgrade)
	actSell    = int(tokenizer.ActSell)
	actWave    = int(tokenizer.ActWave)
	actWarden  = int(tokenizer.ActWarden)
	actEvent   = int(tokenizer.ActEvent)
	actSkill   = int(tokenizer.ActSkill)
	actWait    = int(tokenizer.ActWait)
)

// LLMStrategy bridges the LLM engine with the autoplay Strategy interface.
type LLMStrategy struct {
	model     *engine.Model
	vocab     *tokenizer.Vocab
	sampler   engine.SampleConfig
	modelPath string

	// State diff detection for event-driven decisions
	lastGoldBucket  int
	lastWaveActive  bool
	lastLives       int
	lastTowerCount  int
	tickSinceDecide int
	initialized     bool
}

// NewLLMStrategy creates a new LLM-powered autoplay strategy.
func NewLLMStrategy(modelPath, vocabPath string) (*LLMStrategy, error) {
	// Load vocab
	vf, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("open vocab: %w", err)
	}
	defer vf.Close()
	vocab, err := tokenizer.LoadVocab(vf)
	if err != nil {
		return nil, fmt.Errorf("load vocab: %w", err)
	}

	// Load model weights
	mf, err := os.Open(modelPath)
	if err != nil {
		return nil, fmt.Errorf("open model: %w", err)
	}
	defer mf.Close()
	wf, err := weights.Load(mf)
	if err != nil {
		return nil, fmt.Errorf("load weights: %w", err)
	}

	model, err := engine.NewModel(wf)
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}

	return &LLMStrategy{
		model:     model,
		vocab:     vocab,
		modelPath: modelPath,
		sampler:   engine.SampleConfig{Temperature: 0.3, TopK: 5},
	}, nil
}

func (s *LLMStrategy) Name() string { return "llm" }

func (s *LLMStrategy) Init(state *GameState) {
	s.model.ResetKVCache()
	s.snapshot(state)
	s.initialized = true
}

func (s *LLMStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	// Warden selection (bypass LLM -- always pick prince for now)
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: "prince"}}
	}

	// Event selection (bypass LLM -- rotate like greedy)
	if state.InteractMode == 6 { // modeEvent
		return []Action{{Type: ActionChooseEvent, EventIndex: state.Wave % 3}}
	}

	s.tickSinceDecide++
	if !s.shouldDecide(state) {
		return nil
	}
	s.tickSinceDecide = 0

	// 1. Convert GameState -> EncodeInput
	input := s.buildEncodeInput(state)

	// 2. Encode -> token IDs
	tokens := tokenizer.Encode(s.vocab, input)

	// 3. Reset KV cache (no cross-decision memory)
	s.model.ResetKVCache()

	// 4. Prefill all input tokens
	var logits []float32
	for i, tok := range tokens {
		logits = s.model.Forward(tok, i)
	}

	// 5. Autoregressive decode
	var actionTokens []int
	pos := len(tokens)
	eosID := s.vocab.EOS()

	lastToken := s.sampler.Sample(logits, nil)
	if lastToken == eosID {
		return nil
	}
	actionTokens = append(actionTokens, lastToken)
	pos++

	for range 14 { // max 15 total action tokens
		logits = s.model.Forward(lastToken, pos)
		next := s.sampler.Sample(logits, nil)
		if next == eosID {
			break
		}
		actionTokens = append(actionTokens, next)
		lastToken = next
		pos++
	}

	// 6. Decode action tokens -> DecodedActions
	tokDecoded := tokenizer.DecodeActions(s.vocab, actionTokens)

	// 7. Convert tokenizer types to internal decodedAction
	decoded := make([]decodedAction, len(tokDecoded))
	for i, d := range tokDecoded {
		decoded[i] = decodedAction{
			typ:       int(d.Type),
			towerKey:  d.TowerKey,
			row:       d.Row,
			col:       d.Col,
			wardenKey: d.WardenKey,
			eventIdx:  d.EventIdx,
			skillName: d.SkillName,
		}
	}

	// 8. Convert to autoplay.Action + filter valid
	actions := s.convertActions(decoded, state)
	return s.filterValid(actions, state)
}

// shouldDecide returns true when a state change warrants a new LLM decision.
// When triggered, it resets the snapshot and tick counter.
// When not triggered, it preserves the tick counter for accumulation.
func (s *LLMStrategy) shouldDecide(state *GameState) bool {
	gb := goldToBucket(state.Gold)
	waveChanged := state.WaveActive != s.lastWaveActive
	livesDropped := state.Lives < s.lastLives
	towersChanged := len(state.Towers) != s.lastTowerCount
	intervalHit := s.tickSinceDecide >= 60

	triggered := waveChanged || gb != s.lastGoldBucket ||
		livesDropped || towersChanged || intervalHit

	if triggered {
		s.snapshot(state)
	}
	return triggered
}

// snapshot records current state for diff detection.
func (s *LLMStrategy) snapshot(state *GameState) {
	s.lastGoldBucket = goldToBucket(state.Gold)
	s.lastWaveActive = state.WaveActive
	s.lastLives = state.Lives
	s.lastTowerCount = len(state.Towers)
	s.tickSinceDecide = 0
}

// goldToBucket maps gold amount to a coarse bucket for change detection.
func goldToBucket(gold int) int {
	switch {
	case gold < 50:
		return 0
	case gold < 100:
		return 1
	case gold < 200:
		return 2
	case gold < 400:
		return 3
	case gold < 800:
		return 4
	default:
		return 5
	}
}

// buildEncodeInput converts GameState to tokenizer.EncodeInput.
func (s *LLMStrategy) buildEncodeInput(state *GameState) *tokenizer.EncodeInput {
	input := &tokenizer.EncodeInput{
		Gold:        state.Gold,
		Lives:       state.Lives,
		MaxLives:    20, // default
		Wave:        state.Wave,
		WaveActive:  state.WaveActive,
		GameSpeed:   state.GameSpeed,
		MapPixelW:   state.MapPixelW,
		WardenReady: state.WardenReady,
	}

	for _, e := range state.Enemies {
		if !e.Active || e.Dying {
			continue
		}
		input.Enemies = append(input.Enemies, tokenizer.EncodeEnemy{
			X: e.X, Y: e.Y,
			HP: e.HP, MaxHP: e.MaxHP,
			Archetype:  e.Archetype,
			IsSlowed:   e.IsSlowed,
			IsStunned:  e.IsStunned,
			IsBurning:  e.IsBurning,
			IsBleeding: e.IsBleeding,
			IsRooted:   e.IsRooted,
		})
	}

	for _, t := range state.Towers {
		input.Towers = append(input.Towers, tokenizer.EncodeTower{
			Key:         t.Key,
			Row:         t.Row,
			Col:         t.Col,
			Strength:    t.Strength,
			AttackStyle: t.AttackStyle,
			HasTarget:   t.HasTarget,
		})
	}

	for _, c := range state.BuildCells {
		input.Cells = append(input.Cells, tokenizer.EncodeCell{Row: c.Row, Col: c.Col})
	}

	return input
}

// convertActions converts internal decodedActions to autoplay.Actions.
func (s *LLMStrategy) convertActions(decoded []decodedAction, state *GameState) []Action {
	var actions []Action
	for _, d := range decoded {
		var a Action
		switch d.typ {
		case actBuild:
			a = Action{Type: ActionBuild, TowerKey: d.towerKey, Row: d.row, Col: d.col}
			// Find matching cell for pixel coordinates
			for _, c := range state.BuildCells {
				if c.Row == d.row && c.Col == d.col {
					a.Cell = c
					break
				}
			}
		case actUpgrade:
			a = Action{Type: ActionUpgrade, Row: d.row, Col: d.col}
		case actSell:
			a = Action{Type: ActionSell, Row: d.row, Col: d.col}
		case actWave:
			a = Action{Type: ActionStartWave}
		case actWarden:
			a = Action{Type: ActionSelectWarden, WardenKey: d.wardenKey}
		case actEvent:
			a = Action{Type: ActionChooseEvent, EventIndex: d.eventIdx}
		case actSkill:
			a = Action{Type: ActionAssignSkill, SkillName: d.skillName}
		case actWait:
			a = Action{Type: ActionNoop}
		}
		actions = append(actions, a)
	}
	return actions
}

// filterValid removes actions that are not valid given the current state.
func (s *LLMStrategy) filterValid(actions []Action, state *GameState) []Action {
	var valid []Action
	for _, a := range actions {
		switch a.Type {
		case ActionBuild:
			cost := towerCost(a.TowerKey, state)
			if cost > 0 && state.Gold >= cost && cellAvailable(a.Row, a.Col, state) {
				valid = append(valid, a)
			}
		case ActionUpgrade:
			if towerExists(a.Row, a.Col, state) {
				valid = append(valid, a)
			}
		case ActionSell:
			if towerExists(a.Row, a.Col, state) {
				valid = append(valid, a)
			}
		case ActionStartWave:
			if !state.WaveActive && state.Wave < state.MaxWaves {
				valid = append(valid, a)
			}
		case ActionNoop:
			valid = append(valid, a)
		default:
			valid = append(valid, a)
		}
	}
	return valid
}

func towerCost(key string, state *GameState) int {
	for _, td := range state.TowerDefs {
		if td.Key == key {
			return td.Cost
		}
	}
	return 0
}

func cellAvailable(row, col int, state *GameState) bool {
	for _, c := range state.BuildCells {
		if c.Row == row && c.Col == col {
			return true
		}
	}
	return false
}

func towerExists(row, col int, state *GameState) bool {
	for _, t := range state.Towers {
		if t.Row == row && t.Col == col {
			return true
		}
	}
	return false
}
