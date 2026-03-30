package event

// OnTyped 订阅类型化事件，返回取消函数。
// 当 payload 类型不匹配时静默跳过（不 panic）。
func OnTyped[T any](b *Bus, evt string, fn func(T)) func() {
	return b.On(evt, func(args ...interface{}) {
		if len(args) > 0 {
			if payload, ok := args[0].(T); ok {
				fn(payload)
			}
		}
	})
}
