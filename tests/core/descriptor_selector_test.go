// descriptor_selector_test.go — Selector 类型测试。
//
// 使用 mock EnemyQuerier/TowerQuerier 验证 6 种 Selector 的行为：
// CurrentTarget / AoeRadius / Chain / AllInRange / NearbyAllies / SelfTower
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── mock 实现 ──────────────────────────────────────────

// mockEnemyQuerier 提供可控的敌人查询结果。
type mockEnemyQuerier struct {
	enemies []descriptor.EnemyRef
}

func (m *mockEnemyQuerier) QueryRadius(cx, cy, radius float64) []descriptor.EnemyRef {
	var result []descriptor.EnemyRef
	r2 := radius * radius
	for _, e := range m.enemies {
		if !e.Active {
			continue
		}
		dx, dy := e.X-cx, e.Y-cy
		if dx*dx+dy*dy <= r2 {
			result = append(result, e)
		}
	}
	return result
}

func (m *mockEnemyQuerier) AllActive() []descriptor.EnemyRef {
	var result []descriptor.EnemyRef
	for _, e := range m.enemies {
		if e.Active {
			result = append(result, e)
		}
	}
	return result
}

// mockTowerQuerier 提供可控的塔查询结果。
type mockTowerQuerier struct {
	towers []descriptor.TowerRef
}

func (m *mockTowerQuerier) QueryRadius(cx, cy, radius float64) []descriptor.TowerRef {
	var result []descriptor.TowerRef
	r2 := radius * radius
	for _, tw := range m.towers {
		if !tw.Active {
			continue
		}
		dx, dy := tw.X-cx, tw.Y-cy
		if dx*dx+dy*dy <= r2 {
			result = append(result, tw)
		}
	}
	return result
}

// ── 测试 helpers ───────────────────────────────────────

func baseSelectorCtx() descriptor.SelectorCtx {
	return descriptor.SelectorCtx{
		CurrentEnemyIdx: 0,
		HitX:            100,
		HitY:            100,
		TowerX:          50,
		TowerY:          50,
		TowerRange:      200,
		Strength:        100,
	}
}

// ── CurrentTargetSelector ──────────────────────────────

func TestCurrentTargetSelector(t *testing.T) {
	sel := descriptor.CurrentTargetSelector{}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 7

	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].EnemyIdx != 7 {
		t.Errorf("expected EnemyIdx=7, got %d", targets[0].EnemyIdx)
	}
	if targets[0].TowerIdx != -1 {
		t.Errorf("expected TowerIdx=-1, got %d", targets[0].TowerIdx)
	}
	if targets[0].X != ctx.HitX || targets[0].Y != ctx.HitY {
		t.Errorf("expected position (%v,%v), got (%v,%v)",
			ctx.HitX, ctx.HitY, targets[0].X, targets[0].Y)
	}
}

// ── AoeRadiusSelector ──────────────────────────────────

func TestAoeRadiusSelectorFiltersCorrectly(t *testing.T) {
	// 5 个敌人，3 个在半径 50 内，2 个在外面
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 110, Y: 100, Active: true, Index: 0}, // 距 (100,100)=10 ✓
			{X: 130, Y: 100, Active: true, Index: 1}, // 距=30 ✓
			{X: 100, Y: 140, Active: true, Index: 2}, // 距=40 ✓
			{X: 200, Y: 200, Active: true, Index: 3}, // 距≈141 ✗
			{X: 300, Y: 300, Active: true, Index: 4}, // 距≈283 ✗
		},
	}

	sel := descriptor.AoeRadiusSelector{
		Radius: descriptor.FixedScaler{Value: 50},
	}
	ctx := baseSelectorCtx()
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets in radius, got %d", len(targets))
	}
	// 确认返回的都是敌人目标
	for _, tgt := range targets {
		if tgt.EnemyIdx < 0 {
			t.Error("AoE target should have valid EnemyIdx")
		}
		if tgt.TowerIdx != -1 {
			t.Error("AoE target should have TowerIdx=-1")
		}
	}
}

func TestAoeRadiusSelectorScalesWithStrength(t *testing.T) {
	// 敌人在距离 80 处，只有高强度时半径才够大
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 180, Y: 100, Active: true, Index: 0}, // 距 (100,100)=80
		},
	}
	// base=50, potential=50 → str=0 → radius=50（不够）; str=100 → radius=100（够）
	sel := descriptor.AoeRadiusSelector{
		Radius: descriptor.LinearScaler{Base: 50, Potential: 50},
	}
	ctx := baseSelectorCtx()
	ctx.Enemies = enemies

	ctx.Strength = 0
	targets := sel.Select(ctx)
	if len(targets) != 0 {
		t.Errorf("str=0: expected 0 targets, got %d", len(targets))
	}

	ctx.Strength = 100
	targets = sel.Select(ctx)
	if len(targets) != 1 {
		t.Errorf("str=100: expected 1 target, got %d", len(targets))
	}
}

