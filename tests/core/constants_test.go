package core_test

import (
	"testing"

	"defense2/internal/core/game"
)

func TestScreenDimensions(t *testing.T) {
	if game.ScreenWidth <= 0 || game.ScreenHeight <= 0 {
		t.Fatal("screen dimensions must be positive")
	}
	// Tower defense is landscape
	if game.ScreenWidth <= game.ScreenHeight {
		t.Fatal("expected landscape orientation (width > height)")
	}
}
