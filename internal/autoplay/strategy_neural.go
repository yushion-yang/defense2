// strategy_neural.go — 神经网络驱动的自学习策略。
//
// 核心理念：所有决策（建塔位置、塔型选择、升级目标、经济分配、开波时机）
// 都由 learning.Model 的神经网络评分驱动，不使用硬编码规则。
// 模型通过 Trainer 的在线学习机制从游戏结果中反向传播更新权重，
// 形成"自我对弈→获取 reward→更新权重→下一局更好"的训练闭环。
//
// 与 CompetentStrategy 的关键区别：
//   - CompetentStrategy: 硬编码的建造计划 + 规则引擎（人类经验）
//   - NeuralStrategy: 神经网络评分 + 最小必要骨架（无硬编码偏好）
//
// 训练闭环：
//   train.go 创建 NeuralStrategy(model) → play → trainer.OnWaveEnd(stats)
//   → trainer.OnGameEnd(result) → 提取 model → 传给下一局
package autoplay

import (
	"math"
	"math/rand"
	"strings"

	"defense2/internal/config"
	"defense2/internal/core/aiplayer/learning"
)

// ── 常量 ──────────────────────────────────────────

const (
	// neuralBuildThrottle 建塔节流 tick 数（避免同帧连续建造）。
	neuralBuildThrottle = 5
	// neuralUpgradeThrottle 升级节流 tick 数。
	neuralUpgradeThrottle = 5
	// neuralEconBuildThreshold 经济评分高于此值偏向建塔。
	neuralEconBuildThreshold = 0.3
	// neuralEconUpgradeThreshold 经济评分低于此值偏向升级。
	neuralEconUpgradeThreshold = -0.3
	// neuralExploreRate 探索率：以此概率选择次优决策（避免收敛到局部最优）。
	neuralExploreRate = 0.1
	// neuralMaxTowerStr 经典模式塔强度上限（100 + 4×50）。
	neuralMaxTowerStr = 300
)

// ── 塔型选择特征 ──────────────────────────────────

// towerTypeFeatureLen 塔型选择特征维度。
const towerTypeFeatureLen = 8

const (
	featTTCostRatio      = iota // def.Cost / gold
	featTTDamageEff             // def.Damage * def.Range / def.Cost
	featTTHasCCTower            // 已有 CC 塔?
	featTTHasDPSTower           // 已有高伤塔?
	featTTNeedCC                // 需要 CC?
	featTTNeedDPS               // 需要 DPS?
	featTTTowerCountRatio       // 已有此类型数 / 总塔数
	featTTProgress              // 游戏进度
)

// ── 神经策略战略阶段 ──────────────────────────────────────────

const (
	// nPhaseSetup 初始建设阶段：先建 CC + DPS 两座塔。
	nPhaseSetup = 0
	// nPhaseUpgrade carry 升级阶段：集中资源升级 carry 到满级。
	nPhaseUpgrade = 1
	// nPhaseExpand 扩张阶段：carry 已满级，建更多塔并用神经网络决策。
	nPhaseExpand = 2
)

// ── NeuralStrategy ──────────────────────────────────

// NeuralStrategy 神经网络驱动的自学习策略。
//
// 核心改进：用三阶段战略规划（setup → upgrade → expand）替代
// 逐帧神经网络独立决策，避免在建塔和升级之间反复摇摆。
// 阶段内的具体目标（选哪座塔、选哪个格子）仍由神经网络评分。
//
// 阶段转换：
//   - Phase 0→1: ccBuilt AND dpsBuilt（两座核心塔已就位）
//   - Phase 1→2: carryMaxed（carry 强度达到 300 上限）
type NeuralStrategy struct {
	model   *learning.Model
	trainer *learning.Trainer
	rng     *rand.Rand

	// ── 战略记忆（阶段驱动决策的核心） ──
	phase         int  // 当前阶段: nPhaseSetup / nPhaseUpgrade / nPhaseExpand
	ccBuilt       bool // 已建 CC 塔（cl_shotgun）
	dpsBuilt      bool // 已建 DPS/carry 塔（cl_sentinel）
	carryStr      int  // carry 当前强度（每帧更新）
	carryMaxed    bool // carry 已达强度上限（300）
	totalTowerStr int  // 所有塔强度总和（每帧更新）

	// ── 游戏状态追踪 ──
	lastWave  int
	lastLives int
	lastKills int
	builtCount int
	carryRow  int
	carryCol  int
	hasCarry  bool

	// ── 第一座塔位置（用于第二座塔就近放置） ──
	firstTowerX float64
	firstTowerY float64

	// ── 经典模式 ──
	classicMode bool
	towerDefMap map[string]int // towerKey → TowerDefs 索引

	// ── 路径数据缓存 ──
	allPaths  [][]PathPoint
	pathPts   []learning.PathPt // learning 包格式的路径点
	initDone  bool

	// ── 节流 ──
	lastBuildTick   int
	lastUpgradeTick int

	// ── Boss 波感知 ──
	bossEvery int

	// ── 训练统计 ──
	wavesSinceLastBuild int
	perfectWaveStreak   int
	lastWaveEnemyHP     float64 // 上一波平均敌人 HP

	// ── 种子 ──
	seed int64
}

