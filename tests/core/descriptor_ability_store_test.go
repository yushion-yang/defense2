// descriptor_ability_store_test.go — AbilityStore CRUD 测试。
//
// 验证自定义能力持久化存储的增删改查、数量上限、跨实例持久化。
// 全部使用 MemoryStorage 以避免文件系统依赖。
package core_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"defense2/internal/core/persistence"
	"defense2/internal/core/tower/descriptor"
)

// newTestCustomAbility 创建测试用自定义能力，id 和 name 由参数指定。
func newTestCustomAbility(id, name string) descriptor.CustomAbility {
	return descriptor.CustomAbility{
		ID:   id,
		Name: name,
		Desc: descriptor.AbilityDescriptor{
			ID:          id,
			Label:       name,
			Cost:        5,
			Tags:        []string{"damage", "aoe"},
			AttackStyle: "scatter",
			SpriteKey:   "shotgun",
			AttackParams: json.RawMessage(`{"count":3}`),
		},
	}
}

func TestAbilityStore_CRUDCycle(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	ca := newTestCustomAbility("ab1", "My Ability")

	// Save
	if err := store.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get
	got, err := store.Get("ab1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "My Ability" {
		t.Errorf("expected name %q, got %q", "My Ability", got.Name)
	}
	if got.Desc.AttackStyle != "scatter" {
		t.Errorf("expected attackStyle %q, got %q", "scatter", got.Desc.AttackStyle)
	}
	if len(got.Desc.Tags) != 2 || got.Desc.Tags[0] != "damage" {
		t.Errorf("expected tags [damage aoe], got %v", got.Desc.Tags)
	}

	// Delete
	if err := store.Delete("ab1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 删除后应为空
	list := store.List()
	if len(list) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(list))
	}
}

func TestAbilityStore_List(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	for i := 0; i < 3; i++ {
		ca := newTestCustomAbility(fmt.Sprintf("ab%d", i), fmt.Sprintf("Ability %d", i))
		if err := store.Save(ca); err != nil {
			t.Fatalf("Save ab%d failed: %v", i, err)
		}
	}

	list := store.List()
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
}

func TestAbilityStore_Update(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	ca := newTestCustomAbility("ab1", "Original Name")
	if err := store.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 同 ID 不同名称 → 更新
	ca.Name = "Updated Name"
	if err := store.Save(ca); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Get("ab1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("expected updated name %q, got %q", "Updated Name", got.Name)
	}

	// 更新不应增加数量
	if store.Count() != 1 {
		t.Errorf("expected count 1 after update, got %d", store.Count())
	}
}

func TestAbilityStore_MaxLimit(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	// 保存 50 个自定义能力（上限）
	for i := 0; i < 50; i++ {
		ca := newTestCustomAbility(fmt.Sprintf("ab%d", i), fmt.Sprintf("Ability %d", i))
		if err := store.Save(ca); err != nil {
			t.Fatalf("Save ab%d failed: %v", i, err)
		}
	}

	// 第 51 个应失败
	ca51 := newTestCustomAbility("ab_overflow", "Overflow Ability")
	err := store.Save(ca51)
	if err == nil {
		t.Fatal("expected error when exceeding max limit, got nil")
	}
}

func TestAbilityStore_GetNotFound(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestAbilityStore_DeleteNotFound(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent ID, got nil")
	}
}

func TestAbilityStore_Persistence(t *testing.T) {
	mem := persistence.NewMemoryStorage()

	// 第一个实例：保存
	store1 := descriptor.NewAbilityStore(mem)
	ca := newTestCustomAbility("ab_persist", "Persistent Ability")
	if err := store1.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 第二个实例：用同一个 storage 创建，应能读到数据
	store2 := descriptor.NewAbilityStore(mem)
	got, err := store2.Get("ab_persist")
	if err != nil {
		t.Fatalf("Get from new store failed: %v", err)
	}
	if got.Name != "Persistent Ability" {
		t.Errorf("expected %q, got %q", "Persistent Ability", got.Name)
	}
}

func TestAbilityStore_EmptyStore(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	list := store.List()
	if list == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 items, got %d", len(list))
	}
}

