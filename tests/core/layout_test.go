package core_test

import (
	"testing"

	"defense2/internal/render/hud"
)

func TestLayoutRegionsNoOverlap(t *testing.T) {
	l := hud.Layout
	if l.TopBarY < 0 {
		t.Fatal("TopBarY should be >= 0")
	}
	if l.BuildMenuY <= l.TopBarY+l.TopBarH {
		t.Fatal("BuildMenuY should be below TopBar")
	}
}
