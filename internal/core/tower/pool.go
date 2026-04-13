// pool.go — 塔对象池与空间索引。
//
// 本文件管理所有已放置塔的生命周期（创建/销毁/遍历/查找）。
// 采用固定大小数组实现零分配对象池，避免频繁 GC（塔防游戏中塔的创建/销毁是热路径）。
//
// 核心设计：
//   - towers[]: 预分配的塔槽位数组，Active=false 的槽位可被复用
//   - grid[][]: 2D 空间索引（行×列），支持 At(row,col) O(1) 查找
//   - ByInstanceKey(): 解析 "key_row_col" 字符串反查塔，O(1) 替代每帧构建 map
//   - Place/Remove: 完整的生命周期管理（初始化→随机属性→空间索引注册→清理）
//
// 与其他文件的关系：
//   - tower.go: Tower struct 定义（Pool 的元素类型）
//   - randomize.go: Place 时调用 RollTowerStats/ApplyRandomStats 设置随机属性
//   - lifecycle.go: Place/Remove 时触发生命周期钩子
//   - targeting.go: 通过 Pool.Each 遍历塔进行索敌
package tower

import (
	"fmt"
	"strconv"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/buff"
	"defense2/internal/core/game"
	"defense2/internal/core/strength"
)

// 空间索引尺寸上限（覆盖最大地图 40 列 × 18 行，并留有余量）。
// 这些常量决定 grid 数组大小，超出范围的 row/col 不会注册到空间索引。
const (
	gridMaxRows = 20
	gridMaxCols = 42
)

