// damage_colors.go — 伤害类型 → 显示颜色映射。
// 从 combat 包分离，因为颜色是纯渲染关注点。
package render

import (
	"image/color"

	"defense2/internal/core/combat"
)

// DamageTypeColor 返回伤害类型对应的显示颜色。
func DamageTypeColor(t string) color.RGBA {
	switch t {
	case combat.DmgPhysical:
		return color.RGBA{0xef, 0x44, 0x44, 0xff} // 红色
	case combat.DmgMagic:
		return color.RGBA{0xa8, 0x55, 0xf7, 0xff} // 紫色
	case combat.DmgTrue:
		return color.RGBA{0xfb, 0xbf, 0x24, 0xff} // 金色
	case combat.DmgPure:
		return color.RGBA{0xec, 0x48, 0x99, 0xff} // 洋红
	default:
		return color.RGBA{0xef, 0x44, 0x44, 0xff}
	}
}
