// lifecycle.go — 塔生命周期钩子系统。
// 提供创建/销毁/升级的观察者模式，外部模块通过注册函数响应塔事件。
package tower

import "log"

// 模块级钩子数组（观察者模式）。
var (
	onCreateHooks  []func(t *Tower) // 创建时触发
	onDestroyHooks []func(t *Tower) // 销毁时触发
	onUpgradeHooks []func(t *Tower) // 升级时触发
)

// OnTowerCreate 注册塔创建钩子。
func OnTowerCreate(hook func(t *Tower)) {
	onCreateHooks = append(onCreateHooks, hook)
}

// OnTowerDestroy 注册塔销毁钩子。
func OnTowerDestroy(hook func(t *Tower)) {
	onDestroyHooks = append(onDestroyHooks, hook)
}

// OnTowerUpgrade 注册塔升级钩子。
func OnTowerUpgrade(hook func(t *Tower)) {
	onUpgradeHooks = append(onUpgradeHooks, hook)
}

// EmitCreate 执行所有创建钩子（内部 recover 防止 panic 扩散）。
func EmitCreate(t *Tower) {
	executeHooks(onCreateHooks, t)
}

// EmitDestroy 执行所有销毁钩子。
func EmitDestroy(t *Tower) {
	executeHooks(onDestroyHooks, t)
}

// EmitUpgrade 执行所有升级钩子。
func EmitUpgrade(t *Tower) {
	executeHooks(onUpgradeHooks, t)
}

// ResetHooks 清空所有钩子（测试辅助函数）。
func ResetHooks() {
	onCreateHooks = nil
	onDestroyHooks = nil
	onUpgradeHooks = nil
}

// executeHooks 安全执行一组钩子，每个钩子独立 recover。
func executeHooks(hooks []func(t *Tower), t *Tower) {
	for _, hook := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("tower lifecycle hook panic: %v", r)
				}
			}()
			hook(t)
		}()
	}
}
