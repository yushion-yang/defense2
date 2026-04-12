// pool.go — 塔对象池。
// 固定大小数组实现零分配对象池，支持放置、出售、遍历和按格查找。
package tower

import (
	"fmt"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/game"
	"defense2/internal/core/strength"
)

// grid index dimensions (covers max map size 40 cols × 18 rows with margin)
const (
	gridMaxRows = 20
	gridMaxCols = 42
)

// Pool 固定大小的塔对象池。
type Pool struct {
	towers     []Tower // 预分配的塔槽位数组
	Count      int     // 当前已放置的塔数量
	grid       [gridMaxRows][gridMaxCols]*Tower // spatial index for O(1) At() lookup
	RemoveHook func(instanceKey string) // 可选：塔移除时的清理回调（由上层设置，避免循环依赖）
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

// initTower 初始化塔的公共属性（Place 和 PlaceFromSnapshot 共享）。
// 清零 → 设置定义属性 → 设置攻击方式 → 初始化战力。
func initTower(t *Tower, row, col int, cx, cy float64, def TowerDef) {
	*t = Tower{} // 清零所有字段，防止复用残留

	// 定义属性
	t.Row = row
	t.Col = col
	t.X = cx
	t.Y = cy
	t.Range = def.Range
	t.Damage = def.Damage
	t.AttackSpeed = def.AttackSpeed
	t.Cost = def.Cost
	t.Key = def.Key
	t.InstanceKey = fmt.Sprintf("%s_%d_%d", def.Key, row, col)
	t.Label = def.Label
	t.Abilities = def.Abilities
	t.Color = def.Color
	t.Active = true

	// 攻击方式
	t.AttackStyleID = def.AttackStyleID
	t.ProjectileSpeed = def.ProjectileSpeed
	t.Level = 1
	t.SpriteKey = spriteKeyForStyle(def.AttackStyleID)
	t.UnlockOrder = RollUnlockOrder()

	// 战力系统
	t.Strength = strength.NewStrengthData()
	t.Buffs = buff.NewDefaultBuffList()
}

// allocSlot 从池中分配一个空闲槽位，注册到空间索引。池满时返回 nil。
func (p *Pool) allocSlot(t *Tower, row, col int) *Tower {
	p.Count++
	if row >= 0 && row < gridMaxRows && col >= 0 && col < gridMaxCols {
		p.grid[row][col] = t
	}
	return t
}

// Place 在指定网格位置放置一座塔，从 TowerDef 初始化属性。
// 池满时返回 nil。
func (p *Pool) Place(row, col int, cx, cy float64, def TowerDef) *Tower {
	for i := range p.towers {
		if !p.towers[i].Active {
			t := &p.towers[i]
			initTower(t, row, col, cx, cy, def)

			// 随机属性（ApplyRandomStats 从 tier-presets 设置 Base/Potential/Specialty 并 RecalcStats）
			stats := RollTowerStats()
			ApplyRandomStats(t, stats)
			t.DamageTier = stats.DamageTier
			t.SpeedTier = stats.SpeedTier
			t.RangeTier = stats.RangeTier

			return p.allocSlot(t, row, col)
		}
	}
	return nil
}

// PlaceFromSnapshot places a tower from a saved snapshot, skipping random rolls.
// Uses the snapshot's base/potential/tier values directly instead of RollTowerStats.
func (p *Pool) PlaceFromSnapshot(row, col int, cx, cy float64, def TowerDef, snap config.TowerSnapshot) *Tower {
	for i := range p.towers {
		if !p.towers[i].Active {
			t := &p.towers[i]
			initTower(t, row, col, cx, cy, def)

			// 快照属性（跳过随机 roll）
			t.BaseDamage = snap.BaseDamage
			t.PotentialDamage = snap.PotentialDamage
			t.BaseSpeed = snap.BaseSpeed
			t.PotentialSpeed = snap.PotentialSpeed
			t.BaseRange = snap.BaseRange
			t.PotentialRange = snap.PotentialRange
			t.DamageTier = snap.DamageTier
			t.SpeedTier = snap.SpeedTier
			t.RangeTier = snap.RangeTier
			t.Specialty = snap.Specialty

			return p.allocSlot(t, row, col)
		}
	}
	return nil
}

// Remove 出售塔（标记为非存活，回收槽位）。
func (p *Pool) Remove(t *Tower) {
	if t.Active {
		if p.RemoveHook != nil {
			p.RemoveHook(t.InstanceKey)
		}
		if t.Row >= 0 && t.Row < gridMaxRows && t.Col >= 0 && t.Col < gridMaxCols {
			p.grid[t.Row][t.Col] = nil
		}
		t.Active = false
		t.Target = nil
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

// At 返回指定网格位置 (row, col) 上的塔，无塔则返回 nil。O(1) via grid index.
func (p *Pool) At(row, col int) *Tower {
	if row < 0 || row >= gridMaxRows || col < 0 || col >= gridMaxCols {
		return nil
	}
	t := p.grid[row][col]
	if t != nil && !t.Active {
		return nil
	}
	return t
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

	// 攻击方式
	AttackStyleID   AttackStyle // 攻击方式
	ProjectileSpeed float64     // 弹射物速度（px/s）
	// 战力基础值+潜力值（从 JSON 扁平字段）
	// Base* 是强度0时的底线，Potential* 是每100强度的增量
	// Damage/AttackSpeed/Range 是强度100时的合值（Base+Potential）
	CfgBaseDamage   float64
	CfgBaseSpeed    float64
	CfgBaseRange    float64
	PotentialDamage float64
	PotentialSpeed  float64
	PotentialRange  float64

	// 升级费用
	UpgradeCosts []int // 每次升级费用（索引0=第1次升级，索引5=第6次升级）
}

// spriteKeyForStyle maps attack style to initial sprite key.
func spriteKeyForStyle(style AttackStyle) string {
	switch style {
	case StyleScatter:
		return "shotgun"
	case StyleWideBeam:
		return "prism"
	case StyleSpinAoE:
		return "cyclone"
	case StyleBarrage:
		return "gatling"
	default:
		return "sentinel"
	}
}

