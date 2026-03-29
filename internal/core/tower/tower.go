// tower.go — 塔实体定义。
// 定义已放置塔的核心属性：位置、攻击参数、能力列表等。
package tower

import (
	"defense2/internal/core/enemy"
	"defense2/internal/core/strength"
)

// StrengthBuyCost 购买 10 点强度的金币花费。
const StrengthBuyCost = 10

// AttackStyle 攻击方式标识。
type AttackStyle = string

const (
	StyleProjectile AttackStyle = "projectile" // 标准追踪弹
	StyleLaser      AttackStyle = "laser"      // 即时光束
	StyleWideBeam   AttackStyle = "wideBeam"   // 宽光束（贯穿）
	StyleScatter    AttackStyle = "scatter"    // 锥形散射
	StyleCharge     AttackStyle = "charge"     // 蓄力重弹
	StyleSpinAoE    AttackStyle = "spin_aoe"   // 旋转范围伤害
	StyleAuraDot    AttackStyle = "aura_dot"   // 持续范围毒伤
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
	InstanceKey string   // 实例唯一标识（"key_row_col"，用于弹射物来源匹配）
	Label       string   // 显示名称
	Active      bool     // 是否存活（对象池复用标记）
	Abilities   []string // 该塔拥有的能力名称列表
	Color       [3]uint8 // 显示颜色 RGB
	Faction     string   // 阵营标识（用于资源路径）
	FireAnim    float64  // 射击动画计时器（射击时设为 0.15，逐帧衰减）
	Angle       float64  // 朝向角度（弧度，0=向上，顺时针）
	// 战力缩放参数（来自 JSON 配置，放置后不变）
	// 公式: attr = Base + Potential * (strength / 100)
	BaseDamage      float64 // JSON baseDamage
	BaseRange       float64 // JSON baseRange
	BaseSpeed       float64 // JSON baseAttackSpeed
	PotentialDamage float64 // JSON potentialDamage
	PotentialRange  float64 // JSON potentialRange
	PotentialSpeed  float64 // JSON potentialAttackSpeed

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式（"projectile"/"laser"/...）
	ProjectileSpeed float64     // 弹射物速度（px/s，0=默认300）

	// Charge 运行时状态（由 handler_charge 管理）
	ChargeProgress float64 // 蓄力进度 0-1
	ChargeReady    bool    // 蓄力完成

	// SpinAoE 运行时状态
	SpinAngle  float64 // 旋转角度（弧度）
	SpinActive float64 // 旋转激活计时器

	// AuraDot 运行时状态
	AuraPulse float64 // 脉冲动画计时

	// GoldPassive 运行时状态
	GoldCooldown float64 // 被动产金冷却计时器

	// 分支特化
	Branch string // 分支特化标识（空=未特化，一次性选择）

	// 战力系统
	Strength *strength.StrengthData // 战力运行时数据

	// 光环加成（每帧由 Ticker 能力重置+重算）
	CritBonus float64 // 暴击光环加成的暴击率（由 critAura 设置）

	// 索敌锁定
	Target              *enemy.Enemy // 当前锁定目标
	LastPercentHpTarget int          // 上次触发 percentHpDamage 的敌人 ID（切换目标首击）

	// 叠伤计数（stackDamage 能力：连续命中同目标递增）
	StackTarget int // 当前叠伤目标 ID
	StackCount  int // 叠伤层数
}

// BuyStrength 花费金币购买 10 点永久强度。返回实际花费。
// 需要塔已挂载 StrengthData（通过 Strength 字段）。
func (t *Tower) BuyStrength() int {
	cost := StrengthBuyCost
	t.Cost += cost // 累计投入（影响卖价）
	// 通过 strength 包的接口添加永久加成（由调用方做类型断言）
	return cost
}

// RecalcStats 根据当前强度重算 Damage/AttackSpeed/Range。
// 公式与 AbilityDef 一致: value = base + potential * (strength / 100)
// 无 Strength 时使用强度100的默认值（base + potential）。
func (t *Tower) RecalcStats() {
	ratio := 1.0 // 默认强度100
	if t.Strength != nil {
		ratio = t.Strength.Effective() / 100.0
	}
	t.Damage = t.BaseDamage + t.PotentialDamage*ratio
	t.AttackSpeed = t.BaseSpeed + t.PotentialSpeed*ratio
	t.Range = t.BaseRange + t.PotentialRange*ratio
	t.CritBonus = 0 // 每帧重置，由 critAura OnTick 重新设置
}

// DPS 返回当前每秒伤害。
func (t *Tower) DPS() float64 {
	return t.Damage * t.AttackSpeed
}
