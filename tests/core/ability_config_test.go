package core_test

import (
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

	// 验证几个关键能力
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
		if def.Label != tt.label {
			t.Errorf("%s label=%s, 期望%s", tt.key, def.Label, tt.label)
		}
		if def.Category != tt.category {
			t.Errorf("%s category=%s, 期望%s", tt.key, def.Category, tt.category)
		}
	}
}

func TestAbilityDef_GetParam(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	slow := table["onHitSlow"]
	if slow == nil {
		t.Fatal("onHitSlow 不存在")
	}
	factor := slow.GetParam("factor", 0)
	if factor != 0.32 {
		t.Errorf("factor=%.2f, 期望0.32", factor)
	}
	missing := slow.GetParam("nonExistent", 99)
	if missing != 99 {
		t.Errorf("缺失参数应返回默认值99, 实际%.0f", missing)
	}
}