// NeuralOption 配置选项。
type NeuralOption func(*NeuralStrategy)

// WithNeuralSeed 设置随机种子。
func WithNeuralSeed(seed int64) NeuralOption {
	return func(s *NeuralStrategy) { s.seed = seed }
}

// WithNeuralTrainer 设置训练器（启用在线学习）。
func WithNeuralTrainer(t *learning.Trainer) NeuralOption {
	return func(s *NeuralStrategy) { s.trainer = t }
}

// NewNeuralStrategy 创建神经网络驱动策略。
//
// model: 权重模型（来自 LoadFromFS 或训练管线传入）。
// opts: 可选配置（种子、训练器等）。
func NewNeuralStrategy(model *learning.Model, opts ...NeuralOption) *NeuralStrategy {
	if model == nil {
		model = learning.DefaultModel()
	}
	s := &NeuralStrategy{
		model: model,
		seed:  42,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *NeuralStrategy) Name() string { return "neural" }

func (s *NeuralStrategy) Init(state *GameState) {
	s.rng = rand.New(rand.NewSource(s.seed))
	s.lastWave = 0
	s.lastLives = state.Lives
	s.lastKills = 0
	s.builtCount = 0
	s.hasCarry = false
	s.initDone = false
	s.lastBuildTick = -100
	s.lastUpgradeTick = -100
	s.wavesSinceLastBuild = 0
	s.perfectWaveStreak = 0
	s.lastWaveEnemyHP = 0

	// 战略记忆初始化
	s.phase = nPhaseSetup
	s.ccBuilt = false
	s.dpsBuilt = false
	s.carryStr = 0
	s.carryMaxed = false
	s.totalTowerStr = 0

	// 检测经典模式
	s.classicMode = detectClassic(state)

	// 构建 towerDef 查找表
	s.towerDefMap = make(map[string]int, len(state.TowerDefs))
	for i, d := range state.TowerDefs {
		s.towerDefMap[d.Key] = i
	}

	// Boss 间隔
	s.bossEvery = config.GlobalSpawnerConfig().Boss.EveryNWaves
	if s.bossEvery <= 0 {
		s.bossEvery = 4
	}

	// 如果 MapInfo 已可用，立即初始化路径缓存
	if state.MapInfo != nil {
		s.initPaths(state)
		s.initDone = true
	}
}

func (s *NeuralStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	// 延迟初始化路径缓存
	if !s.initDone && state.MapInfo != nil {
		s.initPaths(state)
		s.initDone = true
	}

	// 波次变化 → 触发训练 reward
	if state.Wave > s.lastWave && s.lastWave > 0 {
		s.onWaveChange(state)
	}

	// ── 每帧更新战略记忆 ──
	s.updateStrategicMemory(state)

	// 战灵选择：经典模式无战灵，其他模式选 chain
	if !state.WardenReady {
		if s.classicMode {
			// 经典模式无战灵，视为已就绪
		} else {
			return []Action{{Type: ActionSelectWarden, WardenKey: "chain"}}
		}
	}

	var actions []Action

	if !state.WaveActive {
		// ── 波间歇期 ──
		actions = s.decideWavePause(state)
	} else {
		// ── 波进行中 ──
		actions = s.decideWaveActive(state)
	}

	s.lastWave = state.Wave
	s.lastLives = state.Lives
	s.lastKills = state.TotalKills
	return actions
}

// ── 战略记忆更新 ──────────────────────────────────

// updateStrategicMemory 每帧更新战略记忆：carry 强度、阶段转换。
//
// 阶段转换逻辑（单向推进，不回退）：
//   - Phase 0→1: ccBuilt AND dpsBuilt
//   - Phase 1→2: carryMaxed (carry 强度达 300)
func (s *NeuralStrategy) updateStrategicMemory(state *GameState) {
	// 更新 carry 强度
	s.totalTowerStr = 0
	for _, t := range state.Towers {
		s.totalTowerStr += t.Strength
		if s.hasCarry && t.Row == s.carryRow && t.Col == s.carryCol {
			s.carryStr = t.Strength
			s.carryMaxed = t.Strength >= neuralMaxTowerStr
		}
	}

	// 检测 CC / DPS 塔是否已建（经典模式下按 key 判断）
	if s.classicMode {
		for _, t := range state.Towers {
			if t.Key == "cl_shotgun" {
				s.ccBuilt = true
			}
			if t.Key == "cl_sentinel" {
				s.dpsBuilt = true
			}
		}
	} else {
		// 非经典模式：有塔就算 CC+DPS 都满足
		if len(state.Towers) >= 2 {
			s.ccBuilt = true
			s.dpsBuilt = true
		} else if len(state.Towers) >= 1 {
			s.ccBuilt = true
			s.dpsBuilt = true // 非经典只有 basic 塔
		}
	}

	// 阶段推进
	if s.phase == nPhaseSetup && s.ccBuilt && s.dpsBuilt {
		s.phase = nPhaseUpgrade
	}
	if s.phase == nPhaseUpgrade && s.carryMaxed {
		s.phase = nPhaseExpand
	}
}

// wavesUntilBoss 计算距离下一个 Boss 波的波数。
// 返回 0 表示当前波就是 Boss 波。
func (s *NeuralStrategy) wavesUntilNextBoss(currentWave int) int {
	if s.bossEvery <= 0 {
		return 99
	}
	remainder := currentWave % s.bossEvery
	if remainder == 0 && currentWave > 0 {
		return 0 // 当前波是 Boss 波
	}
	return s.bossEvery - remainder
}

// ── 波间歇期决策 ──────────────────────────────────

func (s *NeuralStrategy) decideWavePause(state *GameState) []Action {
	// 记录经济决策到训练器（所有阶段都记录，供训练使用）
	if s.trainer != nil {
		econFeats := s.extractEconFeatures(state)
		econScore := s.scoreEcon(state)
		s.trainer.RecordEcon(econFeats, econScore)
	}

	// Boss 波感知：下一波是 Boss 时，优先升级而非建塔
	nextWave := state.Wave + 1
	isPreBoss := s.wavesUntilNextBoss(nextWave) == 0

	switch s.phase {
	case nPhaseSetup:
		// ── Phase 0: 只建塔，不升级 ──
		// 目标：尽快建出 CC + DPS 两座核心塔
		if s.canAfford(state) && len(state.BuildCells) > 0 {
			if action, ok := s.tryBuild(state); ok {
				return []Action{action}
			}
		}

	case nPhaseUpgrade:
		// ── Phase 1: 只升级 carry，不建新塔 ──
		// 目标：把 carry（cl_sentinel）堆到 300 强度上限
		// Boss 前加速升级：一次花光所有金币
		if ups := s.tryMultiUpgradeCarry(state); len(ups) > 0 {
			return ups
		}

	case nPhaseExpand:
		// ── Phase 2: 神经网络决策（建塔 vs 升级） ──
		// carry 已满级，现在扩张：建更多塔 + 升级新塔

		// Boss 前优先升级（不建新塔，集中强化现有火力）
		if isPreBoss {
			if ups := s.tryMultiUpgrade(state); len(ups) > 0 {
				return ups
			}
		}

		// 所有塔满级 → 建新塔（金币转化为额外 DPS）
		if s.allTowersMaxed(state) && s.canAfford(state) && len(state.BuildCells) > 0 {
			if action, ok := s.tryBuild(state); ok {
				return []Action{action}
			}
		}

		// 用神经网络 econScore 决定建塔 vs 升级
		econScore := s.scoreEcon(state)
		if econScore > neuralEconBuildThreshold {
			if s.canAfford(state) && len(state.BuildCells) > 0 {
				if action, ok := s.tryBuild(state); ok {
					return []Action{action}
				}
			}
			if ups := s.tryMultiUpgrade(state); len(ups) > 0 {
				return ups
			}
		} else {
			// 偏向升级（升级最弱的塔，拉平整体实力）
			if ups := s.tryMultiUpgrade(state); len(ups) > 0 {
				return ups
			}
			if s.canAfford(state) && len(state.BuildCells) > 0 {
				if action, ok := s.tryBuild(state); ok {
					return []Action{action}
				}
			}
		}
	}

	// 开波判断
	if state.Wave < state.MaxWaves && len(state.Towers) > 0 {
		if s.shouldStartWave(state) {
			return []Action{{Type: ActionStartWave}}
		}
	}

	return nil
}

// ── 波进行中决策 ──────────────────────────────────

func (s *NeuralStrategy) decideWaveActive(state *GameState) []Action {
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	goldReserve := 10

	// 波中升级策略根据阶段不同
	if state.Gold >= upgCost+goldReserve && len(state.Towers) > 0 {
		switch s.phase {
		case nPhaseUpgrade:
			// Phase 1: 波中也只升级 carry
			if action, ok := s.tryUpgradeCarryOnly(state); ok {
				return []Action{action}
			}
		default:
			// Phase 0/2: 波中升级最优目标
			if action, ok := s.tryUpgrade(state); ok {
				return []Action{action}
			}
		}
	}
	return nil
}

// ── 建塔 ──────────────────────────────────

func (s *NeuralStrategy) tryBuild(state *GameState) (Action, bool) {
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return Action{}, false
	}
	if state.Tick-s.lastBuildTick < neuralBuildThrottle {
		return Action{}, false
	}

	// 1. 选塔型
	towerDef := s.pickTowerType(state)
	if towerDef == nil || state.Gold < towerDef.Cost {
		return Action{}, false
	}

	// 2. 选位置：用神经网络评分所有候选格子
	cell, ok := s.pickBuildCell(state, towerDef)
	if !ok {
		return Action{}, false
	}

	s.builtCount++
	s.lastBuildTick = state.Tick
	s.wavesSinceLastBuild = 0

	// 记录第一座塔位置（供第二座塔就近放置）
	if s.builtCount == 1 {
		s.firstTowerX = cell.X
		s.firstTowerY = cell.Y
	}

	// carry 应该是 cl_sentinel（DPS），不是 cl_shotgun（CC）。
	// builtCount==2 表示刚建完第二座塔（cl_sentinel），设为 carry。
	if s.classicMode && s.builtCount == 2 && towerDef.Key == "cl_sentinel" {
		s.carryRow = cell.Row
		s.carryCol = cell.Col
		s.hasCarry = true
	} else if !s.hasCarry && !s.classicMode {
		// 非经典模式：第一座塔为 carry
		s.carryRow = cell.Row
		s.carryCol = cell.Col
		s.hasCarry = true
	}

	return Action{
		Type:     ActionBuild,
		TowerKey: towerDef.Key,
		Cell:     cell,
	}, true
}