func TestAbilityStore_GetReturnsCopy(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	ca := newTestCustomAbility("ab1", "Original")
	if err := store.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 修改返回值不应影响 store 内部数据
	got, _ := store.Get("ab1")
	got.Name = "Mutated"
	got.Desc.Tags[0] = "MUTATED"

	got2, _ := store.Get("ab1")
	if got2.Name != "Original" {
		t.Errorf("Get should return copy; name mutation leaked: got %q", got2.Name)
	}
	if got2.Desc.Tags[0] != "damage" {
		t.Errorf("Get should return copy; tags mutation leaked: got %q", got2.Desc.Tags[0])
	}
}

// TestAbilityStore_PipelineRoundtrip 验证含 Pipeline 的 CustomAbility 经过
// Save→Load 后 Pipeline 数据不丢失（即 Condition/Selector/Effect 正确序列化）。
func TestAbilityStore_PipelineRoundtrip(t *testing.T) {
	mem := persistence.NewMemoryStorage()

	// 构建一个含完整管线的自定义能力
	ca := descriptor.CustomAbility{
		ID:   "roundtrip",
		Name: "Roundtrip Test",
		Desc: descriptor.AbilityDescriptor{
			ID:          "roundtrip",
			Label:       "Roundtrip Test",
			Cost:        10,
			Tags:        []string{"damage", "cc"},
			AttackStyle: "",
			Pipelines: []descriptor.Pipeline{
				{
					Trigger: descriptor.TriggerOnHit,
					Conditions: []descriptor.Condition{
						descriptor.ChanceCondition{Rate: descriptor.LinearScaler{Base: 0.3, Potential: 0.5}},
					},
					Selector: descriptor.AoeRadiusSelector{Radius: descriptor.FixedScaler{Value: 80}},
					Effects: []descriptor.Effect{
						descriptor.DamageEffect{Mode: 0, Value: descriptor.LinearScaler{Base: 10, Potential: 5}},
						descriptor.SlowEffect{
							Factor:   descriptor.FixedScaler{Value: 0.3},
							Duration: descriptor.LinearScaler{Base: 1, Potential: 0.5},
						},
					},
				},
				{
					Trigger:    descriptor.TriggerOnKill,
					Conditions: []descriptor.Condition{},
					Selector:   descriptor.CurrentTargetSelector{},
					Effects: []descriptor.Effect{
						descriptor.GoldEffect{Amount: descriptor.FixedScaler{Value: 5}},
					},
				},
			},
		},
	}

	// 第一个 store 实例保存
	store1 := descriptor.NewAbilityStore(mem)
	if err := store1.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 第二个 store 实例从同一 storage 加载
	store2 := descriptor.NewAbilityStore(mem)
	got, err := store2.Get("roundtrip")
	if err != nil {
		t.Fatalf("Get from new store failed: %v", err)
	}

	// 验证顶层字段
	if got.Desc.Label != "Roundtrip Test" {
		t.Errorf("label: expected %q, got %q", "Roundtrip Test", got.Desc.Label)
	}
	if got.Desc.Cost != 10 {
		t.Errorf("cost: expected 10, got %d", got.Desc.Cost)
	}
	if len(got.Desc.Tags) != 2 {
		t.Errorf("tags: expected 2, got %d", len(got.Desc.Tags))
	}

	// 验证 pipeline 数量
	if len(got.Desc.Pipelines) != 2 {
		t.Fatalf("pipelines: expected 2, got %d", len(got.Desc.Pipelines))
	}

	// 验证第一条 pipeline
	p0 := got.Desc.Pipelines[0]
	if p0.Trigger != descriptor.TriggerOnHit {
		t.Errorf("p0 trigger: expected onHit, got %v", p0.Trigger)
	}
	if len(p0.Conditions) != 1 {
		t.Fatalf("p0 conditions: expected 1, got %d", len(p0.Conditions))
	}
	// 验证 condition 类型还原
	if _, ok := p0.Conditions[0].(descriptor.ChanceCondition); !ok {
		t.Errorf("p0 condition[0]: expected ChanceCondition, got %T", p0.Conditions[0])
	}
	// 验证 selector 类型还原
	aoe, ok := p0.Selector.(descriptor.AoeRadiusSelector)
	if !ok {
		t.Fatalf("p0 selector: expected AoeRadiusSelector, got %T", p0.Selector)
	}
	// 验证 scaler 值（radius=80 在 strength=0 时应为 80）
	if r := aoe.Radius.Calc(0); r != 80 {
		t.Errorf("p0 selector radius: expected 80, got %v", r)
	}
	// 验证 effects 数量和类型
	if len(p0.Effects) != 2 {
		t.Fatalf("p0 effects: expected 2, got %d", len(p0.Effects))
	}
	if _, ok := p0.Effects[0].(descriptor.DamageEffect); !ok {
		t.Errorf("p0 effect[0]: expected DamageEffect, got %T", p0.Effects[0])
	}
	if _, ok := p0.Effects[1].(descriptor.SlowEffect); !ok {
		t.Errorf("p0 effect[1]: expected SlowEffect, got %T", p0.Effects[1])
	}

	// 验证第二条 pipeline
	p1 := got.Desc.Pipelines[1]
	if p1.Trigger != descriptor.TriggerOnKill {
		t.Errorf("p1 trigger: expected onKill, got %v", p1.Trigger)
	}
	if _, ok := p1.Selector.(descriptor.CurrentTargetSelector); !ok {
		t.Errorf("p1 selector: expected CurrentTargetSelector, got %T", p1.Selector)
	}
	if len(p1.Effects) != 1 {
		t.Fatalf("p1 effects: expected 1, got %d", len(p1.Effects))
	}
	gold, ok := p1.Effects[0].(descriptor.GoldEffect)
	if !ok {
		t.Fatalf("p1 effect[0]: expected GoldEffect, got %T", p1.Effects[0])
	}
	if amount := gold.Amount.Calc(0); amount != 5 {
		t.Errorf("p1 gold amount: expected 5, got %v", amount)
	}
}

