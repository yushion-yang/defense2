// descriptor_ability_test.go — DescriptorAbility 包装器测试。
//
// 验证 descriptor 驱动的能力正确实现 tower.Ability/Ticker 接口，
// 并能通过 OnHit/OnTick 产出预期的 HitResult/TickResult。
//
// mock EnemyQuerier/TowerQuerier 已在 descriptor_selector_test.go 中定义，
// 本文件直接复用（同属 core_test 包）。
package core_test

import (
	"testing"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// ── 接口合规性（编译时检查）──────────────────────────────────

var _ tower.Ability = (*descriptor.DescriptorAbilityHit)(nil)
var _ tower.Ability = (*descriptor.DescriptorAbilityFull)(nil)
var _ tower.Ticker = (*descriptor.DescriptorAbilityFull)(nil)

// ── 辅助：创建最小化测试对象 ─────────────────────────────────

// newTestTower 创建用于测试的最小化塔实例。
func newTestTower() *tower.Tower {
	return &tower.Tower{
		X:        100,
		Y:        100,
		Range:    200,
		Damage:   50,
		Active:   true,
		Strength: strength.NewStrengthData(),
	}
}

// newTestEnemy 创建用于测试的最小化敌人实例。
func newTestEnemy() *enemy.Enemy {
	return &enemy.Enemy{
		Active: true,
		X:      120,
		Y:      130,
		HP:     80,
		MaxHP:  100,
	}
}

// newTestProjectile 创建用于测试的最小化弹射物实例。
func newTestProjectile() *projectile.Projectile {
	return &projectile.Projectile{
		Active: true,
		Damage: 50,
		X:      120,
		Y:      130,
	}
}

// ── 测试用例 ──────────────────────────────────────────────────

// TestDescriptorAbility_Name 验证 Name() 返回描述符 ID。
func TestDescriptorAbility_Name(t *testing.T) {
	data := []byte(`{
		"id": "testStun",
		"label": "Test Stun",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)
	if ability.Name() != "testStun" {
		t.Errorf("Name() = %q, want %q", ability.Name(), "testStun")
	}
}

// TestDescriptorAbility_HitOnlyNotTicker 验证仅 onHit 的描述符不实现 Ticker。
func TestDescriptorAbility_HitOnlyNotTicker(t *testing.T) {
	data := []byte(`{
		"id": "stunOnly",
		"label": "Stun Only",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 1.0}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	if _, ok := ability.(tower.Ticker); ok {
		t.Error("hit-only descriptor should NOT implement Ticker, but it does")
	}
}

// TestDescriptorAbility_FullIsTicker 验证含 onTick 的描述符实现 Ticker。
func TestDescriptorAbility_FullIsTicker(t *testing.T) {
	data := []byte(`{
		"id": "goldPassive",
		"label": "Gold Passive",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [
					{"type": "cooldown", "seconds": 5}
				],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 10}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	if _, ok := ability.(tower.Ticker); !ok {
		t.Error("full descriptor (with onTick) should implement Ticker, but it does not")
	}
}

// TestDescriptorAbility_OnHitProducesStun 验证 OnHit 产出包含 Stun 的 HitResult。
func TestDescriptorAbility_OnHitProducesStun(t *testing.T) {
	data := []byte(`{
		"id": "stunChance",
		"label": "Stun Chance",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 1.0}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	tw := newTestTower()
	en := newTestEnemy()
	pr := newTestProjectile()

	hr := ability.OnHit(tw, pr, en)
	if hr == nil {
		t.Fatal("OnHit returned nil, want non-nil HitResult with Stun")
	}
	if hr.Stun == nil {
		t.Fatal("HitResult.Stun is nil, want non-nil StunEffect")
	}
	if hr.Stun.Duration != 0.5 {
		t.Errorf("Stun.Duration = %v, want 0.5", hr.Stun.Duration)
	}
}

// TestDescriptorAbility_OnHitNilForZeroChance 验证 chance=0 时 OnHit 返回 nil。
func TestDescriptorAbility_OnHitNilForZeroChance(t *testing.T) {
	data := []byte(`{
		"id": "neverStun",
		"label": "Never Stun",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 0.0}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	tw := newTestTower()
	en := newTestEnemy()
	pr := newTestProjectile()

	hr := ability.OnHit(tw, pr, en)
	if hr != nil {
		t.Errorf("OnHit returned non-nil HitResult for chance=0, want nil")
	}
}

// TestDescriptorAbility_OnTickProducesGold 验证 OnTick 通过 cooldown 后产出 GoldEarned。
// CooldownCondition 行为：首次调用无条件通过，之后累计 DT 直到达到冷却时间。
func TestDescriptorAbility_OnTickProducesGold(t *testing.T) {
	data := []byte(`{
		"id": "goldPassive",
		"label": "Gold Passive",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [
					{"type": "cooldown", "seconds": 1.0}
				],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 10}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	ticker, ok := ability.(tower.Ticker)
	if !ok {
		t.Fatal("goldPassive should implement Ticker")
	}

	tw := newTestTower()
	ctx := &tower.TickContext{DT: 0.5}

	// 第 1 tick: CooldownCondition 首次调用无条件通过 → 产金
	tr := ticker.OnTick(tw, ctx)
	if tr == nil {
		t.Fatal("tick 1: cooldown first call should pass unconditionally")
	}
	if tr.GoldEarned != 10 {
		t.Errorf("tick 1 GoldEarned = %d, want 10", tr.GoldEarned)
	}

	// 第 2 tick: 累计 0.5 < 1.0 → nil
	tr = ticker.OnTick(tw, ctx)
	if tr != nil {
		t.Errorf("tick 2 (cumulated=0.5): got non-nil TickResult, want nil (cooldown not reached)")
	}

	// 第 3 tick: 累计 1.0 >= 1.0 → 产金
	tr = ticker.OnTick(tw, ctx)
	if tr == nil {
		t.Fatal("tick 3 (cumulated=1.0): got nil TickResult, want non-nil with GoldEarned")
	}
	if tr.GoldEarned != 10 {
		t.Errorf("GoldEarned = %d, want 10", tr.GoldEarned)
	}
}

// TestDescriptorAbility_OnHitWithNilEnemy 验证 OnHit 在 enemy=nil 时不 panic。
func TestDescriptorAbility_OnHitWithNilEnemy(t *testing.T) {
	data := []byte(`{
		"id": "selfDamage",
		"label": "Self Damage",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 10}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	tw := newTestTower()
	pr := newTestProjectile()

	// 不 panic 即通过
	_ = ability.OnHit(tw, pr, nil)
}

// TestDescriptorAbility_DispatchType 验证 NewDescriptorAbility 返回正确的具体类型。
func TestDescriptorAbility_DispatchType(t *testing.T) {
	// 仅 onHit → DescriptorAbilityHit
	hitOnly := []byte(`{
		"id": "hitOnly",
		"label": "Hit Only",
		"cost": 1,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(hitOnly)
	if err != nil {
		t.Fatalf("ParseDescriptor hitOnly: %v", err)
	}
	ability := descriptor.NewDescriptorAbility(desc)
	if _, ok := ability.(*descriptor.DescriptorAbilityHit); !ok {
		t.Errorf("hitOnly: want *DescriptorAbilityHit, got %T", ability)
	}

	// 含 onTick → DescriptorAbilityFull
	withTick := []byte(`{
		"id": "withTick",
		"label": "With Tick",
		"cost": 1,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			},
			{
				"trigger": "onTick",
				"conditions": [{"type": "cooldown", "seconds": 5}],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 5}}
				]
			}
		]
	}`)

	desc2, err := descriptor.ParseDescriptor(withTick)
	if err != nil {
		t.Fatalf("ParseDescriptor withTick: %v", err)
	}
	ability2 := descriptor.NewDescriptorAbility(desc2)
	if _, ok := ability2.(*descriptor.DescriptorAbilityFull); !ok {
		t.Errorf("withTick: want *DescriptorAbilityFull, got %T", ability2)
	}
}

// TestDescriptorAbility_FullOnHit 验证 DescriptorAbilityFull 的 OnHit 也能正常工作。
func TestDescriptorAbility_FullOnHit(t *testing.T) {
	data := []byte(`{
		"id": "fullAbility",
		"label": "Full Ability",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 1.0}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.3}}
				]
			},
			{
				"trigger": "onTick",
				"conditions": [{"type": "cooldown", "seconds": 5}],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)

	tw := newTestTower()
	en := newTestEnemy()
	pr := newTestProjectile()

	hr := ability.OnHit(tw, pr, en)
	if hr == nil {
		t.Fatal("FullAbility OnHit returned nil, want non-nil HitResult with Stun")
	}
	if hr.Stun == nil {
		t.Fatal("HitResult.Stun is nil")
	}
	if hr.Stun.Duration != 0.3 {
		t.Errorf("Stun.Duration = %v, want 0.3", hr.Stun.Duration)
	}
}

// TestDescriptorAbility_CooldownAccumulates 验证 CooldownCondition 正确累计 DT。
func TestDescriptorAbility_CooldownAccumulates(t *testing.T) {
	// cooldown=2.0s, 每 tick dt=0.6s
	// 第 1 tick: 首次无条件通过 → 产金
	// 第 2~4 tick: 累计 0.6, 1.2, 1.8 → 均 < 2.0 → nil
	// 第 5 tick: 累计 2.4 >= 2.0 → 产金
	data := []byte(`{
		"id": "slowGold",
		"label": "Slow Gold",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [{"type": "cooldown", "seconds": 2.0}],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 7}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	ability := descriptor.NewDescriptorAbility(desc)
	ticker := ability.(tower.Ticker)
	tw := newTestTower()
	ctx := &tower.TickContext{DT: 0.6}

	// 第 1 tick: 首次无条件通过
	tr := ticker.OnTick(tw, ctx)
	if tr == nil || tr.GoldEarned != 7 {
		t.Fatalf("tick 1 (first call): want GoldEarned=7, got %v", tr)
	}

	// 第 2~4 tick: 累计 0.6, 1.2, 1.8 → 均 < 2.0 → nil
	for i := 2; i <= 4; i++ {
		tr := ticker.OnTick(tw, ctx)
		if tr != nil {
			t.Errorf("tick %d: got non-nil TickResult, want nil (cooldown not reached)", i)
		}
	}

	// 第 5 tick: 累计 2.4 >= 2.0 → 产金
	tr = ticker.OnTick(tw, ctx)
	if tr == nil {
		t.Fatal("tick 5 (cumulated=2.4): got nil TickResult, want non-nil")
	}
	if tr.GoldEarned != 7 {
		t.Errorf("GoldEarned = %d, want 7", tr.GoldEarned)
	}
}

// clearSpawnTimer 清除出生动画计时器（测试辅助，使敌人可被索敌系统选中）。
func clearSpawnTimer(pool *enemy.Pool) {
	pool.EachActive(func(e *enemy.Enemy) {
		e.SpawnTimer = 0
	})
}

// TestPoolEnemyQuerier_QueryRadius 验证 poolEnemyQuerier 适配器的半径查询。
func TestPoolEnemyQuerier_QueryRadius(t *testing.T) {
	pool := enemy.NewPool(8)
	// 生成 3 个敌人在不同位置
	pool.Spawn(100, 100, 100, 50, 1, "normal", nil) // 距 (100,100)=0
	pool.Spawn(120, 100, 100, 50, 1, "normal", nil) // 距 (100,100)=20
	pool.Spawn(300, 300, 100, 50, 1, "normal", nil) // 距 (100,100)~283
	clearSpawnTimer(pool) // 清除出生动画，否则 querier 跳过 spawning 敌人

	querier := descriptor.NewPoolEnemyQuerier(pool)

	// 半径 50：应包含前 2 个
	refs := querier.QueryRadius(100, 100, 50)
	if len(refs) != 2 {
		t.Errorf("QueryRadius(50): got %d refs, want 2", len(refs))
	}

	// 半径 500：应包含全部 3 个
	refs = querier.QueryRadius(100, 100, 500)
	if len(refs) != 3 {
		t.Errorf("QueryRadius(500): got %d refs, want 3", len(refs))
	}
}

// TestPoolEnemyQuerier_AllActive 验证 poolEnemyQuerier 返回所有活跃敌人。
func TestPoolEnemyQuerier_AllActive(t *testing.T) {
	pool := enemy.NewPool(8)
	pool.Spawn(100, 100, 100, 50, 1, "normal", nil)
	pool.Spawn(200, 200, 100, 50, 1, "runner", nil)
	clearSpawnTimer(pool)

	querier := descriptor.NewPoolEnemyQuerier(pool)
	refs := querier.AllActive()
	if len(refs) != 2 {
		t.Errorf("AllActive: got %d refs, want 2", len(refs))
	}
}

// TestPoolTowerQuerier_QueryRadius 验证 poolTowerQuerier 适配器的半径查询。
func TestPoolTowerQuerier_QueryRadius(t *testing.T) {
	pool := tower.NewPool(8)

	// 手动放置塔（简化，直接设置 Active 和坐标）
	// 由于 pool.Place 需要 TowerDef 和 MapDef，直接操作 pool 的 Each 不方便，
	// 这里验证空池返回空结果
	selfTower := &tower.Tower{X: 100, Y: 100, Active: true}
	querier := descriptor.NewPoolTowerQuerier(pool, selfTower)

	// 空池 → 空结果
	refs := querier.QueryRadius(100, 100, 200)
	if len(refs) != 0 {
		t.Errorf("empty pool QueryRadius: got %d refs, want 0", len(refs))
	}
}
