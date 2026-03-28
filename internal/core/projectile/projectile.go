// projectile.go — 弹射物实体定义。
// 定义飞行中的弹射物属性：位置、速度、伤害、生存时间等。
package projectile

// Projectile 飞行中的弹射物。
type Projectile struct {
	X, Y    float64 // 当前像素位置
	VX, VY  float64 // 速度分量（像素/秒）
	Damage  float64 // 伤害值
	Radius  float64 // 碰撞半径（像素）
	Speed   float64 // 飞行速度（像素/秒）
	Active  bool    // 是否存活（对象池复用标记）
	Life    float64 // 剩余存活时间（秒）
	MaxLife float64 // 最大存活时间（秒）
}
