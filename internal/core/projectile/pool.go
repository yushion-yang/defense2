// pool.go — 弹射物对象池。
// 环形缓冲区实现，写入时覆盖最老的槽位，支持发射、移动更新和释放。
package projectile

import (
	"math"

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
// 自动计算归一化速度方向，最大存活时间 3 秒。
func (p *Pool) Fire(sx, sy, tx, ty, damage, speed, radius float64) {
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

	p.Count++
	p.cursor = (p.cursor + 1) % len(p.projectiles)
}

// Update 每帧调用，移动所有存活弹射物并回收超时/越界的。
func (p *Pool) Update(dt float64) {
	for i := range p.projectiles {
		proj := &p.projectiles[i]
		if !proj.Active {
			continue
		}
		proj.X += proj.VX * dt
		proj.Y += proj.VY * dt
		proj.Life -= dt

		// 超时或飞出屏幕边界则回收
		if proj.Life <= 0 || proj.X < -50 || proj.X > float64(game.ScreenWidth)+50 ||
			proj.Y < -50 || proj.Y > float64(game.ScreenHeight)+50 {
			proj.Active = false
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

// ClearAll 清空所有弹射物（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.projectiles {
		p.projectiles[i].Active = false
	}
	p.Count = 0
	p.cursor = 0
}
