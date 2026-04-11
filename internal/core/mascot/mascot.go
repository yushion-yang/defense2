package mascot

import "math/rand"

// MascotVM is the view-model snapshot for rendering.
type MascotVM struct {
	Visible    bool
	HasDialog  bool
	Text       string
	Expression string
	CanClick   bool // true = click to advance
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
}

// NewGuide creates a Guide preloaded with dialogs.
// shownIDs may be nil; if non-nil it restores previously-seen Once dialog IDs.
func NewGuide(dialogs []Dialog, shownIDs map[string]bool) *Guide {
	shown := shownIDs
	if shown == nil {
		shown = make(map[string]bool)
	}
	return &Guide{
		dialogs:  dialogs,
		shownIDs: shown,
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

// Tick advances auto-advance timers. dt is in seconds.
func (g *Guide) Tick(dt float64) {
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
		Visible: true,
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
func (g *Guide) tryTrigger(event string) {
	if g.active != nil {
		return
	}
	var best *Dialog
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
		if best == nil || d.Priority > best.Priority {
			best = d
		}
	}
	if best != nil {
		g.startDialog(best)
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
	g.active = nil
	g.lineIdx = 0
	g.timer = 0
}
