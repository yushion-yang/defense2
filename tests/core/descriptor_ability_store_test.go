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
