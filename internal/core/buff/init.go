// init.go — 全局堆叠规则单例管理。
//
// 游戏启动时，config 包通过 go:embed 加载 buff-stack.json 并调用 InitGlobalRules() 一次。
// 此后所有 NewDefaultBuffList() 调用共享同一份规则映射，无需每次传参。
//
// 生命周期：
//  1. config.Init() → InitGlobalRules(jsonBytes) — 启动时一次
//  2. enemy/pool.go Spawn → NewDefaultBuffList()  — 每个敌人创建时
//  3. tower.NewTower() → NewDefaultBuffList()      — 每座塔创建时
//
// sync.Once 保证并发安全且仅初始化一次（即使多次调用 InitGlobalRules 也不会重复解析）。
package buff

import "sync"

var (
	globalRules     map[string]StackRule // 全局堆叠规则（进程级单例）
	globalRulesOnce sync.Once            // 保证 InitGlobalRules 仅执行一次
)

// InitGlobalRules 解析 buff-stack.json 并将结果存为进程级单例。
// 由 sync.Once 保护：首次调用解析并存储，后续调用直接返回 nil（无操作）。
// 若 JSON 解析失败，返回错误但 sync.Once 已消耗，后续调用不会重试。
// 因此调用方必须检查返回值，解析失败应视为致命错误。
func InitGlobalRules(jsonData []byte) error {
	var initErr error
	globalRulesOnce.Do(func() {
		rules, err := LoadRules(jsonData)
		if err != nil {
			initErr = err
			return
		}
		globalRules = rules
	})
	return initErr
}

// GlobalRules 返回已加载的全局堆叠规则映射。
// 若 InitGlobalRules 尚未调用则返回 nil。
// 主要供测试或需要自定义规则的场景使用；生产代码应使用 NewDefaultBuffList()。
func GlobalRules() map[string]StackRule {
	return globalRules
}

// NewDefaultBuffList 使用全局堆叠规则创建 BuffList。
// 这是生产代码创建 BuffList 的推荐方式。
// 若全局规则未初始化（InitGlobalRules 未调用），退化为空规则集——
// 所有 buff 将使用 Override 模式，这是安全的兜底行为（测试场景常见）。
func NewDefaultBuffList() *BuffList {
	rules := globalRules
	if rules == nil {
		rules = map[string]StackRule{}
	}
	return NewBuffList(rules)
}
