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

	// Click inside first card (no tabs, tabCount=0).
	x := gridX + 5
	y := gridY + 10
	hit := hud.BuildMenuHitTest(x, y, cardCount, cardCount, 0)
	if hit.CardIdx != 0 {
		t.Fatalf("expected slot 0, got %d (click at %.0f,%.0f)", hit.CardIdx, x, y)
	}
	if hit.TabIdx != -1 {
		t.Fatalf("expected TabIdx -1 (no tabs), got %d", hit.TabIdx)
	}

	// Click outside the panel.
	hit = hud.BuildMenuHitTest(0, 0, cardCount, cardCount, 0)
	if hit.CardIdx != -1 {
		t.Fatalf("expected -1 for outside click, got %d", hit.CardIdx)
	}
	if hit.TabIdx != -1 {
		t.Fatalf("expected TabIdx -1 for outside click, got %d", hit.TabIdx)
	}
}

// TestBuildMenuHitTestWithTabs 验证有 tab 时面板高度增大且卡片网格下移。
func TestBuildMenuHitTestWithTabs(t *testing.T) {
	cardCount := 4

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

	tabCount := 2
	tabH := float32(theme.BuildFactionTabH) + float32(theme.BuildFactionTabGap) + 4

	cols := bpCols
	if cardCount < cols {
		cols = cardCount
	}
	gridW := float32(cols)*bpCardW + float32(cols-1)*bpCardGap
	gridH := float32(1) * bpCardH
	panelW := gridW + bpPadX*2
	panelH := bpTitleH + bpHeaderH + tabH + gridH + bpPadY*2
	panelX := (float32(theme.CanvasW) - panelW) / 2
	panelY := float32(theme.CanvasH) - panelH - float32(theme.BottomMargin) - float32(theme.ActionBarH) - 4
	gridX := panelX + bpPadX
	gridY := panelY + bpTitleH + bpHeaderH + tabH

	// Click inside first card (卡片网格因 tab 栏下移)
	x := gridX + 5
	y := gridY + 10
	hit := hud.BuildMenuHitTest(x, y, cardCount, cardCount, tabCount)
	if hit.CardIdx != 0 {
		t.Fatalf("expected slot 0 with tabs, got CardIdx=%d (click at %.0f,%.0f)", hit.CardIdx, x, y)
	}
	if hit.TabIdx != -1 {
		t.Fatalf("expected TabIdx -1 for card click, got %d", hit.TabIdx)
	}

	// Click outside panel
	hit = hud.BuildMenuHitTest(0, 0, cardCount, cardCount, tabCount)
	if hit.CardIdx != -1 {
		t.Fatalf("expected CardIdx -1 for outside click with tabs, got %d", hit.CardIdx)
	}

	// 面板有 tab 时高度应比无 tab 时大
	noTabPanelH := bpTitleH + bpHeaderH + gridH + bpPadY*2
	if panelH <= noTabPanelH {
		t.Fatalf("panel with tabs (%.0f) should be taller than without (%.0f)", panelH, noTabPanelH)
	}
}
