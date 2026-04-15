// descriptor_interpreter_test.go — Interpreter 运行时解释器测试。
//
// 验证 Interpreter 能正确执行预编译的 descriptor 管线：
// 触发匹配 → 条件门 → 目标选择 → 效果生成 → 结果收集。
//
// mock EnemyQuerier/TowerQuerier 已在 descriptor_selector_test.go 中定义，
// 本文件直接复用（同属 core_test 包）。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── 测试用例 ──────────────────────────────────────────────────

// TestInterpreter_StunOnHit 解析 stunChance（chance=1.0 确定性），
// ExecOnHit → 应产出 EffTypeStun 结果。
func TestInterpreter_StunOnHit(t *testing.T) {
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

	interp := descriptor.NewInterpreter(desc)

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange: 200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 130, HpRatio: 0.8, Active: true},
		HitX:        120, HitY: 130,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 1 {
		t.Fatalf("results count = %d, want 1", len(results))
	}
	if results[0].Type != descriptor.EffTypeStun {
		t.Errorf("result[0].Type = %v, want EffTypeStun", results[0].Type)
	}
	if results[0].StunDur != 0.5 {
		t.Errorf("result[0].StunDur = %v, want 0.5", results[0].StunDur)
	}
}

// TestInterpreter_SplashDamage 解析 splash，3 个 mock 敌人在半径内，
// ExecOnHit → 应产出 3 个 EffTypeDamage 结果。
func TestInterpreter_SplashDamage(t *testing.T) {
	data := []byte(`{
		"id": "splash",
		"label": "Splash",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "aoeRadius", "radius": {"scaler": "fixed", "value": 100}},
				"effects": [
					{"type": "damage", "mode": "ratio", "value": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{Index: 0, X: 110, Y: 110, HpRatio: 1.0, Active: true},
			{Index: 1, X: 120, Y: 120, HpRatio: 0.5, Active: true},
			{Index: 2, X: 130, Y: 130, HpRatio: 0.3, Active: true},
		},
	}

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 80,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 110, Y: 110, HpRatio: 1.0, Active: true},
		HitX:        110, HitY: 110,
		Enemies:     enemies,
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Type != descriptor.EffTypeDamage {
			t.Errorf("result[%d].Type = %v, want EffTypeDamage", i, r.Type)
		}
		// 0.5 * 80 = 40
		if r.Damage != 40 {
			t.Errorf("result[%d].Damage = %v, want 40", i, r.Damage)
		}
	}
}

// TestInterpreter_GoldPassive_Cooldown 解析 goldPassive（onTick + cooldown 3s），
// 验证首次触发→冷却中不触发→冷却结束触发。
func TestInterpreter_GoldPassive_Cooldown(t *testing.T) {
	data := []byte(`{
		"id": "goldPassive",
		"label": "Gold Generator",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [
					{"type": "cooldown", "seconds": 3.0}
				],
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

	interp := descriptor.NewInterpreter(desc)

	makeCtx := func(elapsed float64) descriptor.TriggerContext {
		return descriptor.TriggerContext{
			Type:     descriptor.TriggerOnTick,
			Strength: 100,
			TowerX:   100, TowerY: 100,
			TowerRange: 200,
			Enemies:    &mockEnemyQuerier{},
			Towers:     &mockTowerQuerier{},
			Elapsed:    elapsed,
		}
	}

	// 首次触发：CooldownCondition 首次无条件通过
	r1 := interp.ExecOnTick(makeCtx(0))
	if len(r1) != 1 {
		t.Fatalf("first tick: results count = %d, want 1", len(r1))
	}
	if r1[0].Type != descriptor.EffTypeGold {
		t.Errorf("first tick: type = %v, want EffTypeGold", r1[0].Type)
	}
	if r1[0].GoldAmount != 5 {
		t.Errorf("first tick: goldAmount = %v, want 5", r1[0].GoldAmount)
	}

	// 冷却中（累计 1s < 3s）→ 不触发
	r2 := interp.ExecOnTick(makeCtx(1.0))
	if len(r2) != 0 {
		t.Errorf("cooldown tick: results count = %d, want 0", len(r2))
	}

	// 冷却中（累计 1+1=2s < 3s）→ 不触发
	r3 := interp.ExecOnTick(makeCtx(1.0))
	if len(r3) != 0 {
		t.Errorf("cooldown tick 2: results count = %d, want 0", len(r3))
	}

	// 冷却结束（累计 2+1.5=3.5s >= 3s）→ 触发
	r4 := interp.ExecOnTick(makeCtx(1.5))
	if len(r4) != 1 {
		t.Fatalf("after cooldown: results count = %d, want 1", len(r4))
	}
	if r4[0].Type != descriptor.EffTypeGold {
		t.Errorf("after cooldown: type = %v, want EffTypeGold", r4[0].Type)
	}
}

// TestInterpreter_MultiPipeline 解析含 2 条 onHit 管线的描述符，
// ExecOnHit → 两条管线的结果都应返回。
func TestInterpreter_MultiPipeline(t *testing.T) {
	data := []byte(`{
		"id": "stunAndDmg",
		"label": "Stun + Extra Damage",
		"cost": 4,
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
				"trigger": "onHit",
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 20}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 0.6, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 2 {
		t.Fatalf("results count = %d, want 2", len(results))
	}

	// 管线 0: stun
	if results[0].Type != descriptor.EffTypeStun {
		t.Errorf("result[0].Type = %v, want EffTypeStun", results[0].Type)
	}
	if results[0].StunDur != 0.3 {
		t.Errorf("result[0].StunDur = %v, want 0.3", results[0].StunDur)
	}

	// 管线 1: damage
	if results[1].Type != descriptor.EffTypeDamage {
		t.Errorf("result[1].Type = %v, want EffTypeDamage", results[1].Type)
	}
	if results[1].Damage != 20 {
		t.Errorf("result[1].Damage = %v, want 20", results[1].Damage)
	}
}

// TestInterpreter_ConditionBlocksExecution 解析 hpBelow(0.5)，
// 目标血量 70% → 条件不通过 → 无结果。
func TestInterpreter_ConditionBlocksExecution(t *testing.T) {
	data := []byte(`{
		"id": "execute",
		"label": "Execute",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "hpBelow", "threshold": {"scaler": "fixed", "value": 0.5}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 999}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	// 目标 70% HP > 50% → 条件不通过
	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 0.7, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 0 {
		t.Errorf("results count = %d, want 0 (condition should block)", len(results))
	}
}

// TestInterpreter_ConditionAllows 验证 hpBelow(0.5) 在目标 30% HP 时通过。
func TestInterpreter_ConditionAllows(t *testing.T) {
	data := []byte(`{
		"id": "execute",
		"label": "Execute",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "hpBelow", "threshold": {"scaler": "fixed", "value": 0.5}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 999}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	// 目标 30% HP < 50% → 条件通过
	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 0.3, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 1 {
		t.Fatalf("results count = %d, want 1", len(results))
	}
	if results[0].Damage != 999 {
		t.Errorf("damage = %v, want 999", results[0].Damage)
	}
}

// TestInterpreter_HasOnHitOnTick 验证 HasOnHit/HasOnTick 标志。
func TestInterpreter_HasOnHitOnTick(t *testing.T) {
	// stunChance → HasOnHit=true, HasOnTick=false
	stunData := []byte(`{
		"id": "stunChance",
		"label": "Stun Chance",
		"cost": 3,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 0.3}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 0.5}}
				]
			}
		]
	}`)

	desc1, err := descriptor.ParseDescriptor(stunData)
	if err != nil {
		t.Fatalf("ParseDescriptor(stun): %v", err)
	}
	interp1 := descriptor.NewInterpreter(desc1)

	if !interp1.HasOnHit() {
		t.Error("stunChance: HasOnHit() = false, want true")
	}
	if interp1.HasOnTick() {
		t.Error("stunChance: HasOnTick() = true, want false")
	}

	// goldPassive → HasOnHit=false, HasOnTick=true
	goldData := []byte(`{
		"id": "goldPassive",
		"label": "Gold Generator",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
				"conditions": [
					{"type": "cooldown", "seconds": 3.0}
				],
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 5}}
				]
			}
		]
	}`)

	desc2, err := descriptor.ParseDescriptor(goldData)
	if err != nil {
		t.Fatalf("ParseDescriptor(gold): %v", err)
	}
	interp2 := descriptor.NewInterpreter(desc2)

	if interp2.HasOnHit() {
		t.Error("goldPassive: HasOnHit() = true, want false")
	}
	if !interp2.HasOnTick() {
		t.Error("goldPassive: HasOnTick() = false, want true")
	}
}

// TestInterpreter_ChanceZero 概率 0 → ExecOnHit 返回空切片。
func TestInterpreter_ChanceZero(t *testing.T) {
	data := []byte(`{
		"id": "neverStun",
		"label": "Never Stun",
		"cost": 1,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "chance", "rate": {"scaler": "fixed", "value": 0.0}}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "stun", "duration": {"scaler": "fixed", "value": 1.0}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 1.0, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 0 {
		t.Errorf("results count = %d, want 0 (chance=0 should never trigger)", len(results))
	}
}

