package mascot

import (
	"testing"
)

// helper: build a Guide with two dialogs for "title" scene.
func newTestGuide() *Guide {
	dialogs := []Dialog{
		{
			ID:      "welcome",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "Welcome!", Expression: "happy", AutoAdvance: 0},
				{Text: "Click to start.", Expression: "idle", AutoAdvance: 0},
			},
			Once:     false,
			Priority: 1,
		},
		{
			ID:      "wave_tip",
			Scene:   "stage",
			Trigger: "wave_start",
			Lines: []Line{
				{Text: "Enemies incoming!", Expression: "surprised", AutoAdvance: 2.0},
			},
			Once:     false,
			Priority: 1,
		},
	}
	return NewGuide(dialogs, nil)
}

func TestSetSceneTriggersEnterDialog(t *testing.T) {
	g := newTestGuide()
	g.SetScene("title")

	vm := g.VM()
	if !vm.HasDialog {
		t.Fatal("expected dialog after SetScene(\"title\")")
	}
	if vm.Text != "Welcome!" {
		t.Fatalf("expected text \"Welcome!\", got %q", vm.Text)
	}
	if vm.Expression != "happy" {
		t.Fatalf("expected expression \"happy\", got %q", vm.Expression)
	}
}

func TestTriggerMatchesEvent(t *testing.T) {
	g := newTestGuide()
	g.SetScene("stage")

	// No dialog yet — scene_enter has no match for "stage" in our test data.
	vm := g.VM()
	if vm.HasDialog {
		t.Fatal("expected no dialog before trigger")
	}

	g.Trigger("wave_start")
	vm = g.VM()
	if !vm.HasDialog {
		t.Fatal("expected dialog after Trigger(\"wave_start\")")
	}
	if vm.Text != "Enemies incoming!" {
		t.Fatalf("expected text \"Enemies incoming!\", got %q", vm.Text)
	}
}

func TestClickAdvance(t *testing.T) {
	g := newTestGuide()
	g.SetScene("title") // starts "welcome" dialog (2 lines)

	vm := g.VM()
	if vm.Text != "Welcome!" {
		t.Fatalf("line 0: expected \"Welcome!\", got %q", vm.Text)
	}

	g.ClickAdvance()
	vm = g.VM()
	if !vm.HasDialog {
		t.Fatal("expected dialog still active after first ClickAdvance")
	}
	if vm.Text != "Click to start." {
		t.Fatalf("line 1: expected \"Click to start.\", got %q", vm.Text)
	}

	g.ClickAdvance()
	vm = g.VM()
	if vm.HasDialog {
		t.Fatal("expected dialog to finish after advancing past last line")
	}
}

func TestAutoAdvance(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "auto",
			Scene:   "stage",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "Line A", Expression: "talk", AutoAdvance: 1.0},
				{Text: "Line B", Expression: "idle", AutoAdvance: 0.5},
			},
		},
	}
	g := NewGuide(dialogs, nil)
	g.SetScene("stage")

	if vm := g.VM(); vm.Text != "Line A" {
		t.Fatalf("expected \"Line A\", got %q", vm.Text)
	}

	// Not enough time yet.
	g.Tick(0.5)
	if vm := g.VM(); vm.Text != "Line A" {
		t.Fatalf("expected still \"Line A\" after 0.5s, got %q", vm.Text)
	}

	// Enough time to advance.
	g.Tick(0.6)
	if vm := g.VM(); vm.Text != "Line B" {
		t.Fatalf("expected \"Line B\" after 1.1s total, got %q", vm.Text)
	}

	// Auto-advance Line B (0.5s threshold).
	g.Tick(0.6)
	if vm := g.VM(); vm.HasDialog {
		t.Fatal("expected dialog to finish after auto-advancing last line")
	}
}

func TestOnceDialogNotRepeated(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "once_tip",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "First time!", Expression: "happy"},
			},
			Once:     true,
			Priority: 1,
		},
	}
	g := NewGuide(dialogs, nil)

	// First time: should show.
	g.SetScene("title")
	if vm := g.VM(); !vm.HasDialog {
		t.Fatal("expected once dialog to show first time")
	}
	g.ClickAdvance() // finish dialog

	// Second time: should NOT show.
	g.SetScene("select") // switch away
	g.SetScene("title")  // switch back
	if vm := g.VM(); vm.HasDialog {
		t.Fatal("expected once dialog NOT to show second time")
	}

	// Verify ShownIDs reports it.
	shown := g.ShownIDs()
	if !shown["once_tip"] {
		t.Fatal("expected \"once_tip\" in ShownIDs()")
	}
}

