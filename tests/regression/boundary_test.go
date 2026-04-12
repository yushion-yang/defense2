// boundary_test.go — 边界条件回归测试。
// 验证极端/边缘情况下系统不会 panic，且行为符合预期。
package regression_test

import (
	"math"
	"testing"

	"defense2/internal/core/buff"
	"defense2/internal/core/combat"
	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
	"defense2/internal/core/gamemap"
)

// ═══════════════════════════════════════
// 1. 对象池满：Spawn 256 个敌人后再 Spawn 应返回 nil
// ═══════════════════════════════════════

func TestBoundary_PoolFull_ReturnsNil(t *testing.T) {
	pool := enemy.DefaultPool() // cap = MaxEnemies = 256

	for i := 0; i < game.MaxEnemies; i++ {
		e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
		if e == nil {
			t.Fatalf("第 %d 次 Spawn 返回 nil，预期应成功（池未满）", i+1)
		}
	}
	if pool.Count != game.MaxEnemies {
		t.Errorf("池满后 Count=%d, want %d", pool.Count, game.MaxEnemies)
	}

	// 第 257 次应返回 nil，不 panic
	e := pool.Spawn(0, 0, 100, 60, 1, "overflow", nil)
	if e != nil {
		t.Error("池满后 Spawn 应返回 nil")
	}
}

func TestBoundary_PoolFull_CountStable(t *testing.T) {
	pool := enemy.NewPool(4) // 小池便于测试

	for i := 0; i < 4; i++ {
		pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	}
	if pool.Count != 4 {
		t.Fatalf("Count=%d, want 4", pool.Count)
	}

	// 溢出 Spawn 不应改变 Count
	pool.Spawn(0, 0, 100, 60, 1, "overflow", nil)
	if pool.Count != 4 {
		t.Errorf("溢出后 Count=%d, want 4", pool.Count)
	}
}

// ═══════════════════════════════════════
// 2. 空池操作：TickBehaviors / MoveAlongPath 在空池上不 panic
// ═══════════════════════════════════════

func TestBoundary_EmptyPool_TickBehaviors(t *testing.T) {
	pool := enemy.NewPool(16)
	// 空池调用 TickBehaviors — 不应 panic
	events := enemy.TickBehaviors(pool, 1.0/60.0)
	if events.Berserks != 0 || len(events.Regens) != 0 {
		t.Error("空池不应产生任何事件")
	}
}

func TestBoundary_EmptyPool_Each(t *testing.T) {
	pool := enemy.NewPool(16)
	count := 0
	pool.Each(func(e *enemy.Enemy) {
		count++
	})
	if count != 0 {
		t.Errorf("空池 Each 回调执行了 %d 次, want 0", count)
	}
}

func TestBoundary_EmptyPool_MoveAlongPath(t *testing.T) {
	// MoveAlongPath on a non-active enemy should be safe
	e := &enemy.Enemy{Active: true, Speed: 60, BaseSpeed: 60, PathIndex: 1}
	e.Buffs = buff.NewDefaultBuffList()
	waypoints := []gamemap.Point{
		{X: 0, Y: 0}, {X: 100, Y: 0},
	}
	// Should not panic
	enemy.MoveAlongPath(e, waypoints, 1.0/60.0)
}

// ═══════════════════════════════════════
// 3. 迭代中击杀：Pool.Each + Kill 不 panic
// ═══════════════════════════════════════

func TestBoundary_KillDuringIteration(t *testing.T) {
	pool := enemy.NewPool(16)
	for i := 0; i < 8; i++ {
		e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
		if e != nil {
			e.SpawnTimer = 0 // 跳过出生动画
		}
	}
	if pool.Count != 8 {
		t.Fatalf("Count=%d, want 8", pool.Count)
	}

	// 在 Each 中杀死偶数 ID 的敌人 — 不应 panic
	pool.Each(func(e *enemy.Enemy) {
		if e.ID%2 == 0 {
			pool.Kill(e)
		}
	})

	// Kill 减少 Count（进入 dying 状态）
	if pool.Count >= 8 {
		t.Errorf("杀死部分敌人后 Count=%d, 应小于 8", pool.Count)
	}
}

// ═══════════════════════════════════════
// 4. 极端伤害：ApplyDamage(MaxFloat64) 不 panic，HP 钳位到 0
// ═══════════════════════════════════════

func TestBoundary_ExtremeDamage_MaxFloat64(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 1, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0

	result := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: math.MaxFloat64,
	})
	if !result.Killed {
		t.Error("极端伤害应击杀目标")
	}
	if e.HP != 0 {
		t.Errorf("HP 应为 0, got %f", e.HP)
	}
	if result.FinalDamage < 0 {
		t.Error("FinalDamage 不应为负")
	}
}

