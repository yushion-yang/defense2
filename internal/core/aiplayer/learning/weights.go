// weights.go — 权重模型与神经网络评分。
//
// 支持两种评分模式：
//   1. 线性模型（旧版，向后兼容）：score = dot(weights, features)
//   2. 神经网络（新版，优先使用）：score = Network.Forward(features)
//     架构: Input(N) → Hidden(16, ReLU) → Output(1, linear)
//     三组独立网络: build/upgrade/econ
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
	"math"
	"math/rand"
)

// ── 神经网络 ──────────────────────────────────────────

// Layer 全连接层。
// Weights[i][j] 表示第 i 个输出神经元对第 j 个输入的权重。
type Layer struct {
	Weights [][]float64 `json:"weights"` // [outputSize][inputSize]
	Biases  []float64   `json:"biases"`  // [outputSize]
}

// Network 单隐藏层前馈神经网络。
// 架构: Input(N) → Hidden(H, ReLU) → Output(1, linear)
type Network struct {
	Hidden Layer `json:"hidden"` // input → hidden (ReLU 激活)
	Output Layer `json:"output"` // hidden → output (线性激活)
}

// Forward 前向传播，返回标量评分。
//
// 计算流程:
//  1. hidden[i] = ReLU(sum(W1[i][j] * input[j]) + b1[i])
//  2. output = sum(W2[0][i] * hidden[i]) + b2[0]
func (n *Network) Forward(input []float64) float64 {
	hiddenSize := len(n.Hidden.Biases)
	hidden := make([]float64, hiddenSize)
	for i := 0; i < hiddenSize; i++ {
		sum := n.Hidden.Biases[i]
		row := n.Hidden.Weights[i]
		for j := 0; j < len(row) && j < len(input); j++ {
			sum += row[j] * input[j]
		}
		hidden[i] = relu(sum)
	}

	out := n.Output.Biases[0]
	row := n.Output.Weights[0]
	for i := 0; i < len(row) && i < hiddenSize; i++ {
		out += row[i] * hidden[i]
	}
	return out
}

// forwardCached 前向传播，返回隐藏层输出和激活前的值（反向传播用）。
func (n *Network) forwardCached(input []float64) (hidden, preAct []float64) {
	hiddenSize := len(n.Hidden.Biases)
	hidden = make([]float64, hiddenSize)
	preAct = make([]float64, hiddenSize)
	for i := 0; i < hiddenSize; i++ {
		sum := n.Hidden.Biases[i]
		row := n.Hidden.Weights[i]
		for j := 0; j < len(row) && j < len(input); j++ {
			sum += row[j] * input[j]
		}
		preAct[i] = sum
		hidden[i] = relu(sum)
	}
	return hidden, preAct
}

// Backward 反向传播，用梯度信号更新网络权重。
//
// gradient: 输出层的梯度信号（lr * reward，正=强化，负=削弱）。
// input: 本次前向传播时的输入特征。
//
// 更新公式（标准反向传播）:
//   dW2[0][i] += gradient * hidden[i]
//   db2[0]    += gradient
//   dHidden[i] = gradient * W2[0][i] * relu'(preAct[i])
//   dW1[i][j] += dHidden[i] * input[j]
//   db1[i]    += dHidden[i]
func (n *Network) Backward(input []float64, gradient float64) {
	hidden, preAct := n.forwardCached(input)
	hiddenSize := len(n.Hidden.Biases)

	// ── 输出层梯度 ──
	outRow := n.Output.Weights[0]
	n.Output.Biases[0] = clampWeight(n.Output.Biases[0] + gradient)
	for i := 0; i < len(outRow) && i < hiddenSize; i++ {
		outRow[i] = clampWeight(outRow[i] + gradient*hidden[i])
	}

	// ── 隐藏层梯度 ──
	for i := 0; i < hiddenSize; i++ {
		// relu 导数: preAct > 0 → 1, 否则 → 0
		if preAct[i] <= 0 {
			continue
		}
		var w2i float64
		if i < len(outRow) {
			w2i = outRow[i]
		}
		dHidden := gradient * w2i

		n.Hidden.Biases[i] = clampWeight(n.Hidden.Biases[i] + dHidden)
		row := n.Hidden.Weights[i]
		for j := 0; j < len(row) && j < len(input); j++ {
			row[j] = clampWeight(row[j] + dHidden*input[j])
		}
	}
}

