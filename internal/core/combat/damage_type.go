// damage_type.go — 伤害类型系统。
// 4 种伤害类型：physical / magic / true / pure，各有不同的穿透规则。
// 颜色映射已移至 render.DamageTypeColor（渲染关注点）。
package combat

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
