// bus.go — 全局事件总线。
// 发布/订阅模式，用于游戏系统间的松耦合通信。
// 任何系统可以 Emit 事件，其他系统通过 On 订阅。
//
// 注意：Bus 设计为单线程使用（Ebitengine 的 Update/Draw 同一 goroutine），
// 不使用 mutex。如果需要跨 goroutine 使用，需外部加锁。
package event

// 预定义事件名称常量
const (
	EvtTowerBuilt    = "towerBuilt"    // 塔建造完成
	EvtTowerUpgraded = "towerUpgraded" // 塔升级完成
	EvtTowerSold     = "towerSold"     // 塔出售
	EvtEnemyKilled   = "enemyKilled"   // 敌人被击杀
	EvtEnemyLeaked   = "enemyLeaked"   // 敌人泄漏到终点
	EvtWaveStarted   = "waveStarted"   // 波次开始
	EvtWaveCleared   = "waveCleared"   // 波次全部清除
)

// BusHandler 事件总线处理函数类型（区别于 handler.go 中的事件效果 Handler）。
type BusHandler func(args ...any)

// Bus 事件总线，单线程发布/订阅中心。
type Bus struct {
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
	b.nextID++
	id := b.nextID
	b.listeners[event] = append(b.listeners[event], busEntry{id: id, fn: fn})

	// 返回取消函数（闭包捕获 id）
	return func() {
		b.removeByID(event, id)
	}
}

// Off 移除指定事件的所有处理器。
func (b *Bus) Off(event string) {
	delete(b.listeners, event)
}

// Emit 触发指定事件，按注册顺序调用所有处理器。
// 内部复制一份处理器列表后再遍历，防止回调中修改订阅导致迭代错乱。
func (b *Bus) Emit(event string, args ...any) {
	entries := b.listeners[event]
	n := len(entries)
	if n == 0 {
		return
	}
	// 栈数组避免堆分配（最多 16 个处理器走快速路径）
	var buf [16]busEntry
	var snapshot []busEntry
	if n <= len(buf) {
		snapshot = buf[:n]
	} else {
		snapshot = make([]busEntry, n)
	}
	copy(snapshot, entries)

	for _, entry := range snapshot {
		entry.fn(args...)
	}
}

// Clear 清除所有事件的所有处理器。
func (b *Bus) Clear() {
	clear(b.listeners)
}

// ListenerCount 返回指定事件的订阅者数量（调试用）。
func (b *Bus) ListenerCount(event string) int {
	return len(b.listeners[event])
}

// removeByID 按 ID 移除特定处理器。
func (b *Bus) removeByID(event string, id int) {
	entries := b.listeners[event]
	for i, e := range entries {
		if e.id == id {
			b.listeners[event] = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// OnTyped 订阅类型化事件，返回取消函数。
// 当 payload 类型不匹配时静默跳过（不 panic）。
func OnTyped[T any](b *Bus, evt string, fn func(T)) func() {
	return b.On(evt, func(args ...any) {
		if len(args) > 0 {
			if payload, ok := args[0].(T); ok {
				fn(payload)
			}
		}
	})
}

// ── 事件载荷 ──────────────────────────────────────

// TowerBuiltPayload 塔建造完成事件载荷。
type TowerBuiltPayload struct {
	TowerKey string // 塔类型 ID（如 "sentinel", "prism"）
	Cost     int    // 建造花费
}

// TowerUpgradedPayload 塔升级完成事件载荷。
type TowerUpgradedPayload struct {
	TowerKey string // 塔类型 ID
	Spent    int    // 本次升级花费
}

// TowerSoldPayload 塔出售事件载荷。
type TowerSoldPayload struct {
	TowerKey string // 塔类型 ID
	Refund   int    // 退还金额
}

// EnemyKilledPayload 敌人击杀事件载荷。
type EnemyKilledPayload struct {
	IsBoss    bool   // 是否 Boss
	KillerID  string // 击杀来源（"projectile"/"warden"）
	GoldValue int    // 击杀金币（预计算）
	Archetype string // 敌人原型 ID（图鉴统计用）
}

// EnemyLeakedPayload 敌人泄漏事件载荷。
type EnemyLeakedPayload struct{}

// WaveStartedPayload 波次开始事件载荷。
type WaveStartedPayload struct {
	Wave   int  // 波次序号
	IsBoss bool // 是否 Boss 波（wave%5==0）
}

// WaveClearedPayload 波次清除事件载荷。
type WaveClearedPayload struct {
	Wave    int  // 已清除的波次序号
	Perfect bool // 是否完美（无泄漏）
}