// relu 激活函数。
func relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

// NewNetwork 创建一个隐藏层神经网络，Xavier 初始化。
//
// Xavier 初始化标准差: sqrt(2 / (fan_in + fan_out))。
// 使用固定 seed 的 RNG 保证可复现。
func NewNetwork(inputSize, hiddenSize int, rng *rand.Rand) *Network {
	net := &Network{
		Hidden: Layer{
			Weights: make([][]float64, hiddenSize),
			Biases:  make([]float64, hiddenSize),
		},
		Output: Layer{
			Weights: make([][]float64, 1),
			Biases:  make([]float64, 1),
		},
	}

	// Hidden layer: Xavier init
	stdH := math.Sqrt(2.0 / float64(inputSize+hiddenSize))
	for i := 0; i < hiddenSize; i++ {
		net.Hidden.Weights[i] = make([]float64, inputSize)
		for j := 0; j < inputSize; j++ {
			net.Hidden.Weights[i][j] = rng.NormFloat64() * stdH
		}
		// biases 初始化为 0（标准做法）
	}

	// Output layer: Xavier init
	stdO := math.Sqrt(2.0 / float64(hiddenSize+1))
	net.Output.Weights[0] = make([]float64, hiddenSize)
	for i := 0; i < hiddenSize; i++ {
		net.Output.Weights[0][i] = rng.NormFloat64() * stdO
	}

	return net
}

// InputSize 返回网络输入维度。
func (n *Network) InputSize() int {
	if len(n.Hidden.Weights) == 0 {
		return 0
	}
	return len(n.Hidden.Weights[0])
}

// HiddenSize 返回隐藏层大小。
func (n *Network) HiddenSize() int {
	return len(n.Hidden.Biases)
}

// ── 权重模型 ──────────────────────────────────────────

// WeightSet 一组决策场景的权重（线性模型，向后兼容）。
type WeightSet struct {
	Build   []float64 `json:"build"`   // 造塔位置评分权重
	Upgrade []float64 `json:"upgrade"` // 升级评分权重
	Econ    []float64 `json:"econ"`    // 经济决策权重（造塔 vs 升级偏好）
}

// Model 完整的权重模型，支持线性权重和神经网络双模式。
//
// 评分优先级：Networks 存在时用神经网络，否则 fallback 到线性权重。
// JSON 序列化同时保存两者，保证旧版消费方仍可用 linear 权重。
type Model struct {
	Weights  WeightSet              `json:"weights"`           // 线性权重（向后兼容）
	Networks map[string]*Network    `json:"networks,omitempty"` // 神经网络（新版）
	Version  int                    `json:"version"`           // 模型版本号
	Episodes int                    `json:"episodes"`          // 已训练的局数
}

// UseNeural 返回模型是否配置了神经网络。
func (m *Model) UseNeural() bool {
	return len(m.Networks) > 0
}

// GetNetwork 获取指定场景的神经网络（不存在返回 nil）。
func (m *Model) GetNetwork(name string) *Network {
	if m.Networks == nil {
		return nil
	}
	return m.Networks[name]
}

// defaultNeuralSeed 默认神经网络初始化种子（固定值，保证可复现）。
const defaultNeuralSeed int64 = 20260416

// DefaultHiddenSize 默认隐藏层大小。
const DefaultHiddenSize = 16