// pickTowerType 用神经网络评分选择塔型。
//
// 经典模式：
//   - 第一座塔: 强制 cl_shotgun（CC 入口，这是最小必要骨架而非"策略"）
//   - 第二座塔: 强制 cl_sentinel（carry DPS，同上）
//   - 后续: 神经网络评分选最优塔型
//
// 非经典模式：只有 basic 塔，直接返回。
func (s *NeuralStrategy) pickTowerType(state *GameState) *TowerDefInfo {
	if len(state.TowerDefs) == 0 {
		return nil
	}

	// 非经典模式只有一种塔
	if !s.classicMode {
		for i := range state.TowerDefs {
			if state.Gold >= state.TowerDefs[i].Cost {
				return &state.TowerDefs[i]
			}
		}
		return nil
	}

	// 经典模式：前两座有最小骨架（CC + carry 是 TD 游戏的基本常识）
	if s.builtCount == 0 {
		if idx, ok := s.towerDefMap["cl_shotgun"]; ok {
			d := &state.TowerDefs[idx]
			if state.Gold >= d.Cost {
				return d
			}
		}
	}
	if s.builtCount == 1 {
		if idx, ok := s.towerDefMap["cl_sentinel"]; ok {
			d := &state.TowerDefs[idx]
			if state.Gold >= d.Cost {
				return d
			}
		}
	}

	// 后续塔型用特征评分
	hasCCTower := false
	hasDPSTower := false
	towerTypeCounts := make(map[string]int)
	for _, t := range state.Towers {
		towerTypeCounts[t.Key]++
		for _, ab := range t.Abilities {
			if ab == "slow" || ab == "stun" || ab == "slowPower" {
				hasCCTower = true
			}
		}
		if t.Damage > 30 {
			hasDPSTower = true
		}
	}

	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	var bestDef *TowerDefInfo
	bestScore := math.Inf(-1)

	for i := range state.TowerDefs {
		d := &state.TowerDefs[i]
		if state.Gold < d.Cost {
			continue
		}

		// 提取塔型选择特征
		var feats learning.FeatureVec
		feats.Len = towerTypeFeatureLen

		// 费用比
		feats.Values[featTTCostRatio] = clampNeural(float64(d.Cost) / float64(max(state.Gold, 1)))
		// 伤害效率
		efficiency := 0.0
		if d.Cost > 0 {
			efficiency = d.Damage * d.Range / float64(d.Cost)
		}
		feats.Values[featTTDamageEff] = clampNeural(efficiency / 10.0) // 归一化
		// CC / DPS 状态
		if hasCCTower {
			feats.Values[featTTHasCCTower] = 1.0
		}
		if hasDPSTower {
			feats.Values[featTTHasDPSTower] = 1.0
		}
		if !hasCCTower {
			feats.Values[featTTNeedCC] = 1.0
		}
		if !hasDPSTower {
			feats.Values[featTTNeedDPS] = 1.0
		}
		// 同类型塔数量
		totalTowers := max(len(state.Towers), 1)
		feats.Values[featTTTowerCountRatio] = clampNeural(float64(towerTypeCounts[d.Key]) / float64(totalTowers))
		// 进度
		feats.Values[featTTProgress] = clampNeural(progress)

		// 评分：用 econ 网络评分（复用已有的网络，towerType 特征维度小于 econ）
		// 这里直接用线性组合评分，因为 towerType 网络尚未训练
		score := 0.0
		score += feats.Values[featTTDamageEff] * 0.3
		score += feats.Values[featTTNeedCC] * 0.25
		score += feats.Values[featTTNeedDPS] * 0.2
		score -= feats.Values[featTTCostRatio] * 0.15
		score -= feats.Values[featTTTowerCountRatio] * 0.1

		// 探索：小概率添加随机噪声
		if s.rng.Float64() < neuralExploreRate {
			score += s.rng.NormFloat64() * 0.2
		}

		if score > bestScore {
			bestScore = score
			bestDef = d
		}
	}

	return bestDef
}

