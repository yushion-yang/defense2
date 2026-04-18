// strategy_neural.go — 神经网络驱动的自学习策略，融合领域知识加速收敛。
//
// 核心理念：用领域知识（chokepoint 放塔、CC+DPS 重叠、激进经济）为神经网络
// 提供强力先验，使其从第 0 局就能表现出"懂 TD"的水准。
// 神经网络仍可通过训练在先验基础上发现更优策略。
//
// 编码的关键领域知识：
//   1. 路径拐点（chokepoint）是最高价值放塔位置（3x 评分加成）
//   2. Phase 0: CC 放在第一拐点，DPS 紧贴 CC（乘法伤害）
//   3. 激进经济：花光金币才开波，NEVER delay
//   4. Reward shaping: 奖励完美波/Boss 存活，惩罚囤金/不建塔
//
// 与 CompetentStrategy 的区别：
//   - CompetentStrategy: 完全硬编码的建造计划 + 规则引擎
//   - NeuralStrategy: 领域知识先验 + 神经网络评分（可持续学习改进）
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
	// neuralPhase2MaxTowers Phase 2 建塔上限。
	neuralPhase2MaxTowers = 5
	// neuralUncoveredBendRadius 判定拐点"已覆盖"的半径（像素）。
	neuralUncoveredBendRadius = 150.0
)

// ── Phase 2 DPS 塔型优先级（经典模式） ──────────────────────────────────

