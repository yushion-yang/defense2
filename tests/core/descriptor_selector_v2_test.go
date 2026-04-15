// descriptor_selector_v2_test.go — Phase 2 新增 3 种选择器测试。
//
// 覆盖 Cone / Ring360 / Random 选择器。
// 复用 descriptor_selector_test.go 中的 mock 类型。
package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ============================================================
// ConeSelector 测试
// ============================================================

func TestConeSelector_FiltersWithinCone(t *testing.T) {
	// 塔在 (0,0)，命中点在 (100,0)（面向右方）
	// 锥角 90 度 → ±45 度
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 80, Y: 0, Active: true, Index: 0},   // 正前方，距 80 ✓
			{X: 60, Y: 50, Active: true, Index: 1},   // 角度 ~40°，距 ~78 ✓
			{X: 0, Y: 80, Active: true, Index: 2},    // 角度 90°，不在锥内 ✗
			{X: -50, Y: 0, Active: true, Index: 3},   // 身后，角度 180° ✗
			{X: 50, Y: 55, Active: true, Index: 4},   // 角度 ~48°，刚好超出 ✗
		},
	}

	sel := descriptor.ConeSelector{
		Angle:  90,
		Radius: descriptor.FixedScaler{Value: 100},
	}
	ctx := descriptor.SelectorCtx{
		TowerX:   0,
		TowerY:   0,
		HitX:     100,
		HitY:     0,
		Strength: 100,
		Enemies:  enemies,
	}

	targets := sel.Select(ctx)
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets in cone, got %d", len(targets))
	}
}

func TestConeSelector_RadiusFilter(t *testing.T) {
	// 敌人在锥角内但超出半径
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 50, Y: 0, Active: true, Index: 0},  // 距=50, 在半径内 ✓
			{X: 200, Y: 0, Active: true, Index: 1}, // 距=200, 超出半径 ✗
		},
	}

	sel := descriptor.ConeSelector{
		Angle:  90,
		Radius: descriptor.FixedScaler{Value: 100},
	}
	ctx := descriptor.SelectorCtx{
		TowerX:   0,
		TowerY:   0,
		HitX:     100,
		HitY:     0,
		Strength: 100,
		Enemies:  enemies,
	}

	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target within radius, got %d", len(targets))
	}
	if targets[0].EnemyIdx != 0 {
		t.Errorf("expected EnemyIdx=0, got %d", targets[0].EnemyIdx)
	}
}

func TestConeSelector_360DegreeCone(t *testing.T) {
	// 锥角 360 度 = 全范围选择
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 50, Y: 0, Active: true, Index: 0},
			{X: -50, Y: 0, Active: true, Index: 1},
			{X: 0, Y: 50, Active: true, Index: 2},
		},
	}

	sel := descriptor.ConeSelector{
		Angle:  360,
		Radius: descriptor.FixedScaler{Value: 100},
	}
	ctx := descriptor.SelectorCtx{
		TowerX:   0,
		TowerY:   0,
		HitX:     100,
		HitY:     0,
		Strength: 100,
		Enemies:  enemies,
	}

	targets := sel.Select(ctx)
	if len(targets) != 3 {
		t.Fatalf("360 度锥应选中全部 3 个目标, got %d", len(targets))
	}
}

func TestConeSelector_NilEnemies(t *testing.T) {
	sel := descriptor.ConeSelector{
		Angle:  90,
		Radius: descriptor.FixedScaler{Value: 100},
	}
	ctx := descriptor.SelectorCtx{
		TowerX: 0, TowerY: 0, HitX: 100, HitY: 0,
		Strength: 100, Enemies: nil,
	}
	targets := sel.Select(ctx)
	if targets != nil {
		t.Error("Enemies=nil 应返回 nil")
	}
}

// ============================================================
// Ring360Selector 测试
// ============================================================

func TestRing360Selector_GeneratesCorrectCount(t *testing.T) {
	sel := descriptor.Ring360Selector{
		Count: descriptor.FixedScaler{Value: 8},
	}
	ctx := descriptor.SelectorCtx{
		TowerX:     100,
		TowerY:     100,
		TowerRange: 150,
		Strength:   100,
	}

	targets := sel.Select(ctx)
	if len(targets) != 8 {
		t.Fatalf("expected 8 targets, got %d", len(targets))
	}
}

func TestRing360Selector_EvenlySpaced(t *testing.T) {
	sel := descriptor.Ring360Selector{
		Count: descriptor.FixedScaler{Value: 4},
	}
	ctx := descriptor.SelectorCtx{
		TowerX:     0,
		TowerY:     0,
		TowerRange: 100,
		Strength:   100,
	}

	targets := sel.Select(ctx)
	if len(targets) != 4 {
		t.Fatalf("expected 4 targets, got %d", len(targets))
	}

	// 4 个方向应该在圆周上等间距分布
	// 验证所有点到塔的距离都等于 TowerRange
	for i, tgt := range targets {
		dist := math.Sqrt(tgt.X*tgt.X + tgt.Y*tgt.Y)
		if math.Abs(dist-100) > 1e-6 {
			t.Errorf("target[%d]: 距离=%v, 期望 100", i, dist)
		}
	}

	// 验证相邻目标之间的角度间距为 90 度
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		// 两个向量的点积 = cos(angle) * r^2
		dot := targets[i].X*targets[j].X + targets[i].Y*targets[j].Y
		cosAngle := dot / (100 * 100)
		// 相邻 90 度 → cos(90°) ≈ 0
		if math.Abs(cosAngle) > 1e-6 {
			t.Errorf("target[%d] 和 target[%d] 的角度不是 90°: cos=%v", i, j, cosAngle)
		}
	}
}

