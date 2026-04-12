package mascot

import "math/rand"

// MascotVM is the view-model snapshot for rendering.
type MascotVM struct {
	Visible          bool
	HasDialog        bool
	Text             string
	Expression       string
	CanClick         bool // true = click to advance
	AbilityReady     bool
	AbilityHintShown bool    // true when ability-hint bubble is visible (click = use ability)
	CooldownPct      float64 // 0.0 = ready, 1.0 = full cooldown
}

// Guide is the mascot state machine that drives dialog playback.
type Guide struct {
	dialogs  []Dialog
	scene    string          // current scene name
	shownIDs map[string]bool // Once dialog IDs that have been shown

	// active dialog state
	active  *Dialog
	lineIdx int
	timer   float64

	// Condition evaluation
	condFuncs    []ConditionFunc
	condState    *ConditionState
	evalInterval float64 // seconds between evaluations (default 5.0)
	evalTimer    float64
	lastCtx      *GameContext

	// Ability system
	abilityCooldown    float64 // remaining seconds (0 = ready)
	abilityCooldownMax float64 // default 30.0
	pendingAction      *MascotAction
	abilityHinted      bool // true after hint dialog was shown this cooldown cycle
	abilityHintActive  bool // true while the hint dialog is being displayed
}

// NewGuide creates a Guide preloaded with dialogs.
// shownIDs may be nil; if non-nil it restores previously-seen Once dialog IDs.
func NewGuide(dialogs []Dialog, shownIDs map[string]bool) *Guide {
	shown := shownIDs
	if shown == nil {
		shown = make(map[string]bool)
	}
	return &Guide{
		dialogs:            dialogs,
		shownIDs:           shown,
		evalInterval:       5.0,
		abilityCooldownMax: 30.0,
	}
}

// SetScene switches the current scene and auto-triggers any "scene_enter" dialog.
func (g *Guide) SetScene(name string) {
	g.scene = name
	g.tryTrigger("scene_enter")
}

// Trigger fires a named event and starts the best matching dialog (if idle).
func (g *Guide) Trigger(event string) {
	g.tryTrigger(event)
}

// Tick advances auto-advance timers and decrements ability cooldown. dt is in seconds.
func (g *Guide) Tick(dt float64) {
	// Decrement ability cooldown.
	if g.abilityCooldown > 0 {
		g.abilityCooldown -= dt
		if g.abilityCooldown < 0 {
			g.abilityCooldown = 0
		}
	}

	// Auto-show ability hint when ready and idle.
	// Skip auto-advance this frame so the hint is visible for at least one tick.
	if g.AbilityReady() && !g.abilityHinted && g.active == nil {
		g.abilityHinted = true
		g.abilityHintActive = true
		g.ForceTrigger("mascot_ability_hint")
		return
	}

	// Decrement condition eval timer.
	if g.condFuncs != nil {
		g.evalTimer -= dt
	}

	if g.active == nil {
		return
	}
	line := &g.active.Lines[g.lineIdx]
	if line.AutoAdvance <= 0 {
		return
	}
	g.timer += dt
	if g.timer >= line.AutoAdvance {
		g.advance()
	}
}

// ClickAdvance manually advances to the next line (or finishes the dialog).
// Works on any line, including auto-advance lines.
func (g *Guide) ClickAdvance() {
	if g.active == nil {
		return
	}
	g.advance()
}

// VM returns a snapshot of the current mascot state for rendering.
func (g *Guide) VM() MascotVM {
	vm := MascotVM{
		Visible:          true,
		AbilityReady:     g.AbilityReady(),
		AbilityHintShown: g.abilityHintActive,
	}
	if g.abilityCooldownMax > 0 && g.abilityCooldown > 0 {
		vm.CooldownPct = g.abilityCooldown / g.abilityCooldownMax
		if vm.CooldownPct > 1.0 {
			vm.CooldownPct = 1.0
		}
	}
	if g.active == nil {
		return vm
	}
	line := &g.active.Lines[g.lineIdx]
	vm.HasDialog = true
	vm.Text = line.Text
	vm.Expression = line.Expression
	vm.CanClick = true
	return vm
}

// ShownIDs returns the set of Once dialog IDs that have been shown.
// The caller should persist this map and pass it back via NewGuide on restart.
func (g *Guide) ShownIDs() map[string]bool {
	out := make(map[string]bool, len(g.shownIDs))
	for k, v := range g.shownIDs {
		out[k] = v
	}
	return out
}

// InitConditions sets up condition evaluation with the given condition functions.
// If funcs is nil, no automatic condition evaluation occurs.
func (g *Guide) InitConditions(funcs []ConditionFunc) {
	g.condFuncs = funcs
	g.condState = NewConditionState()
	g.evalTimer = g.evalInterval
}

