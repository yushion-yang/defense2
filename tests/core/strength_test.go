package core_test

import (
	"math"
	"testing"

	"defense2/internal/core/strength"
)

// ============================================================
// 战力数据测试
// ============================================================

func TestStrength_Default(t *testing.T) {
	s := strength.NewStrengthData()
	if math.Abs(s.Effective()-100) > 1e-9 {
		t.Errorf("默认战力=%.1f, 期望100", s.Effective())
	}
	if math.Abs(s.DamageMultiplier()-1.0) > 1e-9 {
		t.Errorf("默认伤害乘数=%.2f, 期望1.0", s.DamageMultiplier())
	}
}

func TestStrength_Permanent(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(50)
	// 100 + 50 = 150
	if math.Abs(s.Effective()-150) > 1e-9 {
		t.Errorf("永久加成后=%.1f, 期望150", s.Effective())
	}
}

func TestStrength_Temp(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetTemp("aura1", 20)
	s.SetTemp("aura2", 30)
	// 100 + 20 + 30 = 150
	if math.Abs(s.Effective()-150) > 1e-9 {
		t.Errorf("临时加成后=%.1f, 期望150", s.Effective())
	}

	s.RemoveTemp("aura1")
	// 100 + 30 = 130
	if math.Abs(s.Effective()-130) > 1e-9 {
		t.Errorf("移除后=%.1f, 期望130", s.Effective())
	}
}

func TestStrength_EnemyDebuffs(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetEnemyMul("debuff1", 0.5) // 减半
	// 100 * 0.5 = 50
	if math.Abs(s.Effective()-50) > 1e-9 {
		t.Errorf("乘法减益后=%.1f, 期望50", s.Effective())
	}

	s.SetEnemySub("flat1", 10)
	// 100 * 0.5 - 10 = 40
	if math.Abs(s.Effective()-40) > 1e-9 {
		t.Errorf("减法减益后=%.1f, 期望40", s.Effective())
	}
}

func TestStrength_EffectiveFloor(t *testing.T) {
	s := strength.NewStrengthData()
	s.SetEnemySub("massive", 200)
	// max(0, 100 - 200) = 0
	if s.Effective() != 0 {
		t.Errorf("有效值应为0(下限), 实际%.1f", s.Effective())
	}
}

func TestStrength_ClearTransient(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(20)
	s.SetTemp("chain", 30)
	s.SetEnemyMul("curse", 0.5)
	s.SetEnemySub("drain", 5)

	s.ClearTransient()
	// 只保留 base(100) + permanent(20) = 120
	if math.Abs(s.Effective()-120) > 1e-9 {
		t.Errorf("ClearTransient后=%.1f, 期望120", s.Effective())
	}
}

func TestStrength_DamageMultiplier(t *testing.T) {
	s := strength.NewStrengthData()
	s.AddPermanent(100) // 200 effective
	// 200/100 = 2.0
	if math.Abs(s.DamageMultiplier()-2.0) > 1e-9 {
		t.Errorf("200战力伤害乘数=%.2f, 期望2.0", s.DamageMultiplier())
	}
}

func TestStrength_SpeedMultiplier(t *testing.T) {
	s := strength.NewStrengthData()
	// 100: 速度乘数=1.0
	if math.Abs(s.SpeedMultiplier()-1.0) > 1e-9 {
		t.Errorf("100战力速度乘数=%.3f, 期望1.0", s.SpeedMultiplier())
	}

	s.AddPermanent(100) // 200 effective, excess=100
	// 1.0 + 100*0.005 = 1.5
	if math.Abs(s.SpeedMultiplier()-1.5) > 1e-9 {
		t.Errorf("200战力速度乘数=%.3f, 期望1.5", s.SpeedMultiplier())
	}
}

// ============================================================
// 战力配置测试
// ============================================================

