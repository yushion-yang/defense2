// shield.go — 多重护盾系统。
// 支持多层护盾叠加，伤害按剩余时间升序消耗（先消耗即将过期的护盾）。
package enemy

import "sort"

// Shield 单层护盾。
type Shield struct {
	Value    float64 // 当前护盾值
	MaxValue float64 // 初始护盾值
	Duration float64 // 剩余持续时间（秒，<=0 表示永久）
	Source   string  // 来源标识
}

// Threshold HP阈值触发器。
// 当敌人 HP 比例降到 Ratio 以下时触发一次。
type Threshold struct {
	Type      string  // 触发器类型标识
	Ratio     float64 // 触发比例（如 0.5 = 50% HP）
	Triggered bool    // 是否已触发
}

// InitShields 初始化护盾列表（如果尚未初始化）。
func InitShields(e *Enemy) {
	if e.Shields == nil {
		e.Shields = make([]Shield, 0, 4)
	}
}

// AddShield 添加一层护盾。
func AddShield(e *Enemy, value, duration float64, source string) {
	InitShields(e)
	e.Shields = append(e.Shields, Shield{
		Value:    value,
		MaxValue: value,
		Duration: duration,
		Source:   source,
	})
}

// TickShields 每帧更新护盾持续时间，移除过期护盾。
func TickShields(e *Enemy, dt float64) {
	if len(e.Shields) == 0 {
		return
	}
	alive := e.Shields[:0]
	for i := range e.Shields {
		s := &e.Shields[i]
		if s.Duration > 0 {
			s.Duration -= dt
			if s.Duration <= 0 {
				continue // 过期，不保留
			}
		}
		if s.Value > 0 {
			alive = append(alive, *s)
		}
	}
	e.Shields = alive
}

// AbsorbShields 护盾吸收伤害。
// 按剩余时间升序消耗（先消耗即将过期的），返回被护盾吸收的总伤害。
// 同时扣减旧版单层 ShieldHP。
func AbsorbShields(e *Enemy, damage float64) float64 {
	absorbed := 0.0

	// 先处理多重护盾
	if len(e.Shields) > 0 {
		// 按剩余时间升序排列（永久护盾排最后）
		sort.Slice(e.Shields, func(i, j int) bool {
			di := e.Shields[i].Duration
			dj := e.Shields[j].Duration
			if di <= 0 {
				return false // 永久排后面
			}
			if dj <= 0 {
				return true
			}
			return di < dj
		})

		for i := range e.Shields {
			if damage <= 0 {
				break
			}
			s := &e.Shields[i]
			if s.Value <= 0 {
				continue
			}
			if damage <= s.Value {
				s.Value -= damage
				absorbed += damage
				damage = 0
			} else {
				absorbed += s.Value
				damage -= s.Value
				s.Value = 0
			}
		}
	}

	// 再处理旧版单层 ShieldHP
	if damage > 0 && e.ShieldHP > 0 {
		if damage <= e.ShieldHP {
			e.ShieldHP -= damage
			absorbed += damage
			damage = 0
		} else {
			absorbed += e.ShieldHP
			damage -= e.ShieldHP
			e.ShieldHP = 0
		}
	}

	return absorbed
}

// TotalShieldHP 返回所有护盾的总值（含旧版 ShieldHP）。
func TotalShieldHP(e *Enemy) float64 {
	total := e.ShieldHP
	for _, s := range e.Shields {
		total += s.Value
	}
	return total
}

// AddThreshold 注册一个 HP 阈值触发器。
func AddThreshold(e *Enemy, typ string, ratio float64) {
	if e.Thresholds == nil {
		e.Thresholds = make([]Threshold, 0, 4)
	}
	e.Thresholds = append(e.Thresholds, Threshold{
		Type:  typ,
		Ratio: ratio,
	})
}

// CheckThresholds 检查并返回本次伤害触发的阈值列表。
func CheckThresholds(e *Enemy) []Threshold {
	if len(e.Thresholds) == 0 || e.MaxHP <= 0 {
		return nil
	}
	ratio := e.HP / e.MaxHP
	var triggered []Threshold
	for i := range e.Thresholds {
		th := &e.Thresholds[i]
		if !th.Triggered && ratio <= th.Ratio {
			th.Triggered = true
			triggered = append(triggered, *th)
		}
	}
	return triggered
}
