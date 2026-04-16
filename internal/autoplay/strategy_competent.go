// strategy_competent.go — 仿真测试"合理玩家"策略。
//
// 目标：模拟有经验的玩家行为，实现人类玩家的 3 大关键协同：
//   1. 能力选择（slow + DPS 的乘法效应）
//   2. 资源集中到 1-2 座核心塔（carry 系统）
//   3. 火力交叉覆盖（chokepoint 重叠放塔）
//
// 路径感知放塔、分阶段经济管理、Boss 波意识、自主控制开波时机。
// 用于 simulation 模式下的数值平衡验证。
package autoplay

import (
	"math"
	"math/rand"
	"sort"

	"defense2/internal/config"
)

// ── 状态机阶段 ────────────────────────────────

type competentPhase int

const (
	phaseSetup    competentPhase = iota // 开波前初始建造
	phasePlaying                        // 常规阶段
	phaseDesperate                      // 低生命紧急阶段
)

// ── 经济常量 ──────────────────────────────────

const (
	goldReserve       = 10 // 波中保留的应急升级储备金
	nearAffordMargin  = 20 // 差多少金币就能再建一座时延迟开波
	sellCheckWave     = 5  // 从第几波开始检查卖塔
	bossEveryFallback = 4  // Boss 间隔默认值（配置读取失败时）
)

// ── 能力分配优先列表 ─────────────────────────────
// 按类别索引组织，每个 slice 内按优先级降序排列。
// 策略遍历候选列表，选第一个对应 slot 为空的能力。

var (
	// cat 0: 攻击方式 — scatter 覆盖面广适合大多数情况
	abilityPriorityAttack = []string{"scatter", "barrage", "spinAoe", "radial", "wideBeam"}
	// cat 1: CC — slowPower 是最关键的协同（减速 = 更长驻留 = 更多伤害）
	abilityPriorityCC = []string{"slowPower", "stunChance", "slowDuration"}
	// cat 2: 伤害 — crit 提供乘法伤害加成
	abilityPriorityDamage = []string{"crit", "splash", "flatDamage", "momentum"}
	// cat 3: 光环 — damageUpAura 对 chokepoint 内的塔群有乘法效应
	abilityPriorityBuff = []string{"damageUpAura", "attackSpeedAura", "rangeAura"}
	// cat 4: DoT — bleedDot 对高血量敌人持续输出
	abilityPriorityDoT = []string{"bleedDot", "burn", "poison"}
	// cat 5: 区域 — poisonZone 持续范围伤害
	abilityPriorityZone = []string{"poisonZone", "silenceZone", "weakenZone"}
)

// abilityPriorities 按波次解锁顺序排列的能力优先列表。
// 索引 0 = cat 0(攻击)，索引 1 = cat 1(CC)，...
// 策略根据 wavesCleared 推算已解锁 slot 数，逐 slot 分配。
var abilityPriorities = [6][]string{
	abilityPriorityAttack,
	abilityPriorityCC,
	abilityPriorityDamage,
	abilityPriorityBuff,
	abilityPriorityDoT,
	abilityPriorityZone,
}

// abilityAssignOrder 定义各 slot 分配给 carry 和 support 的顺序。
// carry 优先拿攻击 + 伤害 + CC；support 优先拿 CC + 攻击 + 光环。
var (
	carrySlotOrder   = []int{0, 2, 1, 4, 3, 5} // 攻击→伤害→CC→DoT→光环→区域
	supportSlotOrder = []int{0, 1, 3, 4, 2, 5}  // 攻击→CC→光环→DoT→伤害→区域
)

// ── 函数式选项 ──────────────────────────────────

// CompetentOpt 策略配置选项。
type CompetentOpt func(*CompetentStrategy)

// WithCompetentWarden 设置战灵类型。
func WithCompetentWarden(k string) CompetentOpt {
	return func(s *CompetentStrategy) { s.wardenKey = k }
}

// WithCompetentMaxTowers 设置最大建塔数（0=自动计算）。
func WithCompetentMaxTowers(n int) CompetentOpt {
	return func(s *CompetentStrategy) { s.maxTowers = n }
}

// WithCompetentAbilities 设置要分配的能力列表（空=不分配）。
func WithCompetentAbilities(names []string) CompetentOpt {
	return func(s *CompetentStrategy) { s.abilities = names }
}

// WithCompetentSeed 设置随机种子。
func WithCompetentSeed(seed int64) CompetentOpt {
	return func(s *CompetentStrategy) { s.seed = seed }
}

