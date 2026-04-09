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

func (m *mockScene) Update() error        { m.updateCount++; return nil }
func (m *mockScene) Draw(_ *ebiten.Image) { m.drawCount++ }

func TestGameSceneSwitch(t *testing.T) {
	g := scene.NewGame()

	// Initial scene — run one tick so current is active.
	if err := g.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Switch to mock scene — triggers fade-out/fade-in transition.
	// transSpeed = 1/18, so fade-out takes 18 ticks, fade-in takes 18 more.
	mock := &mockScene{}
	g.SwitchScene(mock)

	// Run enough ticks to complete the fade-out (18 ticks).
	// On tick 18, transAlpha reaches 1.0, g.current switches to mock,
	// and mock.Update() is called for the first time.
	for i := 0; i < 18; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("update tick %d failed: %v", i, err)
		}
	}

	// After fade-out completes, mock should have been updated at least once.
	if mock.updateCount < 1 {
		t.Fatalf("expected mock update count >= 1 after fade-out, got %d", mock.updateCount)
	}
	countAfterFadeOut := mock.updateCount

	// One more tick — mock continues receiving updates during fade-in.
	if err := g.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if mock.updateCount != countAfterFadeOut+1 {
		t.Fatalf("expected mock update count %d, got %d", countAfterFadeOut+1, mock.updateCount)
	}
}

func TestGameLayout(t *testing.T) {
	g := scene.NewGame()
	w, h := g.Layout(0, 0)
	if w != 1200 || h != 540 {
		t.Fatalf("expected 1200x540, got %dx%d", w, h)
	}
}
