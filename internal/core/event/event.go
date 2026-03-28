// event.go — 事件系统 v2。
// 事件是玩家在特定波次检查点选择的增益效果，分为两个等级：
//
//	Tier 1: 经济类（额外金币、建造折扣）
//	Tier 2: 全局属性增益（伤害、射程、攻速提升）
//
// 事件通过加权随机从事件池中抽取，玩家三选一。
package event

import (
	"math/rand"
)

// Event 单个事件定义。
type Event struct {
	ID          string  // 事件唯一标识
	Label       string  // 显示名称
	Description string  // 效果描述
	Kind        string  // 处理器类型标识（如 "bonusGold"、"outputUp"）
	Tier        int     // 等级（1=经济, 2=全局）
	Value       float64 // 效果数值
	Weight      float64 // 抽取权重（越大越容易抽到）
	MinWave     int     // 最早出现波次（0=无限制）
	MaxWave     int     // 最晚出现波次（0=无限制）
}

// Pool 事件池，持有所有可用事件。
type Pool struct {
	events []Event // 全部事件
}

// NewPool 从事件列表创建事件池。
func NewPool(events []Event) *Pool {
	return &Pool{events: events}
}

// Pick 从池中按权重随机抽取 n 个不重复事件，可按波次过滤。
func (p *Pool) Pick(n int, wave int) []Event {
	// 过滤可用事件
	eligible := make([]Event, 0)
	for _, e := range p.events {
		if e.MinWave > 0 && wave < e.MinWave {
			continue
		}
		if e.MaxWave > 0 && wave > e.MaxWave {
			continue
		}
		eligible = append(eligible, e)
	}
	if len(eligible) == 0 {
		return nil
	}

	// 加权随机抽取不重复
	picked := make([]Event, 0, n)
	remaining := make([]Event, len(eligible))
	copy(remaining, eligible)

	for i := 0; i < n && len(remaining) > 0; i++ {
		idx := weightedRandom(remaining)
		picked = append(picked, remaining[idx])
		remaining = append(remaining[:idx], remaining[idx+1:]...)
	}
	return picked
}

// weightedRandom 按权重随机选择一个索引。
func weightedRandom(events []Event) int {
	totalWeight := 0.0
	for _, e := range events {
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}
	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for i, e := range events {
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if r <= cumulative {
			return i
		}
	}
	return len(events) - 1
}

// PickTiered 按 Tier 优先级抽取事件（1 个 Tier3 + 2 个 Tier2，不足时用 Tier1 补齐）。
func (p *Pool) PickTiered(wave int) []Event {
	t1 := filterByTier(p.events, 1, wave)
	t2 := filterByTier(p.events, 2, wave)
	t3 := filterByTier(p.events, 3, wave)

	var result []Event

	// 优先 1 个 Tier3
	if len(t3) > 0 {
		idx := weightedRandom(t3)
		result = append(result, t3[idx])
	}

	// 2 个 Tier2
	remaining2 := make([]Event, len(t2))
	copy(remaining2, t2)
	for i := 0; i < 2 && len(remaining2) > 0; i++ {
		idx := weightedRandom(remaining2)
		result = append(result, remaining2[idx])
		remaining2 = append(remaining2[:idx], remaining2[idx+1:]...)
	}

	// 不足 3 个用 Tier1 补齐
	remaining1 := make([]Event, len(t1))
	copy(remaining1, t1)
	for len(result) < 3 && len(remaining1) > 0 {
		idx := weightedRandom(remaining1)
		result = append(result, remaining1[idx])
		remaining1 = append(remaining1[:idx], remaining1[idx+1:]...)
	}

	return result
}

// filterByTier 过滤指定等级的事件。
func filterByTier(events []Event, tier int, wave int) []Event {
	var result []Event
	for _, e := range events {
		if e.Tier != tier {
			continue
		}
		if e.MinWave > 0 && wave < e.MinWave {
			continue
		}
		if e.MaxWave > 0 && wave > e.MaxWave {
			continue
		}
		result = append(result, e)
	}
	return result
}

// DefaultAllyEvents 返回内置的增益事件列表。
func DefaultAllyEvents() []Event {
	return []Event{
		// Tier 1 经济类
		{ID: "bonus-gold", Label: "额外金币", Kind: "bonusGold", Tier: 1, Value: 80, Weight: 1},
		{ID: "build-discount", Label: "建造折扣", Kind: "buildDiscount", Tier: 1, Value: 0.15, Weight: 1},
		{ID: "kill-reward-up", Label: "击杀赏金", Kind: "killRewardUp", Tier: 1, Value: 5, Weight: 1},

		// Tier 2 全局增益
		{ID: "output-surge", Label: "伤害激增", Kind: "outputUp", Tier: 2, Value: 0.15, Weight: 1},
		{ID: "range-extend", Label: "射程扩展", Kind: "rangeUp", Tier: 2, Value: 0.10, Weight: 1},
		{ID: "speed-boost", Label: "攻速提升", Kind: "speedUp", Tier: 2, Value: 0.12, Weight: 1},
		{ID: "durability", Label: "敌人减速", Kind: "enemySlow", Tier: 2, Value: 0.10, Weight: 0.8},
	}
}

// RewardWaves 返回触发事件选择的波次列表。
func RewardWaves() []int {
	return []int{4, 8, 12, 16, 20}
}
