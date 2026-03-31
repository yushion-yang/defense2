// pool.go — 塔对象池。
// 固定大小数组实现零分配对象池，支持放置、出售、遍历和按格查找。
package tower

import (
	"fmt"

	"defense2/internal/core/game"
	"defense2/internal/core/strength"
)

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
			// 初始属性 = base+potential（强度100默认值），RecalcStats 会覆盖
			t.Range = def.Range
			t.Damage = def.Damage
			t.AttackSpeed = def.AttackSpeed
			t.FireTimer = 0
			t.Cost = def.Cost
			t.Key = def.Key
			t.InstanceKey = fmt.Sprintf("%s_%d_%d", def.Key, row, col)
			t.Label = def.Label
			t.Abilities = def.Abilities
			t.Color = def.Color
			t.Active = true
			// 战力缩放参数（JSON 原始 base/potential）
			t.BaseDamage = def.CfgBaseDamage
			t.BaseRange = def.CfgBaseRange
			t.BaseSpeed = def.CfgBaseSpeed
			t.PotentialDamage = def.PotentialDamage
			t.PotentialSpeed = def.PotentialSpeed
			t.PotentialRange = def.PotentialRange
			// 攻击方式
			t.AttackStyleID = def.AttackStyleID
			t.ProjectileSpeed = def.ProjectileSpeed
			t.ChargeProgress = 0
			t.ChargeReady = false
			t.SpinAngle = 0
			t.SpinActive = 0
			t.AuraPulse = 0
			t.Branch = ""
			t.Level = 1
			t.AbilitySlots = [6]string{}
			t.UnlockOrder = RollUnlockOrder()
			t.MomentumCount = 0
			t.Target = nil

			// 随机属性（tier-presets 驱动，总能力均衡但分布不同）
			stats := RollTowerStats()
			ApplyRandomStats(t, stats)
			t.DamageTier = stats.DamageTier
			t.SpeedTier = stats.SpeedTier
			t.RangeTier = stats.RangeTier
			t.BuildAnim = 0
			t.SellAnim = 0
			t.Selling = false
			t.Strength = strength.NewStrengthData() // Base=100，确保强度系统从放置起就生效
			t.Buffs = nil
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
