package core_test

import (
	"testing"

	"defense2/internal/scene"

	"github.com/hajimehoshi/ebiten/v2"
)

// mockScene tracks Update/Draw calls for testing.
type mockScene struct {
	updateCount int
	drawCount   int
}

func (m *mockScene) Update() error { m.updateCount++; return nil }
func (m *mockScene) Draw(_ *ebiten.Image) { m.drawCount++ }

func TestGameSceneSwitch(t *testing.T) {
	g := scene.NewGame()

	// Initial scene should be set (TitleScene)
	if err := g.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Switch to a mock scene
	mock := &mockScene{}
	g.SwitchScene(mock)

	// First update applies the switch
	if err := g.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if mock.updateCount != 1 {
		t.Fatalf("expected mock update count 1, got %d", mock.updateCount)
	}

	// Second update continues with mock
	if err := g.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if mock.updateCount != 2 {
		t.Fatalf("expected mock update count 2, got %d", mock.updateCount)
	}
}

func TestGameLayout(t *testing.T) {
	g := scene.NewGame()
	w, h := g.Layout(0, 0)
	if w != 1200 || h != 540 {
		t.Fatalf("expected 1200x540, got %dx%d", w, h)
	}
}
