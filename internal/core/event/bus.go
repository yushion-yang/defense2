// bus.go — 全局事件总线。
// 发布/订阅模式，用于游戏系统间的松耦合通信。
// 任何系统可以 Emit 事件，其他系统通过 On 订阅。
package event

import "sync"

// 预定义事件名称常量
const (
	EvtTowerBuilt    = "towerBuilt"    // 塔建造完成
	EvtTowerUpgraded = "towerUpgraded" // 塔升级完成
	EvtTowerSold     = "towerSold"     // 塔出售
	EvtEnemyKilled   = "enemyKilled"   // 敌人被击杀
	EvtEnemyLeaked   = "enemyLeaked"   // 敌人泄漏到终点
	EvtWaveStarted   = "waveStarted"   // 波次开始
	EvtWaveCleared   = "waveCleared"   // 波次全部清除
	EvtDamageDealt   = "damageDealt"   // 伤害产生
	EvtGoldChanged   = "goldChanged"   // 金币变更
)

// BusHandler 事件总线处理函数类型（区别于 handler.go 中的事件效果 Handler）。
type BusHandler func(args ...interface{})

// Bus 事件总线，线程安全的发布/订阅中心。
type Bus struct {
	mu        sync.RWMutex          // 读写锁保护 listeners
	listeners map[string][]busEntry // 事件名 → 处理器列表
	nextID    int                   // 处理器 ID 自增器
}

// busEntry 带 ID 的处理器条目，用于精确取消订阅。
type busEntry struct {
	id int        // 唯一标识
	fn BusHandler // 处理函数
}

// NewBus 创建一个空的事件总线。
func NewBus() *Bus {
	return &Bus{
		listeners: make(map[string][]busEntry, 16),
	}
}

// On 订阅指定事件，返回取消订阅函数。
// 同一处理器可多次订阅，每次返回独立的取消函数。
func (b *Bus) On(event string, fn BusHandler) func() {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.listeners[event] = append(b.listeners[event], busEntry{id: id, fn: fn})
	b.mu.Unlock()

	// 返回取消函数（闭包捕获 id）
	return func() {
		b.removeByID(event, id)
	}
}

// Off 移除指定事件的所有处理器。
func (b *Bus) Off(event string) {
	b.mu.Lock()
	delete(b.listeners, event)
	b.mu.Unlock()
}

// Emit 触发指定事件，按注册顺序调用所有处理器。
// 内部复制一份处理器列表后再遍历，防止回调中修改订阅导致死锁。
func (b *Bus) Emit(event string, args ...interface{}) {
	b.mu.RLock()
	entries := b.listeners[event]
	if len(entries) == 0 {
		b.mu.RUnlock()
		return
	}
	// 复制一份避免并发修改
	snapshot := make([]busEntry, len(entries))
	copy(snapshot, entries)
	b.mu.RUnlock()

	for _, entry := range snapshot {
		entry.fn(args...)
	}
}

// Clear 清除所有事件的所有处理器。
func (b *Bus) Clear() {
	b.mu.Lock()
	b.listeners = make(map[string][]busEntry)
	b.mu.Unlock()
}

// ListenerCount 返回指定事件的订阅者数量（调试用）。
func (b *Bus) ListenerCount(event string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.listeners[event])
}

// removeByID 按 ID 移除特定处理器。
func (b *Bus) removeByID(event string, id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entries := b.listeners[event]
	for i, e := range entries {
		if e.id == id {
			b.listeners[event] = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}
