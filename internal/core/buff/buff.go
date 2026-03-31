// buff.go — 通用 Buff 系统。
// 提供实体（塔/敌人/战灵）的增益/减益管理。
// 支持 6 种堆叠模式、持续时间、回调（onApply/onExpire/onTick）和优先级。
package buff

import (
	"fmt"
	"math"

	tel "defense2/internal/core/telemetry"
	"sync/atomic"
)

// 全局 buff ID 自增器
var nextBuffID atomic.Int64

// Buff 单个 buff 实例。
type Buff struct {
	ID          string  // buff 唯一标识（自动生成）
	Type        string  // buff 类型（如 "slow"、"damageUp"）
	Source      string  // 来源标识（如塔 ID、技能名）
	SourceType  string  // 来源类型（如 "tower"、"skill"、"warden"）
	Value       float64 // 效果数值
	Duration    float64 // 剩余持续时间（秒，<=0 表示永久）
	MaxDuration float64 // 初始持续时间
	Stacks      int     // 当前层数
	MaxStacks   int     // 最大可堆叠层数（0=不限）
	Refreshable bool    // 是否可刷新持续时间
	Dispellable bool    // 是否可被净化移除
	Priority    float64 // 优先级（高优先 buff 不被低优先覆盖）
	Active      bool    // 是否生效中

	// 回调函数（可选）
	OnApply  func(target interface{}) // 应用时触发
	OnExpire func(target interface{}) // 过期时触发
	OnTick   func(target interface{}) // 周期性触发

	TickInterval float64 // OnTick 触发间隔（秒，0=不触发）
	TickDamage   float64 // 周期伤害（DOT 用）
	tickTimer    float64 // OnTick 内部计时器
}

// NewBuff 创建一个新的 buff 实例，自动分配唯一 ID。
func NewBuff(typ, source string, value, duration float64) Buff {
	id := nextBuffID.Add(1)
	b := Buff{
		ID:          fmt.Sprintf("buff_%d", id),
		Type:        typ,
		Source:      source,
		Value:       value,
		Duration:    duration,
		MaxDuration: duration,
		Stacks:      1,
		Dispellable: true,
		Active:      true,
	}
	return b
}

// BuffList 实体持有的 buff 列表。
type BuffList struct {
	Buffs []Buff               // buff 切片
	Rules map[string]StackRule // 堆叠规则（nil 则使用 DefaultStackRules）
}

// NewBuffList 创建空 buff 列表，使用默认堆叠规则。
func NewBuffList() *BuffList {
	return &BuffList{
		Rules: DefaultStackRules,
	}
}

// NewBuffListWithRules 创建使用自定义规则的 buff 列表。
func NewBuffListWithRules(rules map[string]StackRule) *BuffList {
	return &BuffList{Rules: rules}
}

// Add 按堆叠规则添加 buff。
// target: buff 挂载的实体（传入 OnApply 回调）。
func (bl *BuffList) Add(b Buff, target interface{}) {
	b.Active = true
	if b.Stacks == 0 {
		b.Stacks = 1
	}

	// 遥测：记录 buff 类型和堆叠模式
	if b.Type != "" {
		tel.T.Record("buff_type", b.Type)
	}
	rule := GetRule(b.Type, bl.Rules)
	tel.T.Record("buff_stack_mode", stackModeName(rule.Mode))

	switch rule.Mode {
	case ModeOverride:
		// 移除所有同类型旧 buff
		for i := range bl.Buffs {
			if bl.Buffs[i].Active && bl.Buffs[i].Type == b.Type {
				bl.deactivate(i, target)
			}
		}
		bl.insert(b, target)

	case ModeIndependent:
		// 直接添加
		bl.insert(b, target)

	case ModeIndependentPerSource:
		// 同来源刷新，不同来源独立
		for i := range bl.Buffs {
			existing := &bl.Buffs[i]
			if existing.Active && existing.Type == b.Type && existing.Source == b.Source {
				// 刷新持续时间和数值
				existing.Duration = b.MaxDuration
				existing.MaxDuration = b.MaxDuration
				existing.Value = b.Value
				return
			}
		}
		bl.insert(b, target)

	case ModeStrongest:
		// 检查是否有更强的已存在
		for i := range bl.Buffs {
			existing := &bl.Buffs[i]
			if existing.Active && existing.Type == b.Type {
				if math.Abs(b.Value) > math.Abs(existing.Value) {
					// 新的更强，替换
					bl.deactivate(i, target)
					bl.insert(b, target)
				} else if existing.Refreshable {
					// 已有更强，但刷新持续时间
					existing.Duration = existing.MaxDuration
				}
				return
			}
		}
		bl.insert(b, target)

	case ModeAdditive:
		// 检查是否达到堆叠上限
		existingCount := 0
		for i := range bl.Buffs {
			if bl.Buffs[i].Active && bl.Buffs[i].Type == b.Type {
				existingCount++
			}
		}
		if b.MaxStacks > 0 && existingCount >= b.MaxStacks {
			// 刷新最旧的
			for i := range bl.Buffs {
				existing := &bl.Buffs[i]
				if existing.Active && existing.Type == b.Type && existing.Refreshable {
					existing.Duration = existing.MaxDuration
					return
				}
			}
			return
		}
		bl.insert(b, target)

	case ModeMultiplicative:
		bl.insert(b, target)
	}
}

