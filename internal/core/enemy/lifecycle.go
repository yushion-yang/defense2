// lifecycle.go — 敌人生命周期钩子系统。
// 支持死亡/出生/受伤等事件的回调注册与触发，用于扩展敌人行为。
package enemy

import "fmt"

// LifecycleHandler 单个生命周期回调处理器。
type LifecycleHandler struct {
	ID    string                                    // 处理器唯一标识（用于去重和移除）
	Apply func(e *Enemy, ctx interface{}) interface{} // 回调函数
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

// ResolveEvent 触发指定事件的所有回调，返回各回调的结果。
// 单个回调 panic 不会影响后续回调执行。
func ResolveEvent(e *Enemy, event string, ctx interface{}) []interface{} {
	if e.Lifecycle == nil {
		return nil
	}

	handlers := getHandlers(e, event)
	if handlers == nil || len(*handlers) == 0 {
		return nil
	}

	results := make([]interface{}, 0, len(*handlers))
	for _, h := range *handlers {
		result := safeApply(h, e, ctx)
		results = append(results, result)
	}
	return results
}

// getHandlers 根据事件名获取对应的处理器切片指针。
func getHandlers(e *Enemy, event string) *[]LifecycleHandler {
	switch event {
	case "death":
		return &e.Lifecycle.OnDeath
	case "spawn":
		return &e.Lifecycle.OnSpawn
	case "damaged":
		return &e.Lifecycle.OnDamaged
	default:
		return nil
	}
}

// safeApply 安全执行回调，捕获 panic 防止崩溃。
func safeApply(h LifecycleHandler, e *Enemy, ctx interface{}) (result interface{}) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Errorf("lifecycle handler %q panicked: %v", h.ID, r)
		}
	}()
	return h.Apply(e, ctx)
}
