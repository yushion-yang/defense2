package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/gamemap"
)

// ── PushBack: 基本回推 ──────────────────────────────────────

func TestPushBack_BasicRewind(t *testing.T) {
	// 路径: (0,0) → (100,0) → (200,0)
	// 敌人在 (80,0)，PathIndex=1（朝 waypoints[1] 前进）
	// 回推 30 像素 → 应到 (50,0)
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 200, Y: 0}}
	e := &enemy.Enemy{
		X:         80,
		Y:         0,
		PathIndex: 1,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, 30)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50", e.X)
	}
	if math.Abs(e.Y-0) > 0.01 {
		t.Errorf("Y = %f, want 0", e.Y)
	}
	if e.PathIndex != 1 {
		t.Errorf("PathIndex = %d, want 1", e.PathIndex)
	}
}

func TestPushBack_CrossSegment(t *testing.T) {
	// 路径: (0,0) → (100,0) → (200,0)
	// 敌人在 (120,0)，PathIndex=2（已过 waypoints[1]，朝 waypoints[2]）
	// 回推 50 → 应过路径点 (100,0) 并继续回退到 (70,0)，PathIndex=1
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 200, Y: 0}}
	e := &enemy.Enemy{
		X:         120,
		Y:         0,
		PathIndex: 2,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, 50)

	// 从 (120,0) 回退 20 到 (100,0)=waypoints[1]，再回退 30 到 (70,0)
	if math.Abs(e.X-70) > 0.01 {
		t.Errorf("X = %f, want 70", e.X)
	}
	if e.PathIndex != 1 {
		t.Errorf("PathIndex = %d, want 1", e.PathIndex)
	}
}

func TestPushBack_ClampAtStart(t *testing.T) {
	// 路径: (0,0) → (100,0)
	// 敌人在 (20,0)，PathIndex=1
	// 回推 200 → 应停在起点附近
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	e := &enemy.Enemy{
		X:         20,
		Y:         0,
		PathIndex: 1,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, 200)

	if math.Abs(e.X-0) > 0.01 {
		t.Errorf("X = %f, want 0 (clamped to start)", e.X)
	}
	if math.Abs(e.Y-0) > 0.01 {
		t.Errorf("Y = %f, want 0", e.Y)
	}
}

func TestPushBack_ZeroDistance(t *testing.T) {
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	e := &enemy.Enemy{
		X:         50,
		Y:         0,
		PathIndex: 1,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, 0)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50 (unchanged)", e.X)
	}
}

func TestPushBack_NegativeDistance(t *testing.T) {
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	e := &enemy.Enemy{
		X:         50,
		Y:         0,
		PathIndex: 1,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, -10)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50 (unchanged for negative dist)", e.X)
	}
}

func TestPushBack_DummyEnemy(t *testing.T) {
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	e := &enemy.Enemy{
		X:         50,
		Y:         0,
		PathIndex: 1,
		Path:      waypoints,
		IsDummy:   true,
	}

	enemy.PushBack(e, nil, 30)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50 (dummy should not move)", e.X)
	}
}

func TestPushBack_NoPath(t *testing.T) {
	// 无路径 + 无 fallback → 应安全退出不 panic
	e := &enemy.Enemy{
		X:         50,
		Y:         0,
		PathIndex: 1,
	}

	enemy.PushBack(e, nil, 30)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50 (no path, no-op)", e.X)
	}
}

func TestPushBack_UsesFallbackWaypoints(t *testing.T) {
	// e.Path 为空，使用 fallback
	fallback := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 200, Y: 0}}
	e := &enemy.Enemy{
		X:         80,
		Y:         0,
		PathIndex: 1,
	}

	enemy.PushBack(e, fallback, 30)

	if math.Abs(e.X-50) > 0.01 {
		t.Errorf("X = %f, want 50", e.X)
	}
}

func TestPushBack_DiagonalPath(t *testing.T) {
	// 对角线路径: (0,0) → (100,100)
	// 敌人在 (70.71, 70.71)（约距起点 100 像素），PathIndex=1
	// 回推 50 → 应沿对角线回退
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 100}}
	startDist := 100.0 * (math.Sqrt(2) / 2) // ~70.71
	e := &enemy.Enemy{
		X:         startDist,
		Y:         startDist,
		PathIndex: 1,
		Path:      waypoints,
	}

	enemy.PushBack(e, nil, 50)

	// 回退 50 后，离起点距离应约为 50（从 ~70.71*sqrt(2)=100 回退 50 变 50）
	distFromStart := math.Hypot(e.X, e.Y)
	expected := math.Hypot(startDist, startDist) - 50 // 100 - 50 = 50
	if math.Abs(distFromStart-expected) > 0.1 {
		t.Errorf("distance from start = %f, want ~%f", distFromStart, expected)
	}
}
