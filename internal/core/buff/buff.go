// buff.go — 通用 Buff 系统。
// 提供实体（塔/敌人）的增益/减益管理，支持持续时间、堆叠、刷新和优先级。
package buff

// Buff 单个 buff 实例。
type Buff struct {
	ID          string  // buff 唯一标识
	Type        string  // buff 类型（如 "slow"、"damageUp"）
	Source      string  // 来源标识（如塔 ID）
	Value       float64 // 效果数值
	Duration    float64 // 剩余持续时间（秒，<=0 表示永久）
	MaxDuration float64 // 初始持续时间
	Stacks      int     // 当前层数
	MaxStacks   int     // 最大可堆叠层数（0=不限）
	Refreshable bool    // 是否可刷新持续时间
	Active      bool    // 是否生效中
}

// BuffList 实体持有的 buff 列表。
type BuffList struct {
	Buffs []Buff // buff 切片
}

// NewBuffList 创建空 buff 列表。
func NewBuffList() *BuffList {
	return &BuffList{}
}

// Add 添加一个 buff。如果同 Type+Source 已存在且可堆叠，则叠层或刷新。
func (bl *BuffList) Add(b Buff) {
	b.Active = true
	if b.Stacks == 0 {
		b.Stacks = 1
	}

	// 查找已有同类 buff
	for i := range bl.Buffs {
		existing := &bl.Buffs[i]
		if existing.Active && existing.Type == b.Type && existing.Source == b.Source {
			// 堆叠
			if existing.MaxStacks > 0 && existing.Stacks < existing.MaxStacks {
				existing.Stacks++
				existing.Value += b.Value
			}
			// 刷新持续时间
			if existing.Refreshable {
				existing.Duration = existing.MaxDuration
			}
			return
		}
	}

	// 新 buff：复用已失效的槽位
	for i := range bl.Buffs {
		if !bl.Buffs[i].Active {
			bl.Buffs[i] = b
			return
		}
	}
	bl.Buffs = append(bl.Buffs, b)
}

// Tick 每帧更新所有 buff 持续时间，过期的标记为非活跃。
func (bl *BuffList) Tick(dt float64) {
	for i := range bl.Buffs {
		b := &bl.Buffs[i]
		if !b.Active {
			continue
		}
		if b.Duration > 0 {
			b.Duration -= dt
			if b.Duration <= 0 {
				b.Active = false
			}
		}
	}
}

// SumByType 统计指定类型所有活跃 buff 的 Value 之和。
func (bl *BuffList) SumByType(typ string) float64 {
	total := 0.0
	for _, b := range bl.Buffs {
		if b.Active && b.Type == typ {
			total += b.Value
		}
	}
	return total
}

// HasType 检查是否存在指定类型的活跃 buff。
func (bl *BuffList) HasType(typ string) bool {
	for _, b := range bl.Buffs {
		if b.Active && b.Type == typ {
			return true
		}
	}
	return false
}

// RemoveBySource 移除指定来源的所有 buff。
func (bl *BuffList) RemoveBySource(source string) {
	for i := range bl.Buffs {
		if bl.Buffs[i].Source == source {
			bl.Buffs[i].Active = false
		}
	}
}

// ClearAll 清除所有 buff。
func (bl *BuffList) ClearAll() {
	for i := range bl.Buffs {
		bl.Buffs[i].Active = false
	}
}

// ActiveCount 返回活跃 buff 数量。
func (bl *BuffList) ActiveCount() int {
	count := 0
	for _, b := range bl.Buffs {
		if b.Active {
			count++
		}
	}
	return count
}