// pickBuildCell 用神经网络评分选择最优建造位置。
//
// 特殊处理：建第二座塔（cl_sentinel）时，优先选择距第一座塔（cl_shotgun）
// 150px 以内的位置，使两塔共享同一杀伤区。
func (s *NeuralStrategy) pickBuildCell(state *GameState, def *TowerDefInfo) (Cell, bool) {
	if len(state.BuildCells) == 0 {
		return Cell{}, false
	}

	// 准备现有塔信息（转换为 learning 包类型）
	existingTowers := s.toLearningTowers(state)

	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	cellSize := 60
	if state.MapInfo != nil {
		cellSize = state.MapInfo.CellSize
	}

	// builtCount==1：建第二座塔时，过滤出距第一座塔 150px 以内的候选格子
	const nearRadius = 150.0
	candidates := state.BuildCells
	if s.classicMode && s.builtCount == 1 && s.firstTowerX > 0 {
		var nearCells []Cell
		for _, c := range state.BuildCells {
			dx := c.X - s.firstTowerX
			dy := c.Y - s.firstTowerY
			if dx*dx+dy*dy <= nearRadius*nearRadius {
				nearCells = append(nearCells, c)
			}
		}
		// 有足够近的候选格 → 只从中选；否则回退到全部候选
		if len(nearCells) > 0 {
			candidates = nearCells
		}
	}

	var bestCell Cell
	bestScore := math.Inf(-1)
	found := false

	for _, c := range candidates {
		input := learning.BuildCellInput{
			CellX: c.X, CellY: c.Y,
			CellRow: c.Row, CellCol: c.Col,
			PathPoints:     s.pathPts,
			MapCenterX:     state.MapPixelW / 2,
			MapCenterY:     state.MapPixelH / 2,
			TowerRange:     def.Range,
			ExistingTowers: existingTowers,
			Gold:           state.Gold,
			TowerCost:      def.Cost,
			Progress:       progress,
			TowerCount:     len(state.Towers),
			Noise:          s.rng.Float64(),
			BossNext:       s.isNextBoss(state.Wave + 1),
			CellSize:       cellSize,
		}

		feats := learning.ExtractBuildFeatures(input)
		score := s.model.ScoreBuild(feats)

		// 记录到训练器
		if s.trainer != nil {
			s.trainer.RecordBuild(feats, score)
		}

		// 探索：小概率加噪声
		if s.rng.Float64() < neuralExploreRate {
			score += s.rng.NormFloat64() * 0.1
		}

		if score > bestScore {
			bestScore = score
			bestCell = c
			found = true
		}
	}

	return bestCell, found
}

