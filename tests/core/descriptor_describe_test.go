package core_test

import (
	"strings"
	"testing"

	"defense2/internal/core/tower/descriptor"
)

// ── GenerateDescription 基础测试 ──

func TestGenerateDescription_Nil(t *testing.T) {
	if got := descriptor.GenerateDescription(nil); got != "" {
		t.Errorf("expected empty string for nil descriptor, got %q", got)
	}
}

func TestGenerateDescription_NoPipeline_LabelOnly(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "test",
		Label: "测试能力",
	}
	if got := descriptor.GenerateDescription(desc); got != "测试能力" {
		t.Errorf("expected label only, got %q", got)
	}
}

func TestGenerateDescription_AttackStyleOnly(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:          "test_scatter",
		Label:       "散射攻击",
		AttackStyle: "scatter",
	}
	got := descriptor.GenerateDescription(desc)
	if !strings.Contains(got, "散射攻击") || !strings.Contains(got, "散射") {
		t.Errorf("expected scatter label, got %q", got)
	}
}

func TestGenerateDescription_SinglePipeline_StunAllInRange(t *testing.T) {
	// 模拟 "永久冻结场": 每帧 → 全范围 → 眩晕 0.1s
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_perma_freeze",
		Label: "永久冻结场",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.AllInRangeSelector{},
				Effects: []descriptor.Effect{
					descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.1}},
				},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	// 应包含: "每帧", "全范围", "眩晕"
	if !strings.Contains(got, "每帧") {
		t.Errorf("missing trigger label '每帧' in %q", got)
	}
	if !strings.Contains(got, "全范围") {
		t.Errorf("missing selector label '全范围' in %q", got)
	}
	if !strings.Contains(got, "眩晕") {
		t.Errorf("missing effect label '眩晕' in %q", got)
	}
}

func TestGenerateDescription_WithConditions(t *testing.T) {
	// 模拟 "Boss猎手" 第一管线: 命中时 [是Boss] → 当前目标 → 净化×3 + 沉默
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_boss_hunter",
		Label: "Boss猎手",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:    descriptor.TriggerOnHit,
				Conditions: []descriptor.Condition{descriptor.IsBossCondition{}},
				Selector:   descriptor.CurrentTargetSelector{},
				Effects: []descriptor.Effect{
					descriptor.PurgeEffect{Count: 3},
					descriptor.SilenceEffect{},
				},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	if !strings.Contains(got, "命中时") {
		t.Errorf("missing '命中时' in %q", got)
	}
	if !strings.Contains(got, "是Boss") {
		t.Errorf("missing '是Boss' in %q", got)
	}
	if !strings.Contains(got, "当前目标") {
		t.Errorf("missing '当前目标' in %q", got)
	}
	if !strings.Contains(got, "净化×3") {
		t.Errorf("missing '净化×3' in %q", got)
	}
	if !strings.Contains(got, "沉默") {
		t.Errorf("missing '沉默' in %q", got)
	}
}

func TestGenerateDescription_MultiplePipelines(t *testing.T) {
	// 两条管线用 "; " 连接
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_multi",
		Label: "多管线",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnHit,
				Selector: descriptor.CurrentTargetSelector{},
				Effects:  []descriptor.Effect{descriptor.SilenceEffect{}},
			},
			{
				Trigger:  descriptor.TriggerOnKill,
				Selector: descriptor.AllInRangeSelector{},
				Effects:  []descriptor.Effect{descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 1}}},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	if !strings.Contains(got, ";") {
		t.Errorf("expected '; ' separator for multiple pipelines, got %q", got)
	}
	if !strings.Contains(got, "命中时") || !strings.Contains(got, "击杀时") {
		t.Errorf("expected both triggers, got %q", got)
	}
}

func TestGenerateDescription_LinearScaler(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_linear",
		Label: "线性",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnHit,
				Selector: descriptor.CurrentTargetSelector{},
				Effects: []descriptor.Effect{
					descriptor.DamageEffect{
						Mode:  descriptor.DmgRatio,
						Value: descriptor.LinearScaler{Base: 1.5, Potential: 0.5},
					},
				},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	// 线性 scaler: base=1.5 显示为 "1.5"，potential=0.5 显示为 "50%"（因为 <1 视为百分比）
	if !strings.Contains(got, "1.5+50%*str") {
		t.Errorf("expected linear scaler format with percentage, got %q", got)
	}
}

func TestGenerateDescription_BuffEffect(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_buff",
		Label: "增益",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnTick,
				Selector: descriptor.NearbyAlliesSelector{Radius: 150},
				Effects: []descriptor.Effect{
					descriptor.BuffEffect{
						Stat:  "damage",
						Bonus: descriptor.FixedScaler{Value: 0.05},
					},
				},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	if !strings.Contains(got, "友方塔(150)") {
		t.Errorf("missing selector label, got %q", got)
	}
	if !strings.Contains(got, "增益damage") {
		t.Errorf("missing buff effect label, got %q", got)
	}
}

func TestGenerateDescription_AoeSelector(t *testing.T) {
	desc := &descriptor.AbilityDescriptor{
		ID:    "ca_aoe",
		Label: "范围",
		Pipelines: []descriptor.Pipeline{
			{
				Trigger:  descriptor.TriggerOnKill,
				Selector: descriptor.AoeRadiusSelector{Radius: descriptor.FixedScaler{Value: 80}},
				Effects: []descriptor.Effect{
					descriptor.StunEffect{Duration: descriptor.FixedScaler{Value: 0.8}},
				},
			},
		},
	}
	got := descriptor.GenerateDescription(desc)
	if !strings.Contains(got, "范围(80)") {
		t.Errorf("missing aoe selector label, got %q", got)
	}
}
