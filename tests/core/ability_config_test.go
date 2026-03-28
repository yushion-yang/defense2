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

	tests := []struct {
		key      string
		label    string
		category string
		scaleDim string
	}{
		{"onHitSlow", "减速", "control", "factor"},
		{"bounce", "弹射", "combat", "range"},
		{"splash", "溅射", "combat", "radius"},
		{"flatDamage", "固伤", "combat", "damage"},
	}
	for _, tt := range tests {
		def := table[tt.key]
		if def == nil {
			t.Errorf("能力 %s 不存在", tt.key)
			continue
		}
		if def.Type != tt.key {
			t.Errorf("%s type=%s, 期望%s", tt.key, def.Type, tt.key)
		}
		if def.Label != tt.label {
			t.Errorf("%s label=%s, 期望%s", tt.key, def.Label, tt.label)
		}
		if def.ScaleDim != tt.scaleDim {
			t.Errorf("%s scaleDim=%s, 期望%s", tt.key, def.ScaleDim, tt.scaleDim)
		}
	}
}

func TestAbilityDef_CalcScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	slow := table["onHitSlow"]

	// 强度100: 0.10 + 0.22 * 1.0 = 0.32
	v100 := slow.CalcScale(100)
	if math.Abs(v100-0.32) > 1e-9 {
		t.Errorf("强度100: factor=%.3f, 期望0.32", v100)
	}

	// 强度200: 0.10 + 0.22 * 2.0 = 0.54
	v200 := slow.CalcScale(200)
	if math.Abs(v200-0.54) > 1e-9 {
		t.Errorf("强度200: factor=%.3f, 期望0.54", v200)
	}

	// 强度0: 0.10
	v0 := slow.CalcScale(0)
	if math.Abs(v0-0.10) > 1e-9 {
		t.Errorf("强度0: factor=%.3f, 期望0.10", v0)
	}
}

func TestAbilityDef_GetParam(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	slow := table["onHitSlow"]

	dur := slow.GetParam("duration", 0)
	if dur != 1.4 {
		t.Errorf("duration=%.1f, 期望1.4", dur)
	}

	missing := slow.GetParam("nonExistent", 99)
	if missing != 99 {
		t.Errorf("缺失参数应返回99, 实际%.0f", missing)
	}
}

func TestAbilityDef_NoScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	mt := table["multiTarget"]

	if mt.HasScale() {
		t.Error("multiTarget不应有缩放维度")
	}
	if mt.CalcScale(200) != 0 {
		t.Error("无缩放时CalcScale应返回0")
	}
}

func TestAbilityDef_FormatScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()

	// 百分比类（base < 1）
	slow := table["onHitSlow"]
	s := slow.FormatScale(100)
	if s == "" {
		t.Error("onHitSlow应有缩放展示")
	}

	// 绝对值类（base >= 1）
	splash := table["splash"]
	s2 := splash.FormatScale(100)
	if s2 == "" {
		t.Error("splash应有缩放展示")
	}

	// 无缩放
	mt := table["multiTarget"]
	if mt.FormatScale(100) != "" {
		t.Error("multiTarget不应有缩放展示")
	}
}
