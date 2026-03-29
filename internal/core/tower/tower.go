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
	BaseDamage      float64 // 基础伤害
	BaseRange       float64 // 基础范围
	BaseSpeed       float64 // 基础攻速
	PotentialDamage float64 // 潜力伤害（随强度缩放的部分）
	PotentialRange  float64 // 潜力范围
	PotentialSpeed  float64 // 潜力攻速

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式（"projectile"/"laser"/...）
	ProjectileSpeed float64     // 弹射物速度（px/s，0=默认300）

	// Charge 运行时状态（由 handler_charge 管理）
	ChargeProgress float64 // 蓄力进度 0-1
	ChargeReady    bool    // 蓄力完成

	// SpinAoE 运行时状态
	SpinAngle     float64 // 旋转角度（弧度）
	SpinActive    float64 // 旋转激活计时器

	// AuraDot 运行时状态
	AuraPulse float64 // 脉冲动画计时

	// 分支特化
	Branch string // 分支特化标识（空=未特化，一次性选择）

	// 战力系统（塔的独立战力数据，不与 Damage 字段混用）
	Strength    *strength.StrengthData   // 战力运行时数据
	StrengthCfg *strength.StrengthConfig // 战力绑定配置

	// 索敌锁定
	Target *enemy.Enemy // 当前锁定目标
}

// BuyStrength 花费金币购买 10 点永久强度。返回实际花费。
// 需要塔已挂载 StrengthData（通过 Strength 字段）。
func (t *Tower) BuyStrength() int {
	cost := StrengthBuyCost
	t.Cost += cost // 累计投入（影响卖价）
	// 通过 strength 包的接口添加永久加成（由调用方做类型断言）
	return cost
}

// DPS 返回当前每秒伤害。
func (t *Tower) DPS() float64 {
	return t.Damage * t.AttackSpeed
}
