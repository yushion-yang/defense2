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

	// ModifyStatEffect multiplier 是总倍率（如 1.5 = ×1.5），
	// Param 应存储增量（0.5 = +50%），这样 {p%} → 0.5*100 = "50%"。
	// Bug 回归检测：不做 -1 转换时 Param >= 1.0（显示为 >=100%）。
	if def.Param <= 0 || def.Param >= 1.0 {
		t.Errorf("enhance Param = %v, want (0, 1.0) 增量; Param>=1 说明未做 multiplier-1 转换", def.Param)
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

	// crit 管线: ChanceCondition(rate=linear) + CritEffect(multiplier=2.0)
	// ChanceCondition 填充了 ScaleDim="chance"，CritEffect 将 multiplier 放入 Param。
	// display 模板 "造成{p}倍暴击" 中 {p}=2.0 意为"造成 2 倍暴击伤害"。
	if def.Param < 1.99 || def.Param > 2.01 {
		t.Errorf("crit Param = %v, want ~2.0 (暴击倍率); 0 导致显示 '造成0倍暴击'", def.Param)
	}
}

// TestRegression_StunChanceDurationNotZero 验证 stunChance 的 Param 应为 0.5（持续秒数），不是 0。
// 旧值: ScaleDim="chance", Base=0.1, Potential=0.05, Param=0.5, ParamDim="duration"
func TestRegression_StunChanceDurationNotZero(t *testing.T) {
	table := descriptor.DeriveAbilityTable()
	def, ok := table["stunChance"]
	if !ok {
		t.Fatal("stunChance not found in derived table")
	}

	// stunChance: ChanceCondition(rate=linear 0.1/0.05) + StunEffect(duration=fixed 0.5)
	// ChanceCondition 应填主 scaler（chance），StunEffect 应填 Param（duration）。
	// Bug: StunEffect 无条件覆盖主 scaler，导致 chance 丢失、Param=0。
	if def.ScaleDim != "chance" {
		t.Errorf("stunChance ScaleDim = %q, want chance", def.ScaleDim)
	}
	if def.Param < 0.49 || def.Param > 0.51 {
		t.Errorf("stunChance Param = %v, want ~0.5 (duration); 0 导致显示 '眩晕0秒'", def.Param)
	}
	if def.ParamDim != "duration" {
		t.Errorf("stunChance ParamDim = %q, want duration", def.ParamDim)
	}
}

// TestRegression_StunDurationChanceNotZero 验证 stunDuration 的 Param 应为 0.25（概率），不是 0。
// 旧值: ScaleDim="duration", Base=0.2, Potential=0.1, Param=0.25, ParamDim="chance"
func TestRegression_StunDurationChanceNotZero(t *testing.T) {
	table := descriptor.DeriveAbilityTable()
	def, ok := table["stunDuration"]
	if !ok {
		t.Fatal("stunDuration not found in derived table")
	}

	// stunDuration: ChanceCondition(rate=fixed 0.25) + StunEffect(duration=linear 0.2/0.1)
	// StunEffect 的 duration 是主 scaler，ChanceCondition 应填 Param。
	if def.ScaleDim != "duration" {
		t.Errorf("stunDuration ScaleDim = %q, want duration", def.ScaleDim)
	}
	if def.Base < 0.19 || def.Base > 0.21 {
		t.Errorf("stunDuration Base = %v, want ~0.2", def.Base)
	}
	if def.Param < 0.24 || def.Param > 0.26 {
		t.Errorf("stunDuration Param = %v, want ~0.25 (chance); 0 导致显示 '有0%%概率'", def.Param)
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
