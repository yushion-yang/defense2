// tower_buff.go — 炮塔 buff 管理。
// 轻量 buff 追踪层，记录每个 buff 的来源/描述/数值/时长，供 HUD 展示。
// buff 同时同步写入 Strength.SetTemp，移除时调用 RemoveTemp。
package tower

import "strings"

// TowerBuff 炮塔上的一个增益效果。
type TowerBuff struct {
	Key       string  // 唯一标识（同 Strength.SetTemp 的 key）
	Source    string  // 来源显示名（如 "聚能战灵"）
	Desc      string  // 效果描述（如 "+30 强度 (3塔串联)"）
	Value     float64 // 加成数值
	Duration  float64 // 总持续时间（-1 = 永久，直到来源移除）
	Remaining float64 // 剩余时间（-1 = 永久）
}

// ApplyBuff 施加或更新一个 buff。同 Key 覆盖，新 Key 追加。
// 同时写入 Strength.SetTemp。
func (t *Tower) ApplyBuff(b TowerBuff) {
	// 同步到 Strength
	if t.Strength != nil {
		t.Strength.SetTemp(b.Key, b.Value)
	}

	// 同 Key 覆盖
	for i := range t.Buffs {
		if t.Buffs[i].Key == b.Key {
			t.Buffs[i].Source = b.Source
			t.Buffs[i].Desc = b.Desc
			t.Buffs[i].Value = b.Value
			t.Buffs[i].Duration = b.Duration
			t.Buffs[i].Remaining = b.Remaining
			return
		}
	}
	// 新 buff
	t.Buffs = append(t.Buffs, b)
}

// RemoveBuff 移除指定 Key 的 buff，同时清除 Strength.Temp。
func (t *Tower) RemoveBuff(key string) {
	if t.Strength != nil {
		t.Strength.RemoveTemp(key)
	}
	for i := range t.Buffs {
		if t.Buffs[i].Key == key {
			t.Buffs = append(t.Buffs[:i], t.Buffs[i+1:]...)
			return
		}
	}
}

// RemoveBuffsByPrefix 移除所有 Key 以 prefix 开头的 buff。
// 用于来源消失时批量清除（如战灵卸载、塔卖出）。
func (t *Tower) RemoveBuffsByPrefix(prefix string) {
	n := 0
	for i := range t.Buffs {
		if strings.HasPrefix(t.Buffs[i].Key, prefix) {
			if t.Strength != nil {
				t.Strength.RemoveTemp(t.Buffs[i].Key)
			}
		} else {
			t.Buffs[n] = t.Buffs[i]
			n++
		}
	}
	t.Buffs = t.Buffs[:n]
}

// TickBuffs 递减有时限 buff 的剩余时间，到期自动移除。
// 永久 buff（Remaining == -1）不受影响。
func (t *Tower) TickBuffs(dt float64) {
	n := 0
	for i := range t.Buffs {
		b := &t.Buffs[i]
		if b.Remaining >= 0 {
			b.Remaining -= dt
			if b.Remaining <= 0 {
				// 到期移除
				if t.Strength != nil {
					t.Strength.RemoveTemp(b.Key)
				}
				continue
			}
		}
		t.Buffs[n] = t.Buffs[i]
		n++
	}
	t.Buffs = t.Buffs[:n]
}