// TestAbilityStore_AllScalerTypes 验证所有 Scaler 类型的序列化往返。
func TestAbilityStore_AllScalerTypes(t *testing.T) {
	mem := persistence.NewMemoryStorage()

	// 用不同 Scaler 类型构建能力
	ca := descriptor.CustomAbility{
		ID:   "scaler-test",
		Name: "Scaler Types",
		Desc: descriptor.AbilityDescriptor{
			ID:    "scaler-test",
			Label: "Scaler Types",
			Pipelines: []descriptor.Pipeline{
				{
					Trigger:    descriptor.TriggerOnHit,
					Conditions: []descriptor.Condition{},
					Selector:   descriptor.CurrentTargetSelector{},
					Effects: []descriptor.Effect{
						// LinearScaler
						descriptor.DamageEffect{Value: descriptor.LinearScaler{Base: 10, Potential: 5}},
						// FixedScaler
						descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 2}},
						// DiminishingScaler
						descriptor.RootEffect{Duration: descriptor.DiminishingScaler{Base: 1, Potential: 3, K: 100}},
						// CappedScaler
						descriptor.GoldEffect{Amount: descriptor.CappedScaler{Base: 1, Potential: 10, Cap: 50}},
					},
				},
			},
		},
	}

	store1 := descriptor.NewAbilityStore(mem)
	if err := store1.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	store2 := descriptor.NewAbilityStore(mem)
	got, err := store2.Get("scaler-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	effects := got.Desc.Pipelines[0].Effects

	// LinearScaler: base=10, potential=5 → Calc(100)=15
	dmg, ok := effects[0].(descriptor.DamageEffect)
	if !ok {
		t.Fatalf("effect[0]: expected DamageEffect, got %T", effects[0])
	}
	if v := dmg.Value.Calc(100); v != 15 {
		t.Errorf("LinearScaler: expected 15, got %v", v)
	}

	// FixedScaler: value=2 → Calc(any)=2
	stun, ok := effects[1].(descriptor.StunEffect)
	if !ok {
		t.Fatalf("effect[1]: expected StunEffect, got %T", effects[1])
	}
	if v := stun.Duration.Calc(999); v != 2 {
		t.Errorf("FixedScaler: expected 2, got %v", v)
	}

	// DiminishingScaler: base=1, potential=3, k=100
	root, ok := effects[2].(descriptor.RootEffect)
	if !ok {
		t.Fatalf("effect[2]: expected RootEffect, got %T", effects[2])
	}
	// Calc(0) = 1 + 3*(1-exp(0)) = 1
	if v := root.Duration.Calc(0); v != 1 {
		t.Errorf("DiminishingScaler at 0: expected 1, got %v", v)
	}

	// CappedScaler: base=1, potential=10, cap=50
	gold, ok := effects[3].(descriptor.GoldEffect)
	if !ok {
		t.Fatalf("effect[3]: expected GoldEffect, got %T", effects[3])
	}
	// Calc(10000) = min(1 + 10*100, 50) = 50
	if v := gold.Amount.Calc(10000); v != 50 {
		t.Errorf("CappedScaler at 10000: expected 50, got %v", v)
	}
}

