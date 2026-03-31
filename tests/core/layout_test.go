// layout_test.go — HUD layout and hit-test verification.
package core_test

import (
	"testing"

	"defense2/internal/render/hud"
	"defense2/internal/render/theme"
)

func TestLayoutRegionsNoOverlap(t *testing.T) {
	// TopBar is at the top of the canvas.
	if theme.TopBarY < 0 {
		t.Fatal("TopBarY should be >= 0")
	}
	// Build menu at the bottom should not overlap the top bar.
	buildMenuY := float32(theme.CanvasH) - float32(theme.BuildCardH) - 16 - float32(theme.BottomMargin)
	topBarBottom := float32(theme.TopBarY) + float32(theme.TopBarH)
	if buildMenuY <= topBarBottom {
		t.Fatalf("BuildMenuY (%.0f) should be below TopBar bottom (%.0f)", buildMenuY, topBarBottom)
	}
}

func TestBuildMenuHitTest(t *testing.T) {
	// The build menu is centered; compute the first card's approximate position.
	cardCount := 4
	cardW := float32(theme.BuildCardW)
	cardGap := float32(theme.BuildCardGap)
	pad := float32(8)

	n := float32(cardCount)
	innerW := n*cardW + (n-1)*cardGap
	panelW := innerW + pad*2
	panelH := float32(theme.BuildCardH) + pad*2
	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin)
	cardStartX := panelX + pad
	cardStartY := panelY + pad

	// Click inside first card.
	x := cardStartX + 5
	y := cardStartY + 10
	idx := hud.BuildMenuHitTest(x, y, cardCount, cardCount)
	if idx != 0 {
		t.Fatalf("expected slot 0, got %d", idx)
	}

	// Click outside the panel.
	idx = hud.BuildMenuHitTest(0, 0, cardCount, cardCount)
	if idx != -1 {
		t.Fatalf("expected -1 for outside click, got %d", idx)
	}
}