// DefaultModel 返回硬编码的默认权重模型。
//
// 同时包含线性权重（向后兼容）和神经网络（新版评分）。
// 线性权重从 decision.go 现有的手工调参评分系统提取。
// 神经网络用 Xavier 初始化 + 固定种子，保证可复现。
func DefaultModel() *Model {
	rng := rand.New(rand.NewSource(defaultNeuralSeed))

	return &Model{
		Version: 0,
		Weights: WeightSet{
			// 造塔权重（线性，向后兼容 14 维）
			// [pathDist, coverage, spread, redundancy, synergy, noise, progress, goldRatio, towerCount,
			//  chokepoint, pathBendDist, nearestTowerDist, waveProgress, bossNext]
			Build: []float64{
				0.20, // FeatBuildPathDist
				0.30, // FeatBuildCoverage
				0.15, // FeatBuildSpread
				0.15, // FeatBuildRedundancy
				0.10, // FeatBuildSynergy
				0.10, // FeatBuildNoise
				0.00, // FeatBuildProgress
				0.00, // FeatBuildGoldRatio
				0.00, // FeatBuildTowerCount
				0.15, // FeatBuildChokepoint
				0.05, // FeatBuildPathBendDist
				0.05, // FeatBuildNearestTowerDist
				0.00, // FeatBuildWaveProgress
				0.00, // FeatBuildBossNext
			},
			// 升级权重（线性，向后兼容 11 维）
			// [roi, pathCoverage, headroom, kills, abilityCount, progress, threatLevel,
			//  costEfficiency, abilitySlotsFull, isChokepoint, goldAfterUpgrade]
			Upgrade: []float64{
				0.25, // FeatUpgROI
				0.25, // FeatUpgPathCoverage
				0.20, // FeatUpgHeadroom
				0.15, // FeatUpgKills
				0.15, // FeatUpgAbilityCount
				0.00, // FeatUpgProgress
				0.00, // FeatUpgThreatLevel
				0.10, // FeatUpgCostEfficiency
				0.05, // FeatUpgAbilitySlotsFull
				0.10, // FeatUpgIsChokepoint
				0.00, // FeatUpgGoldAfterUpgrade
			},
			// 经济决策权重（线性，向后兼容 15 维）
			// [progress, towerCount, goldRatio, urgency, threat, aggression, economy,
			//  adviceBuild, adviceUpgrade, bossNext,
			//  wavesSinceLastBuild, enemyHPTrend, livesPercent, towerDPSvsEnemyHP, perfectWaveStreak]
			Econ: []float64{
				-0.20, // FeatEconProgress
				-0.15, // FeatEconTowerCount
				0.10,  // FeatEconGoldRatio
				0.15,  // FeatEconUrgency
				0.00,  // FeatEconThreatLevel
				0.20,  // FeatEconAggression
				-0.20, // FeatEconEconomy
				0.30,  // FeatEconAdviceBuild
				-0.30, // FeatEconAdviceUpgrade
				-0.10, // FeatEconBossNext
				0.10,  // FeatEconWavesSinceLastBuild
				-0.10, // FeatEconEnemyHPTrend
				0.05,  // FeatEconLivesPercent
				0.10,  // FeatEconTowerDPSvsEnemyHP
				0.05,  // FeatEconPerfectWaveStreak
			},
		},
		Networks: map[string]*Network{
			"build":   NewNetwork(FeatBuildLen, DefaultHiddenSize, rng),
			"upgrade": NewNetwork(FeatUpgLen, DefaultHiddenSize, rng),
			"econ":    NewNetwork(FeatEconLen, DefaultHiddenSize, rng),
		},
	}
}

// Score 计算特征向量与权重的内积评分（线性模式）。
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

// scoreWithNetwork 优先用神经网络评分，fallback 到线性权重。
func scoreWithNetwork(features FeatureVec, net *Network, linearWeights []float64) float64 {
	if net != nil {
		input := features.Values[:features.Len]
		return net.Forward(input)
	}
	return Score(features, linearWeights)
}

// ScoreBuild 造塔位置评分。优先使用神经网络。
func (m *Model) ScoreBuild(features FeatureVec) float64 {
	return scoreWithNetwork(features, m.GetNetwork("build"), m.Weights.Build)
}

// ScoreUpgrade 升级评分。优先使用神经网络。
func (m *Model) ScoreUpgrade(features FeatureVec) float64 {
	return scoreWithNetwork(features, m.GetNetwork("upgrade"), m.Weights.Upgrade)
}

// ScoreEcon 经济决策评分（>0 偏造塔，<0 偏升级）。优先使用神经网络。
func (m *Model) ScoreEcon(features FeatureVec) float64 {
	return scoreWithNetwork(features, m.GetNetwork("econ"), m.Weights.Econ)
}

// ── JSON 序列化 ──────────────────────────────────────────

