// trainer.go — 在线学习训练器。
//
// 职责：追踪 AI 每次决策及其后续结果，用 reward signal 更新权重。
// 设计：轻量级 online learning，不依赖外部 ML 框架。
//
// 支持两种更新模式：
//   1. 线性模式：weight[i] += lr * reward * feature[i]（旧版）
//   2. 神经网络模式：Network.Backward(input, lr * reward)（新版）
//
// 当模型包含 Networks 时自动切换到神经网络模式，
// 同时也更新线性权重（向后兼容双写）。
//
// Reward 信号来源:
//   - 击杀效率: kills_gained / time_elapsed
//   - 生命损失: -penalty * lives_lost
//   - 通关成功: +bonus
//   - 波次进度: wave_reached / max_waves
//
// 在线学习循环:
//  1. 每次决策时调用 RecordDecision(features, decisionType)
//  2. 每波结束时调用 OnWaveEnd(waveStats) → 用本波 reward 更新权重
//  3. 游戏结束时调用 OnGameEnd(result) → 全局 reward 微调
package learning

import "math"

// ── 训练器 ──────────────────────────────────────────

// Trainer 在线学习训练器。
type Trainer struct {
	model     *Model
	lr        float64 // 学习率
	enabled   bool    // 是否启用在线学习
	decayRate float64 // 学习率衰减（每局衰减倍率）

	// 当前局的决策记录
	buildRecords   []decisionRecord
	upgradeRecords []decisionRecord
	econRecords    []decisionRecord

	// 当前局统计
	waveKills  int // 本波击杀
	waveLives  int // 本波开始时生命
	totalWaves int // 已完成的波数
}

// decisionRecord 单次决策的记录（特征 + 结果）。
type decisionRecord struct {
	features FeatureVec
	score    float64 // 决策时的评分
}

// TrainerConfig 训练器配置。
type TrainerConfig struct {
	Model     *Model
	LR        float64 // 学习率（默认: 线性 0.01, 神经网络 0.001）
	Enabled   bool    // 是否启用在线学习
	DecayRate float64 // 学习率衰减（默认 0.995）
}

// NewTrainer 创建训练器。
// 学习率默认值：神经网络模式 0.001，线性模式 0.01。
func NewTrainer(cfg TrainerConfig) *Trainer {
	lr := cfg.LR
	if lr <= 0 {
		if cfg.Model != nil && cfg.Model.UseNeural() {
			lr = 0.001 // 神经网络需要更小的学习率
		} else {
			lr = 0.01
		}
	}
	decay := cfg.DecayRate
	if decay <= 0 || decay > 1 {
		decay = 0.995
	}
	m := cfg.Model
	if m == nil {
		m = DefaultModel()
	}
	return &Trainer{
		model:     m,
		lr:        lr,
		enabled:   cfg.Enabled,
		decayRate: decay,
	}
}

// Model 返回当前模型（只读引用）。
func (t *Trainer) Model() *Model { return t.model }

// Enabled 返回在线学习是否启用。
func (t *Trainer) Enabled() bool { return t.enabled }

// SetEnabled 启用/禁用在线学习。
func (t *Trainer) SetEnabled(v bool) { t.enabled = v }

// ── 决策记录 ──────────────────────────────────────────

// RecordBuild 记录一次造塔决策。
func (t *Trainer) RecordBuild(features FeatureVec, score float64) {
	if !t.enabled {
		return
	}
	t.buildRecords = append(t.buildRecords, decisionRecord{features: features, score: score})
}

// RecordUpgrade 记录一次升级决策。
func (t *Trainer) RecordUpgrade(features FeatureVec, score float64) {
	if !t.enabled {
		return
	}
	t.upgradeRecords = append(t.upgradeRecords, decisionRecord{features: features, score: score})
}

// RecordEcon 记录一次经济决策。
func (t *Trainer) RecordEcon(features FeatureVec, score float64) {
	if !t.enabled {
		return
	}
	t.econRecords = append(t.econRecords, decisionRecord{features: features, score: score})
}

// ── 波次级学习 ──────────────────────────────────────────

// WaveStats 单波结束时的统计数据。
type WaveStats struct {
	WaveNum       int
	KillsThisWave int
	LivesBefore   int
	LivesAfter    int
	GoldSpent     int
	GoldEarned    int
}

