// economy_spec.go — 经济系统规格加载器。
// 从 config/systems/economy.json 读取波次奖金、完美奖金、评分公式等经济参数。
package config

import (
	"encoding/json"
	"log"
	"sync"
)

// EconomySpec 经济系统规格。
type EconomySpec struct {
	Global struct {
		KillReward      int     `json:"killReward"`
		SellRefundRatio float64 `json:"sellRefundRatio"`
	} `json:"global"`
	Modes map[string]ModeEconomy `json:"modes"`
}

// ModeEconomy 单个游戏模式的经济参数。
type ModeEconomy struct {
	WaveBonus        BonusFormula `json:"waveBonus"`
	PerfectBonus     BonusFormula `json:"perfectBonus"`
	TotalBosses      int          `json:"totalBosses,omitempty"`
	IntermissionSecs float64      `json:"intermissionSecs,omitempty"`
	TargetSeconds    float64      `json:"targetSeconds,omitempty"`
}

// BonusFormula 线性奖金公式：base + perWave * wave。
type BonusFormula struct {
	Base    int `json:"base"`
	PerWave int `json:"perWave"`
}

// Calc 计算指定波次的奖金。
func (f BonusFormula) Calc(wave int) int {
	return f.Base + f.PerWave*wave
}

var (
	economySpec     *EconomySpec
	economySpecOnce sync.Once
)

// GlobalEconomySpec 返回全局经济规格（懒加载单例）。
func GlobalEconomySpec() *EconomySpec {
	economySpecOnce.Do(func() {
		economySpec = loadEconomySpec()
	})
	return economySpec
}

func loadEconomySpec() *EconomySpec {
	spec := &EconomySpec{}
	if dataFS == nil {
		return spec
	}
	data, err := dataFS.ReadFile("config/systems/economy.json")
	if err != nil {
		log.Printf("[economy_spec] load error: %v", err)
		return spec
	}
	if err := json.Unmarshal(data, spec); err != nil {
		log.Printf("[economy_spec] parse error: %v", err)
	}
	return spec
}
