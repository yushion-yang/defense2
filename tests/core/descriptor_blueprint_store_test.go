// descriptor_blueprint_store_test.go — BlueprintStore CRUD 测试。
//
// 验证蓝图持久化存储的增删改查、数量上限、跨实例持久化。
// 全部使用 MemoryStorage 以避免文件系统依赖。
package core_test

import (
	"fmt"
	"testing"

	"defense2/internal/core/persistence"
	"defense2/internal/core/tower/descriptor"
)

// newTestBlueprint 创建测试用蓝图，id 和 name 由参数指定。
func newTestBlueprint(id, name string) descriptor.TowerBlueprint {
	return descriptor.TowerBlueprint{
		ID:          id,
		Name:        name,
		Author:      "tester",
		AttackStyle: "projectile",
		Tiers:       map[string]string{"damage": "S", "atkSpeed": "B", "range": "D"},
		Specialty:   "damage",
		Abilities:   []string{"splash", "crit"},
		BuildCost:   100,
	}
}

func TestBlueprintStore_CRUDCycle(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	bp := newTestBlueprint("bp1", "My Tower")

	// Save
	if err := store.Save(bp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get
	got, err := store.Get("bp1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "My Tower" {
		t.Errorf("expected name %q, got %q", "My Tower", got.Name)
	}
	if got.AttackStyle != "projectile" {
		t.Errorf("expected attackStyle %q, got %q", "projectile", got.AttackStyle)
	}
	if got.Tiers["damage"] != "S" {
		t.Errorf("expected damage tier S, got %q", got.Tiers["damage"])
	}

	// Delete
	if err := store.Delete("bp1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 删除后应为空
	list := store.List()
	if len(list) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(list))
	}
}

func TestBlueprintStore_List(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	for i := 0; i < 3; i++ {
		bp := newTestBlueprint(fmt.Sprintf("bp%d", i), fmt.Sprintf("Tower %d", i))
		if err := store.Save(bp); err != nil {
			t.Fatalf("Save bp%d failed: %v", i, err)
		}
	}

	list := store.List()
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
}

func TestBlueprintStore_Update(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	bp := newTestBlueprint("bp1", "Original Name")
	if err := store.Save(bp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 同 ID 不同名称 → 更新
	bp.Name = "Updated Name"
	if err := store.Save(bp); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Get("bp1")
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

func TestBlueprintStore_MaxLimit(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	// 保存 20 个蓝图（上限）
	for i := 0; i < 20; i++ {
		bp := newTestBlueprint(fmt.Sprintf("bp%d", i), fmt.Sprintf("Tower %d", i))
		if err := store.Save(bp); err != nil {
			t.Fatalf("Save bp%d failed: %v", i, err)
		}
	}

	// 第 21 个应失败
	bp21 := newTestBlueprint("bp_overflow", "Overflow Tower")
	err := store.Save(bp21)
	if err == nil {
		t.Fatal("expected error when exceeding max limit, got nil")
	}
}

func TestBlueprintStore_MaxLimitUpdateExisting(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	// 保存 20 个蓝图（上限）
	for i := 0; i < 20; i++ {
		bp := newTestBlueprint(fmt.Sprintf("bp%d", i), fmt.Sprintf("Tower %d", i))
		if err := store.Save(bp); err != nil {
			t.Fatalf("Save bp%d failed: %v", i, err)
		}
	}

	// 更新已有的应成功（不算新增）
	bp := newTestBlueprint("bp0", "Updated Tower 0")
	if err := store.Save(bp); err != nil {
		t.Fatalf("Update existing at max should succeed, got: %v", err)
	}

	// 验证确实更新了
	got, err := store.Get("bp0")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Updated Tower 0" {
		t.Errorf("expected %q, got %q", "Updated Tower 0", got.Name)
	}
}

func TestBlueprintStore_GetNotFound(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestBlueprintStore_DeleteNotFound(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent ID, got nil")
	}
}

func TestBlueprintStore_Persistence(t *testing.T) {
	mem := persistence.NewMemoryStorage()

	// 第一个实例：保存
	store1 := descriptor.NewBlueprintStore(mem)
	bp := newTestBlueprint("bp_persist", "Persistent Tower")
	if err := store1.Save(bp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 第二个实例：用同一个 storage 创建，应能读到数据
	store2 := descriptor.NewBlueprintStore(mem)
	got, err := store2.Get("bp_persist")
	if err != nil {
		t.Fatalf("Get from new store failed: %v", err)
	}
	if got.Name != "Persistent Tower" {
		t.Errorf("expected %q, got %q", "Persistent Tower", got.Name)
	}
}

func TestBlueprintStore_EmptyStore(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	list := store.List()
	if list == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(list) != 0 {
		t.Errorf("expected 0 items, got %d", len(list))
	}
}

func TestBlueprintStore_Count(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	for i := 0; i < 3; i++ {
		bp := newTestBlueprint(fmt.Sprintf("bp%d", i), fmt.Sprintf("Tower %d", i))
		if err := store.Save(bp); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	if store.Count() != 3 {
		t.Errorf("expected count 3, got %d", store.Count())
	}
}

func TestBlueprintStore_GetReturnsCopy(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	bp := newTestBlueprint("bp1", "Original")
	if err := store.Save(bp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 修改返回值不应影响 store 内部数据
	got, _ := store.Get("bp1")
	got.Name = "Mutated"

	got2, _ := store.Get("bp1")
	if got2.Name != "Original" {
		t.Errorf("Get should return copy; mutation leaked: got %q", got2.Name)
	}
}

func TestBlueprintStore_ListReturnsCopy(t *testing.T) {
	store := descriptor.NewBlueprintStore(persistence.NewMemoryStorage())

	bp := newTestBlueprint("bp1", "Original")
	if err := store.Save(bp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 修改列表不应影响 store 内部数据
	list := store.List()
	list[0].Name = "Mutated"

	list2 := store.List()
	if list2[0].Name != "Original" {
		t.Errorf("List should return copy; mutation leaked: got %q", list2[0].Name)
	}
}
