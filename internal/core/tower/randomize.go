// randomize.go — 塔随机属性生成 + 能力解锁顺序随机化。
package tower

import (
	"math/rand"

	"defense2/internal/config"
)

// ── 可调参数（后续根据数据调整） ──

// BaseRatioMin/Max 基础值占总值的比例范围（剩余为潜力值）。
// 每个属性独立随机，让每座塔的成长曲线不同。
const (
	BaseRatioMin = 0.30
	BaseRatioMax = 0.50
)

// 属性划转参数
const (
	RedistMinRounds = 1   // 最少划转轮数
	RedistMaxRounds = 3   // 最多划转轮数
	RedistMinUnits  = 1.0 // 每轮最少划转单位
	RedistMaxUnits  = 2.0 // 每轮最多划转单位
	RedistMaxTake   = 0.20 // 单轮最多扣除源属性的比例

	// 划转时 base 和 potential 的接收权重。
	// base 多给些（即时生效），potential 少给些（叠强度后收益惊人）。
	RedistBaseWeight = 1.0  // base 接收倍率
	RedistPotWeight  = 0.3  // potential 接收倍率
)

// 换算比例：1 划转单位对应各属性的数值量（基于 B 档参考值的 ~10%）。
// [0]=damage [1]=attackSpeed [2]=range
var AttrUnitSize = [3]float64{
	2.0,   // 2 伤害 = 1 单位
	0.091, // 0.091 攻速 = 1 单位
	14.0,  // 14 射程 = 1 单位
}

// 各属性最小值下限
var AttrMinBase = [3]float64{1, 0.1, 40}      // damage, speed, range 的 base 下限
var AttrMinPot  = [3]float64{0, 0, 0}          // damage, speed, range 的 potential 下限

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
	found := false
	for attempts := 0; attempts < 100; attempts++ {
		dt = config.TierNames[rand.Intn(5)]
		st = config.TierNames[rand.Intn(5)]
		rt = config.TierNames[rand.Intn(5)]
		if tierScore[dt]+tierScore[st]+tierScore[rt] == TierBudget {
			found = true
			break
		}
	}
	if !found {
		dt, st, rt = "B", "B", "B"
	}

	// 档内随机取值（attackSpeed 已是次/秒，无需转换）
	dmg := randInTier(tp.Damage.Tiers[dt])
	atkSpd := randInTier(tp.AttackSpeed.Tiers[st])
	rng := randInTier(tp.Range.Tiers[rt])

	return RandomStats{
		Damage: dmg, AttackSpeed: atkSpd, Range: rng,
		DamageTier: dt, SpeedTier: st, RangeTier: rt,
	}
}

// ApplyRandomStats 将随机属性应用到 Tower 的 Base/Potential 字段。
// 1. 按随机比例拆分为 base/potential
// 2. 做 1-3 轮属性划转：从一项按比例扣除，按换算比例加到另一项
func ApplyRandomStats(t *Tower, stats RandomStats) {
	// 拆分 base/potential（每属性独立随机比例）
	dr := randBaseRatio()
	sr := randBaseRatio()
	rr := randBaseRatio()

	t.BaseDamage = stats.Damage * dr
	t.PotentialDamage = stats.Damage * (1 - dr)
	t.BaseSpeed = stats.AttackSpeed * sr
	t.PotentialSpeed = stats.AttackSpeed * (1 - sr)
	t.BaseRange = stats.Range * rr
	t.PotentialRange = stats.Range * (1 - rr)

	// 属性划转（保持总水平不变）
	redistributeStats(t)

	t.RecalcStats()
}

// ── 属性划转 ──
//
// 从源属性按比例扣除，按换算比例加到目标属性。
// 扣除时 base/potential 等比；接收时 base 多给(×BaseWeight)，potential 少给(×PotWeight)。
// 这样划转后 base 立即有感，但 potential 不会因叠强度而膨胀过度。

func redistributeStats(t *Tower) {
	type attr struct {
		base *float64
		pot  *float64
	}
	attrs := [3]attr{
		{&t.BaseDamage, &t.PotentialDamage},
		{&t.BaseSpeed, &t.PotentialSpeed},
		{&t.BaseRange, &t.PotentialRange},
	}

	rounds := RedistMinRounds + rand.Intn(RedistMaxRounds-RedistMinRounds+1)
	for r := 0; r < rounds; r++ {
		src := rand.Intn(3)
		dst := (src + 1 + rand.Intn(2)) % 3

		units := RedistMinUnits + rand.Float64()*(RedistMaxUnits-RedistMinUnits)

		srcTotal := *attrs[src].base + *attrs[src].pot
		if srcTotal <= 0 {
			continue
		}

		// 扣除量（不超过源属性的 RedistMaxTake）
		takeAmount := units * AttrUnitSize[src]
		if takeAmount > srcTotal*RedistMaxTake {
			takeAmount = srcTotal * RedistMaxTake
		}

		// 接收量（换算到目标属性单位）
		giveAmount := units * AttrUnitSize[dst]

		// 源：按当前 base/potential 比例等比扣除
		srcRatio := *attrs[src].base / srcTotal
		*attrs[src].base -= takeAmount * srcRatio
		*attrs[src].pot -= takeAmount * (1 - srcRatio)

		// 目标：base 多给，potential 少给
		// 先算 base/pot 各应得的原始份额，再乘以权重
		dstTotal := *attrs[dst].base + *attrs[dst].pot
		dstRatio := 0.4 // 默认
		if dstTotal > 0 {
			dstRatio = *attrs[dst].base / dstTotal
		}
		rawBase := giveAmount * dstRatio
		rawPot := giveAmount * (1 - dstRatio)
		*attrs[dst].base += rawBase * RedistBaseWeight
		*attrs[dst].pot += rawPot * RedistPotWeight
	}

	// 确保最小值
	clampMin(&t.BaseDamage, AttrMinBase[0])
	clampMin(&t.PotentialDamage, AttrMinPot[0])
	clampMin(&t.BaseSpeed, AttrMinBase[1])
	clampMin(&t.PotentialSpeed, AttrMinPot[1])
	clampMin(&t.BaseRange, AttrMinBase[2])
	clampMin(&t.PotentialRange, AttrMinPot[2])
}

func clampMin(v *float64, min float64) {
	if *v < min {
		*v = min
	}
}

// randBaseRatio 返回 [BaseRatioMin, BaseRatioMax] 区间的随机比例。
func randBaseRatio() float64 {
	return BaseRatioMin + rand.Float64()*(BaseRatioMax-BaseRatioMin)
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
