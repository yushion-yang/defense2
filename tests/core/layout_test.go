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
	// Replicate calcBuildPanelMetrics logic for cardCount=4 to derive
	// accurate grid coordinates. The build panel uses its own internal
	// constants (bpCols=5, bpCardW=100, bpCardH=80, bpCardGap=8,
	// bpPadX=16, bpPadY=12, bpTitleH=28, bpHeaderH=32) which differ
	// from the theme.Build* display constants.
	cardCount := 4

	// Internal build_menu.go constants (must stay in sync manually).
	const (
		bpCols    = 5
		bpCardW   = float32(100)
		bpCardH   = float32(80)
		bpCardGap = float32(8)
		bpPadX    = float32(16)
		bpPadY    = float32(12)
		bpTitleH  = float32(28)
		bpHeaderH = float32(32)
	)
	cols := bpCols
	if cardCount < cols {
		cols = cardCount
	}
	gridW := float32(cols)*bpCardW + float32(cols-1)*bpCardGap
	gridH := float32(1) * bpCardH // 1 row for 4 cards
	panelW := gridW + bpPadX*2
	panelH := bpTitleH + bpHeaderH + gridH + bpPadY*2
	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin) - float32(theme.ActionBarH) - 4
	gridX := panelX + bpPadX
	gridY := panelY + bpTitleH + bpHeaderH

	// Click inside first card.
	x := gridX + 5
	y := gridY + 10
	idx := hud.BuildMenuHitTest(x, y, cardCount, cardCount)
	if idx != 0 {
		t.Fatalf("expected slot 0, got %d (click at %.0f,%.0f)", idx, x, y)
	}

	// Click outside the panel.
	idx = hud.BuildMenuHitTest(0, 0, cardCount, cardCount)
	if idx != -1 {
		t.Fatalf("expected -1 for outside click, got %d", idx)
	}
}
