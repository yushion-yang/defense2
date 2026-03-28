// hero.go — 英雄系统核心。
// 英雄是玩家控制的唯一战斗单位，驻守基地附近，主动追击进入范围的敌人。
// AI 状态机：空闲巡逻 → 交战追击 → 撤回基地。
package hero

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/projectile"
)

// HeroState AI 状态枚举。
type HeroState int

const (
	StateIdle   HeroState = iota // 在基地附近巡逻
	StateEngage                  // 追击并攻击敌人
	StateReturn                  // 脱离战斗，撤回基地
)

// Hero 英雄实体。
type Hero struct {
	X, Y          float64   // 当前像素位置
	BaseX, BaseY  float64   // 基地锚点（巡逻中心）
	State         HeroState // 当前 AI 状态
	Damage        float64   // 单发伤害
	AttackRange   float64   // 攻击范围（像素）
	EngageRange   float64   // 交战触发范围（进入此范围开始追击）
	LeashRadius   float64   // 牵引半径（超出此距离强制撤回）
	MoveSpeed     float64   // 移动速度（像素/秒）
	FireRate      float64   // 射击间隔（秒/次）
	FireTimer     float64   // 下一次射击倒计时（秒）
	BodyRadius    float64   // 碰撞/显示半径（像素）
	Level         int       // 当前等级
	XP            int       // 当前经验值
	XPToNext      int       // 升级所需经验
	OrbitAngle    float64   // 巡逻轨道当前角度（弧度）
	OrbitRadius   float64   // 巡逻轨道半径（像素）
	OrbitSpeed    float64   // 巡逻角速度（弧度/秒）
	TargetID      int       // 当前锁定的敌人索引（-1 表示无目标）
	Active        bool      // 是否已激活
}

// DefaultHero 创建默认属性的英雄。
func DefaultHero(baseX, baseY float64) *Hero {
	return &Hero{
		X: baseX, Y: baseY,
		BaseX: baseX, BaseY: baseY,
		State:       StateIdle,
		Damage:      15,
		AttackRange: 140,
		EngageRange: 180,
		LeashRadius: 250,
		MoveSpeed:   120,
		FireRate:     0.4,
		BodyRadius:  10,
		Level:       1,
		XP:          0,
		XPToNext:    20,
		OrbitRadius: 40,
		OrbitSpeed:  1.2,
		TargetID:    -1,
		Active:      true,
	}
}

// Update 每帧驱动英雄 AI 状态机和射击。
func (h *Hero) Update(enemies *enemy.Pool, projectiles *projectile.Pool, dt float64) {
	if !h.Active {
		return
	}
	h.FireTimer -= dt

	switch h.State {
	case StateIdle:
		h.tickIdle(enemies, dt)
	case StateEngage:
		h.tickEngage(enemies, projectiles, dt)
	case StateReturn:
		h.tickReturn(dt)
	}
}

// tickIdle 空闲状态：绕基地巡逻，检测是否有敌人进入交战范围。
func (h *Hero) tickIdle(enemies *enemy.Pool, dt float64) {
	// 巡逻轨道运动
	h.OrbitAngle += h.OrbitSpeed * dt
	h.X = h.BaseX + math.Cos(h.OrbitAngle)*h.OrbitRadius
	h.Y = h.BaseY + math.Sin(h.OrbitAngle)*h.OrbitRadius

	// 检测敌人进入交战范围
	nearest, dist := findNearest(h.X, h.Y, enemies)
	if nearest >= 0 && dist <= h.EngageRange {
		h.TargetID = nearest
		h.State = StateEngage
	}
}

// tickEngage 交战状态：追击目标、射击、检查牵引距离。
func (h *Hero) tickEngage(enemies *enemy.Pool, projectiles *projectile.Pool, dt float64) {
	// 检查牵引距离
	distToBase := math.Hypot(h.X-h.BaseX, h.Y-h.BaseY)
	if distToBase > h.LeashRadius {
		h.State = StateReturn
		h.TargetID = -1
		return
	}

	// 查找目标
	targetIdx, targetDist := findNearest(h.X, h.Y, enemies)
	if targetIdx < 0 {
		h.State = StateReturn
		h.TargetID = -1
		return
	}
	h.TargetID = targetIdx

	// 获取目标位置
	var tx, ty float64
	idx := 0
	enemies.Each(func(e *enemy.Enemy) {
		if idx == targetIdx {
			tx, ty = e.X, e.Y
		}
		idx++
	})

	// 向目标移动（保持在攻击范围边缘）
	if targetDist > h.AttackRange*0.8 {
		dx := tx - h.X
		dy := ty - h.Y
		d := math.Hypot(dx, dy)
		if d > 1 {
			h.X += (dx / d) * h.MoveSpeed * dt
			h.Y += (dy / d) * h.MoveSpeed * dt
		}
	}

	// 射击
	if h.FireTimer <= 0 && targetDist <= h.AttackRange {
		projectiles.Fire(h.X, h.Y, tx, ty, h.Damage, 350, 3)
		h.FireTimer = h.FireRate
	}
}

// tickReturn 撤回状态：朝基地移动，到达后切换为空闲。
func (h *Hero) tickReturn(dt float64) {
	dx := h.BaseX - h.X
	dy := h.BaseY - h.Y
	dist := math.Hypot(dx, dy)
	if dist < 5 {
		h.State = StateIdle
		return
	}
	h.X += (dx / dist) * h.MoveSpeed * 1.5 * dt
	h.Y += (dy / dist) * h.MoveSpeed * 1.5 * dt
}

// AwardXP 奖励经验值，自动升级。
func (h *Hero) AwardXP(amount int) {
	h.XP += amount
	for h.XP >= h.XPToNext && h.Level < 10 {
		h.XP -= h.XPToNext
		h.Level++
		h.XPToNext = 20 + h.Level*10
		// 升级属性提升
		h.Damage += 3
		h.AttackRange += 5
		h.MoveSpeed += 5
	}
}

// findNearest 在敌人池中查找距离 (x,y) 最近的存活敌人，返回索引和距离。
func findNearest(x, y float64, enemies *enemy.Pool) (int, float64) {
	bestIdx := -1
	bestDist := math.MaxFloat64
	idx := 0
	enemies.Each(func(e *enemy.Enemy) {
		d := math.Hypot(e.X-x, e.Y-y)
		if d < bestDist {
			bestDist = d
			bestIdx = idx
		}
		idx++
	})
	return bestIdx, bestDist
}
