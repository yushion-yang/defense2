// lifecycle.go — 塔生命周期钩子系统（观察者模式）。
//
// 本文件提供塔创建/销毁/升级三种事件的观察者注册与分发机制。
// 外部模块（如成就系统、统计系统、音效系统）通过注册钩子函数响应塔事件，
// 而不需要 tower 包反向依赖它们（解耦方向：tower → lifecycle hooks ← 外部模块）。
//
// 设计要点：
//   - 模块级变量存储钩子数组（非 Pool 级），因为钩子通常是全局性的
//   - 每个钩子独立 recover，单个钩子 panic 不会影响其他钩子和主游戏循环
//   - ResetHooks() 供测试使用，避免跨测试钩子残留
//   - 注意：BuildAnim/SellAnim 动画字段定义在 tower.go 中，
//     动画播放和经济结算（退还金币）由 stage 层 / pipeline 处理，不在本文件
//
// 与其他文件的关系：
//   - pool.go: Place 后调用 EmitCreate，Remove 前调用 EmitDestroy
//   - upgrade.go: AddAbility 后调用 EmitUpgrade
//   - 外部模块: 通过 OnTowerCreate/OnTowerDestroy/OnTowerUpgrade 注册监听
package tower

import "log"

// 模块级钩子数组（观察者模式）。
// 使用模块级变量而非 Pool 方法，因为钩子通常是全局性的（如成就系统只需注册一次）。
// 钩子在程序启动时注册，运行期间不增减（append-only，无并发问题）。
var (
	onCreateHooks  []func(t *Tower) // 塔创建后触发（参数：新建的塔）
	onDestroyHooks []func(t *Tower) // 塔销毁前触发（参数：即将销毁的塔，字段仍可读）
	onUpgradeHooks []func(t *Tower) // 塔升级后触发（参数：刚升级的塔，AbilitySlots 已更新）
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

// ResetHooks 清空所有钩子（仅测试使用）。
// 必要性：Go 测试在同一进程中运行，模块级变量会跨测试残留。
// 不调用此函数会导致测试 A 注册的钩子在测试 B 中意外触发。
func ResetHooks() {
	onCreateHooks = nil
	onDestroyHooks = nil
	onUpgradeHooks = nil
}

// executeHooks 安全执行一组钩子，每个钩子独立 recover。
// 使用闭包+defer 隔离每个钩子的 panic：
//   - 钩子 A panic → 打日志 → 继续执行钩子 B/C（不中断循环）
//   - 不会向上层传播 panic（游戏循环不会因钩子 bug 崩溃）
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
