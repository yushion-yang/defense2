// randomize.go — 塔随机属性生成 + 能力解锁顺序随机化。
package tower

import (
	"math/rand"

	"defense2/internal/config"
)

// RandomStats 三档独立随机 + 专精属性。
type RandomStats struct {
	DamageTier string
	SpeedTier  string
	RangeTier  string
	Specialty  int // 0=damage, 1=speed, 2=range
}

// RollTowerStats 随机生成一组塔属性。
// 三项属性各独立随机 S/B/D（无预算约束），再随机选择一项为专精。
func RollTowerStats() RandomStats {
	tp := config.GlobalTierPresets()
	if tp == nil {
		return defaultStats()
	}
	tiers := config.TierNames // ["S", "B", "D"]
	dt := tiers[rand.Intn(len(tiers))]
	st := tiers[rand.Intn(len(tiers))]
	rt := tiers[rand.Intn(len(tiers))]

	specialty := rand.Intn(3)

	return RandomStats{
		DamageTier: dt,
		SpeedTier:  st,
		RangeTier:  rt,
		Specialty:  specialty,
	}
}

// defaultStats 无预设表时的兜底值。
func defaultStats() RandomStats {
	return RandomStats{
		DamageTier: "B",
		SpeedTier:  "B",
		RangeTier:  "B",
		Specialty:  0,
	}
}

// ApplyRandomStats 将随机属性应用到 Tower 的 Base/Potential 字段。
// 公式: attr = tier.base + (basePotential + tier.potential) × (strength/100)
// 专精属性的 potential 额外加上 basePotential，非专精属性只用 tier.potential。
func ApplyRandomStats(t *Tower, stats RandomStats) {
	tp := config.GlobalTierPresets()
	if tp == nil {
		return
	}

	dtv := tp.Damage.Tiers[stats.DamageTier]
	stv := tp.AttackSpeed.Tiers[stats.SpeedTier]
	rtv := tp.Range.Tiers[stats.RangeTier]

	// Base values
	t.BaseDamage = dtv.Base
	t.BaseSpeed = stv.Base
	t.BaseRange = rtv.Base

	// Potential values (specialty gets basePotential bonus)
	t.PotentialDamage = dtv.Potential
	t.PotentialSpeed = stv.Potential
	t.PotentialRange = rtv.Potential

	switch stats.Specialty {
	case 0:
		t.PotentialDamage += tp.Damage.BasePotential
	case 1:
		t.PotentialSpeed += tp.AttackSpeed.BasePotential
	case 2:
		t.PotentialRange += tp.Range.BasePotential
	}

	t.Specialty = stats.Specialty
	t.RecalcStats()
}

// ── 能力解锁顺序随机化 ──

// RollUnlockOrder 生成随机的能力类别解锁顺序。
// 第一个始终是攻击模式(0)，后面5个非攻击类别随机排列。
func RollUnlockOrder() [6]int {
	var order [6]int
	order[0] = config.AbilityCatAttack // 攻击模式始终第一个

	// 打乱 [1,2,3,4,5]
	others := []int{
		config.AbilityCatCC,
		config.AbilityCatDamage,
		config.AbilityCatBuff,
		config.AbilityCatDoT,
		config.AbilityCatZone,
	}
	rand.Shuffle(len(others), func(i, j int) {
		others[i], others[j] = others[j], others[i]
	})
	for i, v := range others {
		order[i+1] = v
	}
	return order
}
