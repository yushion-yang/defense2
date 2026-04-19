// projectile.go — 弹射物实体定义。
// 定义飞行中的弹射物属性：位置、速度、伤害、生存时间、追踪目标等。
package projectile

import "defense2/internal/core/enemy"

// TrailLen 拖尾历史帧数。
const TrailLen = 6

// TrailPoint 拖尾历史位置。
type TrailPoint struct {
	X, Y   float64
	Active bool
}

// Projectile 飞行中的弹射物。
type Projectile struct {
	X, Y           float64              // 当前像素位置
	VX, VY         float64              // 速度分量（像素/秒）
	Damage         float64              // 伤害值
	Radius         float64              // 碰撞半径（像素）
	Speed          float64              // 飞行速度（像素/秒）
	Active         bool                 // 是否存活（对象池复用标记）
	Life           float64              // 剩余存活时间（秒）
	MaxLife        float64              // 最大存活时间（秒）
	Target         *enemy.Enemy         // 追踪目标（nil = 直线飞行）
	TargetID       int                  // 发射时锁定目标的 ID（用于检测槽位复用 ABA 问题）
	SourceTowerKey string               // 发射塔的 Key（用于能力触发）
	BounceCount    int                  // 已弹射次数（0 = 原始弹射物）
	BounceHitIDs   []int                // 弹射链已命中敌人 ID（避免弹回已命中目标）
	Trail          [TrailLen]TrailPoint // 拖尾历史位置（环形缓冲）
	TrailCursor    int                  // 下一个写入位置

	// 攻击方式扩展标志
	Penetrate bool  // 直线穿透弹（穿过所有敌人，不追踪）
	PenHitIDs []int // 穿透已命中敌人 ID（避免重复伤害）

	ScatterVisual bool    // 散射视觉弹（旧版，不造成伤害）
	ScatterGroup  int     // 散射组 ID（>0 时为散射弹，同组命中同敌人合并伤害）
	Angle         float64 // 固定飞行角度（scatter/directional）
	MaxRange      float64 // 最大飞行距离
	StartX        float64 // 起始位置 X
	StartY        float64 // 起始位置 Y

	// 斩杀（命中时判定）
	ExecuteHpPct float64 // 斩杀血量阈值（0=禁用，>0 时：非 Boss 且 HP < MaxHP*此值 则秒杀）
}
