package contracts

import (
	"fmt"
	"testing"

	"defense2/internal/core/strength"
	"defense2/internal/core/tower"
)

func TestStrengthDrain_TwoDrainers_AttackInterval(t *testing.T) {
	// 模拟基础塔（与 towers.json basic 一致）
	tw := &tower.Tower{
		BaseDamage:      8,
		PotentialDamage: 14,
		BaseSpeed:       0.4,
		PotentialSpeed:  0.6,
		BaseRange:       150,
		PotentialRange:  30,
		Strength:        strength.NewStrengthData(), // Base=100
	}

	// 无削弱时
	tw.RecalcStats()
	normalSpeed := tw.AttackSpeed
	normalInterval := 1.0 / normalSpeed
	normalDmg := tw.Damage
	normalRange := tw.Range
	fmt.Printf("无削弱:  强度=%.0f  攻速=%.2f/s  间隔=%.2fs  伤害=%.1f  射程=%.0f\n",
		tw.Strength.Effective(), normalSpeed, normalInterval, normalDmg, normalRange)

	// 模拟削弱流程（与 tickStrengthDrain 一致）
	// ClearTransient → Drainer A → Drainer B
	drainRatio := 0.5

	tw.Strength.ClearTransient()
	tw.RecalcStats()

	// Drainer A
	effA := tw.Strength.Effective()
	tw.Strength.SetEnemySub("drainerA", effA*drainRatio)
	tw.RecalcStats()
	fmt.Printf("Drainer A: Sub=%.0f  强度=%.0f  攻速=%.2f  伤害=%.1f  射程=%.0f\n",
		effA*drainRatio, tw.Strength.Effective(), tw.AttackSpeed, tw.Damage, tw.Range)

	// Drainer B
	effB := tw.Strength.Effective()
	tw.Strength.SetEnemySub("drainerB", effB*drainRatio)
	tw.RecalcStats()
	fmt.Printf("Drainer B: Sub=%.0f  强度=%.0f  攻速=%.2f  伤害=%.1f  射程=%.0f\n",
		effB*drainRatio, tw.Strength.Effective(), tw.AttackSpeed, tw.Damage, tw.Range)

	drainedSpeed := tw.AttackSpeed
	drainedInterval := 1.0 / drainedSpeed
	fmt.Printf("\n两个削弱后: 攻速=%.2f/s  间隔=%.2fs  (正常%.2fs)\n", drainedSpeed, drainedInterval, normalInterval)
	fmt.Printf("伤害: %.1f (正常%.1f)  射程: %.0f (正常%.0f)\n", tw.Damage, normalDmg, tw.Range, normalRange)

	// 验证基础值保底
	if tw.AttackSpeed < tw.BaseSpeed {
		t.Errorf("攻速 %.2f 低于基础值 %.2f", tw.AttackSpeed, tw.BaseSpeed)
	}
	if tw.Damage < tw.BaseDamage {
		t.Errorf("伤害 %.1f 低于基础值 %.1f", tw.Damage, tw.BaseDamage)
	}
	if tw.Range < tw.BaseRange {
		t.Errorf("射程 %.0f 低于基础值 %.0f", tw.Range, tw.BaseRange)
	}

	// 验证攻速仍可用
	if drainedSpeed < 0.1 {
		t.Errorf("攻速 %.2f 低于最低可用值 0.1", drainedSpeed)
	}
	if drainedInterval > 10 {
		t.Errorf("攻击间隔 %.2fs 超过 10s，塔几乎无法攻击", drainedInterval)
	}

	// 模拟极端：3个削弱
	tw.Strength.ClearTransient()
	for i := 0; i < 3; i++ {
		eff := tw.Strength.Effective()
		tw.Strength.SetEnemySub(fmt.Sprintf("d%d", i), eff*drainRatio)
	}
	tw.RecalcStats()
	fmt.Printf("\n三个削弱后: 强度=%.1f  攻速=%.2f/s  间隔=%.2fs  伤害=%.1f  射程=%.0f\n",
		tw.Strength.Effective(), tw.AttackSpeed, 1.0/tw.AttackSpeed, tw.Damage, tw.Range)
}