func TestBoundary_ExtremeDamage_VeryLargeHP(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 1e15, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0

	result := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 1e15 + 1,
	})
	if !result.Killed {
		t.Error("伤害超过 HP 应击杀")
	}
	if e.HP != 0 {
		t.Errorf("HP 应被钳位到 0, got %f", e.HP)
	}
}

// ═══════════════════════════════════════
// 5. 零持续时间 Stun/Slow：不应 panic，不应添加 buff
// ═══════════════════════════════════════

func TestBoundary_ZeroDurationStun(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0

	// 韧性为 0 时，duration=0 → actualDuration=0 → 不应添加 buff
	applied := combat.ApplyStun(e, 0, "test")
	if applied {
		t.Error("零持续时间的 Stun 不应被成功施加")
	}
	if e.IsStunned() {
		t.Error("敌人不应处于眩晕状态")
	}
}

func TestBoundary_ZeroDurationSlow(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0

	// duration=0 → actualDuration=0 → 不应添加 buff
	applied := combat.ApplySlow(e, 0.5, 0, "test")
	if applied {
		t.Error("零持续时间的 Slow 不应被成功施加")
	}
	if e.IsSlowed() {
		t.Error("敌人不应处于减速状态")
	}
}

// ═══════════════════════════════════════
// 6. 负数伤害：ApplyDamage 应安全处理
// ═══════════════════════════════════════

func TestBoundary_NegativeDamage(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0

	result := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: -50,
	})

	// 负数伤害不应击杀
	if result.Killed {
		t.Error("负数伤害不应击杀敌人")
	}
	// 不应 panic — 这是核心断言
	// 注意：当前管线不钳位负数伤害（e.HP -= negative = HP增加），
	// 这是已知行为，调用方不应传入负数伤害。
	if e.HP < 0 {
		t.Errorf("HP=%f 不应为负", e.HP)
	}
}

func TestBoundary_ZeroDamage(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0
	hpBefore := e.HP

	result := combat.ApplyDamage(combat.DamageInput{
		Target:    e,
		RawDamage: 0,
	})

	if result.Killed {
		t.Error("零伤害不应击杀")
	}
	// 零伤害的保底规则：rawDamage=0 不触发保底1，HP 应不变
	if e.HP != hpBefore {
		t.Errorf("零伤害后 HP=%f, want %f", e.HP, hpBefore)
	}
}

// ═══════════════════════════════════════
// 7. Nil Buffs：IsStunned 等查询方法应返回 false
// ═══════════════════════════════════════

func TestBoundary_NilBuffs_QueryMethods(t *testing.T) {
	e := &enemy.Enemy{
		Active: true,
		HP:     100,
		MaxHP:  100,
		Buffs:  nil, // 故意设为 nil
	}

	cases := []struct {
		name string
		fn   func() bool
	}{
		{"IsStunned", e.IsStunned},
		{"IsSlowed", e.IsSlowed},
		{"IsRooted", e.IsRooted},
		{"IsBleeding", e.IsBleeding},
		{"IsBurning", e.IsBurning},
		{"IsPoisoned", e.IsPoisoned},
		{"IsWeakened", e.IsWeakened},
		{"HasControlImmunity", e.HasControlImmunity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 不应 panic，且应返回 false
			if tc.fn() {
				t.Errorf("%s 在 Buffs=nil 时应返回 false", tc.name)
			}
		})
	}
}

func TestBoundary_NilBuffs_GetSlowFactor(t *testing.T) {
	e := &enemy.Enemy{Buffs: nil}
	factor := e.GetSlowFactor()
	if factor != 1.0 {
		t.Errorf("GetSlowFactor()=%f, want 1.0", factor)
	}
}

func TestBoundary_NilBuffs_GetWeakenAmplify(t *testing.T) {
	e := &enemy.Enemy{Buffs: nil}
	amp := e.GetWeakenAmplify()
	if amp != 0 {
		t.Errorf("GetWeakenAmplify()=%f, want 0", amp)
	}
}

// ═══════════════════════════════════════
// 8. 空路径：MoveAlongPath 应安全处理
// ═══════════════════════════════════════

func TestBoundary_EmptyWaypoints_NoFallback(t *testing.T) {
	e := &enemy.Enemy{
		Active:    true,
		Speed:     60,
		BaseSpeed: 60,
		PathIndex: 0,
		Buffs:     buff.NewDefaultBuffList(),
	}

	// 空路径 + 空 fallback — 应直接返回 true（到达终点）
	reached := enemy.MoveAlongPath(e, nil, 1.0/60.0)
	if !reached {
		t.Error("空路径应视为到达终点（PathIndex >= len(waypoints)）")
	}
}