func TestRing360Selector_CountZero(t *testing.T) {
	sel := descriptor.Ring360Selector{
		Count: descriptor.FixedScaler{Value: 0},
	}
	ctx := descriptor.SelectorCtx{TowerRange: 100, Strength: 100}

	targets := sel.Select(ctx)
	if len(targets) != 0 {
		t.Errorf("count=0: expected 0 targets, got %d", len(targets))
	}
}

func TestRing360Selector_TargetIndices(t *testing.T) {
	sel := descriptor.Ring360Selector{
		Count: descriptor.FixedScaler{Value: 2},
	}
	ctx := descriptor.SelectorCtx{
		TowerX: 50, TowerY: 50, TowerRange: 100, Strength: 100,
	}

	targets := sel.Select(ctx)
	for _, tgt := range targets {
		if tgt.EnemyIdx != -1 {
			t.Errorf("Ring360 目标应无具体敌人索引，expected EnemyIdx=-1, got %d", tgt.EnemyIdx)
		}
		if tgt.TowerIdx != -1 {
			t.Errorf("Ring360 目标应无具体塔索引，expected TowerIdx=-1, got %d", tgt.TowerIdx)
		}
	}
}

// ============================================================
// RandomSelector 测试
// ============================================================

func TestRandomSelector_SelectsWithinRadius(t *testing.T) {
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 10, Y: 10, Active: true, Index: 0},  // 距(0,0) ≈ 14 ✓
			{X: 20, Y: 0, Active: true, Index: 1},   // 距=20 ✓
			{X: 200, Y: 200, Active: true, Index: 2}, // 距≈283 ✗
		},
	}

	sel := descriptor.RandomSelector{
		Count:  descriptor.FixedScaler{Value: 5}, // 请求 5 个但只有 2 个在范围内
		Radius: 50,
	}
	ctx := descriptor.SelectorCtx{
		TowerX: 0, TowerY: 0, Strength: 100, Enemies: enemies,
	}

	targets := sel.Select(ctx)
	// 只有 2 个在半径内，取 min(5, 2) = 2
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets (capped by available), got %d", len(targets))
	}
}

func TestRandomSelector_RespectsCount(t *testing.T) {
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 10, Y: 0, Active: true, Index: 0},
			{X: 20, Y: 0, Active: true, Index: 1},
			{X: 30, Y: 0, Active: true, Index: 2},
			{X: 40, Y: 0, Active: true, Index: 3},
			{X: 45, Y: 0, Active: true, Index: 4},
		},
	}

	sel := descriptor.RandomSelector{
		Count:  descriptor.FixedScaler{Value: 3},
		Radius: 100,
	}
	ctx := descriptor.SelectorCtx{
		TowerX: 0, TowerY: 0, Strength: 100, Enemies: enemies,
	}

	targets := sel.Select(ctx)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	// 确认无重复
	seen := make(map[int]bool)
	for _, tgt := range targets {
		if seen[tgt.EnemyIdx] {
			t.Errorf("重复目标: EnemyIdx=%d", tgt.EnemyIdx)
		}
		seen[tgt.EnemyIdx] = true
	}
}

func TestRandomSelector_NilEnemies(t *testing.T) {
	sel := descriptor.RandomSelector{
		Count:  descriptor.FixedScaler{Value: 3},
		Radius: 100,
	}
	ctx := descriptor.SelectorCtx{Strength: 100, Enemies: nil}

	targets := sel.Select(ctx)
	if targets != nil {
		t.Error("Enemies=nil 应返回 nil")
	}
}

func TestRandomSelector_CountZero(t *testing.T) {
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 10, Y: 0, Active: true, Index: 0},
		},
	}
	sel := descriptor.RandomSelector{
		Count:  descriptor.FixedScaler{Value: 0},
		Radius: 100,
	}
	ctx := descriptor.SelectorCtx{
		TowerX: 0, TowerY: 0, Strength: 100, Enemies: enemies,
	}

	targets := sel.Select(ctx)
	if len(targets) != 0 {
		t.Errorf("count=0: expected 0 targets, got %d", len(targets))
	}
}

// ============================================================
// 编译期验证接口实现
// ============================================================

var _ descriptor.Selector = descriptor.ConeSelector{}
var _ descriptor.Selector = descriptor.Ring360Selector{}
var _ descriptor.Selector = descriptor.RandomSelector{}
