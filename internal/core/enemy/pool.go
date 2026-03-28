// pool.go — 敌人对象池。
// 固定大小数组实现零分配对象池，通过 Active 标记复用槽位。
package enemy

import "defense2/internal/core/game"

// Pool 固定大小的敌人对象池。
type Pool struct {
	enemies []Enemy // 预分配的敌人槽位数组
	Count   int     // 当前存活敌人数量
}

// NewPool 创建指定容量的敌人对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		enemies: make([]Enemy, cap),
	}
}

// DefaultPool 创建默认容量（MaxEnemies=256）的敌人对象池。
func DefaultPool() *Pool {
	return NewPool(game.MaxEnemies)
}

// Spawn 激活一个空闲槽位并初始化敌人属性。
// pathIndex 通常为 1（敌人从 waypoint[0] 出生，朝 waypoint[1] 移动）。
// 池满时返回 nil。
func (p *Pool) Spawn(x, y, hp, speed, radius float64, pathIndex int) *Enemy {
	for i := range p.enemies {
		if !p.enemies[i].Active {
			e := &p.enemies[i]
			e.X = x
			e.Y = y
			e.HP = hp
			e.MaxHP = hp
			e.Speed = speed
			e.BaseSpeed = speed
			e.Radius = radius
			e.PathIndex = pathIndex
			e.ReachedEnd = false
			e.Active = true
			e.Path = nil
			e.Archetype = "normal"
			e.Boss = false
			e.Reward = 0
			e.StunTimer = 0
			e.SlowTimer = 0
			e.SlowFactor = 1
			e.BleedTimer = 0
			e.BleedDPS = 0
			e.BurnTimer = 0
			e.BurnDPS = 0
			e.ShieldHP = 0
			e.RootTimer = 0
			p.Count++
			return e
		}
	}
	return nil
}

// Kill 将敌人标记为非存活（回收槽位）。
func (p *Pool) Kill(e *Enemy) {
	if e.Active {
		e.Active = false
		p.Count--
	}
}

// Each 遍历所有存活敌人并执行回调。
func (p *Pool) Each(fn func(e *Enemy)) {
	for i := range p.enemies {
		if p.enemies[i].Active {
			fn(&p.enemies[i])
		}
	}
}

// ClearAll 清空所有敌人（重置对象池）。
func (p *Pool) ClearAll() {
	for i := range p.enemies {
		p.enemies[i].Active = false
	}
	p.Count = 0
}
