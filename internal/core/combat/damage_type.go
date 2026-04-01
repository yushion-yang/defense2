// damage_type.go — 伤害类型系统。
// 4 种伤害类型：physical / magic / true / pure，各有不同的穿透规则。
package combat

import "image/color"

// 伤害类型常量
const (
	DmgPhysical = "physical" // 物理伤害（受增减伤影响）
	DmgMagic    = "magic"    // 魔法伤害（受增减伤影响）
	DmgTrue     = "true"     // 真实伤害（忽略增减伤）
	DmgPure     = "pure"     // 纯粹伤害（忽略增减伤+无敌）
)

// IgnoresReduction 该伤害类型是否忽略攻击/防御增减益。
// true/pure 类型忽略所有减伤。
func IgnoresReduction(t string) bool {
	return t == DmgTrue || t == DmgPure
}

// IgnoresInvincible 该伤害类型是否忽略无敌状态。
// 仅 pure 类型可以穿透无敌。
func IgnoresInvincible(t string) bool {
	return t == DmgPure
}

// DamageTypeColor 返回伤害类型对应的显示颜色。
func DamageTypeColor(t string) color.RGBA {
	switch t {
	case DmgPhysical:
		return color.RGBA{0xef, 0x44, 0x44, 0xff} // 红色
	case DmgMagic:
		return color.RGBA{0xa8, 0x55, 0xf7, 0xff} // 紫色
	case DmgTrue:
		return color.RGBA{0xfb, 0xbf, 0x24, 0xff} // 金色
	case DmgPure:
		return color.RGBA{0xec, 0x48, 0x99, 0xff} // 洋红
	default:
		return color.RGBA{0xef, 0x44, 0x44, 0xff}
	}
}
