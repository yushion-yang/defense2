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

	// slowPower: base=0.15, potential=0.25
	// 强度100: 0.15 + 0.25 * 1.0 = 0.40
	v := slow.CalcScale(100)
	if math.Abs(v-0.40) > 1e-9 {
		t.Errorf("强度100: factor=%.3f, 期望0.40", v)
	}

	// 强度200: 0.15 + 0.25 * 2.0 = 0.65
	v2 := slow.CalcScale(200)
	if math.Abs(v2-0.65) > 1e-9 {
		t.Errorf("强度200: factor=%.3f, 期望0.65", v2)
	}
}

func TestAbilityDef_Param(t *testing.T) {
	table, _ := config.LoadAbilityTable()

	// slowPower: param=1.0, paramDim=duration
	slow := table["slowPower"]
	if !slow.HasParam() {
		t.Error("slowPower应有参数")
	}
	if slow.Param != 1.0 {
		t.Errorf("slowPower param=%.1f, 期望1.0", slow.Param)
	}
	if slow.ParamDim != "duration" {
		t.Errorf("slowPower paramDim=%s, 期望duration", slow.ParamDim)
	}

	// stackDamage: 无参数
	sd := table["stackDamage"]
	if sd.HasParam() {
		t.Error("stackDamage不应有参数")
	}
}

func TestAbilityDef_MultiTargetScale(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	mt := table["multiTarget"]

	// multiTarget 现在有缩放维度（targets），base=1 potential=1
	if !mt.HasScale() {
		t.Error("multiTarget应有缩放维度")
	}
	// 强度100: 1 + 1*1 = 2 目标
	if mt.CalcScale(100) != 2 {
		t.Errorf("强度100目标数=%.0f, 期望2", mt.CalcScale(100))
	}
	// 强度200: 1 + 1*2 = 3 目标
	if mt.CalcScale(200) != 3 {
		t.Errorf("强度200目标数=%.0f, 期望3", mt.CalcScale(200))
	}
}

func TestAbilityDef_BounceScalesMaxBounces(t *testing.T) {
	table, _ := config.LoadAbilityTable()
	bounce := table["bounce"]
	if bounce.ScaleDim != "maxBounces" {
		t.Errorf("bounce scaleDim=%s, 期望maxBounces", bounce.ScaleDim)
	}
	// 强度100: 1 + 1*1.0 = 2
	v := bounce.CalcScale(100)
	if math.Abs(v-2) > 1e-9 {
		t.Errorf("强度100: maxBounces=%.1f, 期望2", v)
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
