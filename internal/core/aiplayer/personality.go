// personality.go — AI 个性系统。
//
// 每个 AI 玩家拥有独立个性参数，影响决策偏好、反应速度、经济策略。
// 参数均在 [0,1] 范围内，由正态分布随机生成。
package aiplayer

import (
	"math"
	"math/rand"
)

// Personality AI 个性参数。
// 所有字段范围 [0, 1]，0.5 为中性。
type Personality struct {
	Aggression float64 // 0=防守(多CC/AOE), 1=进攻(多DPS)
	Economy    float64 // 0=激进花钱, 1=攒钱
	Risk       float64 // 0=保守(留金币缓冲), 1=冒险(花光)
	Reaction   float64 // 0=慢(3-4s 决策间隔), 1=快(1.5-2.5s)
	Compliance float64 // 0=无视 ping, 1=总听建议
}

// RandomPersonality 生成随机个性。
// 使用正态分布 mean=0.5, stddev=0.15，钳制到 [0.05, 0.95]。
func RandomPersonality() Personality {
	return Personality{
		Aggression: clampedNormal(0.5, 0.15),
		Economy:    clampedNormal(0.5, 0.15),
		Risk:       clampedNormal(0.5, 0.15),
		Reaction:   clampedNormal(0.5, 0.15),
		Compliance: clampedNormal(0.5, 0.15),
	}
}

// clampedNormal 正态分布采样，钳制到 [0.05, 0.95]。
func clampedNormal(mean, stddev float64) float64 {
	v := rand.NormFloat64()*stddev + mean
	return math.Max(0.05, math.Min(0.95, v))
}