// TestAbilityStore_AllEffectTypes 验证所有 Effect 类型的序列化往返。
func TestAbilityStore_AllEffectTypes(t *testing.T) {
	mem := persistence.NewMemoryStorage()

	fixed := descriptor.FixedScaler{Value: 1}
	linear := descriptor.LinearScaler{Base: 1, Potential: 0.5}

	ca := descriptor.CustomAbility{
		ID:   "all-effects",
		Name: "All Effects",
		Desc: descriptor.AbilityDescriptor{
			ID:    "all-effects",
			Label: "All Effects",
			Pipelines: []descriptor.Pipeline{
				{
					Trigger:    descriptor.TriggerOnHit,
					Conditions: []descriptor.Condition{},
					Selector:   descriptor.CurrentTargetSelector{},
					Effects: []descriptor.Effect{
						descriptor.DamageEffect{Mode: 0, Value: fixed},
						descriptor.SlowEffect{Factor: fixed, Duration: linear},
						descriptor.StunEffect{Duration: fixed},
						descriptor.RootEffect{Duration: fixed},
						descriptor.DotEffect{Subtype: "burn", Mode: 0, Value: fixed, Duration: linear},
						descriptor.WeakenEffect{Amplify: fixed, Duration: linear},
						descriptor.SilenceEffect{},
						descriptor.BuffEffect{Stat: "damage", Bonus: fixed},
						descriptor.SelfBuffEffect{Stat: "speed", Bonus: linear},
						descriptor.GoldEffect{Amount: fixed},
						descriptor.ModifyStatEffect{Stat: "range", Multiplier: 1.5},
						descriptor.CritEffect{Multiplier: 2.0},
						descriptor.PurgeEffect{Count: 3},
						descriptor.TeleportEffect{Distance: fixed},
					},
				},
			},
		},
	}

	store1 := descriptor.NewAbilityStore(mem)
	if err := store1.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	store2 := descriptor.NewAbilityStore(mem)
	got, err := store2.Get("all-effects")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	effects := got.Desc.Pipelines[0].Effects
	expectedTypes := []string{
		"DamageEffect", "SlowEffect", "StunEffect", "RootEffect",
		"DotEffect", "WeakenEffect", "SilenceEffect", "BuffEffect",
		"SelfBuffEffect", "GoldEffect", "ModifyStatEffect", "CritEffect",
		"PurgeEffect", "TeleportEffect",
	}

	if len(effects) != len(expectedTypes) {
		t.Fatalf("expected %d effects, got %d", len(expectedTypes), len(effects))
	}

	// 逐个验证类型还原
	for i, name := range expectedTypes {
		actual := fmt.Sprintf("%T", effects[i])
		expected := "descriptor." + name
		if actual != expected {
			t.Errorf("effect[%d]: expected %s, got %s", i, expected, actual)
		}
	}

	// 抽样验证数值
	dot := effects[4].(descriptor.DotEffect)
	if dot.Subtype != "burn" {
		t.Errorf("dot subtype: expected burn, got %s", dot.Subtype)
	}

	crit := effects[11].(descriptor.CritEffect)
	if crit.Multiplier != 2.0 {
		t.Errorf("crit multiplier: expected 2.0, got %v", crit.Multiplier)
	}

	purge := effects[12].(descriptor.PurgeEffect)
	if purge.Count != 3 {
		t.Errorf("purge count: expected 3, got %d", purge.Count)
	}
}

func TestAbilityStore_ListReturnsCopy(t *testing.T) {
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())

	ca := newTestCustomAbility("ab1", "Original")
	if err := store.Save(ca); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 修改列表不应影响 store 内部数据
	list := store.List()
	list[0].Name = "Mutated"
	list[0].Desc.Tags[0] = "MUTATED"

	list2 := store.List()
	if list2[0].Name != "Original" {
		t.Errorf("List should return copy; name mutation leaked: got %q", list2[0].Name)
	}
	if list2[0].Desc.Tags[0] != "damage" {
		t.Errorf("List should return copy; tags mutation leaked: got %q", list2[0].Desc.Tags[0])
	}
}
