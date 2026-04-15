// action.go — 延迟行动队列。
//
// AI 决策后不立即执行，而是排入队列，经过可配置的延迟后才执行。
// 延迟模拟"思考时间"，让 AI 行为节奏更像真人。
package aiplayer

// DelayedAction 延迟行动。
type DelayedAction struct {
	Delay   float64 // 延迟秒数
	Execute func()  // 执行函数
	Label   string  // 行动标签（用于气泡文案关联）
	elapsed float64 // 已等待时间
}

// ActionQueue 延迟行动队列（FIFO）。
type ActionQueue struct {
	queue []DelayedAction
}

// NewActionQueue 创建空队列。
func NewActionQueue() *ActionQueue {
	return &ActionQueue{}
}

// Enqueue 入队一个延迟行动。
func (q *ActionQueue) Enqueue(a DelayedAction) {
	q.queue = append(q.queue, a)
}

// Tick 每帧更新，到时间的行动自动执行并移除。
func (q *ActionQueue) Tick(dt float64) {
	if len(q.queue) == 0 {
		return
	}
	// 只处理队首（FIFO，一次只执行一个行动）
	q.queue[0].elapsed += dt
	if q.queue[0].elapsed >= q.queue[0].Delay {
		q.queue[0].Execute()
		q.queue = q.queue[1:]
	}
}

// Pending 待执行数量。
func (q *ActionQueue) Pending() int {
	return len(q.queue)
}

// Busy 是否有待执行的行动。
func (q *ActionQueue) Busy() bool {
	return len(q.queue) > 0
}

// PeekLabel 查看队首行动标签。
func (q *ActionQueue) PeekLabel() string {
	if len(q.queue) == 0 {
		return ""
	}
	return q.queue[0].Label
}

// Clear 清空队列。
func (q *ActionQueue) Clear() {
	q.queue = q.queue[:0]
}