// ── 策略实现 ──────────────────────────────────

// CompetentStrategy 模拟有经验玩家的策略。
//
// 核心理念：
//   - 集中资源到 carry 塔（Potential 缩放的乘法效应）
//   - slow + DPS 在同一 chokepoint = 乘法伤害
//   - 主动分配能力（攻击→CC→伤害→光环→DoT→区域）
//   - Boss 波前攒钱、Boss 后激进扩张
//   - 卖掉无效塔回收金币
//   - 开波不磨蹭，但也不在准备不足时冒进
type CompetentStrategy struct {
	wardenKey  string
	maxTowers  int
	abilities  []string // 旧式手动能力列表（保留兼容，为空时启用自动分配）
	seed       int64
	rng        *rand.Rand
	phase      competentPhase
	initDone   bool
	startLives int // 初始生命值

	// 预计算数据
	placementOrder []Cell    // 按评分降序的建造位置
	placementScore []float64 // 对应评分
	buildIdx       int       // 下一个要建造的位置索引
	builtCount     int       // 已建造塔数
	abilityIdx     int       // 下一个要分配的能力索引（旧式）

	// ── carry 塔系统 ──
	// 第一座塔自动成为 carry；后续塔为 support。
	// carry 获得 70%+ 的升级金币，support 只在 carry 强度足够后才升级。
	carryRow    int
	carryCol    int
	hasCarry    bool
	carryStrCap int // carry 强度达到此值后才考虑升级 support（默认 300）

	// 节流
	lastBuildTick    int
	lastUpgradeTick  int
	lastSellTick     int
	lastAbilityTick  int // 能力分配节流（防止同帧多次请求）

	// Boss 波感知
	bossEvery int // Boss 出现间隔波数
}

