// descriptor_contracts_test.go — 能力描述符契约测试。
//
// 验证 ability-descriptors.json 与 abilities.json 的 1:1 映射关系，
// 确保每个旧格式能力都有对应的描述符，且无孤儿描述符。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower"
	"defense2/internal/core/tower/descriptor"
)

// init 在 config_rules_test.go 中已设置 SetDataFS，
// 描述符引擎是能力系统的唯一初始化入口。
func init() {
	if err := descriptor.InitDescriptorAbilities(config.GetDataFS()); err != nil {
		panic("init descriptor abilities: " + err.Error())
	}
}

// TestAllAbilitiesHaveDescriptors 验证 abilities.json 中每个能力都有对应的描述符。
func TestAllAbilitiesHaveDescriptors(t *testing.T) {
	abilityTable := config.GlobalAbilityTable()
	if len(abilityTable) == 0 {
		t.Fatal("能力表为空，config 未初始化")
	}

	descTable := descriptor.GlobalDescriptorTable()
	if len(descTable) == 0 {
		t.Fatal("描述符表为空，loader 未初始化")
	}

	for abilityID := range abilityTable {
		t.Run(abilityID, func(t *testing.T) {
			if _, ok := descTable[abilityID]; !ok {
				t.Errorf("能力 %q 在 abilities.json 中存在，但缺少对应的描述符", abilityID)
			}
		})
	}
}

// TestAllDescriptorsHaveAbilities 验证每个描述符都有对应的能力定义（无孤儿描述符）。
func TestAllDescriptorsHaveAbilities(t *testing.T) {
	abilityTable := config.GlobalAbilityTable()
	descTable := descriptor.GlobalDescriptorTable()

	for descID := range descTable {
		t.Run(descID, func(t *testing.T) {
			if _, ok := abilityTable[descID]; !ok {
				t.Errorf("描述符 %q 不存在于 abilities.json 中（孤儿描述符）", descID)
			}
		})
	}
}

// TestDescriptorIDsMatchAbilityTypes 验证描述符 ID 与能力定义 type 字段一致。
func TestDescriptorIDsMatchAbilityTypes(t *testing.T) {
	abilityTable := config.GlobalAbilityTable()
	descTable := descriptor.GlobalDescriptorTable()

	for descID, desc := range descTable {
		t.Run(descID, func(t *testing.T) {
			abilityDef, ok := abilityTable[descID]
			if !ok {
				t.Skipf("描述符 %q 无对应能力（由其他测试覆盖）", descID)
				return
			}
			if desc.ID != abilityDef.Type {
				t.Errorf("描述符 ID=%q ≠ 能力 Type=%q", desc.ID, abilityDef.Type)
			}
		})
	}
}

// TestDescriptorCount 验证描述符总数恰好为 35（与 abilities.json 中非禁用能力一致）。
// 32 基础 + 3 CD 能力（pulseStun, pushField, curseAura）
func TestDescriptorCount(t *testing.T) {
	descTable := descriptor.GlobalDescriptorTable()
	abilityTable := config.GlobalAbilityTable()

	descCount := len(descTable)
	abilityCount := len(abilityTable)

	if descCount != abilityCount {
		t.Errorf("描述符数=%d ≠ 能力数=%d", descCount, abilityCount)
	}
	if descCount != 35 {
		t.Errorf("描述符数=%d，期望 35", descCount)
	}
}

// TestAllDescriptorsParseSuccessfully 验证每个描述符的基本结构完整性。
// 由 loader 在解析阶段已校验，此处二次确认关键字段。
func TestAllDescriptorsParseSuccessfully(t *testing.T) {
	descTable := descriptor.GlobalDescriptorTable()
	if len(descTable) == 0 {
		t.Fatal("描述符表为空")
	}

	for id, desc := range descTable {
		t.Run(id, func(t *testing.T) {
			if desc.ID == "" {
				t.Error("描述符 ID 为空")
			}
			if desc.Label == "" {
				t.Errorf("描述符 %q 缺少 label", id)
			}
			if len(desc.Tags) == 0 {
				t.Errorf("描述符 %q 缺少 tags", id)
			}

			// 有 attackStyle 的能力可以没有 pipeline（效果由 attackStyle 描述），
			// 没有 attackStyle 的能力必须至少有一条 pipeline。
			if desc.AttackStyle == "" && len(desc.Pipelines) == 0 {
				t.Errorf("描述符 %q: 无 attackStyle 时必须有至少一条 pipeline", id)
			}
		})
	}
}

// TestDescriptorLabelsMatchAbilities 验证描述符 label 与能力定义 label 一致。
func TestDescriptorLabelsMatchAbilities(t *testing.T) {
	abilityTable := config.GlobalAbilityTable()
	descTable := descriptor.GlobalDescriptorTable()

	for descID, desc := range descTable {
		t.Run(descID, func(t *testing.T) {
			abilityDef, ok := abilityTable[descID]
			if !ok {
				t.Skipf("描述符 %q 无对应能力", descID)
				return
			}
			if desc.Label != abilityDef.Label {
				t.Errorf("描述符 label=%q ≠ 能力 label=%q", desc.Label, abilityDef.Label)
			}
		})
	}
}

// TestAllRegisteredAreDescriptorAbilities 验证 Registry 中每个能力都是
// DescriptorAbility 类型（描述符引擎是唯一注册源）。
func TestAllRegisteredAreDescriptorAbilities(t *testing.T) {
	abilityTable := config.GlobalAbilityTable()
	if len(abilityTable) == 0 {
		t.Fatal("能力表为空，config 未初始化")
	}

	for name := range abilityTable {
		t.Run(name, func(t *testing.T) {
			ab, ok := tower.Lookup(name)
			if !ok {
				t.Errorf("%q 未在 Registry 中注册", name)
				return
			}
			switch ab.(type) {
			case *descriptor.DescriptorAbilityHit, *descriptor.DescriptorAbilityFull:
				// OK — 描述符已覆盖 ConfigAbility
			default:
				t.Errorf("%q 类型为 %T，期望 DescriptorAbilityHit 或 DescriptorAbilityFull", name, ab)
			}
		})
	}
}