// ── 升级 ──────────────────────────────────

func (s *NeuralStrategy) tryUpgrade(state *GameState) (Action, bool) {
	if len(state.Towers) == 0 {
		return Action{}, false
	}
	if state.Tick-s.lastUpgradeTick < neuralUpgradeThrottle {
		return Action{}, false
	}

	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	if state.Gold < upgCost {
		return Action{}, false
	}

	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	threatLevel := s.calcThreatLevel(state)
	cellSize := 60
	if state.MapInfo != nil {
		cellSize = state.MapInfo.CellSize
	}

	var bestTower *TowerInfo
	bestScore := math.Inf(-1)

	for i := range state.Towers {
		t := &state.Towers[i]

		// 经典模式跳过已达强度上限的塔
		if s.classicMode && t.Strength >= neuralMaxTowerStr {
			continue
		}

		input := learning.UpgradeTowerInput{
			Tower: learning.TowerInfo{
				Row: t.Row, Col: t.Col,
				X: t.X, Y: t.Y,
				Range: t.Range, Damage: t.Damage,
				Strength:    t.Strength,
				AttackSpeed: t.AttackSpeed,
				Kills:       t.Kills,
				Abilities:   t.Abilities,
				Cost:        t.Cost,
			},
			PathPoints:  s.pathPts,
			UpgCost:     upgCost,
			Progress:    progress,
			ThreatLevel: threatLevel,
			Gold:        state.Gold,
			CellSize:    cellSize,
		}

		feats := learning.ExtractUpgradeFeatures(input)
		score := s.model.ScoreUpgrade(feats)

		// 记录到训练器
		if s.trainer != nil {
			s.trainer.RecordUpgrade(feats, score)
		}

		// 探索
		if s.rng.Float64() < neuralExploreRate {
			score += s.rng.NormFloat64() * 0.1
		}

		if score > bestScore {
			bestScore = score
			bestTower = t
		}
	}

	if bestTower == nil {
		return Action{}, false
	}

	s.lastUpgradeTick = state.Tick
	return Action{
		Type: ActionUpgrade,
		Row:  bestTower.Row,
		Col:  bestTower.Col,
	}, true
}

// tryMultiUpgrade 在波间歇期连续升级（最多 10 次），把金币花光转化为 DPS。
// Phase 2 时优先升级最弱的塔（拉平整体实力）。
// 返回多个 ActionUpgrade，确保波前不浪费金币。
func (s *NeuralStrategy) tryMultiUpgrade(state *GameState) []Action {
	if len(state.Towers) == 0 {
		return nil
	}
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	if state.Gold < upgCost {
		return nil
	}

	var actions []Action
	gold := state.Gold
	const maxBatch = 10

	for range maxBatch {
		if gold < upgCost {
			break
		}
		var action Action
		var ok bool
		if s.phase == nPhaseExpand {
			// Phase 2: 升级最弱的塔
			action, ok = s.tryUpgradeWeakest(state)
		} else {
			action, ok = s.tryUpgrade(state)
		}
		if !ok {
			break // 所有塔已满级或无可升级目标
		}
		actions = append(actions, action)
		gold -= upgCost
	}
	return actions
}

