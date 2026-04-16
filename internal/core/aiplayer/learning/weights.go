// weights.go — 权重模型与线性评分。
//
// 核心设计：每个决策场景对应一组权重向量，通过内积(dot product)计算评分。
// score = sum(weight[i] * feature[i])
//
// 权重来源（优先级递减）：
//   1. 运行时实时更新（Trainer 在线学习）
//   2. 嵌入式 JSON 默认值（config/ai/weights.json）
//   3. 代码硬编码 fallback（保证始终可用）
//
// 线程安全：权重只在决策循环中读取，Trainer 在同一 goroutine 中更新，无需互斥。
package learning

import (
	"encoding/json"
)

// ── 权重模型 ──────────────────────────────────────────

// WeightSet 一组决策场景的权重。
type WeightSet struct {
	Build   []float64 `json:"build"`   // 造塔位置评分权重
	Upgrade []float64 `json:"upgrade"` // 升级评分权重
	Econ    []float64 `json:"econ"`    // 经济决策权重（造塔 vs 升级偏好）
}

// Model 完整的权重模型，包含所有决策场景。
type Model struct {
	Weights  WeightSet `json:"weights"`
	Version  int       `json:"version"`  // 模型版本号（每次训练后递增）
	Episodes int       `json:"episodes"` // 已训练的局数
}

// DefaultModel 返回硬编码的默认权重模型。
// 权重值从 decision.go 现有的手工调参评分系统提取。
// 作为 JSON 加载失败时的 fallback，保证系统始终可用。
func DefaultModel() *Model {
	return &Model{
		Version: 0,
		Weights: WeightSet{
			// 造塔权重：与 scoreBuildCell 的手工权重一致
			// [pathDist, coverage, spread, redundancy, synergy, noise, progress, goldRatio, towerCount]
			Build: []float64{
				0.20, // FeatBuildPathDist
				0.30, // FeatBuildCoverage
				0.15, // FeatBuildSpread
				0.15, // FeatBuildRedundancy
				0.10, // FeatBuildSynergy
				0.10, // FeatBuildNoise
				0.00, // FeatBuildProgress（默认不影响，训练后可学到）
				0.00, // FeatBuildGoldRatio
				0.00, // FeatBuildTowerCount
			},
			// 升级权重：与 scoreUpgradeTower 的手工权重一致
			// [roi, pathCoverage, headroom, kills, abilityCount, progress, threatLevel]
			Upgrade: []float64{
				0.25, // FeatUpgROI
				0.25, // FeatUpgPathCoverage
				0.20, // FeatUpgHeadroom
				0.15, // FeatUpgKills
				0.15, // FeatUpgAbilityCount
				0.00, // FeatUpgProgress
				0.00, // FeatUpgThreatLevel
			},
			// 经济决策权重：输出 > 0 偏造塔，< 0 偏升级
			// [progress, towerCount, goldRatio, urgency, threat, aggression, economy, adviceBuild, adviceUpgrade, bossNext]
			Econ: []float64{
				-0.20, // FeatEconProgress（后期偏升级）
				-0.15, // FeatEconTowerCount（塔多偏升级）
				0.10,  // FeatEconGoldRatio（钱多偏造塔）
				0.15,  // FeatEconUrgency（紧急偏造塔）
				0.00,  // FeatEconThreatLevel
				0.20,  // FeatEconAggression（激进偏造塔）
				-0.20, // FeatEconEconomy（经济偏升级）
				0.30,  // FeatEconAdviceBuild
				-0.30, // FeatEconAdviceUpgrade
				-0.10, // FeatEconBossNext（Boss 来偏升级）
			},
		},
	}
}

// Score 计算特征向量与权重的内积评分。
// 如果权重向量短于特征向量，缺少的维度权重按 0 处理。
// 如果权重向量长于特征向量，多余的权重被忽略。
func Score(features FeatureVec, weights []float64) float64 {
	sum := 0.0
	n := features.Len
	if n > len(weights) {
		n = len(weights)
	}
	for i := 0; i < n; i++ {
		sum += features.Values[i] * weights[i]
	}
	return sum
}