func TestStrengthConfig_CalcAttribute(t *testing.T) {
	cfg := &strength.StrengthConfig{
		Bindings: map[string]*strength.StrengthBinding{
			"attackDamage": {Base: 10, Potential: 50},
			"attackSpeed":  {Base: 0.3, Potential: 0.5},
		},
	}

	// 100 strength: 10 + 50*1.0 = 60
	val := cfg.CalcAttribute("attackDamage", 100, 0)
	if math.Abs(val-60) > 1e-9 {
		t.Errorf("100战力attackDamage=%.1f, 期望60", val)
	}

	// 200 strength: 10 + 50*2.0 = 110
	val2 := cfg.CalcAttribute("attackDamage", 200, 0)
	if math.Abs(val2-110) > 1e-9 {
		t.Errorf("200战力attackDamage=%.1f, 期望110", val2)
	}

	// 0 strength: 10 + 50*0 = 10 (只有base)
	val3 := cfg.CalcAttribute("attackDamage", 0, 0)
	if math.Abs(val3-10) > 1e-9 {
		t.Errorf("0战力attackDamage=%.1f, 期望10", val3)
	}

	// 不存在的属性返回fallback
	val4 := cfg.CalcAttribute("unknown", 100, 42)
	if math.Abs(val4-42) > 1e-9 {
		t.Errorf("未知属性=%.1f, 期望fallback=42", val4)
	}
}

func TestStrengthConfig_IsLinked(t *testing.T) {
	cfg := &strength.StrengthConfig{
		Bindings: map[string]*strength.StrengthBinding{
			"attackDamage": {Base: 10, Potential: 50},
		},
	}
	if !cfg.IsLinked("attackDamage") {
		t.Error("attackDamage应有绑定")
	}
	if cfg.IsLinked("range") {
		t.Error("range不应有绑定")
	}
}

// ============================================================
// 链网络测试
// ============================================================

func TestChain_GroupFormation(t *testing.T) {
	towers := []strength.ChainTower{
		{Index: 0, X: 0, Y: 0},
		{Index: 1, X: 100, Y: 0},   // 距离0号100px < 150
		{Index: 2, X: 500, Y: 500}, // 远离
	}

	info := strength.RebuildChainNetwork(towers)

	// 0和1应在同组
	if info[0].GroupSize != 2 {
		t.Errorf("塔0组大小=%d, 期望2", info[0].GroupSize)
	}
	if info[0].GroupID != info[1].GroupID {
		t.Errorf("塔0和塔1应在同组, ID=%d/%d", info[0].GroupID, info[1].GroupID)
	}

	// bonus = 2 * 10 = 20
	if math.Abs(info[0].Bonus-20) > 1e-9 {
		t.Errorf("链加成=%.1f, 期望20", info[0].Bonus)
	}

	// 2号孤立，无加成
	if info[2].GroupSize > 1 {
		t.Errorf("塔2应孤立, 组大小=%d", info[2].GroupSize)
	}
	if info[2].Bonus != 0 {
		t.Errorf("孤立塔加成=%.1f, 期望0", info[2].Bonus)
	}
}

func TestChain_StrengthIntegration(t *testing.T) {
	s1 := strength.NewStrengthData()
	s2 := strength.NewStrengthData()
	towers := []strength.ChainTower{
		{Index: 0, X: 0, Y: 0, Strength: s1},
		{Index: 1, X: 50, Y: 0, Strength: s2},
	}

	strength.RebuildChainNetwork(towers)

	// 两塔链组大小2, bonus=20, 通过 SetTemp("chain", 20) 设置
	if math.Abs(s1.Effective()-120) > 1e-9 {
		t.Errorf("链接后s1战力=%.1f, 期望120", s1.Effective())
	}
	if math.Abs(s2.Effective()-120) > 1e-9 {
		t.Errorf("链接后s2战力=%.1f, 期望120", s2.Effective())
	}
}

func TestChain_LargeGroup(t *testing.T) {
	// 5 个紧密排列的塔
	towers := make([]strength.ChainTower, 5)
	for i := 0; i < 5; i++ {
		towers[i] = strength.ChainTower{Index: i, X: float64(i) * 50, Y: 0}
	}

	info := strength.RebuildChainNetwork(towers)

	// 所有塔应在同一组，大小5，bonus=50
	for i := 0; i < 5; i++ {
		if info[i].GroupSize != 5 {
			t.Errorf("塔%d组大小=%d, 期望5", i, info[i].GroupSize)
		}
		if math.Abs(info[i].Bonus-50) > 1e-9 {
			t.Errorf("塔%d链加成=%.1f, 期望50", i, info[i].Bonus)
		}
	}
}