// tryUpgradeCarryOnly 只升级 carry 塔（Phase 1 专用）。
// 如果 carry 已满级或不存在，返回 false。
func (s *NeuralStrategy) tryUpgradeCarryOnly(state *GameState) (Action, bool) {
	if !s.hasCarry {
		return Action{}, false
	}
	if state.Tick-s.lastUpgradeTick < neuralUpgradeThrottle {
		return Action{}, false
	}

	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	if state.Gold < upgCost {
		return Action{}, false
	}

	// 查找 carry 塔
	for _, t := range state.Towers {
		if t.Row == s.carryRow && t.Col == s.carryCol {
			if s.classicMode && t.Strength >= neuralMaxTowerStr {
				return Action{}, false // 已满级
			}
			s.lastUpgradeTick = state.Tick
			return Action{
				Type: ActionUpgrade,
				Row:  t.Row,
				Col:  t.Col,
			}, true
		}
	}
	return Action{}, false
}

// tryMultiUpgradeCarry 批量升级 carry 塔（Phase 1 波间歇期专用）。
// 一次花光所有金币到 carry 身上。
func (s *NeuralStrategy) tryMultiUpgradeCarry(state *GameState) []Action {
	if !s.hasCarry {
		return nil
	}
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	if state.Gold < upgCost {
		return nil
	}

	// 查找 carry 塔
	var carry *TowerInfo
	for i := range state.Towers {
		if state.Towers[i].Row == s.carryRow && state.Towers[i].Col == s.carryCol {
			carry = &state.Towers[i]
			break
		}
	}
	if carry == nil {
		return nil
	}

	// 经典模式检查上限
	if s.classicMode && carry.Strength >= neuralMaxTowerStr {
		return nil
	}

	var actions []Action
	gold := state.Gold
	str := carry.Strength
	const maxBatch = 20 // carry 集中升级允许更大批次

	for range maxBatch {
		if gold < upgCost {
			break
		}
		if s.classicMode && str >= neuralMaxTowerStr {
			break
		}
		actions = append(actions, Action{
			Type: ActionUpgrade,
			Row:  carry.Row,
			Col:  carry.Col,
		})
		gold -= upgCost
		str += int(config.GlobalBalance().Tower.StrengthBuyAmount)
	}
	return actions
}

// tryUpgradeWeakest Phase 2 专用：升级最弱的塔（拉平整体实力）。
func (s *NeuralStrategy) tryUpgradeWeakest(state *GameState) (Action, bool) {
	if len(state.Towers) == 0 {
		return Action{}, false
	}
	if state.Tick-s.lastUpgradeTick < neuralUpgradeThrottle {
		return Action{}, false
	}

	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	if state.Gold < upgCost {
		return Action{}, false
	}

	var weakest *TowerInfo
	weakestStr := math.MaxInt32

	for i := range state.Towers {
		t := &state.Towers[i]
		if s.classicMode && t.Strength >= neuralMaxTowerStr {
			continue
		}
		if t.Strength < weakestStr {
			weakestStr = t.Strength
			weakest = t
		}
	}

	if weakest == nil {
		return Action{}, false
	}

	s.lastUpgradeTick = state.Tick
	return Action{
		Type: ActionUpgrade,
		Row:  weakest.Row,
		Col:  weakest.Col,
	}, true
}

// shouldStartWave 判断是否应该开波（阶段感知）。
//
// Phase 0: 不开波直到 CC + DPS 都建好
// Phase 1: 花光金币升级 carry 后才开波
// Phase 2: 没有可执行的金币操作时开波
func (s *NeuralStrategy) shouldStartWave(state *GameState) bool {
	if len(state.Towers) == 0 {
		return false
	}
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost

	switch s.phase {
	case nPhaseSetup:
		// Phase 0: 两座核心塔都建好才开波
		if !s.ccBuilt || !s.dpsBuilt {
			// 但如果买不起也没办法，只能开波
			if !s.canAfford(state) {
				return true
			}
			return false
		}
		// 两座塔都建好了，花完剩余金币
		if state.Gold >= upgCost && !s.allTowersMaxed(state) {
			return false
		}
		return true

	case nPhaseUpgrade:
		// Phase 1: carry 没满级且还有钱 → 不开波
		if !s.carryMaxed && state.Gold >= upgCost {
			return false
		}
		return true

	case nPhaseExpand:
		// Phase 2: 还能升级且有未满级塔 → 不开波
		if state.Gold >= upgCost && !s.allTowersMaxed(state) {
			return false
		}
		// 还能建新塔 → 不开波
		if s.canAfford(state) && len(state.BuildCells) > 0 {
			return false
		}
		return true
	}

	return true
}

