// damage_type.go — 伤害类型系统。
// 定义 4 种伤害类型及其穿透规则，用于伤害管线的免疫/减免判定。
//
// 穿透规则矩阵:
//
//	类型       | 忽略减免 | 忽略无敌
//	-----------|---------|--------
//	physical   |   ✗     |   ✗
//	magic      |   ✗     |   ✗
//	true       |   ✓     |   ✗
//	pure       |   ✓     |   ✓
package combat

import "image/color"

// 伤害类型常量
const (
	DmgPhysical = "physical" // 物理伤害（受减免、受无敌）
	DmgMagic    = "magic"    // 魔法伤害（受减免、受无敌）
	DmgTrue     = "true"     // 真实伤害（忽略减免，受无敌）
	DmgPure     = "pure"     // 纯粹伤害（忽略减免、忽略无敌）
)

// IgnoresReduction 该伤害类型是否忽略攻击/防御增减益。
func IgnoresReduction(t string) bool {
	return t == DmgTrue || t == DmgPure
}

// IgnoresInvincible 该伤害类型是否忽略无敌状态。
func IgnoresInvincible(t string) bool {
	return t == DmgPure
}

// 伤害类型颜色映射（用于飘字和 UI 显示）
var damageTypeColors = map[string]color.RGBA{
	DmgPhysical: {0xef, 0x44, 0x44, 0xff}, // 红色
	DmgMagic:    {0xa8, 0x55, 0xf7, 0xff}, // 紫色
	DmgTrue:     {0xfb, 0xbf, 0x24, 0xff}, // 金色
	DmgPure:     {0xf4, 0x3f, 0x5e, 0xff}, // 玫红
}

// DamageTypeColor 返回伤害类型对应的显示颜色。
// 未知类型默认返回物理伤害颜色。
func DamageTypeColor(t string) color.RGBA {
	if c, ok := damageTypeColors[t]; ok {
		return c
	}
	return damageTypeColors[DmgPhysical]
}