func TestAoeRadiusSelectorNoEnemies(t *testing.T) {
	enemies := &mockEnemyQuerier{enemies: nil}
	sel := descriptor.AoeRadiusSelector{
		Radius: descriptor.FixedScaler{Value: 100},
	}
	ctx := baseSelectorCtx()
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

// ── AllInRangeSelector ─────────────────────────────────

func TestAllInRangeSelector(t *testing.T) {
	// 塔在 (50,50)，range=200
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 60, Y: 60, Active: true, Index: 0},   // 距≈14 ✓
			{X: 100, Y: 100, Active: true, Index: 1},  // 距≈71 ✓
			{X: 200, Y: 200, Active: true, Index: 2},  // 距≈212 ✗
			{X: 50, Y: 240, Active: true, Index: 3},   // 距=190 ✓
			{X: 500, Y: 500, Active: true, Index: 4},   // 远 ✗
		},
	}

	sel := descriptor.AllInRangeSelector{}
	ctx := baseSelectorCtx()
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets within tower range, got %d", len(targets))
	}
}

// ── NearbyAlliesSelector ───────────────────────────────

func TestNearbyAlliesSelector(t *testing.T) {
	// 塔在 (50,50)，4 个友方塔，2 个在半径 100 内
	towers := &mockTowerQuerier{
		towers: []descriptor.TowerRef{
			{X: 60, Y: 60, Active: true, Index: 0},   // 距≈14 ✓
			{X: 100, Y: 50, Active: true, Index: 1},   // 距=50 ✓
			{X: 300, Y: 300, Active: true, Index: 2},  // 远 ✗
			{X: 400, Y: 400, Active: true, Index: 3},  // 远 ✗
		},
	}

	sel := descriptor.NearbyAlliesSelector{Radius: 100}
	ctx := baseSelectorCtx()
	ctx.Towers = towers

	targets := sel.Select(ctx)
	if len(targets) != 2 {
		t.Fatalf("expected 2 nearby allies, got %d", len(targets))
	}
	for _, tgt := range targets {
		if tgt.TowerIdx < 0 {
			t.Error("ally target should have valid TowerIdx")
		}
		if tgt.EnemyIdx != -1 {
			t.Error("ally target should have EnemyIdx=-1")
		}
	}
}

func TestNearbyAlliesSelectorExcludesInactive(t *testing.T) {
	towers := &mockTowerQuerier{
		towers: []descriptor.TowerRef{
			{X: 55, Y: 55, Active: false, Index: 0}, // 近但不活跃
			{X: 60, Y: 60, Active: true, Index: 1},  // 近且活跃
		},
	}

	sel := descriptor.NearbyAlliesSelector{Radius: 100}
	ctx := baseSelectorCtx()
	ctx.Towers = towers

	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Fatalf("expected 1 active ally, got %d", len(targets))
	}
	if targets[0].TowerIdx != 1 {
		t.Errorf("expected TowerIdx=1, got %d", targets[0].TowerIdx)
	}
}

// ── SelfTowerSelector ──────────────────────────────────

func TestSelfTowerSelector(t *testing.T) {
	sel := descriptor.SelfTowerSelector{}
	ctx := baseSelectorCtx()

	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	tgt := targets[0]
	if tgt.EnemyIdx != -1 {
		t.Errorf("expected EnemyIdx=-1, got %d", tgt.EnemyIdx)
	}
	// TowerIdx 用 -1 也可以接受（self 是特殊标记），主要看坐标
	if tgt.X != ctx.TowerX || tgt.Y != ctx.TowerY {
		t.Errorf("expected position (%v,%v), got (%v,%v)",
			ctx.TowerX, ctx.TowerY, tgt.X, tgt.Y)
	}
}

// ── ChainSelector ──────────────────────────────────────

func TestChainSelectorBasic(t *testing.T) {
	// 5 个敌人沿 X 轴排列，间距 30
	// 起始命中 index=0 位于 (100,100)
	// maxBounce=2 → 应该弹到 2 个邻近敌人
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0}, // 命中点
			{X: 130, Y: 100, Active: true, Index: 1}, // 距0=30 ✓
			{X: 160, Y: 100, Active: true, Index: 2}, // 距1=30 ✓
			{X: 190, Y: 100, Active: true, Index: 3}, // 距2=30 (超出 bounce)
			{X: 220, Y: 100, Active: true, Index: 4}, // 更远
		},
	}

	sel := descriptor.ChainSelector{
		MaxBounce:  descriptor.FixedScaler{Value: 2},
		ChainRange: 50,
		DecayRatio: 0.7,
	}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 0
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 2 {
		t.Fatalf("expected 2 chain targets, got %d", len(targets))
	}
	// 第一个弹到应该是离命中点最近的（index=1）
	if targets[0].EnemyIdx != 1 {
		t.Errorf("first chain target expected index=1, got %d", targets[0].EnemyIdx)
	}
	// 第二个应该是 index=2
	if targets[1].EnemyIdx != 2 {
		t.Errorf("second chain target expected index=2, got %d", targets[1].EnemyIdx)
	}
}

