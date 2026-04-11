// ids.go — 敌人子系统唯一标识符常量。
// 行为类型、过滤器、生命周期事件的魔法字符串消灭。
package enemy

// 敌人行为类型
const (
	BehaviorHealer      = "healer"
	BehaviorStealth     = "stealth"
	BehaviorBuffer      = "buffer"
	BehaviorSplitter    = "splitter"
	BehaviorBerserk     = "berserk"
	BehaviorRegenerator = "regenerator"
)

// 敌人过滤器（Spawner.EnemyFilter 的合法值）
const (
	FilterNone       = "none"
	FilterMixed      = "mixed"
	FilterGroundOnly = "ground-only"
	FilterFlyingOnly = "flying-only"
	FilterBossOnly   = "boss-only"
	FilterDummy      = "dummy"
	FilterStress     = "stress"
	FilterAllStatic  = "all-static"
)

// 生命周期事件名
const (
	LifecycleDeath   = "death"
	LifecycleSpawn   = "spawn"
	LifecycleDamaged = "damaged"
)
