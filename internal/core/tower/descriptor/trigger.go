// trigger.go — 触发器类型定义及解析。
//
// TriggerType 定义效果的触发时机（命中/每帧/击杀/放置）。
// 由 descriptor 引擎在加载 JSON 描述符时通过 ParseTrigger 解析。
package descriptor

import "fmt"

// TriggerType 效果触发时机枚举。
type TriggerType int

const (
	TriggerOnHit   TriggerType = iota // 命中敌人时
	TriggerOnTick                     // 每帧 tick
	TriggerOnKill                     // 击杀敌人时
	TriggerOnPlace                    // 放置塔时
)

var triggerNames = map[TriggerType]string{
	TriggerOnHit:   "onHit",
	TriggerOnTick:  "onTick",
	TriggerOnKill:  "onKill",
	TriggerOnPlace: "onPlace",
}

var triggerByName = map[string]TriggerType{
	"onHit":   TriggerOnHit,
	"onTick":  TriggerOnTick,
	"onKill":  TriggerOnKill,
	"onPlace": TriggerOnPlace,
}

// String 返回触发器的 camelCase 名称。
func (t TriggerType) String() string {
	if name, ok := triggerNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TriggerType(%d)", t)
}

// ParseTrigger 将字符串解析为 TriggerType，大小写敏感。
func ParseTrigger(s string) (TriggerType, error) {
	if tt, ok := triggerByName[s]; ok {
		return tt, nil
	}
	return 0, fmt.Errorf("unknown trigger type: %q", s)
}