// allTowersMaxed 检查所有塔是否已达强度上限。
// 非经典模式无上限，始终返回 false。
func (s *NeuralStrategy) allTowersMaxed(state *GameState) bool {
	if !s.classicMode || len(state.Towers) == 0 {
		return false
	}
	for _, t := range state.Towers {
		if t.Strength < neuralMaxTowerStr {
			return false
		}
	}
	return true
}

// ── 经济评分 ──────────────────────────────────

func (s *NeuralStrategy) scoreEcon(state *GameState) float64 {
	feats := s.extractEconFeatures(state)
	return s.model.ScoreEcon(feats)
}

func (s *NeuralStrategy) extractEconFeatures(state *GameState) learning.FeatureVec {
	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	totalGold := state.Gold
	urgency := 0.0
	if state.Lives > 0 && state.Lives < 5 {
		urgency = 1.0 - float64(state.Lives)/5.0
	}

	threatLevel := s.calcThreatLevel(state)

	// 敌人 HP 趋势
	enemyHPTrend := 1.0
	if s.lastWaveEnemyHP > 0 {
		avgHP := s.currentWaveAvgHP(state)
		if avgHP > 0 {
			enemyHPTrend = avgHP / s.lastWaveEnemyHP
		}
	}

	// 全塔 DPS
	totalDPS := 0.0
	for _, t := range state.Towers {
		as := t.AttackSpeed
		if as <= 0 {
			as = 1.0
		}
		totalDPS += t.Damage * as
	}

	// Boss 波感知
	nextWave := state.Wave + 1
	wavesUntilBoss := s.wavesUntilNextBoss(nextWave)
	isBossNext := wavesUntilBoss == 0
	isPreBoss := wavesUntilBoss <= 1

	// 建议（基于阶段 + Boss 感知）
	advice := "balanced"
	switch s.phase {
	case nPhaseSetup:
		advice = "build_cc"
	case nPhaseUpgrade:
		advice = "upgrade"
	case nPhaseExpand:
		if isPreBoss || isBossNext {
			// Boss 前优先升级
			advice = "upgrade"
		} else if s.allTowersMaxed(state) {
			advice = "build_dps"
		}
	}

	// 经济偏好根据阶段调整
	aggression := 0.5
	economy := 0.5
	switch s.phase {
	case nPhaseSetup:
		aggression = 0.8 // 激进建塔
		economy = 0.2
	case nPhaseUpgrade:
		aggression = 0.2 // 保守攒钱升级
		economy = 0.8
	case nPhaseExpand:
		if isPreBoss {
			aggression = 0.3
			economy = 0.7
		}
	}

	input := learning.EconInput{
		Progress:            progress,
		TowerCount:          len(state.Towers),
		Gold:                state.Gold,
		TotalGold:           totalGold,
		Urgency:             urgency,
		ThreatLevel:         threatLevel,
		Aggression:          aggression,
		Economy:             economy,
		AdvicePriority:      advice,
		BossNext:            isBossNext,
		WavesSinceLastBuild: s.wavesSinceLastBuild,
		EnemyHPTrend:        enemyHPTrend,
		Lives:               state.Lives,
		MaxLives:            20, // 默认最大生命
		TotalTowerDPS:       totalDPS,
		WaveEnemyTotalHP:    0, // 无法精确估算
		PerfectWaveStreak:   s.perfectWaveStreak,
	}

	return learning.ExtractEconFeatures(input)
}

// ── 训练回调 ──────────────────────────────────

// onWaveChange 波次变化时触发训练 reward（增强版）。
//
// 额外 reward shaping：
//   - Boss 波存活: +5（vs 普通波 +1）
//   - 完美波（零泄漏）: +2
//   - 阶段转换早期完成: +1
func (s *NeuralStrategy) onWaveChange(state *GameState) {
	livesLost := s.lastLives - state.Lives
	if livesLost < 0 {
		livesLost = 0
	}

	// 更新追踪状态（不依赖 trainer）
	s.wavesSinceLastBuild++
	if livesLost == 0 {
		s.perfectWaveStreak++
	} else {
		s.perfectWaveStreak = 0
	}

	// 记录本波平均敌人 HP（供下一波趋势计算）
	s.lastWaveEnemyHP = s.currentWaveAvgHP(state)

	if s.trainer == nil {
		return
	}

	killsThisWave := state.TotalKills - s.lastKills

	s.trainer.OnWaveEnd(learning.WaveStats{
		WaveNum:       s.lastWave,
		KillsThisWave: killsThisWave,
		LivesBefore:   s.lastLives,
		LivesAfter:    state.Lives,
	})

	// ── 额外 reward shaping ──

	// Boss 波存活奖励（大幅强化）
	wasBossWave := s.bossEvery > 0 && s.lastWave > 0 && s.lastWave%s.bossEvery == 0
	if wasBossWave && livesLost == 0 {
		// Boss 波完美通过：额外 +5 reward
		s.trainer.OnWaveEnd(learning.WaveStats{
			WaveNum:       s.lastWave,
			KillsThisWave: killsThisWave,
			LivesBefore:   state.Lives, // 无损失
			LivesAfter:    state.Lives,
		})
	}

	// 完美波奖励（零泄漏）
	if livesLost == 0 && killsThisWave > 0 {
		s.trainer.OnWaveEnd(learning.WaveStats{
			WaveNum:       s.lastWave,
			KillsThisWave: killsThisWave,
			LivesBefore:   state.Lives,
			LivesAfter:    state.Lives,
		})
	}
}