// TestInterpreter_TriggerMismatch onTick 管线不应被 ExecOnHit 触发。
func TestInterpreter_TriggerMismatch(t *testing.T) {
	data := []byte(`{
		"id": "goldPassive",
		"label": "Gold Generator",
		"cost": 5,
		"pipelines": [
			{
				"trigger": "onTick",
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

	interp := descriptor.NewInterpreter(desc)

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 1.0, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	results := interp.ExecOnHit(ctx)

	if len(results) != 0 {
		t.Errorf("results count = %d, want 0 (onTick should not trigger on onHit)", len(results))
	}
}

// TestInterpreter_NilTargetEnemy ExecOnHit 无 TargetEnemy 时不 panic。
func TestInterpreter_NilTargetEnemy(t *testing.T) {
	data := []byte(`{
		"id": "splash",
		"label": "Splash",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"selector": {"type": "aoeRadius", "radius": {"scaler": "fixed", "value": 50}},
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

	interp := descriptor.NewInterpreter(desc)

	enemies := &mockEnemyQuerier{
		enemies: []descriptor.EnemyRef{
			{Index: 0, X: 105, Y: 105, HpRatio: 1.0, Active: true},
		},
	}

	ctx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: nil,
		HitX:        100, HitY: 100,
		Enemies:     enemies,
		Towers:      &mockTowerQuerier{},
	}

	// 不 panic 即通过
	results := interp.ExecOnHit(ctx)

	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
}

// TestInterpreter_NearestAllyDist 验证 findNearestAllyDist 在 NoNearbyTower 条件中生效。
func TestInterpreter_NearestAllyDist(t *testing.T) {
	data := []byte(`{
		"id": "loner",
		"label": "Loner Bonus",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "noNearbyTower", "radius": 100}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 50}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	// 场景 1: 无友方塔 → NearestAllyDist = MaxFloat64 > 100 → 通过
	ctx1 := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 0.8, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	r1 := interp.ExecOnHit(ctx1)
	if len(r1) != 1 {
		t.Errorf("no allies: results count = %d, want 1", len(r1))
	}

	// 场景 2: 有友方塔在 50 距离内 → NearestAllyDist=50 ≤ 100 → 不通过
	ctx2 := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 0.8, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers: &mockTowerQuerier{
			towers: []descriptor.TowerRef{
				{Index: 1, X: 150, Y: 100, Active: true}, // 距离 50
			},
		},
	}

	r2 := interp.ExecOnHit(ctx2)
	if len(r2) != 0 {
		t.Errorf("nearby ally: results count = %d, want 0", len(r2))
	}
}

// TestInterpreter_MixedTriggerTypes 同时含 onHit + onTick 的描述符。
func TestInterpreter_MixedTriggerTypes(t *testing.T) {
	data := []byte(`{
		"id": "hybrid",
		"label": "Hybrid",
		"cost": 5,
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
				"selector": {"type": "selfTower"},
				"effects": [
					{"type": "gold", "amount": {"scaler": "fixed", "value": 3}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	if !interp.HasOnHit() {
		t.Error("HasOnHit() = false, want true")
	}
	if !interp.HasOnTick() {
		t.Error("HasOnTick() = false, want true")
	}

	// ExecOnHit 只触发 onHit 管线
	hitCtx := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      100, TowerY: 100,
		TowerRange:  200,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 120, Y: 120, HpRatio: 1.0, Active: true},
		HitX:        120, HitY: 120,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	hitResults := interp.ExecOnHit(hitCtx)
	if len(hitResults) != 1 {
		t.Fatalf("ExecOnHit: count = %d, want 1", len(hitResults))
	}
	if hitResults[0].Type != descriptor.EffTypeStun {
		t.Errorf("ExecOnHit: type = %v, want EffTypeStun", hitResults[0].Type)
	}

	// ExecOnTick 只触发 onTick 管线
	tickCtx := descriptor.TriggerContext{
		Type:     descriptor.TriggerOnTick,
		Strength: 100,
		TowerX:   100, TowerY: 100,
		TowerRange: 200,
		Enemies:    &mockEnemyQuerier{},
		Towers:     &mockTowerQuerier{},
		Elapsed:    0,
	}

	tickResults := interp.ExecOnTick(tickCtx)
	if len(tickResults) != 1 {
		t.Fatalf("ExecOnTick: count = %d, want 1", len(tickResults))
	}
	if tickResults[0].Type != descriptor.EffTypeGold {
		t.Errorf("ExecOnTick: type = %v, want EffTypeGold", tickResults[0].Type)
	}
}

// TestInterpreter_TargetDistance 验证 ConditionCtx.TargetDistance 正确计算。
func TestInterpreter_TargetDistance(t *testing.T) {
	// distanceMin(100): 塔(0,0) 目标(60,80) → 距离=100 → 刚好通过(>=100)
	data := []byte(`{
		"id": "sniper",
		"label": "Sniper",
		"cost": 2,
		"pipelines": [
			{
				"trigger": "onHit",
				"conditions": [
					{"type": "distanceMin", "distance": 100}
				],
				"selector": {"type": "currentTarget"},
				"effects": [
					{"type": "damage", "mode": "flat", "value": {"scaler": "fixed", "value": 100}}
				]
			}
		]
	}`)

	desc, err := descriptor.ParseDescriptor(data)
	if err != nil {
		t.Fatalf("ParseDescriptor: %v", err)
	}

	interp := descriptor.NewInterpreter(desc)

	// 距离正好 = 100 → 通过
	ctx1 := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      0, TowerY: 0,
		TowerRange:  300,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 60, Y: 80, HpRatio: 1.0, Active: true},
		HitX:        60, HitY: 80,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	r1 := interp.ExecOnHit(ctx1)
	if len(r1) != 1 {
		t.Errorf("dist=100: results count = %d, want 1", len(r1))
	}

	// 距离 50 < 100 → 不通过
	ctx2 := descriptor.TriggerContext{
		Type:        descriptor.TriggerOnHit,
		Strength:    100,
		TowerDamage: 50,
		TowerX:      0, TowerY: 0,
		TowerRange:  300,
		TargetEnemy: &descriptor.EnemyRef{Index: 0, X: 30, Y: 40, HpRatio: 1.0, Active: true},
		HitX:        30, HitY: 40,
		Enemies:     &mockEnemyQuerier{},
		Towers:      &mockTowerQuerier{},
	}

	r2 := interp.ExecOnHit(ctx2)
	if len(r2) != 0 {
		t.Errorf("dist=50: results count = %d, want 0", len(r2))
	}
}