// NewCompetentStrategy 创建合理玩家策略。
func NewCompetentStrategy(opts ...CompetentOpt) *CompetentStrategy {
	s := &CompetentStrategy{
		wardenKey: "prince",
		seed:      42,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *CompetentStrategy) Name() string { return "competent" }

func (s *CompetentStrategy) Init(state *GameState) {
	s.rng = rand.New(rand.NewSource(s.seed))
	s.startLives = state.Lives
	s.phase = phaseSetup
	s.initDone = false
	s.lastBuildTick = -100 // 允许首帧立即建造
	s.lastUpgradeTick = -100
	s.lastSellTick = -100
	s.lastAbilityTick = -100
	s.hasCarry = false
	s.carryStrCap = 300 // carry 强度达到 300 后才考虑升级 support

	// 从配置读取 Boss 间隔
	s.bossEvery = config.GlobalSpawnerConfig().Boss.EveryNWaves
	if s.bossEvery <= 0 {
		s.bossEvery = bossEveryFallback
	}

	// 如果 MapInfo 已可用（测试场景），立即初始化放塔排名
	if state.MapInfo != nil {
		s.initPlacement(state)
		s.initDone = true
	}
}

func (s *CompetentStrategy) Decide(state *GameState) []Action {
	if state.GameOver {
		return nil
	}

	// 延迟初始化：等待 MapInfo 可用
	if !s.initDone && state.MapInfo != nil {
		s.initPlacement(state)
		s.initDone = true
	}

	var actions []Action

	// 战灵选择
	if !state.WardenReady {
		return []Action{{Type: ActionSelectWarden, WardenKey: s.wardenKey}}
	}

	// 阶段判定
	s.updatePhase(state)

	switch s.phase {
	case phaseSetup:
		actions = s.decideSetup(state)
	case phasePlaying:
		actions = s.decidePlaying(state)
	case phaseDesperate:
		actions = s.decideDesperate(state)
	}

	return actions
}

// ── 阶段逻辑 ──────────────────────────────────

func (s *CompetentStrategy) updatePhase(state *GameState) {
	if s.phase == phaseSetup {
		// 建了至少 3 塔 或 金币不够再建且已有 2 塔 → 开始打
		minSetup := 3
		if s.builtCount >= minSetup {
			s.phase = phasePlaying
			return
		}
		// 至少有 2 塔且无法再建时也进入 playing
		if s.builtCount >= 2 && (len(state.BuildCells) == 0 || !s.canAffordBuild(state)) {
			s.phase = phasePlaying
		}
		return
	}
	// 生命低于 30% → 紧急模式
	threshold := float64(s.startLives) * 0.3
	if s.startLives > 0 && float64(state.Lives) <= threshold {
		s.phase = phaseDesperate
	} else {
		s.phase = phasePlaying
	}
}

func (s *CompetentStrategy) decideSetup(state *GameState) []Action {
	// Setup 阶段：建塔优先，同时尽早分配能力（免费操作）
	s.updateCarry(state)

	// 给已建好的塔分配能力（不消耗金币，越早越好）
	if abilActions := s.tryAutoAbilities(state); len(abilActions) > 0 {
		return abilActions
	}

	if action, ok := s.tryBuild(state); ok {
		return []Action{action}
	}
	// 建不了了 → 直接开波（phase 会在下帧切换）
	s.phase = phasePlaying
	return s.decidePlaying(state)
}

func (s *CompetentStrategy) decidePlaying(state *GameState) []Action {
	var actions []Action
	target := s.targetTowers(state)
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	nextWave := state.Wave + 1
	preBoss := s.isPreBossWave(nextWave)

	// 每帧都尝试维护 carry 标记
	s.updateCarry(state)

	if !state.WaveActive {
		// ── 波间歇期决策 ──

		// 0. 能力分配优先（免费，不消耗金币，必须尽早执行）
		if abilActions := s.tryAutoAbilities(state); len(abilActions) > 0 {
			return abilActions
		}

		// 1. 卖掉无效塔（wave 5+ 且有 0 击杀塔）
		if state.Wave >= sellCheckWave {
			if action, ok := s.trySell(state); ok {
				actions = append(actions, action)
				return actions
			}
		}

		// 2. Boss 前策略：不建新塔，集中升级 carry
		if preBoss {
			if state.Gold >= upgCost && len(state.Towers) > 0 {
				if action, ok := s.tryUpgrade(state, false); ok {
					actions = append(actions, action)
					return actions
				}
			}
			// 旧式能力分配（兼容）
			if action, ok := s.tryAssignAbility(state); ok {
				actions = append(actions, action)
				return actions
			}
		} else {
			// 非 Boss 前：建塔优先，然后升级
			if s.builtCount < target && s.canAffordBuild(state) {
				if action, ok := s.tryBuild(state); ok {
					actions = append(actions, action)
					return actions
				}
			}
			if state.Gold >= upgCost && len(state.Towers) > 0 {
				if action, ok := s.tryUpgrade(state, false); ok {
					actions = append(actions, action)
					return actions
				}
			}
			// 旧式能力分配（兼容）
			if action, ok := s.tryAssignAbility(state); ok {
				actions = append(actions, action)
				return actions
			}
		}

		// 3. 开波时机判断
		if state.Wave < state.MaxWaves {
			if s.shouldStartWave(state) {
				actions = append(actions, Action{Type: ActionStartWave})
			}
		}
	} else {
		// ── 波进行中 ──

		// 能力分配（免费，波中也可以做）
		if abilActions := s.tryAutoAbilities(state); len(abilActions) > 0 {
			return abilActions
		}

		// 升级当前正在攻击的 carry（保留 goldReserve 应急）
		if state.Gold >= upgCost+goldReserve {
			if action, ok := s.tryUpgrade(state, true); ok {
				actions = append(actions, action)
			}
		}
	}

	return actions
}

func (s *CompetentStrategy) decideDesperate(state *GameState) []Action {
	var actions []Action
	upgCost := config.GlobalBalance().Tower.StrengthBuyCost
	s.updateCarry(state)

	if !state.WaveActive {
		// 紧急模式下也要尽量分配能力
		if abilActions := s.tryAutoAbilities(state); len(abilActions) > 0 {
			return abilActions
		}

		// 紧急模式：升级已验证的塔（最多击杀）> 建新塔 > 开波
		if state.Gold >= upgCost && len(state.Towers) > 0 {
			if action, ok := s.tryUpgrade(state, false); ok {
				actions = append(actions, action)
				return actions
			}
		}
		// 卖掉无效塔
		if action, ok := s.trySell(state); ok {
			actions = append(actions, action)
			return actions
		}
		// 金币足够且有好位置 → 建塔
		if s.canAffordBuild(state) {
			if action, ok := s.tryBuild(state); ok {
				actions = append(actions, action)
				return actions
			}
		}
		// 不得不开波
		if state.Wave < state.MaxWaves {
			actions = append(actions, Action{Type: ActionStartWave})
		}
	} else {
		// 战斗中也尝试分配能力
		if abilActions := s.tryAutoAbilities(state); len(abilActions) > 0 {
			return abilActions
		}
		// 战斗中：有金就升级，不留储备
		if state.Gold >= upgCost {
			if action, ok := s.tryUpgrade(state, true); ok {
				actions = append(actions, action)
			}
		}
	}

	return actions
}

// ── 操作尝试 ──────────────────────────────────

func (s *CompetentStrategy) tryBuild(state *GameState) (Action, bool) {
	if len(state.BuildCells) == 0 || len(state.TowerDefs) == 0 {
		return Action{}, false
	}
	if s.maxTowers > 0 && s.builtCount >= s.maxTowers {
		return Action{}, false
	}
	// 节流：每 10 tick 最多建一座
	if state.Tick-s.lastBuildTick < 10 {
		return Action{}, false
	}

	// 选塔：性价比最高的
	bestDef := s.pickBestTowerDef(state)
	if bestDef == nil || state.Gold < bestDef.Cost {
		return Action{}, false
	}

	// 选位置：carry 建后，后续塔优先在 carry 附近（chokepoint 火力重叠）
	var cell Cell
	var found bool
	if s.hasCarry && s.builtCount >= 1 {
		cell, found = s.pickNearCarryCell(state)
	}
	if !found {
		cell, found = s.pickBestAvailableCell(state)
	}
	if !found {
		return Action{}, false
	}

	s.builtCount++
	s.buildIdx++
	s.lastBuildTick = state.Tick

	// 第一座塔自动成为 carry
	if !s.hasCarry {
		s.carryRow = cell.Row
		s.carryCol = cell.Col
		s.hasCarry = true
	}

	return Action{
		Type:     ActionBuild,
		TowerKey: bestDef.Key,
		Cell:     cell,
	}, true
}

// tryUpgrade 选择最值得升级的塔。
//
// 核心原则：carry 系统 — 资源高度集中到 carry 塔。
//   - carry str < carryStrCap(300) 时：100% 升级 carry
//   - carry str >= carryStrCap 后：按效能评分选最高分塔
//   - 波中：优先升级正在攻击的 carry / 最强塔
//   - 紧急模式：按击杀数选（已验证的表现者）
func (s *CompetentStrategy) tryUpgrade(state *GameState, preferActive bool) (Action, bool) {
	if len(state.Towers) == 0 {
		return Action{}, false
	}
	// 节流：每 5 tick 最多升一次
	if state.Tick-s.lastUpgradeTick < 5 {
		return Action{}, false
	}

	desperate := s.phase == phaseDesperate

	// carry 集中升级：carry 强度未达上限时，只升 carry
	if s.hasCarry && !desperate {
		carry := s.findCarry(state)
		if carry != nil && carry.Strength < s.carryStrCap {
			// 波中模式下，carry 必须有目标才升级
			if !preferActive || carry.HasTarget {
				s.lastUpgradeTick = state.Tick
				return Action{Type: ActionUpgrade, Row: carry.Row, Col: carry.Col}, true
			}
		}
	}

	// carry 已满或紧急模式 → 按效能评分选最佳塔
	var best *TowerInfo
	bestScore := -1.0

	for i := range state.Towers {
		t := &state.Towers[i]
		if preferActive && !t.HasTarget {
			continue
		}
		score := s.towerEffectivenessScore(t, desperate)
		// carry 塔始终有轻微加分（打破平局）
		if s.isCarry(t) {
			score *= 1.2
		}
		if score > bestScore {
			bestScore = score
			best = t
		}
	}

	// 如果 preferActive 没找到有目标的塔，退而求其次选所有塔中最强的
	if best == nil && preferActive {
		for i := range state.Towers {
			t := &state.Towers[i]
			score := s.towerEffectivenessScore(t, desperate)
			if s.isCarry(t) {
				score *= 1.2
			}
			if score > bestScore {
				bestScore = score
				best = t
			}
		}
	}
	if best == nil {
		return Action{}, false
	}

	s.lastUpgradeTick = state.Tick
	return Action{
		Type: ActionUpgrade,
		Row:  best.Row,
		Col:  best.Col,
	}, true
}

// trySell 卖掉无效塔（0 击杀、波次 >= sellCheckWave）。
// 卖出后 builtCount 减一，让后续帧可以在更好位置重建。
func (s *CompetentStrategy) trySell(state *GameState) (Action, bool) {
	if len(state.Towers) <= 1 {
		return Action{}, false // 至少保留 1 座塔
	}
	// 节流：每 60 tick 最多卖一座
	if state.Tick-s.lastSellTick < 60 {
		return Action{}, false
	}

	// 找 0 击杀且位置评分最低的塔（永不卖 carry）
	var worst *TowerInfo
	worstPosScore := math.MaxFloat64

	for i := range state.Towers {
		t := &state.Towers[i]
		if t.Kills > 0 {
			continue // 有击杀的留着
		}
		if s.isCarry(t) {
			continue // 永不卖 carry
		}
		posScore := s.cellPlacementScore(t.Row, t.Col)
		if posScore < worstPosScore {
			worstPosScore = posScore
			worst = t
		}
	}

	if worst == nil {
		return Action{}, false
	}

	s.lastSellTick = state.Tick
	s.builtCount--
	if s.builtCount < 0 {
		s.builtCount = 0
	}
	return Action{
		Type: ActionSell,
		Row:  worst.Row,
		Col:  worst.Col,
	}, true
}

func (s *CompetentStrategy) tryAssignAbility(state *GameState) (Action, bool) {
	if len(s.abilities) == 0 || s.abilityIdx >= len(s.abilities) {
		return Action{}, false
	}
	if len(state.Towers) == 0 {
		return Action{}, false
	}

	// 轮流给每座塔分配能力
	towerIdx := s.abilityIdx % len(state.Towers)
	t := &state.Towers[towerIdx]
	abilityName := s.abilities[s.abilityIdx]
	s.abilityIdx++

	return Action{
		Type:        ActionAddAbility,
		Row:         t.Row,
		Col:         t.Col,
		AbilityName: abilityName,
	}, true
}

// ── carry 塔系统 ──────────────────────────────

// updateCarry 维护 carry 塔标记。
// 如果 carry 塔被卖掉（不在 Towers 列表中），重新选择击杀最多的塔作为 carry。
func (s *CompetentStrategy) updateCarry(state *GameState) {
	if !s.hasCarry || len(state.Towers) == 0 {
		return
	}
	// 检查 carry 是否还存在
	if s.findCarry(state) != nil {
		return
	}
	// carry 被卖了 → 重新选择击杀最多的塔
	var best *TowerInfo
	bestKills := -1
	for i := range state.Towers {
		t := &state.Towers[i]
		if t.Kills > bestKills {
			bestKills = t.Kills
			best = t
		}
	}
	if best != nil {
		s.carryRow = best.Row
		s.carryCol = best.Col
	}
}

// findCarry 在当前塔列表中找到 carry 塔，找不到返回 nil。
func (s *CompetentStrategy) findCarry(state *GameState) *TowerInfo {
	for i := range state.Towers {
		if state.Towers[i].Row == s.carryRow && state.Towers[i].Col == s.carryCol {
			return &state.Towers[i]
		}
	}
	return nil
}

// isCarry 判断塔是否是 carry。
func (s *CompetentStrategy) isCarry(t *TowerInfo) bool {
	return s.hasCarry && t.Row == s.carryRow && t.Col == s.carryCol
}

// pickNearCarryCell 在 carry 附近（200px 内）找到路径评分最高的可建位置。
// 这实现了 chokepoint 火力重叠策略：多塔射程重叠在同一 kill zone。
func (s *CompetentStrategy) pickNearCarryCell(state *GameState) (Cell, bool) {
	if !s.hasCarry || len(state.BuildCells) == 0 {
		return Cell{}, false
	}

	// 找到 carry 的像素位置
	var carryX, carryY float64
	carryFound := false
	for i := range state.Towers {
		if state.Towers[i].Row == s.carryRow && state.Towers[i].Col == s.carryCol {
			carryX = state.Towers[i].X
			carryY = state.Towers[i].Y
			carryFound = true
			break
		}
	}
	if !carryFound {
		return Cell{}, false
	}

	// 在 carry 附近 200px 内找路径评分最高的格子
	const maxDist = 200.0
	var bestCell Cell
	bestScore := -1.0
	found := false

	for _, c := range state.BuildCells {
		dist := math.Hypot(c.X-carryX, c.Y-carryY)
		if dist > maxDist {
			continue
		}
		// 路径评分 * 距离权重（越近越好，但不能太近以免浪费覆盖面）
		pathScore := s.cellPlacementScore(c.Row, c.Col)
		// 距离 60-120px 是理想范围（射程重叠但不完全重合）
		distWeight := 1.0
		if dist < 60 {
			distWeight = 0.5 // 太近了
		} else if dist < 150 {
			distWeight = 1.5 // 理想距离
		}
		score := pathScore * distWeight
		if score > bestScore {
			bestScore = score
			bestCell = c
			found = true
		}
	}

	return bestCell, found
}

// ── 自动能力分配 ──────────────────────────────

// tryAutoAbilities 自动为需要能力的塔分配能力。
//
// 核心逻辑：
//   - 根据 wavesCleared 推算已解锁的 slot 数
//   - 遍历每座塔，检查已有能力数 vs 可解锁 slot 数
//   - carry 优先分配攻击/伤害/CC，support 优先分配 CC/攻击/光环
//   - 每次只返回 1 个 action（节流，避免爆发请求）
//
// 如果旧式 abilities 列表非空，跳过自动分配（保持兼容）。
func (s *CompetentStrategy) tryAutoAbilities(state *GameState) []Action {
	// 旧式手动列表存在时，不启用自动分配
	if len(s.abilities) > 0 {
		return nil
	}
	if len(state.Towers) == 0 {
		return nil
	}
	// 节流：每 8 tick 最多分配一个能力
	if state.Tick-s.lastAbilityTick < 8 {
		return nil
	}

	// 计算当前可解锁 slot 数（基于已完成的波次）
	wpu := config.GlobalBalance().Tower.WavesPerUnlock
	if wpu <= 0 {
		wpu = 2
	}
	maxSlots := 1 + state.WavesCleared/wpu
	if maxSlots > 6 {
		maxSlots = 6
	}

	// carry 优先分配
	if s.hasCarry {
		carry := s.findCarry(state)
		if carry != nil {
			if action, ok := s.pickAbilityForTower(carry, maxSlots, true); ok {
				s.lastAbilityTick = state.Tick
				return []Action{action}
			}
		}
	}

	// 然后 support 塔
	for i := range state.Towers {
		t := &state.Towers[i]
		if s.isCarry(t) {
			continue // carry 已处理
		}
		if action, ok := s.pickAbilityForTower(t, maxSlots, false); ok {
			s.lastAbilityTick = state.Tick
			return []Action{action}
		}
	}

	return nil
}

// pickAbilityForTower 为指定塔选择下一个应分配的能力。
//
// 逻辑：遍历 slot 分配顺序（carry/support 不同），
// 找到第一个 (1) 已解锁 (2) 尚未填充 的 slot，从对应优先列表中选第一个可用能力。
func (s *CompetentStrategy) pickAbilityForTower(t *TowerInfo, maxSlots int, isCarry bool) (Action, bool) {
	filledSlots := s.countFilledSlots(t)

	// 如果已有能力数 >= 可解锁 slot 数，无需分配
	if filledSlots >= maxSlots {
		return Action{}, false
	}

	// 根据角色选择 slot 分配顺序
	order := supportSlotOrder
	if isCarry {
		order = carrySlotOrder
	}

	// 构建已有能力集合（快速查重）
	abilitySet := make(map[string]bool, len(t.Abilities))
	for _, a := range t.Abilities {
		abilitySet[a] = true
	}

	// 遍历分配顺序，找第一个可填充的 slot
	for _, slotIdx := range order {
		if slotIdx >= maxSlots {
			continue // 该 slot 尚未解锁
		}
		// 检查该类别是否已有能力（通过遍历优先列表对比）
		slotFilled := false
		for _, candidate := range abilityPriorities[slotIdx] {
			if abilitySet[candidate] {
				slotFilled = true
				break
			}
		}
		if slotFilled {
			continue
		}

		// 该 slot 空闲 → 从优先列表中选第一个候选
		for _, candidate := range abilityPriorities[slotIdx] {
			return Action{
				Type:        ActionAddAbility,
				Row:         t.Row,
				Col:         t.Col,
				AbilityName: candidate,
			}, true
		}
	}

	return Action{}, false
}

// countFilledSlots 统计塔已填充的能力 slot 数（通过 Abilities 列表长度推断）。
func (s *CompetentStrategy) countFilledSlots(t *TowerInfo) int {
	return len(t.Abilities)
}

// ── 辅助函数 ──────────────────────────────────

func (s *CompetentStrategy) canAffordBuild(state *GameState) bool {
	if len(state.TowerDefs) == 0 {
		return false
	}
	cheapest := state.TowerDefs[0].Cost
	for _, d := range state.TowerDefs[1:] {
		if d.Cost < cheapest {
			cheapest = d.Cost
		}
	}
	return state.Gold >= cheapest
}

// cheapestTowerCost 返回最便宜的塔造价。
func (s *CompetentStrategy) cheapestTowerCost(state *GameState) int {
	if len(state.TowerDefs) == 0 {
		return math.MaxInt32
	}
	cheapest := state.TowerDefs[0].Cost
	for _, d := range state.TowerDefs[1:] {
		if d.Cost < cheapest {
			cheapest = d.Cost
		}
	}
	return cheapest
}

// targetTowers 根据波次计算目标塔数。
//
// 早期激进建塔（wave 1-3: 2~3 塔），中后期稳步扩张（3 + wave/4）。
// 上限为可用槽位的一半（留空间给位置优化）。
func (s *CompetentStrategy) targetTowers(state *GameState) int {
	maxBuild := len(state.BuildCells) + s.builtCount // 总可用位置
	halfSlots := maxBuild / 2
	if halfSlots < 3 {
		halfSlots = 3
	}

	var target int
	switch {
	case state.Wave <= 3:
		// 早期：2~3 塔，用好起始金
		target = 2 + (state.Wave+1)/2 // wave0=2, wave1=3, wave2=3, wave3=3
		if target > 3 {
			target = 3
		}
	default:
		// 中后期：更激进的扩张
		target = 3 + state.Wave/4
	}

	if s.maxTowers > 0 && target > s.maxTowers {
		target = s.maxTowers
	}
	if target > halfSlots {
		target = halfSlots
	}
	return target
}

// towerEffectivenessScore 计算塔的效能评分，用于升级优先级。
//
// 正常模式：damage * attackSpeed * (1 + kills*0.1)
//   — 高伤害高攻速的塔从升级中获益最多（Potential 乘法效应）
//   — kills 提供经验加权但不主导
//
// 紧急模式：纯击杀数（已验证的实战表现者）
func (s *CompetentStrategy) towerEffectivenessScore(t *TowerInfo, desperate bool) float64 {
	if desperate {
		// 紧急模式：最多击杀的塔是已验证的表现者
		return float64(t.Kills) + 0.001 // +0.001 避免全零时无法区分
	}
	// 正常模式：damage * attackSpeed，击杀数作为经验权重
	killsWeight := 1.0 + float64(t.Kills)*0.1
	return t.Damage * math.Max(t.AttackSpeed, 0.1) * killsWeight
}

// isPreBossWave 判断 nextWave 的前一波是否应该攒钱。
// Boss 出现在 wave % bossEvery == 0 的波次（如 4, 8, 12...）。
// 在 Boss 前一波（如 3, 7, 11...）节约金币，为 Boss 波准备。
func (s *CompetentStrategy) isPreBossWave(nextWave int) bool {
	if nextWave <= 0 {
		return false
	}
	return nextWave%s.bossEvery == 0
}

// shouldStartWave 判断是否应该开波。
//
// 不开波的情况：
//   - 没有任何塔
//   - 金币差一点就能再建一座（nearAffordMargin 内）
//
// 其他情况尽快开波（更快 = 更高 perfect bonus 概率）。
func (s *CompetentStrategy) shouldStartWave(state *GameState) bool {
	// 安全检查：没塔不开
	if len(state.Towers) == 0 {
		return false
	}

	// 快要够钱建新塔时延迟开波
	target := s.targetTowers(state)
	if s.builtCount < target {
		cheapest := s.cheapestTowerCost(state)
		deficit := cheapest - state.Gold
		if deficit > 0 && deficit <= nearAffordMargin {
			return false
		}
	}

	return true
}

// cellPlacementScore 查询格子在预计算排名中的评分。
// 用于 trySell 判断哪座塔的位置最差。
func (s *CompetentStrategy) cellPlacementScore(row, col int) float64 {
	for i, c := range s.placementOrder {
		if c.Row == row && c.Col == col {
			return s.placementScore[i]
		}
	}
	return 0 // 未在排名中 = 最低分
}

func (s *CompetentStrategy) pickBestTowerDef(state *GameState) *TowerDefInfo {
	if len(state.TowerDefs) == 0 {
		return nil
	}
	// 性价比 = (Damage * Range) / Cost
	var best *TowerDefInfo
	bestEff := -1.0
	for i := range state.TowerDefs {
		d := &state.TowerDefs[i]
		if d.Cost > state.Gold {
			continue
		}
		eff := (d.Damage * d.Range) / float64(d.Cost)
		if eff > bestEff {
			bestEff = eff
			best = d
		}
	}
	return best
}

func (s *CompetentStrategy) pickBestAvailableCell(state *GameState) (Cell, bool) {
	if len(state.BuildCells) == 0 {
		return Cell{}, false
	}

	// 如果有预计算排名 → 按排名找第一个仍可用的
	if len(s.placementOrder) > 0 {
		available := make(map[[2]int]Cell)
		for _, c := range state.BuildCells {
			available[[2]int{c.Row, c.Col}] = c
		}
		for _, ranked := range s.placementOrder {
			if c, ok := available[[2]int{ranked.Row, ranked.Col}]; ok {
				return c, true
			}
		}
	}

	// Fallback：返回第一个可用位置
	return state.BuildCells[0], true
}

// ── 路径分析与格子评分 ────────────────────────────

func (s *CompetentStrategy) initPlacement(state *GameState) {
	mi := state.MapInfo
	if mi == nil {
		return
	}

	// 收集所有路径
	var allPaths [][]PathPoint
	if mi.MultiPath && len(mi.Paths) > 0 {
		for _, p := range mi.Paths {
			allPaths = append(allPaths, p.Waypoints)
		}
	}
	if len(allPaths) == 0 && len(mi.Waypoints) > 0 {
		allPaths = [][]PathPoint{mi.Waypoints}
	}

	// 识别拐角
	var allCorners [][]bool
	for _, wps := range allPaths {
		allCorners = append(allCorners, findCorners(wps))
	}

	// 对所有可建造格子评分
	// 收集所有可建造位置（包括已建和未建的）
	allCells := make([]Cell, len(state.BuildCells))
	copy(allCells, state.BuildCells)
	for _, t := range state.Towers {
		allCells = append(allCells, Cell{Row: t.Row, Col: t.Col, X: t.X, Y: t.Y})
	}

	type scored struct {
		cell  Cell
		score float64
	}
	var entries []scored
	for _, c := range allCells {
		sc := scoreCell(c, allPaths, allCorners)
		entries = append(entries, scored{cell: c, score: sc})
	}

	// 按分数降序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	s.placementOrder = make([]Cell, len(entries))
	s.placementScore = make([]float64, len(entries))
	for i, e := range entries {
		s.placementOrder[i] = e.cell
		s.placementScore[i] = e.score
	}
}

// findCorners 识别路径上的拐角点（方向变化 > 30 度）。
func findCorners(wps []PathPoint) []bool {
	corners := make([]bool, len(wps))
	if len(wps) < 3 {
		return corners
	}
	const angleThresh = 30.0 * math.Pi / 180.0

	for i := 1; i < len(wps)-1; i++ {
		// 向量: prev->curr, curr->next
		dx1 := wps[i].X - wps[i-1].X
		dy1 := wps[i].Y - wps[i-1].Y
		dx2 := wps[i+1].X - wps[i].X
		dy2 := wps[i+1].Y - wps[i].Y

		// 两向量夹角
		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
		if mag1 < 1e-6 || mag2 < 1e-6 {
			continue
		}
		cosAngle := dot / (mag1 * mag2)
		// clamp to [-1, 1]
		if cosAngle > 1 {
			cosAngle = 1
		}
		if cosAngle < -1 {
			cosAngle = -1
		}
		angle := math.Acos(cosAngle)
		if angle > angleThresh {
			corners[i] = true
		}
	}
	return corners
}

// scoreCell 计算建造格子的路径亲和度评分。
// 拐角处权重 x2，多路径交汇处额外加分。
func scoreCell(c Cell, allPaths [][]PathPoint, allCorners [][]bool) float64 {
	const maxRange = 200.0 // 最大考虑距离
	score := 0.0
	pathsInRange := 0

	for pathIdx, wps := range allPaths {
		pathScore := 0.0
		for i, wp := range wps {
			dist := math.Hypot(c.X-wp.X, c.Y-wp.Y)
			if dist > maxRange {
				continue
			}
			proximity := 1.0 / math.Max(dist, 30.0) // 避免除以零尖峰
			cornerBonus := 1.0
			if pathIdx < len(allCorners) && i < len(allCorners[pathIdx]) && allCorners[pathIdx][i] {
				cornerBonus = 2.0 // 拐角处敌人停留更久
			}
			pathScore += proximity * cornerBonus
		}
		if pathScore > 0 {
			pathsInRange++
		}
		score += pathScore
	}

	// 多路径交汇奖励
	if pathsInRange > 1 {
		score *= 1.0 + 0.5*float64(pathsInRange-1)
	}

	return score
}