func TestOnceDialogPreloaded(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "already_seen",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "You won't see this.", Expression: "idle"},
			},
			Once:     true,
			Priority: 1,
		},
	}
	// Pre-load shown IDs (simulating persistence restore).
	shown := map[string]bool{"already_seen": true}
	g := NewGuide(dialogs, shown)

	g.SetScene("title")
	if vm := g.VM(); vm.HasDialog {
		t.Fatal("expected pre-loaded once dialog NOT to trigger")
	}
}

func TestVMWhenIdle(t *testing.T) {
	g := newTestGuide()
	g.SetScene("stage") // no scene_enter dialog for "stage" in test data

	vm := g.VM()
	if !vm.Visible {
		t.Fatal("expected Visible=true even when idle")
	}
	if vm.HasDialog {
		t.Fatal("expected HasDialog=false when idle")
	}
	if vm.Text != "" {
		t.Fatalf("expected empty Text when idle, got %q", vm.Text)
	}
}

func TestPrioritySelection(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "low",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines:   []Line{{Text: "Low priority", Expression: "idle"}},
			Priority: 1,
		},
		{
			ID:      "high",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines:   []Line{{Text: "High priority", Expression: "happy"}},
			Priority: 10,
		},
	}
	g := NewGuide(dialogs, nil)
	g.SetScene("title")

	vm := g.VM()
	if vm.Text != "High priority" {
		t.Fatalf("expected high-priority dialog, got %q", vm.Text)
	}
}

func TestWildcardScene(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "global_hint",
			Scene:   "*",
			Trigger: "hint",
			Lines:   []Line{{Text: "Global hint!", Expression: "talk"}},
			Priority: 1,
		},
	}
	g := NewGuide(dialogs, nil)

	g.SetScene("whatever_scene")
	g.Trigger("hint")

	vm := g.VM()
	if !vm.HasDialog {
		t.Fatal("expected wildcard dialog to match any scene")
	}
	if vm.Text != "Global hint!" {
		t.Fatalf("expected \"Global hint!\", got %q", vm.Text)
	}
}

func TestNoInterruptActiveDialog(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "first",
			Scene:   "stage",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "Line 1", Expression: "idle"},
				{Text: "Line 2", Expression: "idle"},
			},
			Priority: 1,
		},
		{
			ID:      "second",
			Scene:   "stage",
			Trigger: "wave_start",
			Lines:   []Line{{Text: "Should not appear", Expression: "idle"}},
			Priority: 5,
		},
	}
	g := NewGuide(dialogs, nil)
	g.SetScene("stage") // starts "first" dialog

	// Try to trigger "second" while "first" is playing.
	g.Trigger("wave_start")

	vm := g.VM()
	if vm.Text != "Line 1" {
		t.Fatalf("expected active dialog to be preserved, got %q", vm.Text)
	}
}

func TestClickAdvanceOnAutoAdvanceLine(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "auto_click",
			Scene:   "stage",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "Auto line", Expression: "talk", AutoAdvance: 5.0},
				{Text: "Done", Expression: "idle"},
			},
		},
	}
	g := NewGuide(dialogs, nil)
	g.SetScene("stage")

	// Should be able to click-advance even though line has autoAdvance.
	g.ClickAdvance()
	vm := g.VM()
	if vm.Text != "Done" {
		t.Fatalf("expected click to advance auto-advance line, got %q", vm.Text)
	}
}

func TestVMCanClick(t *testing.T) {
	dialogs := []Dialog{
		{
			ID:      "canclick",
			Scene:   "title",
			Trigger: "scene_enter",
			Lines: []Line{
				{Text: "Click me", Expression: "idle", AutoAdvance: 0},
			},
		},
	}
	g := NewGuide(dialogs, nil)
	g.SetScene("title")

	vm := g.VM()
	if !vm.CanClick {
		t.Fatal("expected CanClick=true when dialog is active")
	}
}

func TestTickNoDialogNoPanic(t *testing.T) {
	g := newTestGuide()
	g.SetScene("stage") // no scene_enter for stage

	// Tick without active dialog should not panic.
	g.Tick(1.0)
	g.ClickAdvance()
}
