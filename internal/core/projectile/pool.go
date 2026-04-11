// pool.go — 弹射物对象池。
// Ring buffer pool (1024 slots). Chosen for: high churn rate (projectiles
// created/destroyed rapidly), FIFO ordering ensures oldest slots are
// reclaimed first, and cursor-based insertion avoids linear scan overhead.
//
// ── 弹射物生命周期（塔防模型） ──
//
// 本游戏是塔防，不是弹幕射击。弹射物行为遵循塔防惯例：
//
//  1. 发射时锁定目标（Target != nil），每帧追踪目标飞行
//  2. 碰撞检测只对锁定目标生效，穿过其他敌人（见 tick_combat.go）
//  3. 命中目标 → 触发能力 + 伤害 → 回收
//  4. 目标被其他弹先杀死 → 本弹直接消失（不继续飞行）
//
// 例外：穿透弹（Penetrate=true）对路径上所有敌人做碰撞检测。
package projectile

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
)

const (
	normalProjectileMaxLife   = 3.0  // 普通弹射物最大存活时间(秒)
	bounceProjectileMaxLife   = 2.0  // 弹射弹射物最大存活时间(秒)
	penetrateProjectileRadius = 4    // 穿透弹射物碰撞半径(像素)
	penetrateLifeOvershoot    = 1.1  // 穿透弹射物寿命余量倍率
	projectileBoundaryMargin  = 50.0 // 弹射物屏幕外回收边距(像素)
)

// Pool 环形缓冲区弹射物对象池。
type Pool struct {
	projectiles []Projectile // 预分配的弹射物槽位数组
	cursor      int          // 下一个写入位置（环形）
	Count       int          // 当前存活弹射物数量
}

// NewPool 创建指定容量的弹射物对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		projectiles: make([]Projectile, cap),
	}
}

// DefaultPool 创建默认容量（MaxProjectiles=1024）的弹射物对象池。
func DefaultPool() *Pool {
	return NewPool(game.MaxProjectiles)
}

// Fire 从 (sx,sy) 向 (tx,ty) 发射一颗弹射物。
// target 非 nil 时启用追踪（每帧重新计算朝向），否则直线飞行。
// towerKey 用于碰撞时查找来源塔触发能力。
func (p *Pool) Fire(sx, sy, tx, ty, damage, speed, radius float64, target *enemy.Enemy, towerKey string) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零所有旧字段，防止复用槽位残留

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = normalProjectileMaxLife
	proj.Life = proj.MaxLife
	proj.Target = target
	if target != nil {
		proj.TargetID = target.ID
	}
	proj.SourceTowerKey = towerKey

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FireBounce 发射一颗弹射子弹（从前一次命中位置飞向新目标）。
func (p *Pool) FireBounce(sx, sy float64, target *enemy.Enemy, damage, speed, radius float64, towerKey string, bounceCount int, hitIDs []int) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零所有旧字段

	dx := target.X - sx
	dy := target.Y - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = radius
	proj.Active = true
	proj.MaxLife = bounceProjectileMaxLife
	proj.Life = proj.MaxLife
	proj.Target = target
	if target != nil {
		proj.TargetID = target.ID
	}
	proj.SourceTowerKey = towerKey
	proj.BounceCount = bounceCount
	proj.BounceHitIDs = hitIDs

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// Tick 每帧调用，追踪目标 → 移动 → 回收超时/越界弹射物。
func (p *Pool) Tick(dt float64) {
	for i := range p.projectiles {
		proj := &p.projectiles[i]
		if !proj.Active {
			continue
		}

		// 追踪：所有有目标的弹（普通弹/弹射弹/蓄力弹）统一行为：
		//   - 目标存活且 ID 匹配 → 每帧重算朝向，完美追踪
		//   - 目标死亡或 ID 不匹配（槽位被复用 ABA 问题）→ 弹射物立即消失
		if proj.Target != nil {
			if proj.Target.Active && proj.Target.ID == proj.TargetID {
				dx := proj.Target.X - proj.X
				dy := proj.Target.Y - proj.Y
				dist := math.Hypot(dx, dy)
				if dist > 1 {
					proj.VX = (dx / dist) * proj.Speed
					proj.VY = (dy / dist) * proj.Speed
				}
			} else {
				proj.Active = false
				proj.Target = nil
				p.Count--
				continue
			}
		}

		// 记录拖尾位置（移动前）
		proj.Trail[proj.TrailCursor] = TrailPoint{X: proj.X, Y: proj.Y, Active: true}
		proj.TrailCursor = (proj.TrailCursor + 1) % TrailLen

		proj.X += proj.VX * dt
		proj.Y += proj.VY * dt
		proj.Life -= dt

		// 散射弹飞行距离限制
		if proj.MaxRange > 0 {
			dx := proj.X - proj.StartX
			dy := proj.Y - proj.StartY
			if math.Hypot(dx, dy) >= proj.MaxRange {
				proj.Active = false
				proj.Target = nil
				p.Count--
				continue
			}
		}

		// 超时或飞出屏幕边界则回收
		if proj.Life <= 0 || proj.X < -projectileBoundaryMargin || proj.X > float64(game.ScreenWidth)+projectileBoundaryMargin ||
			proj.Y < -projectileBoundaryMargin || proj.Y > float64(game.ScreenHeight)+projectileBoundaryMargin {
			proj.Active = false
			proj.Target = nil
			p.Count--
		}
	}
}

// Each 遍历所有存活弹射物并执行回调。
func (p *Pool) Each(fn func(proj *Projectile)) {
	for i := range p.projectiles {
		if p.projectiles[i].Active {
			fn(&p.projectiles[i])
		}
	}
}

// Release 释放弹射物（命中后回收）。
func (p *Pool) Release(proj *Projectile) {
	if proj.Active {
		proj.Active = false
		p.Count--
	}
}

// FirePenetrate 发射一颗直线穿透弹（穿过所有敌人，不追踪，不衰减）。
func (p *Pool) FirePenetrate(sx, sy, tx, ty, damage, speed float64, towerKey string) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零

	dx := tx - sx
	dy := ty - sy
	dist := math.Hypot(dx, dy)
	if dist < 1 {
		dist = 1
	}

	proj.X = sx
	proj.Y = sy
	proj.VX = (dx / dist) * speed
	proj.VY = (dy / dist) * speed
	proj.Damage = damage
	proj.Speed = speed
	proj.Radius = penetrateProjectileRadius
	proj.Active = true
	proj.MaxLife = dist / speed * penetrateLifeOvershoot // 飞到终点后稍微多一点余量
	proj.Life = proj.MaxLife
	proj.Target = nil // 直线飞行，不追踪
	proj.SourceTowerKey = towerKey
	proj.Penetrate = true
	proj.StartX = sx
	proj.StartY = sy
	proj.MaxRange = dist

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// ClearAll 清空所有弹射物（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.projectiles {
		p.projectiles[i].Active = false
		p.projectiles[i].Target = nil
	}
	p.Count = 0
	p.cursor = 0
}