// UpdateContext stores the latest game context and evaluates conditions when the
// evaluation timer has elapsed.
func (g *Guide) UpdateContext(ctx GameContext) {
	ctxCopy := ctx
	g.lastCtx = &ctxCopy

	if g.condFuncs == nil || g.condState == nil {
		return
	}

	if g.evalTimer > 0 {
		return
	}

	// Reset eval timer.
	g.evalTimer = g.evalInterval

	// Evaluate conditions: first non-empty result wins.
	for _, fn := range g.condFuncs {
		trigger := fn(&ctx, g.condState)
		if trigger != "" {
			g.condState.markFired(trigger, ctx.SessionSecs)
			g.tryTrigger(trigger)
			break
		}
	}

	// Update tracking state.
	g.condState.PrevWave = ctx.Wave
	g.condState.PrevInStage = ctx.InStage
	g.condState.PrevInteractMode = ctx.InteractMode
	g.condState.PrevQuality = ctx.QualityLevel

	// Meta tracking: gold/lives/kills for anomaly detection.
	if ctx.InStage {
		g.condState.PrevGold = ctx.Gold
		g.condState.PrevGoldInit = true
		g.condState.PrevLives = ctx.Lives
		g.condState.PrevLivesInit = true
		g.condState.PrevKills = ctx.Kills
	}

	// Meta tracking: pause accumulation (evalInterval is the tick period).
	if ctx.InStage && ctx.Paused {
		g.condState.PauseAccum += g.evalInterval
	} else {
		g.condState.PauseAccum = 0
	}

	// Meta tracking: defeat counting.
	if ctx.Defeat && !g.condState.PrevDefeat {
		g.condState.StageDefeats++
	}
	g.condState.PrevVictory = ctx.Victory
	g.condState.PrevDefeat = ctx.Defeat
}

// HasActiveDialog returns true if a dialog is currently being displayed.
func (g *Guide) HasActiveDialog() bool {
	return g.active != nil
}

// AbilityReady returns true if the mascot ability can be used.
func (g *Guide) AbilityReady() bool {
	return g.abilityCooldown <= 0 && g.lastCtx != nil && g.lastCtx.InStage
}

// IsAbilityHintActive returns true if the ability-hint dialog is currently showing.
// When true, clicking the mascot should trigger the ability instead of advancing dialog.
func (g *Guide) IsAbilityHintActive() bool {
	return g.abilityHintActive
}

// RequestHelp requests the mascot to perform its battle assistance ability.
// Returns the action if ability is ready, or nil if on cooldown / not in stage.
func (g *Guide) RequestHelp() *MascotAction {
	if !g.AbilityReady() {
		g.ForceTrigger("mascot_not_ready")
		return nil
	}

	action := &MascotAction{Type: ActionKillWeakEnemy}
	g.pendingAction = action
	g.abilityCooldown = g.abilityCooldownMax
	g.abilityHinted = false
	g.abilityHintActive = false
	g.ForceTrigger("mascot_help")
	return action
}

// ConsumeAction returns and clears the pending mascot action.
func (g *Guide) ConsumeAction() *MascotAction {
	a := g.pendingAction
	g.pendingAction = nil
	return a
}

// NotifyActionComplete informs the mascot that an action was completed.
func (g *Guide) NotifyActionComplete(t ActionType) {
	_ = t // reserved for future action-type-specific responses
	g.ForceTrigger("mascot_kill_success")
}

// ForceTrigger fires a named event, interrupting any active dialog.
// Used for high-priority events like panic recovery.
// Randomly picks among all matching dialogs (not just highest priority).
func (g *Guide) ForceTrigger(event string) {
	// Finish current dialog if any.
	if g.active != nil {
		g.finishDialog()
	}
	var candidates []*Dialog
	for i := range g.dialogs {
		d := &g.dialogs[i]
		if d.Trigger != event {
			continue
		}
		if d.Scene != "*" && d.Scene != g.scene {
			continue
		}
		candidates = append(candidates, d)
	}
	if len(candidates) > 0 {
		pick := candidates[rand.Intn(len(candidates))]
		g.startDialog(pick)
	}
}

// tryTrigger finds the best matching dialog for the given event and starts it.
// Does nothing if a dialog is already active.
// When multiple dialogs share the highest priority, one is chosen at random.
func (g *Guide) tryTrigger(event string) {
	if g.active != nil {
		return
	}
	var candidates []*Dialog
	bestPri := -1
	for i := range g.dialogs {
		d := &g.dialogs[i]
		if d.Trigger != event {
			continue
		}
		if d.Scene != "*" && d.Scene != g.scene {
			continue
		}
		if d.Once && g.shownIDs[d.ID] {
			continue
		}
		if d.Priority > bestPri {
			bestPri = d.Priority
			candidates = candidates[:0]
			candidates = append(candidates, d)
		} else if d.Priority == bestPri {
			candidates = append(candidates, d)
		}
	}
	if len(candidates) > 0 {
		pick := candidates[rand.Intn(len(candidates))]
		g.startDialog(pick)
	}
}

// startDialog begins playback of a dialog.
func (g *Guide) startDialog(d *Dialog) {
	g.active = d
	g.lineIdx = 0
	g.timer = 0
}

// advance moves to the next line or finishes the dialog.
func (g *Guide) advance() {
	g.lineIdx++
	g.timer = 0
	if g.lineIdx >= len(g.active.Lines) {
		g.finishDialog()
	}
}

// finishDialog ends the current dialog and marks Once dialogs as shown.
func (g *Guide) finishDialog() {
	if g.active.Once {
		g.shownIDs[g.active.ID] = true
	}
	if g.active.Trigger == "mascot_ability_hint" {
		g.abilityHintActive = false
	}
	g.active = nil
	g.lineIdx = 0
	g.timer = 0
}
