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
	if len(table) < 26 {
		t.Errorf("能力数=%d, 期望>=26", len(table))
	}

	// 验证结构统一性：每个能力都有 type/label/category
	for key, def := range table {
		if def.Type != key {
			t.Errorf("%s: type=%s 不匹配 key", key, def.Type)
		}
		if def.Label == "" {
			t.Errorf("%s: label为空", key)
		}
		if def.Category == "" {
			t.Errorf("%s: category为空", key)
		}
	}
}

func TestAbilityDef_CalcScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	slow := table["slowPower"]

	// 自洽性验证：CalcScale 结果必须符合公式 base + potential * (strength / 100)
	expected100 := slow.Base + slow.Potential*(100.0/100.0)
	if math.Abs(slow.CalcScale(100)-expected100) > 1e-9 {
		t.Errorf("强度100: factor=%.6f, 期望%.6f (base+potential*1.0)", slow.CalcScale(100), expected100)
	}

	expected200 := slow.Base + slow.Potential*(200.0/100.0)
	if math.Abs(slow.CalcScale(200)-expected200) > 1e-9 {
		t.Errorf("强度200: factor=%.6f, 期望%.6f (base+potential*2.0)", slow.CalcScale(200), expected200)
	}
}

func TestAbilityDef_Param(t *testing.T) {
	table, _ := config.LoadAbilityTable()

	// slowPower: 有参数，paramDim=duration（语义不变式）
	slow := table["slowPower"]
	if !slow.HasParam() {
		t.Error("slowPower应有参数")
	}
	if slow.Param <= 0 {
		t.Errorf("slowPower param=%.3f, 期望 > 0", slow.Param)
	}
	if slow.ParamDim != "duration" {
		t.Errorf("slowPower paramDim=%s, 期望duration", slow.ParamDim)
	}

	// stackDamage: 无参数（结构性断言）
	sd := table["stackDamage"]
	if sd.HasParam() {
		t.Error("stackDamage不应有参数")
	}
}

func TestAbilityDef_MultiTargetScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	mt := table["multiTarget"]

	// multiTarget 有缩放维度（targets）
	if !mt.HasScale() {
		t.Error("multiTarget应有缩放维度")
	}

	// 自洽性验证：CalcScale 结果符合公式
	expected100 := mt.Base + mt.Potential*(100.0/100.0)
	if mt.CalcScale(100) != expected100 {
		t.Errorf("强度100目标数=%.1f, 期望%.1f (base+potential*1.0)", mt.CalcScale(100), expected100)
	}

	expected200 := mt.Base + mt.Potential*(200.0/100.0)
	if mt.CalcScale(200) != expected200 {
		t.Errorf("强度200目标数=%.1f, 期望%.1f (base+potential*2.0)", mt.CalcScale(200), expected200)
	}
}

func TestAbilityDef_BounceScalesMaxBounces(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	bounce := table["bounce"]
	if bounce.ScaleDim != "maxBounces" {
		t.Errorf("bounce scaleDim=%s, 期望maxBounces", bounce.ScaleDim)
	}

	// 自洽性验证：CalcScale 结果符合公式
	expected := bounce.Base + bounce.Potential*(100.0/100.0)
	if math.Abs(bounce.CalcScale(100)-expected) > 1e-9 {
		t.Errorf("强度100: maxBounces=%.3f, 期望%.3f (base+potential*1.0)", bounce.CalcScale(100), expected)
	}
}

func TestAbilityDef_SplashScalesRatio(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	splash := table["splash"]
	if splash.ScaleDim != "ratio" {
		t.Errorf("splash scaleDim=%s, 期望ratio", splash.ScaleDim)
	}
	// 固定参数是 radius
	if splash.ParamDim != "radius" {
		t.Errorf("splash paramDim=%s, 期望radius", splash.ParamDim)
	}
}

func TestAbilityDef_PercentHpNoBossCap(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	php := table["percentHpDamage"]
	// bossCap 已移除为全局规则，不在能力参数中
	if php.HasParam() {
		t.Error("percentHpDamage不应有参数(bossCap是全局规则)")
	}
}
