// descriptor_metadata_test.go — 描述符元数据完整性契约测试。
//
// 验证所有预制描述符包含完整的展示元数据（icon/display/prebuilt/tags），
// 确保从描述符派生 AbilityDef 时数据不丢失。
package contracts_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// TestDescriptorMetadataComplete 验证所有预制描述符包含完整的展示元数据。
func TestDescriptorMetadataComplete(t *testing.T) {
	// init() 已初始化 ConfigAbilities + DescriptorAbilities
	table := descriptor.GlobalDescriptorTable()
	if len(table) < 32 {
		t.Fatalf("expected >= 32 descriptors, got %d", len(table))
	}

	for _, desc := range table {
		if desc.Icon == "" {
			t.Errorf("descriptor %q missing icon", desc.ID)
		}
		if desc.Display == "" {
			t.Errorf("descriptor %q missing display", desc.ID)
		}
		if !desc.Prebuilt {
			t.Errorf("descriptor %q should be prebuilt", desc.ID)
		}
		if len(desc.Tags) == 0 {
			t.Errorf("descriptor %q missing tags", desc.ID)
		}
	}
}