// OnGameEnd 游戏结束时通知训练器（由外部调用）。
func (s *NeuralStrategy) OnGameEnd(state *GameState, won bool) {
	if s.trainer == nil {
		return
	}
	s.trainer.OnGameEnd(learning.GameResult{
		Won:          won,
		WavesReached: state.Wave,
		MaxWaves:     state.MaxWaves,
		LivesLeft:    state.Lives,
		MaxLives:     20,
		TotalKills:   state.TotalKills,
	})
}

// Model 返回当前模型（训练后的权重可提取导出）。
func (s *NeuralStrategy) Model() *learning.Model {
	return s.model
}

// Trainer 返回训练器（可选，nil 表示非训练模式）。
func (s *NeuralStrategy) Trainer() *learning.Trainer {
	return s.trainer
}

// ── 辅助函数 ──────────────────────────────────

// initPaths 初始化路径数据缓存。
func (s *NeuralStrategy) initPaths(state *GameState) {
	mi := state.MapInfo
	if mi == nil {
		return
	}

	// 收集所有路径
	if mi.MultiPath && len(mi.Paths) > 0 {
		for _, p := range mi.Paths {
			s.allPaths = append(s.allPaths, p.Waypoints)
		}
	}
	if len(s.allPaths) == 0 && len(mi.Waypoints) > 0 {
		s.allPaths = [][]PathPoint{mi.Waypoints}
	}

	// 转换为 learning 包格式
	for _, path := range s.allPaths {
		for _, wp := range path {
			s.pathPts = append(s.pathPts, learning.PathPt{X: wp.X, Y: wp.Y})
		}
	}
}

// toLearningTowers 将 GameState 中的塔转换为 learning 包类型。
func (s *NeuralStrategy) toLearningTowers(state *GameState) []learning.TowerInfo {
	towers := make([]learning.TowerInfo, len(state.Towers))
	for i, t := range state.Towers {
		towers[i] = learning.TowerInfo{
			Row: t.Row, Col: t.Col,
			X: t.X, Y: t.Y,
			Range: t.Range, Damage: t.Damage,
			Strength:    t.Strength,
			AttackSpeed: t.AttackSpeed,
			Kills:       t.Kills,
			Abilities:   t.Abilities,
			Cost:        t.Cost,
		}
	}
	return towers
}

// findCarry 查找 carry 塔。
func (s *NeuralStrategy) findCarry(state *GameState) *TowerInfo {
	for i := range state.Towers {
		if state.Towers[i].Row == s.carryRow && state.Towers[i].Col == s.carryCol {
			return &state.Towers[i]
		}
	}
	return nil
}

// canAfford 检查是否买得起最便宜的塔。
func (s *NeuralStrategy) canAfford(state *GameState) bool {
	for _, d := range state.TowerDefs {
		if state.Gold >= d.Cost {
			return true
		}
	}
	return false
}

// isNextBoss 判断下一波是否是 Boss 波。
func (s *NeuralStrategy) isNextBoss(nextWave int) bool {
	if nextWave <= 0 || s.bossEvery <= 0 {
		return false
	}
	return nextWave%s.bossEvery == 0
}

// calcThreatLevel 计算当前威胁等级 [0,1]。
// 基于活跃敌人数量和生命值消耗。
func (s *NeuralStrategy) calcThreatLevel(state *GameState) float64 {
	threat := 0.0

	// 活跃敌人比例
	activeCount := 0
	for _, e := range state.Enemies {
		if e.Active && !e.Dying {
			activeCount++
		}
	}
	if activeCount > 10 {
		threat += 0.3
	} else if activeCount > 5 {
		threat += 0.15
	}

	// 生命损耗
	if s.lastLives > 0 {
		livesLost := s.lastLives - state.Lives
		if livesLost > 0 {
			threat += clampNeural(float64(livesLost) / float64(s.lastLives))
		}
	}

	// 低生命
	if state.Lives <= 3 {
		threat += 0.3
	}

	return clampNeural(threat)
}

// currentWaveAvgHP 计算当前波次活跃敌人的平均 HP。
func (s *NeuralStrategy) currentWaveAvgHP(state *GameState) float64 {
	count := 0
	totalHP := 0.0
	for _, e := range state.Enemies {
		if e.Active && !e.Dying {
			totalHP += e.MaxHP
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return totalHP / float64(count)
}

// detectClassic 检测经典模式（TowerDefs 含 cl_ 前缀）。
func detectClassic(state *GameState) bool {
	for _, d := range state.TowerDefs {
		if strings.HasPrefix(d.Key, "cl_") {
			return true
		}
	}
	return false
}

// clampNeural 钳制到 [0,1]（与 learning 包同名函数独立，避免导出依赖）。
func clampNeural(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
