// derive_ability_table_test.go — DeriveAbilityTable 契约测试。
//
// 验证从描述符派生的 AbilityTable 与旧 abilities.json 的元数据一致，
// 确保迁移后 AddAbility/HUD/info panel 行为不变。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/persistence"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// TestDeriveAbilityTableMatchesOld 验证从描述符派生的 AbilityTable 与旧 abilities.json 一致。
func TestDeriveAbilityTableMatchesOld(t *testing.T) {
	// init() 已初始化 ConfigAbilities + DescriptorAbilities
	oldTable := config.GlobalAbilityTable()
	if len(oldTable) == 0 {
		t.Fatal("old ability table is empty")
	}

	derived := descriptor.DeriveAbilityTable()

	// 每个旧能力必须在派生表中存在且核心元数据匹配
	for id, oldDef := range oldTable {
		newDef, ok := derived[id]
		if !ok {
			t.Errorf("ability %q missing from derived table", id)
			continue
		}
		if newDef.Category != oldDef.Category {
			t.Errorf("%s: category mismatch: got %q, want %q", id, newDef.Category, oldDef.Category)
		}
		if newDef.Icon != oldDef.Icon {
			t.Errorf("%s: icon mismatch: got %q, want %q", id, newDef.Icon, oldDef.Icon)
		}
		if newDef.Label != oldDef.Label {
			t.Errorf("%s: label mismatch: got %q, want %q", id, newDef.Label, oldDef.Label)
		}
		if newDef.Display != oldDef.Display {
			t.Errorf("%s: display mismatch:\n  got:  %q\n  want: %q", id, newDef.Display, oldDef.Display)
		}
	}
}

// TestGlobalAbilityTableContainsDerived 验证 InitDescriptorAbilities 后
// GlobalAbilityTable 包含描述符派生的能力元数据。
func TestGlobalAbilityTableContainsDerived(t *testing.T) {
	// init() 已调用 InitDescriptorAbilities，如果 wiring 正确，
	// GlobalAbilityTable 应包含描述符派生的数据。
	table := config.GlobalAbilityTable()
	if len(table) < 32 {
		t.Fatalf("expected >= 32 abilities in table, got %d", len(table))
	}

	// 抽查：slowPower 的 category 应为 cc
	if def, ok := table["slowPower"]; !ok {
		t.Error("slowPower missing from GlobalAbilityTable")
	} else if def.Category != "cc" {
		t.Errorf("slowPower category: got %q, want cc", def.Category)
	}

	// 抽查：display 不应为空（说明是从描述符派生的，不是旧 abilities.json）
	if def, ok := table["crit"]; ok && def.Display == "" {
		t.Error("crit display is empty — DeriveAbilityTable not wired?")
	}
}

// TestCustomAbilityInAbilityTable 验证自定义能力注册后出现在 AbilityTable 中。
func TestCustomAbilityInAbilityTable(t *testing.T) {
	// 创建内存存储和自定义能力
	store := descriptor.NewAbilityStore(persistence.NewMemoryStorage())
	ca := descriptor.CustomAbility{
		ID:   "ca_test_001",
		Name: "测试能力",
		Desc: descriptor.AbilityDescriptor{
			ID:    "ca_test_001",
			Label: "测试能力",
			Icon:  "crit",
			Tags:  []string{"damage"},
		},
	}
	_ = store.Save(ca)

	descriptor.RegisterCustomAbilities(store)

	// 验证注入成功
	table := config.GlobalAbilityTable()
	def, ok := table["ca_test_001"]
	if !ok {
		t.Fatal("custom ability ca_test_001 not in AbilityTable")
	}
	if def.Category != "damage" {
		t.Errorf("category: got %q, want damage", def.Category)
	}
	if def.Label != "测试能力" {
		t.Errorf("label: got %q, want 测试能力", def.Label)
	}

	// 清理：移除测试注入的全局状态，避免污染其他测试
	delete(table, "ca_test_001")
	delete(tower.Registry, "ca_test_001")
}
