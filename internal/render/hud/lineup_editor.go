// lineup_editor.go — 塔阵容编辑器。
// 支持阵容序列化/反序列化、波次统计、变速控制。
package hud

import (
	"encoding/json"
	"fmt"
)

// LineupEditor 塔阵容编辑器。
type LineupEditor struct {
	Active     bool       // 是否激活
	WaveStats  []WaveStat // 每波统计
	SpeedLevel int        // 当前速度档位（0=1x, 1=2x, 2=3x, 3=10x）
}

// WaveStat 单波统计。
type WaveStat struct {
	Wave   int     // 波次编号
	Kills  int     // 击杀数
	Leaks  int     // 泄漏数
	Damage float64 // 总伤害
}

// LineupEntry 阵容条目（用于序列化）。
type LineupEntry struct {
	SlotIndex int    `json:"slotIndex"` // 槽位索引
	TowerKey  string `json:"towerKey"`  // 塔键名
	Level     int    `json:"level"`     // 塔等级
}

// speedMultipliers 速度档位对应的倍率。
var speedMultipliers = [4]int{1, 2, 3, 10}

// NewLineupEditor 创建阵容编辑器。
func NewLineupEditor() *LineupEditor {
	return &LineupEditor{
		WaveStats:  make([]WaveStat, 0, 64),
		SpeedLevel: 0,
	}
}

// RecordWaveEnd 记录波次结束统计。
func (e *LineupEditor) RecordWaveEnd(wave, kills, leaks int, damage float64) {
	e.WaveStats = append(e.WaveStats, WaveStat{
		Wave:   wave,
		Kills:  kills,
		Leaks:  leaks,
		Damage: damage,
	})
}

// SerializeLineup 将阵容序列化为 JSON 字符串。
func (e *LineupEditor) SerializeLineup(entries []LineupEntry) (string, error) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化阵容失败: %w", err)
	}
	return string(data), nil
}

// DeserializeLineup 从 JSON 字符串反序列化阵容。
func (e *LineupEditor) DeserializeLineup(data string) ([]LineupEntry, error) {
	var entries []LineupEntry
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return nil, fmt.Errorf("反序列化阵容失败: %w", err)
	}
	return entries, nil
}

// CycleSpeed 切换到下一速度档位，返回新的速度倍率。
// 循环顺序：1x → 2x → 3x → 10x → 1x。
func (e *LineupEditor) CycleSpeed() int {
	e.SpeedLevel = (e.SpeedLevel + 1) % len(speedMultipliers)
	return speedMultipliers[e.SpeedLevel]
}

// GetSpeedMultiplier 返回当前速度倍率。
func (e *LineupEditor) GetSpeedMultiplier() int {
	if e.SpeedLevel < 0 || e.SpeedLevel >= len(speedMultipliers) {
		return 1
	}
	return speedMultipliers[e.SpeedLevel]
}

// RecentWaves 返回最近 N 波的统计。
func (e *LineupEditor) RecentWaves(n int) []WaveStat {
	total := len(e.WaveStats)
	if n <= 0 || total == 0 {
		return nil
	}
	if n > total {
		n = total
	}
	// 返回副本，避免外部修改
	result := make([]WaveStat, n)
	copy(result, e.WaveStats[total-n:])
	return result
}
