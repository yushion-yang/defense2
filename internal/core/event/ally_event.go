// ally_event.go — 关卡事件系统 stub。
// TODO: 完整实现奖励波次事件选择。
package event

// Event 关卡事件（如增益/强化等，奖励波次弹出选择）。
type Event struct {
	ID          string
	Name        string
	Label       string // 显示标签
	Description string
	Tier        int
	Icon        string
}

// Pool 事件池，按波次阶段提供事件抽取。
type Pool struct {
	events []Event
}

// NewPool 创建事件池。
func NewPool(events []Event) *Pool {
	return &Pool{events: events}
}

// PickTiered 按波次阶段随机抽取事件（stub：返回空）。
func (p *Pool) PickTiered(wave int) []Event {
	// TODO: implement tiered event picking
	return nil
}

// Apply 应用事件效果到游戏场景（stub）。
func Apply(ev *Event, target interface{}) {
	// TODO: implement event effects
}

// RewardWaves 返回奖励波次列表（stub：返回空）。
func RewardWaves() []int {
	// TODO: load from config/settings.json rewardWaves
	return nil
}