// ScoreBuild 造塔位置评分。
func (m *Model) ScoreBuild(features FeatureVec) float64 {
	return Score(features, m.Weights.Build)
}

// ScoreUpgrade 升级评分。
func (m *Model) ScoreUpgrade(features FeatureVec) float64 {
	return Score(features, m.Weights.Upgrade)
}

// ScoreEcon 经济决策评分（>0 偏造塔，<0 偏升级）。
func (m *Model) ScoreEcon(features FeatureVec) float64 {
	return Score(features, m.Weights.Econ)
}

// ── JSON 序列化 ──────────────────────────────────────────

// LoadModel 从 JSON 字节加载权重模型。
// 加载失败时返回 DefaultModel。
func LoadModel(data []byte) *Model {
	m := DefaultModel()
	if len(data) == 0 {
		return m
	}
	var loaded Model
	if err := json.Unmarshal(data, &loaded); err != nil {
		return m
	}
	// 验证权重向量长度，不足时用默认值补齐
	loaded.Weights.Build = padWeights(loaded.Weights.Build, m.Weights.Build)
	loaded.Weights.Upgrade = padWeights(loaded.Weights.Upgrade, m.Weights.Upgrade)
	loaded.Weights.Econ = padWeights(loaded.Weights.Econ, m.Weights.Econ)
	return &loaded
}

// MarshalModel 将权重模型序列化为 JSON 字节。
func MarshalModel(m *Model) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// padWeights 补齐权重向量：loaded 短于 defaults 时用 defaults 补齐。
func padWeights(loaded, defaults []float64) []float64 {
	if len(loaded) >= len(defaults) {
		return loaded
	}
	result := make([]float64, len(defaults))
	copy(result, loaded)
	for i := len(loaded); i < len(defaults); i++ {
		result[i] = defaults[i]
	}
	return result
}

// AverageModels 将多个模型的权重取平均，返回新模型。
// 用于训练管线：收集多局 winning model，平均后导出。
// 空输入返回 DefaultModel。
func AverageModels(models []*Model) *Model {
	if len(models) == 0 {
		return DefaultModel()
	}
	if len(models) == 1 {
		return models[0].Clone()
	}

	base := DefaultModel()
	n := float64(len(models))

	// 清零 base 权重（用于累加）
	for i := range base.Weights.Build {
		base.Weights.Build[i] = 0
	}
	for i := range base.Weights.Upgrade {
		base.Weights.Upgrade[i] = 0
	}
	for i := range base.Weights.Econ {
		base.Weights.Econ[i] = 0
	}

	// 累加所有模型的权重
	for _, m := range models {
		for i := 0; i < len(base.Weights.Build) && i < len(m.Weights.Build); i++ {
			base.Weights.Build[i] += m.Weights.Build[i]
		}
		for i := 0; i < len(base.Weights.Upgrade) && i < len(m.Weights.Upgrade); i++ {
			base.Weights.Upgrade[i] += m.Weights.Upgrade[i]
		}
		for i := 0; i < len(base.Weights.Econ) && i < len(m.Weights.Econ); i++ {
			base.Weights.Econ[i] += m.Weights.Econ[i]
		}
		base.Episodes += m.Episodes
	}

	// 取平均
	for i := range base.Weights.Build {
		base.Weights.Build[i] /= n
	}
	for i := range base.Weights.Upgrade {
		base.Weights.Upgrade[i] /= n
	}
	for i := range base.Weights.Econ {
		base.Weights.Econ[i] /= n
	}

	base.Version = models[0].Version + 1
	return base
}

// Clone 深拷贝权重模型。
func (m *Model) Clone() *Model {
	c := &Model{
		Version:  m.Version,
		Episodes: m.Episodes,
	}
	c.Weights.Build = make([]float64, len(m.Weights.Build))
	copy(c.Weights.Build, m.Weights.Build)
	c.Weights.Upgrade = make([]float64, len(m.Weights.Upgrade))
	copy(c.Weights.Upgrade, m.Weights.Upgrade)
	c.Weights.Econ = make([]float64, len(m.Weights.Econ))
	copy(c.Weights.Econ, m.Weights.Econ)
	return c
}