func TestChainSelectorNoDuplicates(t *testing.T) {
	// 3 个敌人形成三角，确保不会重复弹到同一个
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0}, // 起点
			{X: 120, Y: 100, Active: true, Index: 1}, // 距0=20
			{X: 110, Y: 115, Active: true, Index: 2}, // 距0≈18, 距1≈18
		},
	}

	sel := descriptor.ChainSelector{
		MaxBounce:  descriptor.FixedScaler{Value: 5}, // 多余的 bounce，但只有 2 个可用目标
		ChainRange: 50,
		DecayRatio: 0.8,
	}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 0
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 2 {
		t.Fatalf("expected 2 unique chain targets (excl origin), got %d", len(targets))
	}
	// 确认无重复
	seen := make(map[int]bool)
	for _, tgt := range targets {
		if seen[tgt.EnemyIdx] {
			t.Errorf("duplicate chain target: EnemyIdx=%d", tgt.EnemyIdx)
		}
		seen[tgt.EnemyIdx] = true
		if tgt.EnemyIdx == 0 {
			t.Error("chain should not include the origin enemy")
		}
	}
}

func TestChainSelectorScalesWithStrength(t *testing.T) {
	// maxBounce = LinearScaler{Base: 1, Potential: 2}
	// str=0 → 1 bounce, str=200 → 1+2*(200/100)=5 bounces (capped by available enemies=3)
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0},
			{X: 120, Y: 100, Active: true, Index: 1},
			{X: 140, Y: 100, Active: true, Index: 2},
			{X: 160, Y: 100, Active: true, Index: 3},
		},
	}

	sel := descriptor.ChainSelector{
		MaxBounce:  descriptor.LinearScaler{Base: 1, Potential: 2},
		ChainRange: 50,
		DecayRatio: 0.7,
	}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 0
	ctx.Enemies = enemies

	// str=0: base=1 + 2*(0/100) = 1 bounce
	ctx.Strength = 0
	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Errorf("str=0: expected 1 bounce, got %d", len(targets))
	}

	// str=200: base=1 + 2*(200/100) = 5，但只有 3 个可弹目标
	ctx.Strength = 200
	targets = sel.Select(ctx)
	if len(targets) != 3 {
		t.Errorf("str=200: expected 3 bounces, got %d", len(targets))
	}
}

func TestChainSelectorBreaksWhenNoNearby(t *testing.T) {
	// 2 个敌人，第二个太远无法继续弹射
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0},
			{X: 130, Y: 100, Active: true, Index: 1}, // 距0=30 ✓
			{X: 500, Y: 500, Active: true, Index: 2}, // 距1≈495 ✗
		},
	}

	sel := descriptor.ChainSelector{
		MaxBounce:  descriptor.FixedScaler{Value: 5},
		ChainRange: 50,
		DecayRatio: 0.8,
	}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 0
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 1 {
		t.Fatalf("expected 1 chain target (chain broken by distance), got %d", len(targets))
	}
}

// ── 边界条件 ───────────────────────────────────────────

func TestAoeRadiusSelectorZeroRadius(t *testing.T) {
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0},
		},
	}
	sel := descriptor.AoeRadiusSelector{
		Radius: descriptor.FixedScaler{Value: 0},
	}
	ctx := baseSelectorCtx()
	ctx.Enemies = enemies

	// 半径 0 时，只有恰好在命中点上的敌人才能被选中
	targets := sel.Select(ctx)
	// 敌人在 (100,100)，命中点也在 (100,100)，距离=0 ≤ 0
	if len(targets) != 1 {
		t.Errorf("expected 1 target at exact hit point, got %d", len(targets))
	}
}

func TestChainSelectorMaxBounceZero(t *testing.T) {
	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{X: 100, Y: 100, Active: true, Index: 0},
			{X: 110, Y: 100, Active: true, Index: 1},
		},
	}
	sel := descriptor.ChainSelector{
		MaxBounce:  descriptor.FixedScaler{Value: 0},
		ChainRange: 50,
		DecayRatio: 0.8,
	}
	ctx := baseSelectorCtx()
	ctx.CurrentEnemyIdx = 0
	ctx.Enemies = enemies

	targets := sel.Select(ctx)
	if len(targets) != 0 {
		t.Errorf("maxBounce=0: expected 0 targets, got %d", len(targets))
	}
}