// phase2DPSPreference Phase 2 扩张时优先选择的 DPS 塔型。
// 排序依据：cl_gatling（barrage+flat damage） > cl_railgun（wide beam） > cl_mortar（splash+stun）。
var phase2DPSPreference = []string{
	"cl_gatling",
	"cl_railgun",
	"cl_mortar",
}

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
	pathBends []PathPoint       // 预计算的路径拐点（chokepoints）
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
	s.allPaths = nil
	s.pathPts = nil
	s.pathBends = nil
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
		// ── Phase 2: 确定性 build-then-max 循环 ──
		// carry 已满级，激进扩张：先升满现有塔 → 再建新 DPS 塔 → 升满 → 循环。
		//
		// 关键洞察：100 强度的新塔 DPS 很低（约 50），而 300 强度塔 DPS 约 173。
		// 先把现有塔升满再建新塔，确保每座塔尽快达到最大 DPS 贡献。
		//
		// 循环优先级：
		//   1. 有塔未满强度 → 批量升级最弱塔到满（DPS 最大化）
		//   2. 都满了且塔数不足 → 在未覆盖拐点建新 DPS 塔
		//   3. 都满了/都买不起 → 开波

		// 优先升级：有塔未满 → 批量升级最弱到满（花光所有金币）
		if !s.allTowersMaxed(state) {
			if ups := s.tryMultiUpgradeToMax(state); len(ups) > 0 {
				return ups
			}
		}

		// 所有塔已满或没钱升级 → 建新塔
		if s.canAfford(state) && len(state.BuildCells) > 0 {
			if action, ok := s.tryBuildPhase2(state); ok {
				return []Action{action}
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
		case nPhaseExpand:
			// Phase 2: 波中升级最弱塔（拉平整体实力）
			if action, ok := s.tryUpgradeWeakest(state); ok {
				return []Action{action}
			}
		default:
			// Phase 0: 波中升级最优目标
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

// pickBuildCell 用神经网络评分 + 领域知识选择最优建造位置。
//
// 编码的领域知识（在神经网络评分之上叠加）：
//   - Phase 0 第一座塔（CC）：强制放在第一个路径拐点附近（敌人转弯即被减速）
//   - Phase 0 第二座塔（DPS）：强制放在第一座塔 100px 以内（CC+DPS 重叠 = 乘法伤害）
//   - Phase 2 扩张：拐点附近 3x 加成，远离拐点 0.5x 惩罚
func (s *NeuralStrategy) pickBuildCell(state *GameState, def *TowerDefInfo) (Cell, bool) {
	if len(state.BuildCells) == 0 {
		return Cell{}, false
	}

	// ── Phase 0 硬编码：CC 在第一拐点、DPS 紧贴 CC ──
	// 这两个位置永远是最优的，不需要神经网络学习。
	if s.phase == nPhaseSetup && len(s.pathBends) > 0 {
		if s.builtCount == 0 {
			// 第一座塔（CC）：选离第一个拐点最近的格子
			return s.pickCellNearPoint(state.BuildCells, s.pathBends[0].X, s.pathBends[0].Y)
		}
		if s.builtCount == 1 {
			// 第二座塔（DPS）：选离第一座塔最近且在 100px 以内的格子
			best, ok := s.pickCellNearPoint(state.BuildCells, s.firstTowerX, s.firstTowerY)
			if ok {
				dx := best.X - s.firstTowerX
				dy := best.Y - s.firstTowerY
				if dx*dx+dy*dy <= 100*100 {
					return best, true
				}
			}
			// 100px 内无候选 → 放宽到 150px
			return s.pickCellNearPointRadius(state.BuildCells, s.firstTowerX, s.firstTowerY, 150)
		}
	}

	// ── Phase 2+ 及后续塔：神经网络评分 + 拐点加成 ──

	existingTowers := s.toLearningTowers(state)

	progress := 0.0
	if state.MaxWaves > 0 {
		progress = float64(state.Wave) / float64(state.MaxWaves)
	}

	cellSize := 60
	if state.MapInfo != nil {
		cellSize = state.MapInfo.CellSize
	}

	candidates := state.BuildCells

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

		// ── 领域知识：拐点加成（chokepoint bonus） ──
		// 路径拐点是敌人停留最久的位置（减速转弯），
		// 在此放塔 = 每颗子弹命中更多次 = 等效 DPS 翻倍。
		if len(s.pathBends) > 0 {
			bendDist := s.distToNearestBend(c.X, c.Y)
			switch {
			case bendDist < 100:
				score *= 3.0 // 拐点 100px 内 = 3 倍价值
			case bendDist < 200:
				score *= 2.0 // 拐点 200px 内 = 2 倍价值
			default:
				score *= 0.5 // 远离拐点 = 半价（惩罚直线段放塔）
			}
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

// pickCellNearPoint 从候选格子中选出离目标点 (tx,ty) 最近的格子。
func (s *NeuralStrategy) pickCellNearPoint(cells []Cell, tx, ty float64) (Cell, bool) {
	if len(cells) == 0 {
		return Cell{}, false
	}
	best := cells[0]
	bestDist := math.Hypot(cells[0].X-tx, cells[0].Y-ty)
	for _, c := range cells[1:] {
		d := math.Hypot(c.X-tx, c.Y-ty)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	return best, true
}

// pickCellNearPointRadius 从候选格子中选出离目标点最近且在 radius 以内的格子。
// 无候选时回退到全局最近。
func (s *NeuralStrategy) pickCellNearPointRadius(cells []Cell, tx, ty, radius float64) (Cell, bool) {
	if len(cells) == 0 {
		return Cell{}, false
	}
	var best Cell
	bestDist := math.MaxFloat64
	found := false

	for _, c := range cells {
		d := math.Hypot(c.X-tx, c.Y-ty)
		if d <= radius && d < bestDist {
			bestDist = d
			best = c
			found = true
		}
	}
	if found {
		return best, true
	}
	// 回退：radius 内无候选，选全局最近
	return s.pickCellNearPoint(cells, tx, ty)
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

// ── Phase 2 专用建塔/升级 ──────────────────────────────────

// tryBuildPhase2 Phase 2 建塔：优先选 DPS 塔型，放在未覆盖拐点。
func (s *NeuralStrategy) tryBuildPhase2(state *GameState) (Action, bool) {
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return Action{}, false
	}
	if state.Tick-s.lastBuildTick < neuralBuildThrottle {
		return Action{}, false
	}

	// 1. 选塔型：Phase 2 优先 DPS 偏好列表
	towerDef := s.pickPhase2TowerType(state)
	if towerDef == nil || state.Gold < towerDef.Cost {
		return Action{}, false
	}

	// 2. 选位置：优先未覆盖拐点
	cell, ok := s.pickUncoveredBend(state)
	if !ok {
		// 无未覆盖拐点 → 退回常规选址
		cell, ok = s.pickBuildCell(state, towerDef)
		if !ok {
			return Action{}, false
		}
	}

	s.builtCount++
	s.lastBuildTick = state.Tick
	s.wavesSinceLastBuild = 0

	return Action{
		Type:     ActionBuild,
		TowerKey: towerDef.Key,
		Cell:     cell,
	}, true
}

// pickPhase2TowerType Phase 2 塔型选择：按 DPS 偏好列表依次选择。
//
// 优先级：cl_gatling > cl_railgun > cl_mortar > 其他任意可购买塔型。
// 跳过已建过的塔型（避免重复，确保阵容多样性）。
func (s *NeuralStrategy) pickPhase2TowerType(state *GameState) *TowerDefInfo {
	if !s.classicMode {
		// 非经典模式只有一种塔
		for i := range state.TowerDefs {
			if state.Gold >= state.TowerDefs[i].Cost {
				return &state.TowerDefs[i]
			}
		}
		return nil
	}

	// 已建塔型集合
	builtKeys := make(map[string]bool)
	for _, t := range state.Towers {
		builtKeys[t.Key] = true
	}

	// 按偏好列表选择未建过的 DPS 塔
	for _, key := range phase2DPSPreference {
		if builtKeys[key] {
			continue // 已有此型号，跳过
		}
		if idx, ok := s.towerDefMap[key]; ok {
			d := &state.TowerDefs[idx]
			if state.Gold >= d.Cost {
				return d
			}
		}
	}

	// 偏好列表耗尽 → 选任意未建过且买得起的塔
	for i := range state.TowerDefs {
		d := &state.TowerDefs[i]
		if builtKeys[d.Key] {
			continue
		}
		if state.Gold >= d.Cost {
			return d
		}
	}

	// 全建过 → 选最便宜的（允许重复）
	var cheapest *TowerDefInfo
	for i := range state.TowerDefs {
		d := &state.TowerDefs[i]
		if state.Gold >= d.Cost {
			if cheapest == nil || d.Cost < cheapest.Cost {
				cheapest = d
			}
		}
	}
	return cheapest
}

// pickUncoveredBend 选择没有被现有塔覆盖的拐点，返回最近的空建造格子。
//
// "覆盖"定义：拐点 neuralUncoveredBendRadius (150px) 内有已建塔。
// 优先选覆盖最少的拐点，以最大化全路径防御。
func (s *NeuralStrategy) pickUncoveredBend(state *GameState) (Cell, bool) {
	if len(s.pathBends) == 0 || len(state.BuildCells) == 0 {
		return Cell{}, false
	}

	// 找出未覆盖拐点
	var uncoveredBends []PathPoint
	for _, bend := range s.pathBends {
		covered := false
		for _, t := range state.Towers {
			if math.Hypot(t.X-bend.X, t.Y-bend.Y) < neuralUncoveredBendRadius {
				covered = true
				break
			}
		}
		if !covered {
			uncoveredBends = append(uncoveredBends, bend)
		}
	}

	if len(uncoveredBends) == 0 {
		// 全部拐点都已覆盖 → 选离所有塔最远的拐点（分散火力）
		bestDist := -1.0
		var farthest PathPoint
		for _, bend := range s.pathBends {
			minDist := math.MaxFloat64
			for _, t := range state.Towers {
				d := math.Hypot(t.X-bend.X, t.Y-bend.Y)
				if d < minDist {
					minDist = d
				}
			}
			if minDist > bestDist {
				bestDist = minDist
				farthest = bend
			}
		}
		return s.pickCellNearPoint(state.BuildCells, farthest.X, farthest.Y)
	}

	// 选第一个未覆盖拐点（路径顺序靠前 = 敌人先经过 = 更高价值）
	target := uncoveredBends[0]
	return s.pickCellNearPoint(state.BuildCells, target.X, target.Y)
}

// tryMultiUpgradeToMax Phase 2 专用批量升级：把最弱塔升到满（300），
// 然后切换到下一个最弱，直到金币耗尽。
//
// 与 tryMultiUpgrade 的区别：
//   - 每次迭代重新找最弱塔（因为上一轮可能已升满）
//   - 不受 10 次批次限制，充分利用所有金币
//   - 模拟金币和强度变化，确保循环正确终止
func (s *NeuralStrategy) tryMultiUpgradeToMax(state *GameState) []Action {
	if len(state.Towers) == 0 {
		return nil
	}
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	upgAmount := int(config.GlobalBalance().Tower.StrengthBuyAmount)
	if state.Gold < upgCost {
		return nil
	}

	// 构建塔强度快照（模拟升级过程中的强度变化）
	type towerSnap struct {
		row, col int
		str      int
	}
	snaps := make([]towerSnap, len(state.Towers))
	for i, t := range state.Towers {
		snaps[i] = towerSnap{row: t.Row, col: t.Col, str: t.Strength}
	}

	var actions []Action
	gold := state.Gold
	const maxBatch = 50 // Phase 2 允许大批量（花光所有金币）

	for range maxBatch {
		if gold < upgCost {
			break
		}

		// 找当前最弱且未满的塔
		weakestIdx := -1
		weakestStr := math.MaxInt32
		for i, snap := range snaps {
			if s.classicMode && snap.str >= neuralMaxTowerStr {
				continue
			}
			if snap.str < weakestStr {
				weakestStr = snap.str
				weakestIdx = i
			}
		}
		if weakestIdx < 0 {
			break // 所有塔已满级
		}

		actions = append(actions, Action{
			Type: ActionUpgrade,
			Row:  snaps[weakestIdx].row,
			Col:  snaps[weakestIdx].col,
		})
		gold -= upgCost
		snaps[weakestIdx].str += upgAmount
	}

	return actions
}

// shouldStartWave 判断是否应该开波（阶段感知，激进策略）。
//
// 核心原则：NEVER delay — 更快的波次 = 更多金币收入 = 更强防御。
// 一旦当前阶段的金币操作全部完成，立即开波。
//
// Phase 0: CC + DPS 都建好 → 立即开波
// Phase 1: gold < upgCost（花不起升级费） → 立即开波
// Phase 2: gold < towerCost AND gold < upgCost → 立即开波
func (s *NeuralStrategy) shouldStartWave(state *GameState) bool {
	if len(state.Towers) == 0 {
		return false
	}
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost

	switch s.phase {
	case nPhaseSetup:
		// Phase 0: 两座核心塔都建好 → 立即开波（不等升级）
		if s.ccBuilt && s.dpsBuilt {
			return true
		}
		// 买不起塔也只能开波
		if !s.canAfford(state) {
			return true
		}
		return false

	case nPhaseUpgrade:
		// Phase 1: 花不起升级费 → 开波赚钱
		if state.Gold < upgCost || s.carryMaxed {
			return true
		}
		return false

	case nPhaseExpand:
		// Phase 2: 既买不起塔也升不起级 → 开波
		canBuild := s.canAfford(state) && len(state.BuildCells) > 0
		canUpgrade := state.Gold >= upgCost && !s.allTowersMaxed(state)
		if !canBuild && !canUpgrade {
			return true
		}
		return false
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

// onWaveChange 波次变化时触发训练 reward（增强版领域知识 shaping）。
//
// 正向 reward（教导正确行为）：
//   - 存活一波: +2
//   - 完美波（零泄漏）: +3
//   - Boss 波存活: +5
//
// 负向 reward（惩罚错误行为）：
//   - wave > 1 但没建塔: -5（没在建设）
//   - wave > 2 且金币 > 150: -3（囤金不花）
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

	// ── 基础 reward: 波次结果 ──
	s.trainer.OnWaveEnd(learning.WaveStats{
		WaveNum:       s.lastWave,
		KillsThisWave: killsThisWave,
		LivesBefore:   s.lastLives,
		LivesAfter:    state.Lives,
	})

	// ── 正向 reward shaping ──

	// 存活一波: 额外 +2 reward（重复调用 OnWaveEnd 以叠加 reward）
	if livesLost == 0 && killsThisWave > 0 {
		s.trainer.OnWaveEnd(learning.WaveStats{
			WaveNum:       s.lastWave,
			KillsThisWave: killsThisWave,
			LivesBefore:   state.Lives,
			LivesAfter:    state.Lives,
		})
	}

	// 完美波奖励（零泄漏，叠加上面的存活奖励 = +3 总额外）
	if livesLost == 0 && killsThisWave > 0 {
		s.trainer.OnWaveEnd(learning.WaveStats{
			WaveNum:       s.lastWave,
			KillsThisWave: killsThisWave,
			LivesBefore:   state.Lives,
			LivesAfter:    state.Lives,
		})
	}

	// Boss 波存活奖励: 额外 +5（Boss 波是关键检验点）
	wasBossWave := s.bossEvery > 0 && s.lastWave > 0 && s.lastWave%s.bossEvery == 0
	if wasBossWave && livesLost == 0 {
		for range 3 {
			s.trainer.OnWaveEnd(learning.WaveStats{
				WaveNum:       s.lastWave,
				KillsThisWave: killsThisWave,
				LivesBefore:   state.Lives,
				LivesAfter:    state.Lives,
			})
		}
	}

	// ── 负向 reward shaping（惩罚错误行为加速收敛） ──

	// wave > 1 但没建塔: -5（AI 应该尽早建设）
	if s.lastWave > 1 && len(state.Towers) == 0 {
		// 伪造大量生命损失以产生负 reward
		for range 3 {
			s.trainer.OnWaveEnd(learning.WaveStats{
				WaveNum:       s.lastWave,
				KillsThisWave: 0,
				LivesBefore:   20,
				LivesAfter:    15,
			})
		}
	}

	// wave > 2 且金币 > 150: -3（囤金不花 = 浪费经济优势）
	if s.lastWave > 2 && state.Gold > 150 {
		for range 2 {
			s.trainer.OnWaveEnd(learning.WaveStats{
				WaveNum:       s.lastWave,
				KillsThisWave: 0,
				LivesBefore:   20,
				LivesAfter:    17,
			})
		}
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

// initPaths 初始化路径数据缓存并预计算路径拐点。
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

	// 预计算所有路径的拐点（chokepoints）——
	// 拐点是敌人必须减速转弯的位置，是塔防最高价值放置点。
	s.pathBends = nil
	for _, path := range s.allPaths {
		bends := findPathBends(path)
		s.pathBends = append(s.pathBends, bends...)
	}
}

// findPathBends 检测路径中方向变化超过 30° 的拐点。
// 算法与 competent strategy 的 findCorners 一致：计算相邻线段的夹角。
func findPathBends(wps []PathPoint) []PathPoint {
	if len(wps) < 3 {
		return nil
	}
	const angleThresh = 30.0 * math.Pi / 180.0
	var bends []PathPoint

	for i := 1; i < len(wps)-1; i++ {
		// 向量: prev→curr, curr→next
		dx1 := wps[i].X - wps[i-1].X
		dy1 := wps[i].Y - wps[i-1].Y
		dx2 := wps[i+1].X - wps[i].X
		dy2 := wps[i+1].Y - wps[i].Y

		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
		if mag1 < 1e-6 || mag2 < 1e-6 {
			continue
		}

		dot := dx1*dx2 + dy1*dy2
		cosAngle := dot / (mag1 * mag2)
		// clamp [-1, 1]
		if cosAngle > 1 {
			cosAngle = 1
		}
		if cosAngle < -1 {
			cosAngle = -1
		}
		angle := math.Acos(cosAngle)
		if angle > angleThresh {
			bends = append(bends, wps[i])
		}
	}
	return bends
}

// distToNearestBend 计算点 (x,y) 到最近拐点的距离。
// 无拐点时返回 math.MaxFloat64。
func (s *NeuralStrategy) distToNearestBend(x, y float64) float64 {
	best := math.MaxFloat64
	for _, b := range s.pathBends {
		d := math.Hypot(x-b.X, y-b.Y)
		if d < best {
			best = d
		}
	}
	return best
}

// nearestBend 返回离 (x,y) 最近的拐点，以及距离。
// 无拐点时返回零值和 MaxFloat64。
func (s *NeuralStrategy) nearestBend(x, y float64) (PathPoint, float64) {
	best := math.MaxFloat64
	var pt PathPoint
	for _, b := range s.pathBends {
		d := math.Hypot(x-b.X, y-b.Y)
		if d < best {
			best = d
			pt = b
		}
	}
	return pt, best
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
