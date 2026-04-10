// tier_presets_contracts_test.go — 塔属性档位预设契约测试。
// 验证 config/towers/tier-presets.json 的结构完整性、范围合法性和档位排序。
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

// TestTierPresetsAllDimensionsComplete 验证 damage/attackSpeed/range 各有 S/A/B/C/D 五档。
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

// TestTierPresetsRangeValid 验证每档 min > 0, max > min, min <= ref <= max。
func TestTierPresetsRangeValid(t *testing.T) {
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
		for tier, r := range at.Tiers {
			t.Run(dimName+"/"+tier, func(t *testing.T) {
				if r.Min <= 0 {
					t.Errorf("min=%.2f 应 > 0", r.Min)
				}
				if r.Max <= r.Min {
					t.Errorf("max=%.2f 应 > min=%.2f", r.Max, r.Min)
				}
				if r.Ref < r.Min || r.Ref > r.Max {
					t.Errorf("ref=%.2f 应在 [min=%.2f, max=%.2f] 范围内", r.Ref, r.Min, r.Max)
				}
			})
		}
	}
}

// TestTierPresetsOrderDescending 验证 S.ref > A.ref > B.ref > C.ref > D.ref。
func TestTierPresetsOrderDescending(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	order := config.TierNames // S, A, B, C, D
	for dimName, at := range dims {
		t.Run(dimName, func(t *testing.T) {
			for i := 1; i < len(order); i++ {
				prev := at.Tiers[order[i-1]]
				curr := at.Tiers[order[i]]
				if curr.Ref >= prev.Ref {
					t.Errorf("%s.ref=%.2f 应 < %s.ref=%.2f", order[i], curr.Ref, order[i-1], prev.Ref)
				}
			}
		})
	}
}

// TestTierPresetsNonOverlapping 验证高档 min >= 低档 max（档位间无交叉）。
func TestTierPresetsNonOverlapping(t *testing.T) {
	tp, err := config.LoadTierPresets()
	if err != nil {
		t.Fatalf("LoadTierPresets 失败: %v", err)
	}

	dims := map[string]config.AttrTiers{
		"damage":      tp.Damage,
		"attackSpeed": tp.AttackSpeed,
		"range":       tp.Range,
	}
	order := config.TierNames
	for dimName, at := range dims {
		t.Run(dimName, func(t *testing.T) {
			for i := 1; i < len(order); i++ {
				higher := at.Tiers[order[i-1]]
				lower := at.Tiers[order[i]]
				if higher.Min < lower.Max {
					t.Errorf("%s.min=%.2f 应 >= %s.max=%.2f（档位间不应交叉）",
						order[i-1], higher.Min, order[i], lower.Max)
				}
			}
		})
	}
}
