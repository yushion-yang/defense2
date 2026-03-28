package core_test

import (
	"math"
	"testing"

	"defense2/internal/config"
)

func TestLoadAbilityTable(t *testing.T) {
	table, err := config.LoadAbilityTable()
	if err != nil {
		t.Fatalf("加载能力表失败: %v", err)
	}
	if len(table) < 20 {
		t.Errorf("能力数=%d, 期望至少20", len(table))
	}

	// 验证几个关键能力的结构
	tests := []struct {
		key      string
		label    string
		category string
	}{
		{"onHitSlow", "减速", "control"},
		{"bounce", "弹射", "combat"},
		{"splash", "溅射", "combat"},
		{"damageUpAura", "伤害光环", "aura"},
		{"goldPassive", "被动产金", "economy"},
	}
	for _, tt := range tests {
		def, ok := table[tt.key]
		if !ok {
			t.Errorf("能力 %s 不存在", tt.key)
			continue
		}
		if def.Type != tt.key {
			t.Errorf("%s type=%s, 期望%s", tt.key, def.Type, tt.key)
		}
		if def.Label != tt.label {
			t.Errorf("%s label=%s, 期望%s", tt.key, def.Label, tt.label)
		}
		if def.Category != tt.category {
			t.Errorf("%s category=%s, 期望%s", tt.key, def.Category, tt.category)
		}
	}
}

func TestAbilityDef_CalcParam(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	slow := table["onHitSlow"]
	if slow == nil {
		t.Fatal("onHitSlow 不存在")
	}

	// 强度100: base + potential * 1.0
	factor100 := slow.CalcParam("factor", 100, 0)
	expected := slow.GetBase("factor", 0) + slow.GetPotential("factor", 0)*1.0
	if math.Abs(factor100-expected) > 1e-9 {
		t.Errorf("强度100: factor=%.3f, 期望%.3f", factor100, expected)
	}

	// 强度200: base + potential * 2.0
	factor200 := slow.CalcParam("factor", 200, 0)
	expected200 := slow.GetBase("factor", 0) + slow.GetPotential("factor", 0)*2.0
	if math.Abs(factor200-expected200) > 1e-9 {
		t.Errorf("强度200: factor=%.3f, 期望%.3f", factor200, expected200)
	}

	// 强度0: 只有 base
	factor0 := slow.CalcParam("factor", 0, 0)
	if math.Abs(factor0-slow.GetBase("factor", 0)) > 1e-9 {
		t.Errorf("强度0: factor=%.3f, 期望base=%.3f", factor0, slow.GetBase("factor", 0))
	}

	// 不存在的参数返回默认值
	missing := slow.CalcParam("nonExistent", 100, 99)
	if missing != 99 {
		t.Errorf("缺失参数应返回99, 实际%.0f", missing)
	}
}

func TestAbilityDef_CalcAllParams(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	splash := table["splash"]
	if splash == nil {
		t.Fatal("splash 不存在")
	}

	params := splash.CalcAllParams(100)
	// 强度100: base + potential
	expectedRadius := splash.GetBase("radius", 0) + splash.GetPotential("radius", 0)
	if math.Abs(params["radius"]-expectedRadius) > 1e-9 {
		t.Errorf("radius=%.1f, 期望%.1f", params["radius"], expectedRadius)
	}
	expectedRatio := splash.GetBase("ratio", 0) + splash.GetPotential("ratio", 0)
	if math.Abs(params["ratio"]-expectedRatio) > 1e-9 {
		t.Errorf("ratio=%.2f, 期望%.2f", params["ratio"], expectedRatio)
	}
}

func TestAbilityDef_NoParams(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	// shieldIgnore 无参数
	si := table["shieldIgnore"]
	if si == nil {
		t.Fatal("shieldIgnore 不存在")
	}
	params := si.CalcAllParams(100)
	if len(params) != 0 {
		t.Errorf("shieldIgnore 应无参数, 实际%d个", len(params))
	}
}
