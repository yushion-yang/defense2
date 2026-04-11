package mascot

import (
	"fmt"
	"testing"
)

func TestParseDialogs(t *testing.T) {
	raw := `[
		{
			"id": "test_dialog",
			"scene": "title",
			"trigger": "scene_enter",
			"lines": [
				{"text": "Hello!", "expression": "happy", "autoAdvance": 0},
				{"text": "Bye!", "expression": "idle", "autoAdvance": 2.5}
			],
			"once": true,
			"priority": 5
		}
	]`
	dialogs, err := ParseDialogs([]byte(raw))
	if err != nil {
		t.Fatalf("ParseDialogs error: %v", err)
	}
	if len(dialogs) != 1 {
		t.Fatalf("expected 1 dialog, got %d", len(dialogs))
	}
	d := dialogs[0]
	if d.ID != "test_dialog" {
		t.Errorf("ID = %q, want \"test_dialog\"", d.ID)
	}
	if d.Scene != "title" {
		t.Errorf("Scene = %q, want \"title\"", d.Scene)
	}
	if d.Trigger != "scene_enter" {
		t.Errorf("Trigger = %q, want \"scene_enter\"", d.Trigger)
	}
	if len(d.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(d.Lines))
	}
	if d.Lines[0].Text != "Hello!" {
		t.Errorf("Lines[0].Text = %q, want \"Hello!\"", d.Lines[0].Text)
	}
	if d.Lines[0].Expression != "happy" {
		t.Errorf("Lines[0].Expression = %q, want \"happy\"", d.Lines[0].Expression)
	}
	if d.Lines[1].AutoAdvance != 2.5 {
		t.Errorf("Lines[1].AutoAdvance = %f, want 2.5", d.Lines[1].AutoAdvance)
	}
	if !d.Once {
		t.Error("Once = false, want true")
	}
	if d.Priority != 5 {
		t.Errorf("Priority = %d, want 5", d.Priority)
	}
}

func TestParseDialogsInvalidJSON(t *testing.T) {
	_, err := ParseDialogs([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseDialogsEmptyArray(t *testing.T) {
	dialogs, err := ParseDialogs([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialogs) != 0 {
		t.Fatalf("expected 0 dialogs, got %d", len(dialogs))
	}
}

// mockFS implements AssetReader for testing.
type mockFS struct {
	files map[string][]byte
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return data, nil
}

func TestLoadAllDialogsMergesFiles(t *testing.T) {
	fs := &mockFS{files: map[string][]byte{
		"config/mascot/dialogs-title.json": []byte(`[
			{"id": "d1", "scene": "title", "trigger": "scene_enter",
			 "lines": [{"text": "A", "expression": "idle"}]}
		]`),
		"config/mascot/dialogs-stage.json": []byte(`[
			{"id": "d2", "scene": "stage", "trigger": "wave_start",
			 "lines": [{"text": "B", "expression": "talk"}]}
		]`),
	}}
	dialogs, err := LoadAllDialogs(fs)
	if err != nil {
		t.Fatalf("LoadAllDialogs error: %v", err)
	}
	if len(dialogs) != 2 {
		t.Fatalf("expected 2 dialogs, got %d", len(dialogs))
	}
}

func TestLoadAllDialogsSkipsMissing(t *testing.T) {
	// Empty filesystem — all files missing. Should return nil/empty, no error.
	fs := &mockFS{files: map[string][]byte{}}
	dialogs, err := LoadAllDialogs(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialogs) != 0 {
		t.Fatalf("expected 0 dialogs, got %d", len(dialogs))
	}
}

func TestLoadAllDialogsReturnsParseError(t *testing.T) {
	fs := &mockFS{files: map[string][]byte{
		"config/mascot/dialogs-title.json": []byte(`{broken`),
	}}
	_, err := LoadAllDialogs(fs)
	if err == nil {
		t.Fatal("expected error for invalid JSON in file")
	}
}
