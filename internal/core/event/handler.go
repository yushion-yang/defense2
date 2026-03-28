// handler.go — 事件效果处理器注册表。
// 每种事件 Kind 对应一个处理函数，负责将事件效果应用到游戏状态上。
package event

// GameState 事件处理器可修改的游戏状态接口。
// 由 StageScene 实现，避免事件系统直接依赖 scene 包。
type GameState interface {
	AddGold(amount int)                // 增加金币
	SetBuildDiscount(ratio float64)    // 设置建造折扣比例
	SetKillRewardBonus(extra int)      // 设置额外击杀金币
	BuffAllTowersDamage(ratio float64) // 全体塔伤害加成（比例）
	BuffAllTowersRange(ratio float64)  // 全体塔射程加成（比例）
	BuffAllTowersSpeed(ratio float64)  // 全体塔攻速加成（比例）
	SlowAllEnemies(ratio float64)      // 全场敌人减速（比例）
}

// Handler 事件效果处理函数签名。
type Handler func(e *Event, gs GameState)

// 处理器注册表。
var handlers = map[string]Handler{
	"bonusGold":     handleBonusGold,
	"buildDiscount": handleBuildDiscount,
	"killRewardUp":  handleKillRewardUp,
	"outputUp":      handleOutputUp,
	"rangeUp":       handleRangeUp,
	"speedUp":       handleSpeedUp,
	"enemySlow":     handleEnemySlow,
}

// Apply 应用一个事件到游戏状态。
func Apply(e *Event, gs GameState) {
	if h, ok := handlers[e.Kind]; ok {
		h(e, gs)
	}
}

func handleBonusGold(e *Event, gs GameState) {
	gs.AddGold(int(e.Value))
}

func handleBuildDiscount(e *Event, gs GameState) {
	gs.SetBuildDiscount(e.Value)
}

func handleKillRewardUp(e *Event, gs GameState) {
	gs.SetKillRewardBonus(int(e.Value))
}

func handleOutputUp(e *Event, gs GameState) {
	gs.BuffAllTowersDamage(e.Value)
}

func handleRangeUp(e *Event, gs GameState) {
	gs.BuffAllTowersRange(e.Value)
}

func handleSpeedUp(e *Event, gs GameState) {
	gs.BuffAllTowersSpeed(e.Value)
}

func handleEnemySlow(e *Event, gs GameState) {
	gs.SlowAllEnemies(e.Value)
}
