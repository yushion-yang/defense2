// randomize.go — 塔随机属性生成 + 能力解锁顺序随机化。
package tower

import (
	"math/rand"

	"defense2/internal/config"
)

// BaseRatio 基础值占总值的比例（剩余为潜力值）。
// 例如: 总伤害=22, BaseRatio=0.4 → BaseDamage=8.8, PotentialDamage=13.2
const BaseRatio = 0.4

// RandomStats 用 tier-presets 为塔随机生成属性。
// 三项属性 (damage/fireRate/range) 各随机分配一个档位(S~D)，档内随机取值。
// 约束: 三项档位总分 ≤ 预算 (避免全S)。
type RandomStats struct {
	// 总值 (base + potential)
	Damage      float64
	AttackSpeed float64 // 注意 fireRate 是间隔，attackSpeed = 1/fireRate
	Range       float64

	// 各项档位标识
	DamageTier string
	SpeedTier  string
	RangeTier  string
}

// tierScore S=4, A=3, B=2, C=1, D=0
var tierScore = map[string]int{"S": 4, "A": 3, "B": 2, "C": 1, "D": 0}

// TierBudget 三项档位总分固定值。
// 固定为 6 → 保证所有塔"总能力均衡"，只是分布不同。
// 例: S+B+D(4+2+0=6), A+A+C(3+3+0→不行), A+B+C(3+2+1=6), B+B+B(2+2+2=6)
const TierBudget = 6

// RollTowerStats 随机生成一组塔属性。
// 三项属性各分配一个档位，总分固定为 TierBudget（保证均衡）。
func RollTowerStats() RandomStats {
	tp := config.GlobalTierPresets()
	if tp == nil {
		return RandomStats{
			Damage: 16, AttackSpeed: 0.85, Range: 140,
			DamageTier: "B", SpeedTier: "B", RangeTier: "B",
		}
	}

	// 随机分配三个档位，约束总分 == TierBudget
	var dt, st, rt string
	for attempts := 0; attempts < 100; attempts++ {
		dt = config.TierNames[rand.Intn(5)]
		st = config.TierNames[rand.Intn(5)]
		rt = config.TierNames[rand.Intn(5)]
		if tierScore[dt]+tierScore[st]+tierScore[rt] == TierBudget {
			break
		}
	}

	// 档内随机取值
	dmg := randInTier(tp.Damage.Tiers[dt])
	fr := randInTier(tp.FireRate.Tiers[st]) // fireRate 是间隔(秒)
	rng := randInTier(tp.Range.Tiers[rt])

	// fireRate → attackSpeed (次/秒)
	atkSpd := 0.85
	if fr > 0 {
		atkSpd = 1.0 / fr
	}

	return RandomStats{
		Damage: dmg, AttackSpeed: atkSpd, Range: rng,
		DamageTier: dt, SpeedTier: st, RangeTier: rt,
	}
}

// ApplyRandomStats 将随机属性应用到 Tower 的 Base/Potential 字段。
func ApplyRandomStats(t *Tower, stats RandomStats) {
	// Base = total * BaseRatio, Potential = total * (1 - BaseRatio)
	t.BaseDamage = stats.Damage * BaseRatio
	t.PotentialDamage = stats.Damage * (1 - BaseRatio)
	t.BaseSpeed = stats.AttackSpeed * BaseRatio
	t.PotentialSpeed = stats.AttackSpeed * (1 - BaseRatio)
	t.BaseRange = stats.Range * BaseRatio
	t.PotentialRange = stats.Range * (1 - BaseRatio)

	// 强度100时的值 (base + potential = total)
	t.Damage = stats.Damage
	t.AttackSpeed = stats.AttackSpeed
	t.Range = stats.Range

	t.RecalcStats()
}

// randInTier 在档位范围内随机取值。
func randInTier(tr config.TierRange) float64 {
	if tr.Max <= tr.Min {
		return tr.Ref
	}
	return tr.Min + rand.Float64()*(tr.Max-tr.Min)
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