// LoadModel 从 JSON 字节加载权重模型。
// 加载失败时返回 DefaultModel。
// JSON 中没有 networks 字段时，自动用 DefaultModel 的网络补齐。
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

	// 没有 networks 时用默认初始化补齐
	if len(loaded.Networks) == 0 {
		loaded.Networks = m.Networks
	}
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
// 空输入返回 DefaultModel。同时平均线性权重和神经网络权重。
func AverageModels(models []*Model) *Model {
	if len(models) == 0 {
		return DefaultModel()
	}
	if len(models) == 1 {
		return models[0].Clone()
	}

	base := DefaultModel()
	n := float64(len(models))

	// 清零 base 线性权重（用于累加）
	for i := range base.Weights.Build {
		base.Weights.Build[i] = 0
	}
	for i := range base.Weights.Upgrade {
		base.Weights.Upgrade[i] = 0
	}
	for i := range base.Weights.Econ {
		base.Weights.Econ[i] = 0
	}

	// 清零 base 网络权重（用于累加）
	for _, net := range base.Networks {
		zeroNetwork(net)
	}

	// 累加所有模型的线性权重
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

		// 累加网络权重
		for name, baseNet := range base.Networks {
			if m.Networks != nil {
				if mNet, ok := m.Networks[name]; ok {
					addNetwork(baseNet, mNet)
				}
			}
		}

		base.Episodes += m.Episodes
	}

	// 取平均（线性权重）
	for i := range base.Weights.Build {
		base.Weights.Build[i] /= n
	}
	for i := range base.Weights.Upgrade {
		base.Weights.Upgrade[i] /= n
	}
	for i := range base.Weights.Econ {
		base.Weights.Econ[i] /= n
	}

	// 取平均（网络权重）
	for _, net := range base.Networks {
		scaleNetwork(net, 1.0/n)
	}

	base.Version = models[0].Version + 1
	return base
}

// ── 网络算术辅助 ──────────────────────────────────────────

// zeroNetwork 将网络所有权重置零。
func zeroNetwork(n *Network) {
	zeroLayer(&n.Hidden)
	zeroLayer(&n.Output)
}

func zeroLayer(l *Layer) {
	for i := range l.Biases {
		l.Biases[i] = 0
	}
	for i := range l.Weights {
		for j := range l.Weights[i] {
			l.Weights[i][j] = 0
		}
	}
}

// addNetwork 将 src 的权重累加到 dst。
func addNetwork(dst, src *Network) {
	addLayer(&dst.Hidden, &src.Hidden)
	addLayer(&dst.Output, &src.Output)
}

func addLayer(dst, src *Layer) {
	for i := 0; i < len(dst.Biases) && i < len(src.Biases); i++ {
		dst.Biases[i] += src.Biases[i]
	}
	for i := 0; i < len(dst.Weights) && i < len(src.Weights); i++ {
		for j := 0; j < len(dst.Weights[i]) && j < len(src.Weights[i]); j++ {
			dst.Weights[i][j] += src.Weights[i][j]
		}
	}
}

// scaleNetwork 将网络所有权重乘以 s。
func scaleNetwork(n *Network, s float64) {
	scaleLayer(&n.Hidden, s)
	scaleLayer(&n.Output, s)
}

func scaleLayer(l *Layer, s float64) {
	for i := range l.Biases {
		l.Biases[i] *= s
	}
	for i := range l.Weights {
		for j := range l.Weights[i] {
			l.Weights[i][j] *= s
		}
	}
}

// Clone 深拷贝权重模型（含线性权重和神经网络）。
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

	if m.Networks != nil {
		c.Networks = make(map[string]*Network, len(m.Networks))
		for name, net := range m.Networks {
			c.Networks[name] = cloneNetwork(net)
		}
	}
	return c
}

// cloneNetwork 深拷贝单个神经网络。
func cloneNetwork(n *Network) *Network {
	if n == nil {
		return nil
	}
	c := &Network{
		Hidden: cloneLayer(n.Hidden),
		Output: cloneLayer(n.Output),
	}
	return c
}

// cloneLayer 深拷贝单层。
func cloneLayer(l Layer) Layer {
	c := Layer{
		Weights: make([][]float64, len(l.Weights)),
		Biases:  make([]float64, len(l.Biases)),
	}
	copy(c.Biases, l.Biases)
	for i, row := range l.Weights {
		c.Weights[i] = make([]float64, len(row))
		copy(c.Weights[i], row)
	}
	return c
}
