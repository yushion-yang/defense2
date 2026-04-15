// descriptor_primitive_meta_test.go — 基元元数据注册表测试。
//
// 验证 AllTriggerMeta/AllConditionMeta/AllSelectorMeta/AllEffectMeta
// 返回正确数量、非空字段、唯一 ID 及参数范围合法性。
package core_test

import (
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// TestAllTriggerMeta 验证触发器元数据返回 4 项，且所有字段非空。
func TestAllTriggerMeta(t *testing.T) {
	metas := descriptor.AllTriggerMeta()
	if len(metas) != 4 {
		t.Fatalf("AllTriggerMeta() returned %d items, want 4", len(metas))
	}
	for _, m := range metas {
		if m.ID == "" {
			t.Error("trigger meta has empty ID")
		}
		if m.Label == "" {
			t.Errorf("trigger %q has empty Label", m.ID)
		}
	}
}

// TestAllConditionMeta 验证条件元数据返回 11 项。
func TestAllConditionMeta(t *testing.T) {
	metas := descriptor.AllConditionMeta()
	if len(metas) != 11 {
		t.Fatalf("AllConditionMeta() returned %d items, want 11", len(metas))
	}
	for _, m := range metas {
		if m.ID == "" {
			t.Error("condition meta has empty ID")
		}
		if m.Label == "" {
			t.Errorf("condition %q has empty Label", m.ID)
		}
	}
}

// TestAllSelectorMeta 验证选择器元数据返回 9 项。
func TestAllSelectorMeta(t *testing.T) {
	metas := descriptor.AllSelectorMeta()
	if len(metas) != 9 {
		t.Fatalf("AllSelectorMeta() returned %d items, want 9", len(metas))
	}
	for _, m := range metas {
		if m.ID == "" {
			t.Error("selector meta has empty ID")
		}
		if m.Label == "" {
			t.Errorf("selector %q has empty Label", m.ID)
		}
	}
}

// TestAllEffectMeta 验证效果元数据返回 13 项。
func TestAllEffectMeta(t *testing.T) {
	metas := descriptor.AllEffectMeta()
	if len(metas) != 13 {
		t.Fatalf("AllEffectMeta() returned %d items, want 13", len(metas))
	}
	for _, m := range metas {
		if m.ID == "" {
			t.Error("effect meta has empty ID")
		}
		if m.Label == "" {
			t.Errorf("effect %q has empty Label", m.ID)
		}
	}
}

// TestPrimitiveMetaUniqueIDs 验证所有类别的 ID 无重复。
func TestPrimitiveMetaUniqueIDs(t *testing.T) {
	seen := make(map[string]string) // id → category

	check := func(category string, metas []descriptor.PrimitiveMeta) {
		for _, m := range metas {
			if prev, dup := seen[m.ID]; dup {
				t.Errorf("duplicate ID %q: found in %q and %q", m.ID, prev, category)
			}
			seen[m.ID] = category
		}
	}

	check("trigger", descriptor.AllTriggerMeta())
	check("condition", descriptor.AllConditionMeta())
	check("selector", descriptor.AllSelectorMeta())
	check("effect", descriptor.AllEffectMeta())
}

// TestPrimitiveMetaParamRanges 验证参数范围合法：Min <= Max，Default 在 [Min, Max] 内。
func TestPrimitiveMetaParamRanges(t *testing.T) {
	allMetas := []struct {
		category string
		metas    []descriptor.PrimitiveMeta
	}{
		{"trigger", descriptor.AllTriggerMeta()},
		{"condition", descriptor.AllConditionMeta()},
		{"selector", descriptor.AllSelectorMeta()},
		{"effect", descriptor.AllEffectMeta()},
	}

	for _, group := range allMetas {
		for _, m := range group.metas {
			for _, p := range m.Params {
				// string/bool 类型没有数值范围约束
				if p.Type == "string" || p.Type == "bool" {
					continue
				}

				if p.Min > p.Max {
					t.Errorf("%s/%s param %q: Min(%.2f) > Max(%.2f)",
						group.category, m.ID, p.Key, p.Min, p.Max)
				}
				if p.Default < p.Min || p.Default > p.Max {
					t.Errorf("%s/%s param %q: Default(%.2f) not in [%.2f, %.2f]",
						group.category, m.ID, p.Key, p.Default, p.Min, p.Max)
				}
			}
		}
	}
}
