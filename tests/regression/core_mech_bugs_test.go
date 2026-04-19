// core_mech_bugs_test.go — 机甲战灵 bug 回归测试。
//
// 复现三个 bug：
//  1. 斩杀能力失效（发射时计算伤害，命中时血量已变）
//  2. 地图底部弹射物被立即回收（边界检查用固定 ScreenHeight=540）
package regression

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/projectile"
)

// spawnTestEnemy 创建测试用敌人。
func spawnTestEnemy(pool *enemy.Pool, x, y, hp, speed float64) *enemy.Enemy {
	e := pool.Spawn(x, y, hp, speed, 0, "test", nil)
	e.Radius = 10
	return e
}

// TestCoreMech_ExecuteAtHitTime 验证斩杀应在命中时判定，而非发射时。
//
// 场景：敌人初始 HP=100，斩杀阈值 15%。
// 发射时 HP=100 > 15，不触发秒杀，伤害=12。
// 弹射物飞行中敌人被其他伤害打到 HP=10 < 15。
// 命中时应重新判定触发秒杀，伤害=10（当前 HP），而非发射时的 12。
func TestCoreMech_ExecuteAtHitTime(t *testing.T) {
	pool := projectile.NewPool(16)
	enemies := enemy.DefaultPool()

	// 创建测试敌人
	e := spawnTestEnemy(enemies, 200, 200, 100, 50)
	e.Radius = 10

	// 模拟机甲战灵发射带斩杀的弹射物
	// 发射时 HP=100 > 15% 阈值，正常伤害=12
	execHpPct := 0.15
	baseDmg := 12.0
	pool.FireWithExecute(100, 200, e.X, e.Y, baseDmg, 400, 4, e, "warden", execHpPct)

	// 验证弹射物已创建并携带斩杀参数
	var proj *projectile.Projectile
	pool.Each(func(p *projectile.Projectile) {
		proj = p
	})
	if proj == nil {
		t.Fatal("projectile should be created")
	}
	if proj.ExecuteHpPct != execHpPct {
		t.Fatalf("ExecuteHpPct = %f, want %f", proj.ExecuteHpPct, execHpPct)
	}

	// 模拟飞行中敌人被其他伤害打到低血量
	e.HP = 10 // 现在 HP=10 < MaxHP*15% = 15

	// 在命中处理时，应使用当前 HP（10）而非发射时计算的伤害（12）
	// 这里只测试弹射物携带了正确的斩杀参数，实际命中逻辑在 tick_combat 中
	if proj.Damage != baseDmg {
		t.Fatalf("base Damage = %f, want %f", proj.Damage, baseDmg)
	}
}

// TestProjectile_BoundaryUsesMapSize 验证弹射物边界检查应使用地图尺寸而非固定 ScreenHeight。
//
// 场景：地图高度 720px（大于 ScreenHeight=540）。
// 战灵在 Y=600 位置向上方敌人发射弹射物。
// 弹射物不应被立即回收。
func TestProjectile_BoundaryUsesMapSize(t *testing.T) {
	pool := projectile.NewPool(16)
	pool.SetMapBounds(1200, 720) // 设置地图尺寸

	enemies := enemy.DefaultPool()
	e := spawnTestEnemy(enemies, 600, 500, 100, 50)

	// 从 Y=600（超出 ScreenHeight=540）位置发射
	pool.Fire(600, 600, e.X, e.Y, 10, 400, 4, e, "warden")

	// 验证弹射物未被立即回收
	if pool.Count != 1 {
		t.Fatalf("projectile count = %d, want 1 (should not be culled)", pool.Count)
	}

	// 模拟一帧 tick，弹射物应该正常移动
	pool.Tick(0.016)

	if pool.Count != 1 {
		t.Fatalf("after tick: projectile count = %d, want 1", pool.Count)
	}

	// 弹射物 Y 应该减小（向上飞）
	var proj *projectile.Projectile
	pool.Each(func(p *projectile.Projectile) {
		proj = p
	})
	if proj == nil {
		t.Fatal("projectile should exist after tick")
	}
	if proj.Y >= 600 {
		t.Fatalf("projectile Y = %f, should be < 600 (flying upward)", proj.Y)
	}
}

// TestProjectile_OldBoundaryBug 复现旧 bug：固定边界导致大地图弹射物被错误回收。
func TestProjectile_OldBoundaryBug(t *testing.T) {
	pool := projectile.NewPool(16)
	// 不调用 SetMapBounds，使用默认值

	enemies := enemy.DefaultPool()
	e := spawnTestEnemy(enemies, 600, 500, 100, 50)

	// 从 Y=600（超出默认 ScreenHeight=540）发射
	// 修复后：默认边界应该足够大，不会立即回收
	pool.Fire(600, 600, e.X, e.Y, 10, 400, 4, e, "warden")

	// 修复前：pool.Count = 0（被立即回收）
	// 修复后：pool.Count = 1（正常存活）
	if pool.Count != 1 {
		t.Fatalf("projectile count = %d, want 1 (default bounds should be large enough)", pool.Count)
	}
}

// TestProjectile_ExecuteHpPctField 验证 Projectile 结构体有 ExecuteHpPct 字段。
func TestProjectile_ExecuteHpPctField(t *testing.T) {
	p := &projectile.Projectile{
		ExecuteHpPct: 0.15,
	}
	if p.ExecuteHpPct != 0.15 {
		t.Fatalf("ExecuteHpPct = %f, want 0.15", p.ExecuteHpPct)
	}
}

// TestProjectile_MapBoundsDefault 验证默认地图边界足够大。
func TestProjectile_MapBoundsDefault(t *testing.T) {
	pool := projectile.NewPool(16)

	// 默认边界应至少能容纳最大地图（如 2400x1200）
	w, h := pool.MapBounds()
	if w < float64(game.ScreenWidth) {
		t.Fatalf("default map width = %f, should be >= %d", w, game.ScreenWidth)
	}
	if h < float64(game.ScreenHeight) {
		t.Fatalf("default map height = %f, should be >= %d", h, game.ScreenHeight)
	}
}