// AddSimple 简化版添加（无目标回调）。
func (bl *BuffList) AddSimple(b Buff) {
	bl.Add(b, nil)
}

// Tick 每帧更新所有 buff。
// 递减持续时间、触发 OnTick 回调、移除过期 buff 并触发 OnExpire。
// 返回本帧过期的 buff 数量。
func (bl *BuffList) Tick(dt float64, target interface{}) int {
	expired := 0
	for i := range bl.Buffs {
		b := &bl.Buffs[i]
		if !b.Active {
			continue
		}

		// OnTick 周期回调
		if b.OnTick != nil && b.TickInterval > 0 {
			b.tickTimer += dt
			for b.tickTimer >= b.TickInterval {
				b.tickTimer -= b.TickInterval
				b.OnTick(target)
			}
		}

		// 持续时间递减
		if b.Duration > 0 {
			b.Duration -= dt
			if b.Duration <= 0 {
				bl.deactivate(i, target)
				expired++
			}
		}
	}
	return expired
}

// TickSimple 简化版 Tick（无目标回调）。
func (bl *BuffList) TickSimple(dt float64) int {
	return bl.Tick(dt, nil)
}

// GetEffective 按堆叠规则解算指定类型 buff 的生效值。
func (bl *BuffList) GetEffective(typ string) float64 {
	return ResolveStack(typ, bl.Buffs, bl.Rules)
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

// GetBuffsOfType 返回指定类型的所有活跃 buff。
func (bl *BuffList) GetBuffsOfType(typ string) []Buff {
	var result []Buff
	for _, b := range bl.Buffs {
		if b.Active && b.Type == typ {
			result = append(result, b)
		}
	}
	return result
}

// RemoveByID 按 ID 移除 buff。
func (bl *BuffList) RemoveByID(id string, target interface{}) bool {
	for i := range bl.Buffs {
		if bl.Buffs[i].ID == id && bl.Buffs[i].Active {
			bl.deactivate(i, target)
			return true
		}
	}
	return false
}

// RemoveBySource 移除指定来源的所有 buff。
func (bl *BuffList) RemoveBySource(source string, target interface{}) int {
	count := 0
	for i := range bl.Buffs {
		if bl.Buffs[i].Active && bl.Buffs[i].Source == source {
			bl.deactivate(i, target)
			count++
		}
	}
	return count
}

// PurgeDispellable 净化所有可驱散的 buff，返回被移除的数量。
// filterFn 可选，为 nil 则净化所有可驱散的。
func (bl *BuffList) PurgeDispellable(target interface{}, filterFn func(b *Buff) bool) int {
	count := 0
	for i := range bl.Buffs {
		b := &bl.Buffs[i]
		if !b.Active || !b.Dispellable {
			continue
		}
		if filterFn != nil && !filterFn(b) {
			continue
		}
		bl.deactivate(i, target)
		count++
	}
	return count
}

// ClearAll 清除所有 buff（触发所有 OnExpire）。
func (bl *BuffList) ClearAll(target interface{}) {
	for i := range bl.Buffs {
		if bl.Buffs[i].Active {
			bl.deactivate(i, target)
		}
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

// ---- 状态查询快捷方法 ----

// IsUntargetable 是否不可选中。
func (bl *BuffList) IsUntargetable() bool { return bl.HasType("untargetable") }

// IsInvincible 是否无敌。
func (bl *BuffList) IsInvincible() bool { return bl.HasType("invincible") }

// IsDamageImmune 是否伤害免疫。
func (bl *BuffList) IsDamageImmune() bool { return bl.HasType("damageImmune") }

// IsControlImmune 是否控制免疫。
func (bl *BuffList) IsControlImmune() bool { return bl.HasType("controlImmune") }

// IsStunImmune 是否眩晕免疫。
func (bl *BuffList) IsStunImmune() bool { return bl.HasType("stunImmune") || bl.IsControlImmune() }

// IsSlowImmune 是否减速免疫。
func (bl *BuffList) IsSlowImmune() bool { return bl.HasType("slowImmune") || bl.IsControlImmune() }

// IsRootImmune 是否定身免疫。
func (bl *BuffList) IsRootImmune() bool { return bl.HasType("rootImmune") || bl.IsControlImmune() }

// ---- 内部方法 ----

// insert 插入 buff 到列表（复用失效槽位），触发 OnApply。
func (bl *BuffList) insert(b Buff, target interface{}) {
	// 复用失效槽位
	for i := range bl.Buffs {
		if !bl.Buffs[i].Active {
			bl.Buffs[i] = b
			if b.OnApply != nil {
				b.OnApply(target)
			}
			return
		}
	}
	bl.Buffs = append(bl.Buffs, b)
	if b.OnApply != nil {
		b.OnApply(target)
	}
}

// deactivate 将指定索引的 buff 设为失效，触发 OnExpire。
func (bl *BuffList) deactivate(idx int, target interface{}) {
	b := &bl.Buffs[idx]
	if !b.Active {
		return
	}
	b.Active = false
	if b.OnExpire != nil {
		b.OnExpire(target)
	}
}