// OnWaveEnd 波次结束时更新权重。
//
// Reward 计算：
//   - 基础: +0.5（完成一波）
//   - 击杀效率: +0.3 * min(1, kills/10)
//   - 生命损失: -0.5 * livesLost/totalLives
//   - 完美通关: +0.2（零损失）
func (t *Trainer) OnWaveEnd(stats WaveStats) {
	if !t.enabled {
		return
	}

	// 计算 reward
	reward := 0.5 // 完成波次基础 reward

	// 击杀效率
	killRatio := clamp01(float64(stats.KillsThisWave) / 10.0)
	reward += 0.3 * killRatio

	// 生命损失惩罚
	livesLost := stats.LivesBefore - stats.LivesAfter
	if livesLost < 0 {
		livesLost = 0
	}
	if stats.LivesBefore > 0 {
		reward -= 0.5 * float64(livesLost) / float64(stats.LivesBefore)
	}

	// 完美通关奖励
	if livesLost == 0 {
		reward += 0.2
	}

	// 用 reward 更新本波所有决策的权重
	t.applyReward(reward)

	// 清空本波记录
	t.buildRecords = t.buildRecords[:0]
	t.upgradeRecords = t.upgradeRecords[:0]
	t.econRecords = t.econRecords[:0]

	t.totalWaves++
}

// ── 游戏级学习 ──────────────────────────────────────────

// GameResult 游戏结束时的结果。
type GameResult struct {
	Won          bool
	WavesReached int
	MaxWaves     int
	LivesLeft    int
	MaxLives     int
	TotalKills   int
}

// OnGameEnd 游戏结束时的全局权重微调。
//
// Reward 计算：
//   - 通关: +1.0
//   - 未通关: -0.5 + 0.5 * (wavesReached/maxWaves)
//   - 生命残留加成: +0.3 * (livesLeft/maxLives)
func (t *Trainer) OnGameEnd(result GameResult) {
	if !t.enabled {
		return
	}

	var globalReward float64
	if result.Won {
		globalReward = 1.0
		if result.MaxLives > 0 {
			globalReward += 0.3 * float64(result.LivesLeft) / float64(result.MaxLives)
		}
	} else {
		globalReward = -0.5
		if result.MaxWaves > 0 {
			globalReward += 0.5 * float64(result.WavesReached) / float64(result.MaxWaves)
		}
	}

	// 全局微调：对所有权重做小幅调整
	// 方向：赢了就加强当前权重模式，输了就略微随机扰动
	t.applyGlobalAdjustment(globalReward)

	// 更新模型版本
	t.model.Episodes++
	t.model.Version++

	// 学习率衰减
	t.lr *= t.decayRate
	if t.lr < 0.0001 {
		t.lr = 0.0001 // 下限（神经网络需要更小的下限）
	}
}

// Reset 重置训练器状态（新一局开始）。
func (t *Trainer) Reset() {
	t.buildRecords = t.buildRecords[:0]
	t.upgradeRecords = t.upgradeRecords[:0]
	t.econRecords = t.econRecords[:0]
	t.waveKills = 0
	t.waveLives = 0
	t.totalWaves = 0
}

// ── 内部权重更新 ──────────────────────────────────────────

// applyReward 将 reward 反向传播到本波的所有决策记录。
//
// 双模式更新：
//   - 有神经网络 → Network.Backward(input, lr * reward)
//   - 始终更新线性权重（向后兼容双写）
func (t *Trainer) applyReward(reward float64) {
	gradient := t.lr * reward

	// 造塔权重更新
	buildNet := t.model.GetNetwork("build")
	for _, rec := range t.buildRecords {
		if buildNet != nil {
			buildNet.Backward(rec.features.Values[:rec.features.Len], gradient)
		}
		updateWeights(t.model.Weights.Build, rec.features, t.lr, reward)
	}

	// 升级权重更新
	upgradeNet := t.model.GetNetwork("upgrade")
	for _, rec := range t.upgradeRecords {
		if upgradeNet != nil {
			upgradeNet.Backward(rec.features.Values[:rec.features.Len], gradient)
		}
		updateWeights(t.model.Weights.Upgrade, rec.features, t.lr, reward)
	}

	// 经济决策权重更新
	econNet := t.model.GetNetwork("econ")
	for _, rec := range t.econRecords {
		if econNet != nil {
			econNet.Backward(rec.features.Values[:rec.features.Len], gradient)
		}
		updateWeights(t.model.Weights.Econ, rec.features, t.lr, reward)
	}
}

