// preset_ability_test.go — 回归测试：预制塔同 category 能力丢失。
//
// 3 个预制塔有同 category 的能力对，AddAbility 的 slot 限制导致第二个被拒绝。
// cl_hydra: critAura + rangeAura (都是 buff)
// cl_ricochet: attackSpeedAura + goldPassive (都是 buff)
// cl_fortress: curseZone + weakenZone (都是 zone)
package regression_test

import (
	"testing"

	"defense2/internal/core/tower"
)

// TestRegression_PresetTowerKeepsBothAbilities 验证预制塔的所有能力都被保留。
func TestRegression_PresetTowerKeepsBothAbilities(t *testing.T) {
	tests := []struct {
		name      string
		abilities []string
	}{
		{"hydra_buff_pair", []string{"critAura", "rangeAura"}},
		{"ricochet_buff_pair", []string{"attackSpeedAura", "goldPassive"}},
		{"fortress_zone_pair", []string{"curseZone", "weakenZone"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &tower.Tower{}
			tower.ApplyPresetAbilities(tw, tt.abilities)

			for _, want := range tt.abilities {
				found := false
				for _, got := range tw.Abilities {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ability %q not found in tower.Abilities %v", want, tw.Abilities)
				}
			}
		})
	}
}