// Pool 固定大小的塔对象池。
// 使用预分配数组（而非 map/slice append）确保零运行时分配。
// grid 是 2D 空间索引，将 (row,col) 映射到 *Tower 指针，实现 O(1) 位置查找。
type Pool struct {
	towers     []Tower                          // 预分配的塔槽位数组（容量固定，通过 Active 标记区分占用/空闲）
	Count      int                              // 当前已放置的塔数量（用于 UI 显示和容量检查）
	grid       [gridMaxRows][gridMaxCols]*Tower // 2D 空间索引：grid[row][col] 直接指向塔，At()/ByInstanceKey() 依赖此索引
	RemoveHook func(instanceKey string)         // 可选：塔移除时的清理回调（上层设置，用于清理弹射物等关联资源，避免 pool→projectile 的循环依赖）
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
//
// 流程：清零所有字段 → 复制 TowerDef 定义属性 → 设置攻击方式 → 初始化战力系统。
// 关键点：*t = Tower{} 是必须的，因为槽位可能是被 Remove 回收的旧塔，
// 如果不清零，旧塔的 BarrageTarget/PendingChoices 等指针字段会残留导致 bug。
func initTower(t *Tower, row, col int, cx, cy float64, def TowerDef) {
	*t = Tower{} // 清零所有字段，防止对象池复用时旧数据残留

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

	// 攻击方式（建塔时尚未选择攻击能力，使用 TowerDef 的默认值）
	t.AttackStyleID = def.AttackStyleID
	t.ProjectileSpeed = def.ProjectileSpeed
	t.Level = 1
	t.SpriteKey = spriteKeyForStyle(def.AttackStyleID) // 初始精灵（通常是 "sentinel"）
	t.UnlockOrder = RollUnlockOrder()                  // 随机生成 6 个能力类别的解锁顺序（[0] 始终是攻击模式）

	// 战力系统初始化：Strength 基础值 100，BuffList 空容器
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
//
// 完整流程：扫描数组找空闲槽 → initTower 初始化 → RollTowerStats 随机 S/B/D 档位
// → ApplyRandomStats 设置 Base/Potential 并 RecalcStats → allocSlot 注册空间索引。
// 池满时返回 nil（调用方应在建塔前检查 Count < cap）。
func (p *Pool) Place(row, col int, cx, cy float64, def TowerDef) *Tower {
	for i := range p.towers {
		if !p.towers[i].Active {
			t := &p.towers[i]
			initTower(t, row, col, cx, cy, def)

			// 随机属性：独立 roll 三档(S/B/D) + 专精，然后应用到 Base/Potential 并触发 RecalcStats
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

// PlaceFromSnapshot 从存档快照恢复塔，跳过随机 roll。
// 与 Place 的唯一区别：属性使用快照中保存的 Base/Potential/Tier/Specialty，
// 保证存档加载后塔的属性与保存时完全一致（否则每次加载会重新随机）。
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

// Remove 出售/销毁塔（标记为非存活，回收槽位）。
// 流程：触发 RemoveHook（清理关联弹射物等）→ 从空间索引移除 → 标记 Active=false → 清空 Target 避免悬空引用。
// 注意：不清零整个 Tower struct，因为出售动画可能还需要读取部分字段。下次 Place 时 initTower 会完整清零。
func (p *Pool) Remove(t *Tower) {
	if t.Active {
		if p.RemoveHook != nil {
			p.RemoveHook(t.InstanceKey) // 通知上层清理（如移除该塔发射的所有弹射物）
		}
		if t.Row >= 0 && t.Row < gridMaxRows && t.Col >= 0 && t.Col < gridMaxCols {
			p.grid[t.Row][t.Col] = nil // 从空间索引移除
		}
		t.Active = false
		t.Target = nil // 防止悬空指针（目标敌人可能在下一帧被回收）
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

// ByInstanceKey 根据 InstanceKey（格式 "key_row_col"）反查塔。O(1) via grid index.
//
// 解析逻辑：从字符串末尾向前找两个 '_' 分隔符，提取 row 和 col，
// 然后通过 At(row, col) 使用空间索引 O(1) 查找。
// 这比每帧构建 map[string]*Tower（之前的做法）节省大量分配。
//
// 示例："basic_3_7" → row=3, col=7 → p.grid[3][7]
// 边界：key 中塔名本身可能含 '_'（如 "fire_basic_3_7"），所以必须从末尾解析。
func (p *Pool) ByInstanceKey(key string) *Tower {
	// 从末尾找最后两个 '_' 分隔符，提取 row 和 col（从末尾解析是因为 key 前缀可能含 '_'）
	lastUS := strings.LastIndexByte(key, '_')
	if lastUS <= 0 {
		return nil
	}
	col, err := strconv.Atoi(key[lastUS+1:])
	if err != nil {
		return nil
	}
	rest := key[:lastUS]
	secondUS := strings.LastIndexByte(rest, '_')
	if secondUS < 0 {
		return nil
	}
	row, err := strconv.Atoi(rest[secondUS+1:])
	if err != nil {
		return nil
	}
	return p.At(row, col)
}

// At 返回指定网格位置 (row, col) 上的塔，无塔则返回 nil。O(1) via grid index.
// 双重检查：grid 有指针 + 塔确实 Active。grid 残留非 Active 指针是 Remove 后可能的瞬态。
func (p *Pool) At(row, col int) *Tower {
	if row < 0 || row >= gridMaxRows || col < 0 || col >= gridMaxCols {
		return nil
	}
	t := p.grid[row][col]
	if t != nil && !t.Active {
		return nil // 防御性检查：grid 可能残留已 Remove 的指针
	}
	return t
}

// TowerDef 塔类型定义，用于建造时初始化塔属性。
// 由 tower_loader.go 从 JSON 配置文件加载，运行时只读。
// 与 Tower struct 的区别：TowerDef 是"模板"（所有同类塔共享），Tower 是"实例"（每个放置的塔独立）。
// Cfg* 前缀的字段是 JSON 原始值，randomize.go 会用 tier-presets 覆盖实际的 Base*/Potential*。
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
	// 战力基础值+潜力值（从 JSON 扁平字段加载）
	// Base* 是强度 0 时的底线，Potential* 是每 100 强度的增量
	// 注意：Place 时这些值会被 ApplyRandomStats 用 tier-presets 覆盖，TowerDef 中的值仅作为兜底
	CfgBaseDamage   float64
	CfgBaseSpeed    float64
	CfgBaseRange    float64
	PotentialDamage float64
	PotentialSpeed  float64
	PotentialRange  float64

	// 升级费用
	UpgradeCosts []int // 每次升级费用（索引0=第1次升级，索引5=第6次升级）
}

// spriteKeyForStyle 将攻击方式映射到初始精灵标识。
// 仅在 initTower 时使用（设置建塔瞬间的外观），后续选择攻击能力后由 AbilitySpriteKey 覆盖。
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
