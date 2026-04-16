// prebuilt_blueprint_test.go — 预制蓝图与经典预设的一致性契约测试。
//
// 验证从预制蓝图路径生成的 TowerDef 与旧 loadClassicTowerDefs 路径
// 的属性完全一致，确保模式切换不影响游戏平衡。
package contracts_test

import (
	"math"
	"testing"

	"defense2/internal/config"
	"defense2/internal/core/tower/descriptor"
)

// TestPrebuiltBlueprintMatchesClassic 验证预制蓝图和经典预设生成的 TowerDef 属性一致。
func TestPrebuiltBlueprintMatchesClassic(t *testing.T) {
	// 确保 tier-presets 和 classic-presets 已加载
	if config.GlobalTierPresets() == nil {
		if _, err := config.LoadTierPresets(); err != nil {
			t.Fatalf("load tier presets: %v", err)
		}
	}
	if config.GlobalClassicPresets() == nil {
		if err := config.LoadClassicPresets(); err != nil {
			t.Fatalf("load classic presets: %v", err)
		}
	}

	// 旧路径数据
	presets := config.GlobalClassicPresets()
	if presets == nil {
		t.Fatal("classic presets not loaded")
	}
	tp := config.GlobalTierPresets()
	if tp == nil {
		t.Fatal("tier presets not loaded")
	}

	// 新路径数据
	newDefs := descriptor.LoadPrebuiltBlueprintDefs()
	if len(newDefs) == 0 {
		t.Fatal("no prebuilt blueprint defs loaded")
	}

	if len(newDefs) != len(presets.Towers) {
		t.Fatalf("count mismatch: old=%d new=%d", len(presets.Towers), len(newDefs))
	}

	for i, p := range presets.Towers {
		newDef := newDefs[i]

		if p.Key != newDef.Key {
			t.Errorf("[%d] key mismatch: old=%s new=%s", i, p.Key, newDef.Key)
			continue
		}

		// 从旧路径手动计算期望值
		dmgTier := tp.Damage.Tiers[p.Tiers["damage"]]
		spdTier := tp.AttackSpeed.Tiers[p.Tiers["atkSpeed"]]
		rngTier := tp.Range.Tiers[p.Tiers["range"]]

		wantDmgBase := dmgTier.Base
		wantDmgPot := dmgTier.Potential
		wantSpdBase := spdTier.Base
		wantSpdPot := spdTier.Potential
		wantRngBase := rngTier.Base
		wantRngPot := rngTier.Potential

		// 专精加成
		switch p.Specialty {
		case "damage":
			wantDmgPot += tp.Damage.BasePotential
		case "atkSpeed":
			wantSpdPot += tp.AttackSpeed.BasePotential
		case "range":
			wantRngPot += tp.Range.BasePotential
		}

		assertClose(t, p.Key+".CfgBaseDamage", newDef.CfgBaseDamage, wantDmgBase)
		assertClose(t, p.Key+".PotentialDamage", newDef.PotentialDamage, wantDmgPot)
		assertClose(t, p.Key+".CfgBaseSpeed", newDef.CfgBaseSpeed, wantSpdBase)
		assertClose(t, p.Key+".PotentialSpeed", newDef.PotentialSpeed, wantSpdPot)
		assertClose(t, p.Key+".CfgBaseRange", newDef.CfgBaseRange, wantRngBase)
		assertClose(t, p.Key+".PotentialRange", newDef.PotentialRange, wantRngPot)

		if newDef.Label != p.Name {
			t.Errorf("%s: label mismatch: got %q, want %q", p.Key, newDef.Label, p.Name)
		}
		if newDef.Category != p.Category {
			t.Errorf("%s: category mismatch: got %q, want %q", p.Key, newDef.Category, p.Category)
		}
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Errorf("%s: got %.4f, want %.4f", name, got, want)
	}
}