func TestBoundary_EmptyWaypoints_WithFallback(t *testing.T) {
	e := &enemy.Enemy{
		Active:    true,
		Speed:     60,
		BaseSpeed: 60,
		PathIndex: 0,
		Buffs:     buff.NewDefaultBuffList(),
	}

	// 空 path + 空 fallback
	reached := enemy.MoveAlongPath(e, []gamemap.Point{}, 1.0/60.0)
	if !reached {
		t.Error("空 fallback 路径应视为到达终点")
	}
}

func TestBoundary_PathIndex_BeyondEnd(t *testing.T) {
	e := &enemy.Enemy{
		Active:    true,
		Speed:     60,
		BaseSpeed: 60,
		PathIndex: 999, // 远超路径长度
		Buffs:     buff.NewDefaultBuffList(),
	}
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}

	reached := enemy.MoveAlongPath(e, waypoints, 1.0/60.0)
	if !reached {
		t.Error("PathIndex 超过路径长度时应视为到达终点")
	}
}

// ═══════════════════════════════════════
// 9. 补充边界：Dummy 不移动
// ═══════════════════════════════════════

func TestBoundary_DummyEnemy_DoesNotMove(t *testing.T) {
	e := &enemy.Enemy{
		Active:    true,
		IsDummy:   true,
		Speed:     0,
		BaseSpeed: 0,
		PathIndex: 1,
		Buffs:     buff.NewDefaultBuffList(),
	}
	waypoints := []gamemap.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}

	reached := enemy.MoveAlongPath(e, waypoints, 1.0/60.0)
	if reached {
		t.Error("Dummy 敌人不应到达终点")
	}
}

// ═══════════════════════════════════════
// 10. Pool Kill 后重用：Kill 后 Spawn 应复用槽位
// ═══════════════════════════════════════

func TestBoundary_PoolReuse_AfterKillImmediate(t *testing.T) {
	pool := enemy.NewPool(2)

	e1 := pool.Spawn(0, 0, 100, 60, 1, "a", nil)
	e2 := pool.Spawn(0, 0, 100, 60, 1, "b", nil)
	if e1 == nil || e2 == nil {
		t.Fatal("初始 Spawn 失败")
	}

	// 池满
	if pool.Spawn(0, 0, 100, 60, 1, "c", nil) != nil {
		t.Fatal("2 容量的池第 3 次 Spawn 应返回 nil")
	}

	// KillImmediate 立即释放槽位
	pool.KillImmediate(e1)
	if pool.Count != 1 {
		t.Errorf("KillImmediate 后 Count=%d, want 1", pool.Count)
	}

	// 应该能再次 Spawn
	e3 := pool.Spawn(0, 0, 200, 30, 1, "reused", nil)
	if e3 == nil {
		t.Error("KillImmediate 释放槽位后 Spawn 应成功")
	}
	if pool.Count != 2 {
		t.Errorf("重用后 Count=%d, want 2", pool.Count)
	}
}

// ═══════════════════════════════════════
// 11. 高韧性 CC：韧性 100% 时 CC 无效
// ═══════════════════════════════════════

func TestBoundary_FullTenacity_StunFails(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0
	e.Tenacity = 1.0 // 100% 韧性

	applied := combat.ApplyStun(e, 2.0, "test")
	if applied {
		t.Error("100% 韧性时 Stun 不应成功")
	}
	if e.IsStunned() {
		t.Error("敌人不应处于眩晕状态")
	}
}

func TestBoundary_FullTenacity_SlowFails(t *testing.T) {
	pool := enemy.NewPool(4)
	e := pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	if e == nil {
		t.Fatal("Spawn 失败")
	}
	e.SpawnTimer = 0
	e.Tenacity = 1.0

	applied := combat.ApplySlow(e, 0.5, 2.0, "test")
	if applied {
		t.Error("100% 韧性时 Slow 不应成功")
	}
}

// ═══════════════════════════════════════
// 12. ClearAll 后池应完全空
// ═══════════════════════════════════════

func TestBoundary_ClearAll(t *testing.T) {
	pool := enemy.NewPool(16)
	for i := 0; i < 8; i++ {
		pool.Spawn(0, 0, 100, 60, 1, "normal", nil)
	}
	if pool.Count != 8 {
		t.Fatalf("Count=%d, want 8", pool.Count)
	}

	pool.ClearAll()
	if pool.Count != 0 {
		t.Errorf("ClearAll 后 Count=%d, want 0", pool.Count)
	}

	// ClearAll 后应能重新 Spawn
	e := pool.Spawn(0, 0, 100, 60, 1, "fresh", nil)
	if e == nil {
		t.Error("ClearAll 后 Spawn 应成功")
	}
}
