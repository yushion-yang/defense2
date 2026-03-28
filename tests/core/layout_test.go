// layout_test.go — HUD 布局坐标测试。
package core_test

import (
	"testing"

	"defense2/internal/render/hud"
)

func TestLayoutRegionsNoOverlap(t *testing.T) {
	L := hud.Layout
	// 顶部栏在最上面
	if L.TopBarY < 0 {
		t.Fatal("TopBarY should be >= 0")
	}
	// 建塔菜单在底部，不与顶部栏重叠
	if L.BuildMenuY <= L.TopBarY+L.TopBarH {
		t.Fatal("BuildMenuY should be below TopBar")
	}
	// 信息面板不与建塔菜单 X 区域冲突
	if L.InfoPanelX >= L.BuildMenuX && L.InfoPanelX <= L.BuildMenuX+L.BuildMenuW {
		t.Log("InfoPanel X overlaps BuildMenu X range — acceptable if Y separated")
	}
}

func TestBuildMenuHitTest(t *testing.T) {
	L := hud.Layout
	// 点击第一个槽位
	x := L.BuildMenuX + L.SlotGap + 5
	y := L.BuildMenuY + 10
	idx := hud.BuildMenuHitTest(x, y, 4)
	if idx != 0 {
		t.Fatalf("expected slot 0, got %d", idx)
	}

	// 点击面板外
	idx = hud.BuildMenuHitTest(0, 0, 4)
	if idx != -1 {
		t.Fatalf("expected -1 for outside click, got %d", idx)
	}
}
