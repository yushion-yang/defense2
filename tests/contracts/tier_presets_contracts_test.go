// tier_presets_contracts_test.go — 塔属性档位预设契约测试。
// 验证 config/towers/tier-presets.json 的结构完整性和数值合法性。
package contracts_test

import (
	"testing"

	"defense2/internal/config"
)

// TestTierPresetsLoads 验证 tier-presets.json 加载无错误。
func TestTierPresetsLoads(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}
	if tp == nil {
		t.Fatal("LoadTierPresets 返回 nil")
	}
}

// TestTierPresetsAllDimensionsComplete 验证 damage/attackSpeed/range 各有 S/B/D 三档。
func TestTierPresetsAllDimensionsComplete(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	for dimName, at := range dims {
		for _, tier := range config.TierNames {
			if _, ok := at.Tiers[tier]; !ok {
				t.Errorf("%s: 缺少档位 %q", dimName, tier)
			}
		}
	}
}

// TestTierPresetsValuesPositive 验证每档 base > 0, potential >= 0。
func TestTierPresetsValuesPositive(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	for dimName, at := range dims {
		if at.BasePotential < 0 {
			t.Errorf("%s: basePotential=%.2f 应 >= 0", dimName, at.BasePotential)
		}
		for tier, tv := range at.Tiers {
			t.Run(dimName+"/"+tier, func(t *testing.T) {
				if tv.Base <= 0 {
					t.Errorf("base=%.2f 应 > 0", tv.Base)
				}
				if tv.Potential < 0 {
					t.Errorf("potential=%.2f 应 >= 0", tv.Potential)
				}
			})
		}
	}
}

// TestTierPresetsBaseOrderDescending 验证 S.base > B.base > D.base（高档基础值更高）。
func TestTierPresetsBaseOrderDescending(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	order := config.TierNames // S, B, D
	for dimName, at := range dims {
		t.Run(dimName, func(t *testing.T) {
			for i := 1; i < len(order); i++ {
				prev := at.Tiers[order[i-1]]
				curr := at.Tiers[order[i]]
				if curr.Base >= prev.Base {
					t.Errorf("%s.base=%.2f 应 < %s.base=%.2f", order[i], curr.Base, order[i-1], prev.Base)
				}
			}
		})
	}
}

// TestTierPresetsPotentialOrderAscending 验证 S.potential < B.potential < D.potential
// （低档基础值低但潜力更高，高强度时追平）。
func TestTierPresetsPotentialOrderAscending(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	order := config.TierNames // S, B, D
	for dimName, at := range dims {
		t.Run(dimName, func(t *testing.T) {
			for i := 1; i < len(order); i++ {
				prev := at.Tiers[order[i-1]]
				curr := at.Tiers[order[i]]
				if curr.Potential <= prev.Potential {
					t.Errorf("%s.potential=%.2f 应 > %s.potential=%.2f", order[i], curr.Potential, order[i-1], prev.Potential)
				}
			}
		})
	}
}
