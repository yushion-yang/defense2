// event_loader.go — 事件配置→运行时事件转换器。
// 从 config 包的 JSON 事件数据转换为 event.Event 运行时结构。
package loader

import (
	"log"

	"defense2/internal/config"
	"defense2/internal/core/event"
)

// LoadAllyEvents 加载增益事件并转为运行时格式。
func LoadAllyEvents() []event.Event {
	raw, err := config.LoadAllyEvents()
	if err != nil {
		log.Printf("增益事件加载失败，使用默认: %v", err)
		return event.DefaultAllyEvents()
	}
	return convertEvents(raw)
}

// LoadEnemyEvents 加载减益事件并转为运行时格式。
func LoadEnemyEvents() []event.Event {
	raw, err := config.LoadEnemyEvents()
	if err != nil {
		log.Printf("减益事件加载失败: %v", err)
		return nil
	}
	return convertEvents(raw)
}

func convertEvents(raw []config.EventJSON) []event.Event {
	events := make([]event.Event, 0, len(raw))
	for _, r := range raw {
		e := event.Event{
			ID:          r.ID,
			Label:       r.Label,
			Description: r.Description,
			Kind:        r.Kind,
			Tier:        r.Tier,
			Value:       r.Value,
			Weight:      r.Weight,
			MinWave:     r.MinWave,
			MaxWave:     r.MaxWave,
		}
		if e.Weight == 0 {
			e.Weight = 1
		}
		if e.Tier == 0 {
			e.Tier = 1
		}
		events = append(events, e)
	}
	return events
}
