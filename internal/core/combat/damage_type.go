// damage_type.go — 伤害类型系统。
// 当前只有物理伤害，保留扩展接口。
package combat

import "image/color"

// 伤害类型常量
const (
	DmgPhysical = "physical" // 物理伤害
)

// IgnoresReduction 该伤害类型是否忽略攻击/防御增减益。
// 当前只有 physical，不忽略。保留接口供未来扩展。
func IgnoresReduction(t string) bool {
	return false
}

// IgnoresInvincible 该伤害类型是否忽略无敌状态。
func IgnoresInvincible(t string) bool {
	return false
}

// DamageTypeColor 返回伤害类型对应的显示颜色。
func DamageTypeColor(t string) color.RGBA {
	return color.RGBA{0xef, 0x44, 0x44, 0xff} // 红色
}