// applyGlobalAdjustment 全局微调权重。
// 赢了 → 所有权重乘以 (1 + lr*reward*0.1)，略微加强当前模式。
// 输了 → 权重向默认值回归（避免发散）。
// 神经网络模式下缩放因子更保守（0.01 vs 0.1）。
func (t *Trainer) applyGlobalAdjustment(reward float64) {
	if reward >= 0 {
		// 赢了：略微加强当前权重模式
		scaleFactor := 0.1
		if t.model.UseNeural() {
			scaleFactor = 0.01 // 神经网络更保守
		}
		scale := 1.0 + t.lr*reward*scaleFactor

		// 线性权重缩放
		scaleWeightsVec(t.model.Weights.Build, scale)
		scaleWeightsVec(t.model.Weights.Upgrade, scale)
		scaleWeightsVec(t.model.Weights.Econ, scale)

		// 神经网络缩放
		for _, net := range t.model.Networks {
			scaleNetwork(net, scale)
		}
	} else {
		// 输了：权重向默认值回归（防止过拟合失败模式）
		defaults := DefaultModel()
		regressRate := t.lr * math.Abs(reward) * 0.2

		// 线性权重回归
		regressWeights(t.model.Weights.Build, defaults.Weights.Build, regressRate)
		regressWeights(t.model.Weights.Upgrade, defaults.Weights.Upgrade, regressRate)
		regressWeights(t.model.Weights.Econ, defaults.Weights.Econ, regressRate)

		// 神经网络回归
		for name, net := range t.model.Networks {
			if defNet, ok := defaults.Networks[name]; ok {
				regressNetwork(net, defNet, regressRate)
			}
		}
	}
}

// ── 线性权重更新辅助 ──────────────────────────────────────────

// updateWeights 根据特征和 reward 更新权重向量。
// weight[i] += lr * reward * feature[i]
func updateWeights(weights []float64, features FeatureVec, lr, reward float64) {
	n := features.Len
	if n > len(weights) {
		n = len(weights)
	}
	for i := 0; i < n; i++ {
		delta := lr * reward * features.Values[i]
		weights[i] += delta
		// 钳制权重范围，防止发散（[-2, 2]）
		weights[i] = clampWeight(weights[i])
	}
}

// scaleWeightsVec 缩放权重向量。
func scaleWeightsVec(weights []float64, scale float64) {
	for i := range weights {
		weights[i] *= scale
		weights[i] = clampWeight(weights[i])
	}
}

// regressWeights 将权重向目标值回归。
// weight[i] += rate * (target[i] - weight[i])
func regressWeights(weights, targets []float64, rate float64) {
	n := len(weights)
	if n > len(targets) {
		n = len(targets)
	}
	for i := 0; i < n; i++ {
		weights[i] += rate * (targets[i] - weights[i])
		weights[i] = clampWeight(weights[i])
	}
}

// ── 神经网络回归辅助 ──────────────────────────────────────────

// regressNetwork 将网络权重向目标网络回归。
func regressNetwork(net, target *Network, rate float64) {
	regressLayer(&net.Hidden, &target.Hidden, rate)
	regressLayer(&net.Output, &target.Output, rate)
}

func regressLayer(l, target *Layer, rate float64) {
	for i := 0; i < len(l.Biases) && i < len(target.Biases); i++ {
		l.Biases[i] += rate * (target.Biases[i] - l.Biases[i])
		l.Biases[i] = clampWeight(l.Biases[i])
	}
	for i := 0; i < len(l.Weights) && i < len(target.Weights); i++ {
		for j := 0; j < len(l.Weights[i]) && j < len(target.Weights[i]); j++ {
			l.Weights[i][j] += rate * (target.Weights[i][j] - l.Weights[i][j])
			l.Weights[i][j] = clampWeight(l.Weights[i][j])
		}
	}
}

// clampWeight 钳制单个权重到 [-2, 2] 范围。
func clampWeight(w float64) float64 {
	if w < -2.0 {
		return -2.0
	}
	if w > 2.0 {
		return 2.0
	}
	return w
}
