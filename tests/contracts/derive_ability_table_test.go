// derive_ability_table_test.go — DeriveAbilityTable 契约测试。
//
// 验证从描述符派生的 AbilityTable 与旧 abilities.json 的元数据一致，
// 确保迁移后 AddAbility/HUD/info panel 行为不变。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
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
