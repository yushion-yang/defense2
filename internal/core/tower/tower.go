// tower.go — 塔实体定义。
// 定义已放置塔的核心属性：位置、攻击参数、能力列表等。
package tower

// MaxTowerLevel 塔的最高等级。
const MaxTowerLevel = 4

// AbilityUnlock 等级解锁的能力信息。
type AbilityUnlock struct {
	Level int    // 解锁等级
	Type  string // 能力代码名
	Name  string // 显示名称
}

// AttackStyle 攻击方式标识。
type AttackStyle = string

const (
	StyleProjectile AttackStyle = "projectile" // 标准追踪弹
	StyleLaser      AttackStyle = "laser"      // 即时光束
	StyleWideBeam   AttackStyle = "wideBeam"   // 宽光束（贯穿）
	StyleScatter    AttackStyle = "scatter"     // 锥形散射
	StyleCharge     AttackStyle = "charge"      // 蓄力重弹
	StyleSpinAoE    AttackStyle = "spin_aoe"    // 旋转范围伤害
	StylePierce     AttackStyle = "pierce"      // 穿刺弹
	StyleAuraDot    AttackStyle = "aura_dot"    // 持续范围毒伤
)

// Tower 已放置的塔实体。
type Tower struct {
	X, Y        float64  // 像素中心坐标
	Row, Col    int      // 所在网格行列
	Range       float64  // 攻击范围（像素）
	Damage      float64  // 单发伤害
	AttackSpeed float64  // 攻击速度（次/秒）
	FireTimer   float64  // 下一次射击倒计时（秒）
	Cost        int      // 总投入金币（建造+升级累计，用于计算卖价）
	Key         string   // 塔类型标识（如 "basic"、"splash"）
	Label       string   // 显示名称
	Active      bool     // 是否存活（对象池复用标记）
	Abilities   []string // 该塔拥有的能力名称列表
	Color       [3]uint8 // 显示颜色 RGB
	Faction     string   // 阵营标识（用于资源路径）
	FireAnim    float64  // 射击动画计时器（射击时设为 0.15，逐帧衰减）
	Angle       float64  // 朝向角度（弧度，0=向上，顺时针）
	Level       int      // 当前等级（1-4）
	BaseDamage  float64  // Lv1 基础伤害（升级时参照）
	BaseRange   float64  // Lv1 基础范围
	BaseSpeed   float64  // Lv1 基础攻速

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式（"projectile"/"laser"/...）
	ProjectileSpeed float64     // 弹射物速度（px/s，0=默认300）

	// Beam 配置
	BeamDuration float64  // beam 显示时长（秒）
	BeamWidth    float64  // beam 宽度（像素）
	BeamColor    [3]uint8 // beam 颜色 RGB

	// Scatter 配置
	ScatterPellets int     // 弹丸数
	ScatterSpread  float64 // 散射半角（弧度）

	// Charge 运行时状态
	ChargeMult     float64 // 蓄力伤害倍率
	ChargeProgress float64 // 蓄力进度 0-1
	ChargeReady    bool    // 蓄力完成

	// SpinAoE 配置 + 运行时状态
	InnerDmgBonus float64 // 内圈加伤倍率
	InnerRatioR   float64 // 内圈半径比例
	SpinAngle     float64 // 旋转角度（弧度）
	SpinActive    float64 // 旋转激活计时器

	// Pierce 配置
	PierceTargets int     // 最大穿透数
	PierceDecay   float64 // 伤害衰减

	// AuraDot 运行时状态
	AuraPulse float64 // 脉冲动画计时

	// 分支特化
	Branch string // 分支特化标识（空=未特化，一次性选择）

	// 索敌锁定
	Target interface{} // 当前锁定目标（*enemy.Enemy，用 interface{} 避免循环导入）

	// 等级解锁能力（配置数据，不参与逻辑，仅用于 HUD 展示）
	AbilityUnlocks []AbilityUnlock
}

// UpgradeCost 返回升级到下一级的费用。不可升级返回 0。
func (t *Tower) UpgradeCost() int {
	if t.Level >= MaxTowerLevel {
		return 0
	}
	// 升级费用 = 基础建造费 × 等级系数
	multiplier := []float64{0, 0.5, 0.75, 1.0} // Lv1→2: 50%, Lv2→3: 75%, Lv3→4: 100%
	return int(float64(t.Cost) * multiplier[t.Level])
}

// Upgrade 提升一级，按比例增强属性。返回本次升级花费。
func (t *Tower) Upgrade() int {
	cost := t.UpgradeCost()
	if cost == 0 {
		return 0
	}
	t.Level++
	t.Cost += cost // 累计投入

	// 每级属性增长：伤害 +25%，范围 +8%，攻速 +10%
	growthDmg := 0.25
	growthRange := 0.08
	growthSpeed := 0.10

	t.Damage = t.BaseDamage * (1 + growthDmg*float64(t.Level-1))
	t.Range = t.BaseRange * (1 + growthRange*float64(t.Level-1))
	t.AttackSpeed = t.BaseSpeed * (1 + growthSpeed*float64(t.Level-1))

	return cost
}

// NextLevelStats 预览下一级的属性值（不修改塔）。
func (t *Tower) NextLevelStats() (damage, atkSpeed, rng float64) {
	nextLevel := t.Level + 1
	if nextLevel > MaxTowerLevel {
		nextLevel = MaxTowerLevel
	}
	damage = t.BaseDamage * (1 + 0.25*float64(nextLevel-1))
	atkSpeed = t.BaseSpeed * (1 + 0.10*float64(nextLevel-1))
	rng = t.BaseRange * (1 + 0.08*float64(nextLevel-1))
	return
}

// DPS 返回当前每秒伤害。
func (t *Tower) DPS() float64 {
	return t.Damage * t.AttackSpeed
}
