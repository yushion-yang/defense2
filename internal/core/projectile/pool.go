// pool.go — 弹射物对象池。
// 环形缓冲区实现，写入时覆盖最老的槽位，支持发射、移动更新和释放。
package projectile

import (
	"math"

	"defense2/internal/core/enemy"
	"defense2/internal/core/game"
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
		p.Count-- // 覆盖了一颗还在飞的弹射物
	}

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
	proj.MaxLife = 3.0
	proj.Life = proj.MaxLife
	proj.Target = target
	proj.SourceTowerKey = towerKey

	proj.BounceCount = 0
	// 清零攻击方式扩展字段
	proj.Pierce = false
	proj.ScatterVisual = false
	proj.ChargeShot = false
	proj.PierceHitIDs = nil
	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FireBounce 发射一颗弹射子弹（从前一次命中位置飞向新目标）。
func (p *Pool) FireBounce(sx, sy float64, target *enemy.Enemy, damage, speed, radius float64, towerKey string, bounceCount int) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}

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
	proj.MaxLife = 2.0
	proj.Life = proj.MaxLife
	proj.Target = target
	proj.SourceTowerKey = towerKey
	proj.BounceCount = bounceCount
	proj.Trail = [TrailLen]TrailPoint{}
	proj.TrailCursor = 0

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// Update 每帧调用，追踪目标 → 移动 → 回收超时/越界弹射物。
func (p *Pool) Update(dt float64) {
	for i := range p.projectiles {
		proj := &p.projectiles[i]
		if !proj.Active {
			continue
		}

		// 追踪：若目标存活，重新计算速度方向；目标死亡则子弹消失
		if proj.Target != nil {
			if proj.Target.Active {
				dx := proj.Target.X - proj.X
				dy := proj.Target.Y - proj.Y
				dist := math.Hypot(dx, dy)
				if dist > 1 {
					proj.VX = (dx / dist) * proj.Speed
					proj.VY = (dy / dist) * proj.Speed
				}
			} else {
				// 目标已死 → 回收弹射物
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
		if proj.Life <= 0 || proj.X < -50 || proj.X > float64(game.ScreenWidth)+50 ||
			proj.Y < -50 || proj.Y > float64(game.ScreenHeight)+50 {
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

// FireScatter 发射一颗散射视觉弹（不造成伤害）。
func (p *Pool) FireScatter(sx, sy, angle, maxRange, speed float64, towerKey string) {
	proj := &p.projectiles[p.cursor]
	if proj.Active {
		p.Count--
	}
	*proj = Projectile{} // 清零

	proj.X = sx
	proj.Y = sy
	proj.VX = math.Cos(angle) * speed
	proj.VY = math.Sin(angle) * speed
	proj.Speed = speed
	proj.Radius = 3
	proj.Active = true
	proj.MaxLife = 2.0
	proj.Life = proj.MaxLife
	proj.SourceTowerKey = towerKey
	proj.ScatterVisual = true
	proj.Angle = angle
	proj.MaxRange = maxRange
	proj.StartX = sx
	proj.StartY = sy

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FirePierce 发射一颗穿刺弹。
func (p *Pool) FirePierce(sx, sy, tx, ty, damage, speed float64, target *enemy.Enemy, towerKey string, maxPierce int, decay float64) {
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
	proj.Radius = 4
	proj.Active = true
	proj.MaxLife = 3.0
	proj.Life = proj.MaxLife
	proj.Target = target
	proj.SourceTowerKey = towerKey
	proj.Pierce = true
	proj.PierceMax = maxPierce
	proj.PierceDecay = decay

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// FireCharge 发射一颗蓄力弹。
func (p *Pool) FireCharge(sx, sy, tx, ty, damage, speed float64, target *enemy.Enemy, towerKey string) {
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
	proj.Radius = 8 // 蓄力弹更大
	proj.Active = true
	proj.MaxLife = 3.0
	proj.Life = proj.MaxLife
	proj.Target = target
	proj.SourceTowerKey = towerKey
	proj.ChargeShot = true

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
