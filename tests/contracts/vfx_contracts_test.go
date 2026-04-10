// vfx_contracts_test.go — VFX 特效目录契约测试。
// 验证 config/visuals/vfx.json 的结构完整性和字段有效性。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
)

// TestVFXCatalogLoads 验证 vfx.json 加载无错误。
func TestVFXCatalogLoads(t *testing.T) {
	cat, err := config.LoadVFXCatalog()
	if err != nil {
		t.Fatalf("LoadVFXCatalog 失败: %v", err)
	}
	if cat == nil {
		t.Fatal("LoadVFXCatalog 返回 nil")
	}
}

// TestVFXCatalogCategoriesNonEmpty 验证至少有 3 个特效类别。
func TestVFXCatalogCategoriesNonEmpty(t *testing.T) {
	cat, err := config.LoadVFXCatalog()
	if err != nil {
		t.Fatalf("LoadVFXCatalog 失败: %v", err)
	}
	if len(cat.Categories) < 3 {
		t.Errorf("VFX 类别数=%d, 应 >= 3", len(cat.Categories))
	}
}

// TestVFXCatalogCategoriesHaveRequiredFields 验证每个类别有 id/name/label。
func TestVFXCatalogCategoriesHaveRequiredFields(t *testing.T) {
	cat, err := config.LoadVFXCatalog()
	if err != nil {
		t.Fatalf("LoadVFXCatalog 失败: %v", err)
	}
	for _, c := range cat.Categories {
		t.Run(c.ID, func(t *testing.T) {
			if c.ID == "" {
				t.Error("category id 为空")
			}
			if c.Name == "" {
				t.Error("category name 为空")
			}
			if c.Label == "" {
				t.Error("category label 为空")
			}
			if len(c.Effects) == 0 {
				t.Error("category effects 为空")
			}
		})
	}
}

// TestVFXCatalogEffectsHaveRequiredFields 验证每个特效有 id/name/label/description。
func TestVFXCatalogEffectsHaveRequiredFields(t *testing.T) {
	cat, err := config.LoadVFXCatalog()
	if err != nil {
		t.Fatalf("LoadVFXCatalog 失败: %v", err)
	}
	for _, c := range cat.Categories {
		for _, fx := range c.Effects {
			t.Run(c.ID+"/"+fx.ID, func(t *testing.T) {
				if fx.ID == "" {
					t.Error("effect id 为空")
				}
				if fx.Name == "" {
					t.Error("effect name 为空")
				}
				if fx.Label == "" {
					t.Error("effect label 为空")
				}
				if fx.Description == "" {
					t.Error("effect description 为空")
				}
			})
		}
	}
}

// TestVFXCatalogEffectIDsUnique 验证所有特效 ID 全局唯一。
func TestVFXCatalogEffectIDsUnique(t *testing.T) {
	cat, err := config.LoadVFXCatalog()
	if err != nil {
		t.Fatalf("LoadVFXCatalog 失败: %v", err)
	}
	seen := make(map[string]string) // id -> category
	for _, c := range cat.Categories {
		for _, fx := range c.Effects {
			if prev, dup := seen[fx.ID]; dup {
				t.Errorf("effect id %q 重复: 出现在 %q 和 %q", fx.ID, prev, c.ID)
			}
			seen[fx.ID] = c.ID
		}
	}
}
