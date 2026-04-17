// derive_param_extraction_test.go — 回归测试：DeriveAbilityTable 参数提取 bug。
//
// Bug 1: enhance 的 ModifyStatEffect multiplier=1.8 被直接存入 Param，
//         {p%} 显示为 180% 而非 80%。
// Bug 2: crit 的 ChanceCondition 占据了 ScaleDim，导致 DamageEffect 被跳过，
//         {p} 显示为 0 而非 1。
// Bug 3: enhance 有 3 个 effects，只提取 Effects[0]，
//         Param2（range multiplier）丢失，{p2%} 显示为 0%。
package regression_test

import (
	"testing"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/core/tower/descriptor"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	if err := descriptor.InitDescriptorAbilities(config.GetDataFS()); err != nil {
		panic("init descriptor abilities: " + err.Error())
	}
}

// TestRegression_EnhanceMultiplierDisplayAs80Not180 验证 enhance 的 Param 应为 0.8（+80%），不是 1.8（180%）。
func TestRegression_EnhanceMultiplierDisplayAs80Not180(t *testing.T) {
	table := descriptor.DeriveAbilityTable()
	def, ok := table["enhance"]
	if !ok {
		t.Fatal("enhance not found in derived table")
	}

	// ModifyStatEffect multiplier=1.8 意为 "×1.8"（增加80%），
	// Param 应存储增量 0.8，这样 {p%} → 0.8*100 = "80%"。
	// Bug: 存的是 1.8，{p%} → 1.8*100 = "180%"
	if def.Param < 0.79 || def.Param > 0.81 {
		t.Errorf("enhance Param = %v, want ~0.8 (增量); 当前值导致显示为 180%% 而非 80%%", def.Param)
	}
	if def.ParamDim != "statBoost" {
		t.Errorf("enhance ParamDim = %q, want statBoost", def.ParamDim)
	}
}

// TestRegression_CritDamageParamNotZero 验证 crit 的 Param 应为 1.0（1倍暴击伤害），不是 0。
func TestRegression_CritDamageParamNotZero(t *testing.T) {
	table := descriptor.DeriveAbilityTable()
	def, ok := table["crit"]
	if !ok {
		t.Fatal("crit not found in derived table")
	}

	// crit 管线: ChanceCondition(rate=linear) + DamageEffect(ratio, value=1.0)
	// ChanceCondition 填充了 ScaleDim="chance"，之后 DamageEffect 应将 value 放入 Param。
	// Bug: DamageEffect 检查 ScaleDim != "" 后整个跳过，Param 留在 0。
	// DamageEffect ratio value=1.0 表示额外 100% 伤害，Param 存的就是 1.0。
	// display 模板 "造成{p}倍暴击" 中 {p}=1.0 意为"额外 1 倍伤害"（即总共 2 倍）。
	if def.Param < 0.99 || def.Param > 1.01 {
		t.Errorf("crit Param = %v, want ~1.0 (额外倍率); 0 导致显示 '造成0倍暴击'", def.Param)
	}
}

// TestRegression_EnhanceRangeParam2NotZero 验证 enhance 的 Param2 应为 0.2（射程+20%），不是 0。
func TestRegression_EnhanceRangeParam2NotZero(t *testing.T) {
	table := descriptor.DeriveAbilityTable()
	def, ok := table["enhance"]
	if !ok {
		t.Fatal("enhance not found in derived table")
	}

	// enhance 有 3 个 ModifyStatEffect: damage=1.8, speed=1.8, range=1.2
	// 只处理 Effects[0] 导致 range 的 1.2 从未提取到 Param2。
	// Param2 应为 0.2（增量），这样 {p2%} → 0.2*100 = "20%"。
	if def.Param2 < 0.19 || def.Param2 > 0.21 {
		t.Errorf("enhance Param2 = %v, want ~0.2 (射程增量); 0 导致显示 '射程+0%%'", def.Param2)
	}
	if def.Param2Dim != "statBoost" {
		t.Errorf("enhance Param2Dim = %q, want statBoost", def.Param2Dim)
	}
}
