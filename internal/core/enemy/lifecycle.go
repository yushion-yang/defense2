// lifecycle.go — 敌人生命周期钩子系统（观察者模式）。
//
// 提供 death/spawn/damaged 三种事件的回调注册与触发机制，
// 用于解耦敌人核心逻辑与外围扩展行为（如成就统计、音效触发、特殊死亡效果）。
//
// 设计要点：
//   - 每个 Enemy 持有自己的 LifecycleHandlers（懒初始化，多数敌人不需要）
//   - Handler 通过 ID 去重，同一 ID 不会重复注册
//   - EmitEvent 使用 panic recover 保护，单个回调崩溃不影响其他回调
//   - 回调结果收集为 []any 返回，调用方按需类型断言
//
// 调用时机：
//   - death: pool.go Kill() 前由 combat 层触发（用于奖励计算、统计）
//   - spawn: 目前由 stage 层在 Spawn 后主动触发（不在 pool.go 中自动调用）
//   - damaged: apply_hit.go 中受伤后触发（用于受伤反应、阈值检查）
//
// 关联文件：
//   - pool.go: Kill/Spawn 生命周期管理
//   - ids.go: LifecycleDeath/LifecycleSpawn/LifecycleDamaged 常量
package enemy

import "fmt"

// LifecycleHandler 单个生命周期回调处理器。
type LifecycleHandler struct {
	ID    string                      // 处理器唯一标识（用于去重和移除）
	Apply func(e *Enemy, ctx any) any // 回调函数
}

// LifecycleHandlers 敌人生命周期回调集合。
type LifecycleHandlers struct {
	OnDeath   []LifecycleHandler // 死亡时触发
	OnSpawn   []LifecycleHandler // 出生时触发
	OnDamaged []LifecycleHandler // 受伤时触发
}

// InitLifecycle 初始化生命周期回调（如果尚未初始化）。
func InitLifecycle(e *Enemy) {
	if e.Lifecycle == nil {
		e.Lifecycle = &LifecycleHandlers{
			OnDeath:   make([]LifecycleHandler, 0, 4),
			OnSpawn:   make([]LifecycleHandler, 0, 4),
			OnDamaged: make([]LifecycleHandler, 0, 4),
		}
	}
}

// RegisterHandler 注册一个生命周期回调处理器。
// event 支持 "death"、"spawn"、"damaged"。
// 同一 ID 不会重复注册。
func RegisterHandler(e *Enemy, event string, handler LifecycleHandler) {
	InitLifecycle(e)

	handlers := getHandlers(e, event)
	if handlers == nil {
		return
	}

	// 去重：已存在相同 ID 的处理器则跳过
	for _, h := range *handlers {
		if h.ID == handler.ID {
			return
		}
	}

	*handlers = append(*handlers, handler)
}

// RemoveHandler 移除指定 ID 的生命周期回调处理器。
func RemoveHandler(e *Enemy, event string, handlerID string) {
	if e.Lifecycle == nil {
		return
	}

	handlers := getHandlers(e, event)
	if handlers == nil {
		return
	}

	for i, h := range *handlers {
		if h.ID == handlerID {
			// 保持顺序移除
			*handlers = append((*handlers)[:i], (*handlers)[i+1:]...)
			return
		}
	}
}

// EmitEvent 触发指定事件的所有回调，返回各回调的结果列表。
// ctx 是事件上下文（类型由调用方和 handler 约定，如 DamageContext/KillContext）。
// 单个回调 panic 不会影响后续回调执行（safeApply 捕获 panic 返回 error）。
func EmitEvent(e *Enemy, event string, ctx any) []any {
	if e.Lifecycle == nil {
		return nil
	}

	handlers := getHandlers(e, event)
	if handlers == nil || len(*handlers) == 0 {
		return nil
	}

	results := make([]any, 0, len(*handlers))
	for _, h := range *handlers {
		result := safeApply(h, e, ctx)
		results = append(results, result)
	}
	return results
}

// getHandlers 根据事件名获取对应的处理器切片指针。
func getHandlers(e *Enemy, event string) *[]LifecycleHandler {
	switch event {
	case LifecycleDeath:
		return &e.Lifecycle.OnDeath
	case LifecycleSpawn:
		return &e.Lifecycle.OnSpawn
	case LifecycleDamaged:
		return &e.Lifecycle.OnDamaged
	default:
		return nil
	}
}

// safeApply 安全执行回调，捕获 panic 防止崩溃。
func safeApply(h LifecycleHandler, e *Enemy, ctx any) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Errorf("lifecycle handler %q panicked: %v", h.ID, r)
		}
	}()
	return h.Apply(e, ctx)
}
