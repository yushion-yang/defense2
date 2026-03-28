// pool.go — 塔对象池。
// 固定大小数组实现零分配对象池，支持放置、出售、遍历和按格查找。
package tower

import "defense2/internal/core/game"

// Pool 固定大小的塔对象池。
type Pool struct {
	towers []Tower // 预分配的塔槽位数组
	Count  int     // 当前已放置的塔数量
}

// NewPool 创建指定容量的塔对象池。
func NewPool(cap int) *Pool {
	return &Pool{
		towers: make([]Tower, cap),
	}
}

// DefaultPool 创建默认容量（MaxTowers=64）的塔对象池。
func DefaultPool() *Pool {
	return NewPool(game.MaxTowers)
}

// Place 在指定网格位置放置一座塔，从 TowerDef 初始化属性。
// 池满时返回 nil。
func (p *Pool) Place(row, col int, cx, cy float64, def TowerDef) *Tower {
	for i := range p.towers {
		if !p.towers[i].Active {
			t := &p.towers[i]
			t.Row = row
			t.Col = col
			t.X = cx
			t.Y = cy
			t.Range = def.Range
			t.Damage = def.Damage
			t.AttackSpeed = def.AttackSpeed
			t.FireTimer = 0
			t.Cost = def.Cost
			t.Key = def.Key
			t.Label = def.Label
			t.Abilities = def.Abilities
			t.Color = def.Color
			t.Active = true
			p.Count++
			return t
		}
	}
	return nil
}

// Remove 出售塔（标记为非存活，回收槽位）。
func (p *Pool) Remove(t *Tower) {
	if t.Active {
		t.Active = false
		p.Count--
	}
}

// Each 遍历所有已放置的塔并执行回调。
func (p *Pool) Each(fn func(t *Tower)) {
	for i := range p.towers {
		if p.towers[i].Active {
			fn(&p.towers[i])
		}
	}
}

// At 返回指定网格位置 (row, col) 上的塔，无塔则返回 nil。
func (p *Pool) At(row, col int) *Tower {
	for i := range p.towers {
		if p.towers[i].Active && p.towers[i].Row == row && p.towers[i].Col == col {
			return &p.towers[i]
		}
	}
	return nil
}

// TowerDef 塔类型定义，用于建造时初始化塔属性。
type TowerDef struct {
	Key         string   // 塔类型标识
	Label       string   // 显示名称
	Range       float64  // 攻击范围（像素）
	Damage      float64  // 单发伤害
	AttackSpeed float64  // 攻击速度（次/秒）
	Cost        int      // 建造费用
	Abilities   []string // 能力列表
	Color       [3]uint8 // 显示颜色 RGB
}

// BaseTowerDefs 返回 4 种基础塔定义。
func BaseTowerDefs() []TowerDef {
	return []TowerDef{
		{
			Key: "basic", Label: "Arrow",
			Range: 150, Damage: 10, AttackSpeed: 1.5, Cost: 50,
			Color: [3]uint8{80, 140, 220},
		},
		{
			Key: "splash", Label: "Cannon",
			Range: 120, Damage: 20, AttackSpeed: 0.8, Cost: 80,
			Abilities: []string{"splash"},
			Color:     [3]uint8{200, 120, 60},
		},
		{
			Key: "slow", Label: "Frost",
			Range: 130, Damage: 5, AttackSpeed: 1.2, Cost: 60,
			Abilities: []string{"onHitSlow"},
			Color:     [3]uint8{100, 180, 220},
		},
		{
			Key: "sniper", Label: "Sniper",
			Range: 220, Damage: 35, AttackSpeed: 0.5, Cost: 100,
			Abilities: []string{"crit"},
			Color:     [3]uint8{180, 60, 180},
		},
	}
}
