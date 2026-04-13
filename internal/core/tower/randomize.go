// randomize.go — 塔随机属性生成与能力解锁顺序随机化。
//
// 本文件实现建塔时的"随机天赋"系统，让每座塔都有独特的属性偏向。
//
// ═══ S/B/D 三档随机系统 ═══
//
// 每座塔的三项属性（伤害/攻速/射程）各独立随机为 S/B/D 三档之一：
//   - S(Superior): 高 Base + 高 Potential（基础好且成长快）
//   - B(Basic):    中 Base + 中 Potential（均衡）
//   - D(Deficient):低 Base + 低 Potential（该项较弱）
//
// 三项独立随机意味着有 3^3=27 种组合（SSS/SSB/SSD/...），无预算约束。
// 这与传统 RPG 的"总点数固定"不同——玩家可能运气好拿到 SSS，也可能拿到 DDD。
//
// ═══ 专精（Specialty）机制 ═══
//
// 建塔时额外随机选择一项属性作为专精（0=伤害/1=攻速/2=射程）。
// 专精项的 Potential 额外加上该属性的 basePotential（来自 tier-presets.json）。
// 效果：专精属性在高强度时成长显著更快，让塔在 3 项中有一个明确优势方向。
//
// ═══ 能力解锁顺序 ═══
//
// RollUnlockOrder 生成 6 个能力类别的随机解锁顺序。
// [0] 固定为攻击模式（建塔时必须先选攻击方式），后 5 个类别随机排列。
// 这保证了不同塔获得 CC/DoT/光环等能力的时机不同，增加策略多样性。
//
// 数值来源：config/systems/tier-presets.json（S/B/D 各档的 Base/Potential 值）
package tower

import (
	"math/rand"

	"defense2/internal/config"
)

// RandomStats 一次 RollTowerStats 的结果：三档独立随机 + 专精属性。
// 用作中间值传递给 ApplyRandomStats，也保存到 Tower 的 *Tier/Specialty 字段供 UI 显示。
type RandomStats struct {
	DamageTier string // "S"/"B"/"D" — 伤害属性档位
	SpeedTier  string // "S"/"B"/"D" — 攻速属性档位
	RangeTier  string // "S"/"B"/"D" — 射程属性档位
	Specialty  int    // 专精属性索引（0=damage, 1=speed, 2=range）
}

// RollTowerStats 随机生成一组塔属性（建塔时调用）。
// 三项属性各独立随机 S/B/D（无预算约束，均匀分布 1/3 概率），再随机选择一项为专精。
// 如果 tier-presets.json 未加载（配置缺失），回退到全 B 档无专精。
func RollTowerStats() RandomStats {
	tp := config.GlobalTierPresets()
	if tp == nil {
		return defaultStats() // 配置缺失时的安全兜底
	}
	tiers := config.TierNames // ["S", "B", "D"]
	// 三项独立随机：每项 1/3 概率 S、1/3 概率 B、1/3 概率 D
	dt := tiers[rand.Intn(len(tiers))]
	st := tiers[rand.Intn(len(tiers))]
	rt := tiers[rand.Intn(len(tiers))]

	specialty := rand.Intn(3) // 0=伤害专精, 1=攻速专精, 2=射程专精

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

// ApplyRandomStats 将随机属性应用到 Tower 的 Base/Potential 字段并触发 RecalcStats。
//
// 属性公式：
//
//	Base     = tier.base              （档位决定的固定底线值）
//	Potential = tier.potential          （非专精属性的成长系数）
//	Potential = tier.potential + basePotential  （专精属性额外加成）
//
// 然后 RecalcStats 计算最终值：attr = Base + Potential × (Strength/100)
//
// 示例：伤害 S 档 + 伤害专精
//
//	BaseDamage     = S.base = 15
//	PotentialDamage = S.potential(25) + basePotential(10) = 35
//	强度 150 时：Damage = 15 + 35 × 1.5 = 67.5
func ApplyRandomStats(t *Tower, stats RandomStats) {
	tp := config.GlobalTierPresets()
	if tp == nil {
		return
	}

	dtv := tp.Damage.Tiers[stats.DamageTier]
	stv := tp.AttackSpeed.Tiers[stats.SpeedTier]
	rtv := tp.Range.Tiers[stats.RangeTier]

	// Base 值：档位决定的固定底线（强度 0 时的属性值）
	t.BaseDamage = dtv.Base
	t.BaseSpeed = stv.Base
	t.BaseRange = rtv.Base

	// Potential 值：档位决定的成长系数（每 100 强度的属性增量）
	t.PotentialDamage = dtv.Potential
	t.PotentialSpeed = stv.Potential
	t.PotentialRange = rtv.Potential

	// 专精加成：专精属性的 Potential 额外加上 basePotential（让该属性在高强度时成长更快）
	switch stats.Specialty {
	case 0: // 伤害专精
		t.PotentialDamage += tp.Damage.BasePotential
	case 1: // 攻速专精
		t.PotentialSpeed += tp.AttackSpeed.BasePotential
	case 2: // 射程专精
		t.PotentialRange += tp.Range.BasePotential
	}

	t.Specialty = stats.Specialty
	t.RecalcStats()
}

// ── 能力解锁顺序随机化 ──

// RollUnlockOrder 生成随机的能力类别解锁顺序（建塔时调用一次）。
// [0] 固定为 AbilityCatAttack（攻击模式），因为建塔后必须先选攻击方式才能开火。
// [1..5] 是 CC/Damage/Buff/DoT/Zone 五个类别的随机排列。
// 效果：同一局中不同塔获得辅助能力的时机不同，增加策略深度。
func RollUnlockOrder() [6]int {
	var order [6]int
	order[0] = config.AbilityCatAttack // 攻击模式始终第一个（建塔后首先要选择攻击方式）

	// Fisher-Yates 洗牌后 5 个辅助类别
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
