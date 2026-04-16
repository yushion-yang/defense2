// decision.go — AI 决策引擎。
//
// 基于评分系统做造塔/升级/开波/选战灵决策。
// 位置评分综合考虑路径距离、覆盖范围、塔间距、随机噪声。
// 升级评分综合考虑击杀效率、位置重要性、强度提升空间。
// 个性系统影响决策偏好、经济策略、冒险程度。
package aiplayer

import (
	"math"
	"math/rand"
)

// DecisionType 决策类型。
type DecisionType int

const (
	DecisionIdle          DecisionType = iota // 什么都不做
	DecisionBuild                             // 造塔
	DecisionUpgrade                           // 升级
	DecisionStartWave                         // 开始下一波
	DecisionSelectWarden                      // 选择战灵
	DecisionChooseAbility                     // 选择能力（3 选 1）
	DecisionSell                              // 卖塔
	DecisionUseItem                           // 使用道具
	DecisionUnlockSlot                        // 付费解锁能力槽
)

// Decision 决策结果。
type Decision struct {
	Type      DecisionType
	Row, Col  int     // 目标格子
	TowerKey  string  // 塔类型（Build 时使用）/ 战灵键名（SelectWarden 时使用）
	CenterX   float64 // 目标像素中心
	CenterY   float64
	WardenKey   string // 战灵类型键名（SelectWarden 时使用）
	AbilityName string // ChooseAbility: 选中的能力 ID
	SlotIndex   int    // ChooseAbility: 对应的槽位类别索引
	ItemKind    int    // UseItem: 道具类型（item.Kind 枚举值）
	Reason      string // 策略原因（用于气泡文案，由局势感知系统生成）
}

// ── 快照数据类型 ──

// AISnapshot AI 可见的游戏状态快照（纯值，零 core 依赖）。
type AISnapshot struct {
	Gold       int
	Wave       int
	MaxWaves   int
	WaveActive bool
	Lives      int
	Towers     []AITower
	BuildCells []AICell
	TowerDefs  []AITowerDef
	Enemies    []AIEnemy
	PathPoints []AIPathPoint
	MapCenterX float64
	MapCenterY float64
	WardenReady bool // 战灵是否已选择

	// ── Phase 2-3: 人类玩家数据（观战评论用）──
	HumanTowerCount int // 人类玩家的塔数量
	HumanGold       int // 人类玩家的金币
	TotalKills      int // 所有塔的总击杀数（含人类 + AI）

	// ── AI 道具背包 ──
	Items []AIItem

	// ── 付费解锁能力槽 ──
	UnlockCost int // 下一个能力槽解锁费用（0=不可解锁）

	// ── 波次信息 ──
	NextWaveIsBoss bool // 下一波是否有 Boss

	// ── 协作意识：全量塔数据（含人类 + AI，未按 Owner 过滤）──
	AllTowers []AITower
}

// AITower 塔快照。
type AITower struct {
	Row, Col     int
	Damage       float64
	Strength     int
	AttackSpeed  float64
	Range        float64
	Kills        int
	Cost         int              // 塔总投资（含升级费用）
	Owner        int              // 塔所有者 ID
	Abilities    []string         // 该塔已拥有的能力名称列表
	PendingSlots []AIPendingSlot  // 待选能力槽位（有候选选项的）
	CanUnlock    bool             // 是否可以付费解锁新能力槽
	UnlockCost   int              // 解锁下一个槽位的费用
}

// AIPendingSlot 一个待选的能力槽位。
type AIPendingSlot struct {
	SlotIndex int               // 类别索引（0=attack, 1=cc, 2=damage, 3=buff, 4=dot, 5=zone）
	Choices   []AIAbilityChoice // 候选能力列表（通常 3 个）
}

// AIAbilityChoice 单个候选能力。
type AIAbilityChoice struct {
	Name     string // 能力 ID（如 "scatter", "stun"）
	Label    string // 显示名称
	Category string // 所属类别（"attack"/"cc"/"damage"/"buff"/"dot"/"zone"）
}

// AIItem AI 可用道具快照。
type AIItem struct {
	Kind     int    // item.Kind 枚举值
	Name     string // 显示名称
	Category string // "baseDamage"/"potentialDamage"/"baseSpeed"/"potentialSpeed"/"baseRange"/"potentialRange"
	Count    int    // 持有数量
}

// AICell 可建造格子。
type AICell struct {
	Row, Col int
	X, Y     float64
}

// AITowerDef 可建造塔定义。
type AITowerDef struct {
	Key    string
	Cost   int
	Damage float64
	Range  float64
	Index  int
}

// AIEnemy 敌人快照。
type AIEnemy struct {
	X, Y      float64
	HP, MaxHP float64
	Speed     float64
	Boss      bool
	Active    bool
}

// AIPathPoint 路径点（像素坐标）。
type AIPathPoint struct {
	X, Y float64
}

// ── 决策引擎 ──

// DecisionEngine 决策引擎。
type DecisionEngine struct {
	strengthBuyCost int
	personality     Personality
	ownerID         int // AI 的 owner ID，用于协作分析时区分人类/AI 塔

	// 开波延迟计时器（避免波结束后立即开下一波）
	waveStartDelay float64

	// 卖塔节流：每 3 个决策周期才评估一次卖塔（避免频繁卖买）
	sellCycleCounter int

	// 局势感知结果（每次 Evaluate 更新，供外部读取）
	LatestAdvice StrategicAdvice
	LatestCoop   CoopAnalysis
}

// NewDecisionEngine 创建决策引擎。
func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{
		strengthBuyCost: 10,
		personality:     Personality{Aggression: 0.5, Economy: 0.5, Risk: 0.5, Reaction: 0.5, Compliance: 0.5},
	}
}

// SetOwnerID 设置 AI owner ID（协作分析用）。
func (e *DecisionEngine) SetOwnerID(id int) {
	e.ownerID = id
}

// SetStrengthBuyCost 设置升级花费。
func (e *DecisionEngine) SetStrengthBuyCost(cost int) {
	e.strengthBuyCost = cost
}

// SetPersonality 设置 AI 个性。
func (e *DecisionEngine) SetPersonality(p Personality) {
	e.personality = p
}

// 可选战灵列表。
var wardenOptions = []string{"prince", "core", "chain", "skystrike", "envoy"}

// Evaluate 评估当前状态，返回最佳决策。
//
// 决策优先级:
//   1. 战灵未选择 → 选战灵
//   2. 有待选能力 → 选能力（3 选 1）
//   3. 波间歇期 + 满足条件 → 开波
//   4. 根据进度和个性：造塔 / 升级 / idle
//   5. 12% 概率选次优选项（拟人化）
func (e *DecisionEngine) Evaluate(snap AISnapshot) Decision {
	// ── 战灵选择（最高优先级）──
	if !snap.WardenReady {
		key := wardenOptions[rand.Intn(len(wardenOptions))]
		return Decision{Type: DecisionSelectWarden, WardenKey: key}
	}

	// ── 能力选择（优先于经济决策，防止积压待选槽位）──
	if d, ok := e.pickAbility(snap); ok {
		return d
	}

	// ── 道具使用（优先于经济决策，道具即时价值高）──
	if d, ok := e.pickItemUse(snap); ok {
		return d
	}

	// ── 开波判定 ──
	if waveDecision, ok := e.tryStartWave(snap); ok {
		return waveDecision
	}

	// ── 局势感知 ──
	threats := AnalyzeThreats(snap.Enemies)
	gaps := DetectDefenseGaps(snap.Towers, threats, len(snap.PathPoints))
	advice := GenerateAdvice(gaps, threats, snap.Towers, e.personality)

	// ── 协作意识（分析全队配置，合并建议）──
	coop := AnalyzeCooperation(snap.AllTowers, e.ownerID)
	e.LatestCoop = coop
	advice = MergeCoopWithAdvice(coop, advice)
	e.LatestAdvice = advice

	// ── 卖塔评估（每 3 个周期评估一次，避免频繁操作）──
	e.sellCycleCounter++
	if e.sellCycleCounter >= 3 {
		e.sellCycleCounter = 0
		if d, ok := e.pickSell(snap); ok {
			return d
		}
	}

	// ── 付费解锁能力槽（余钱充足时主动解锁）──
	if d, ok := e.pickUnlockSlot(snap, advice); ok {
		return d
	}

	// ── 经济决策（造塔 / 升级，受局势+协作建议影响）──
	return e.pickEconomicAction(snap, advice)
}

// TickWaveDelay 每帧递减开波延迟。由 AIPlayer.Tick 调用。
func (e *DecisionEngine) TickWaveDelay(dt float64) {
	if e.waveStartDelay > 0 {
		e.waveStartDelay -= dt
	}
}

// tryStartWave 判定是否应该开始下一波。
//
// 条件：波不在进行中 + 没有存活敌人 + 至少有 1 座塔 + 延迟已过。
// 优化策略：
//   - 金币耗尽且无操作可做 → 缩短延迟抢 perfect bonus
//   - 下一波 Boss → 有余钱时先延迟升级
//   - 刚建完塔且有待选能力 → 延迟等选完
func (e *DecisionEngine) tryStartWave(snap AISnapshot) (Decision, bool) {
	if snap.WaveActive {
		return Decision{}, false
	}

	// 还有存活敌人，不开波
	for _, en := range snap.Enemies {
		if en.Active {
			return Decision{}, false
		}
	}

	// 至少需要 1 座塔
	if len(snap.Towers) == 0 {
		return Decision{}, false
	}

	// 有待选能力时延迟开波（等选完能力再开）
	for _, t := range snap.Towers {
		if len(t.PendingSlots) > 0 {
			return Decision{}, false
		}
	}

	// 个性影响的开波延迟
	if e.waveStartDelay > 0 {
		return Decision{}, false
	}

	// Boss 波准备：有余钱且可升级时延迟开波（给升级决策机会）
	if snap.NextWaveIsBoss && snap.Gold >= e.strengthBuyCost && len(snap.Towers) > 0 {
		// 仅在高 Economy 个性时延迟（冒险型直接开）
		if e.personality.Economy > 0.4 {
			e.waveStartDelay = 1.5 + rand.Float64()*1.0
			return Decision{}, false
		}
	}

	// 基础延迟：高 Aggression → 短延迟(1-2s)，低 Aggression → 长延迟(3-5s)
	baseDelay := 3.0 - e.personality.Aggression*2.0 // 1.0 ~ 3.0

	// 金币耗尽且无操作可做 → 立即开波抢 perfect bonus
	canDoAnything := false
	for _, d := range snap.TowerDefs {
		if d.Cost <= snap.Gold {
			canDoAnything = true
			break
		}
	}
	if !canDoAnything && snap.Gold < e.strengthBuyCost {
		// 没有经济操作可做，缩短延迟
		baseDelay = 0.5
	}

	e.waveStartDelay = baseDelay + rand.Float64()*1.5

	reason := ""
	if !canDoAnything && snap.Gold < e.strengthBuyCost {
		reason = "wave_early"
	} else if snap.NextWaveIsBoss {
		reason = "boss_prepare"
	}

	return Decision{Type: DecisionStartWave, Reason: reason}, true
}

// pickEconomicAction 根据经济状况、个性和局势建议选择造塔/升级/idle。
func (e *DecisionEngine) pickEconomicAction(snap AISnapshot, advice StrategicAdvice) Decision {
	progress := 0.0
	if snap.MaxWaves > 0 {
		progress = float64(snap.Wave) / float64(snap.MaxWaves)
	}

	// 金币保留缓冲：根据威胁等级和个性动态调整
	// 低威胁 → 保留 10%，中威胁 → 20%，高威胁 → 0%（全力投入），Boss 来 → 保留升级费
	reserveRatio := 0.20 * (1.0 - e.personality.Risk)
	if advice.Urgency > 0.8 {
		reserveRatio = 0 // 紧急状况下花光
	} else if advice.Urgency < 0.4 {
		reserveRatio = 0.10 // 低威胁少保留
	}
	// Boss 准备：保留至少一次升级费用
	if snap.NextWaveIsBoss && reserveRatio*float64(snap.Gold) < float64(e.strengthBuyCost) {
		reserveRatio = float64(e.strengthBuyCost) / float64(snap.Gold+1)
	}
	goldReserve := int(float64(snap.Gold) * reserveRatio)
	availableGold := snap.Gold - goldReserve
	if availableGold < 0 {
		availableGold = 0
	}

	canBuild := false
	for _, d := range snap.TowerDefs {
		if d.Cost <= availableGold {
			canBuild = true
			break
		}
	}
	canUpgrade := len(snap.Towers) > 0 && availableGold >= e.strengthBuyCost

	if !canBuild && !canUpgrade {
		return Decision{Type: DecisionIdle}
	}

	// 造塔 vs 升级的偏好。
	// 基础偏好：高 Economy → 偏升级，低 Economy → 偏造塔，高 Aggression → 偏造塔。
	buildPreference := 0.5 + e.personality.Aggression*0.2 - e.personality.Economy*0.2

	// 早期（<40%进度）更偏造塔，后期更偏升级
	if progress < 0.4 {
		buildPreference += 0.2
	} else {
		buildPreference -= 0.2
	}

	// 塔太少时强制偏造塔
	if len(snap.Towers) < 2 && canBuild && len(snap.BuildCells) > 0 {
		buildPreference = 0.9
	}

	// 局势建议影响偏好
	switch advice.Priority {
	case "build_cc", "build_dps", "build_aoe":
		// 建议造塔 → 提高建造偏好（urgency 越高影响越大）
		buildPreference += advice.Urgency * 0.3
	case "upgrade":
		// 建议升级 → 降低建造偏好
		buildPreference -= advice.Urgency * 0.3
	}
	// 钳制到合理范围
	if buildPreference < 0.1 {
		buildPreference = 0.1
	}
	if buildPreference > 0.95 {
		buildPreference = 0.95
	}

	preferBuild := rand.Float64() < buildPreference

	var bestDecision, secondDecision Decision

	if preferBuild && canBuild && len(snap.BuildCells) > 0 {
		bestDecision = e.pickBuild(snap, availableGold)
		bestDecision.Reason = advice.Reason
		if canUpgrade {
			secondDecision = e.pickUpgrade(snap)
			secondDecision.Reason = advice.Reason
		} else {
			secondDecision = bestDecision
		}
	} else if canUpgrade {
		bestDecision = e.pickUpgrade(snap)
		bestDecision.Reason = advice.Reason
		if canBuild && len(snap.BuildCells) > 0 {
			secondDecision = e.pickBuild(snap, availableGold)
			secondDecision.Reason = advice.Reason
		} else {
			secondDecision = bestDecision
		}
	} else if canBuild && len(snap.BuildCells) > 0 {
		bestDecision = e.pickBuild(snap, availableGold)
		bestDecision.Reason = advice.Reason
		secondDecision = bestDecision
	} else {
		return Decision{Type: DecisionIdle}
	}

	// 拟人化：12% 概率选次优选项
	if rand.Float64() < 0.12 {
		return secondDecision
	}
	return bestDecision
}

// ── 能力选择 ──

// pickAbility 检查是否有塔存在待选能力，如有则评分选择最佳能力。
//
// 策略：
//   - attack 类别（slot 0）：随机选择（决定塔身份，无明确好坏）
//   - cc 类别：防御型 AI（低 Aggression）偏好 +30
//   - damage 类别：进攻型 AI（高 Aggression）偏好 +30
//   - buff 类别：+20 基础分
//   - dot 类别：+25 基础分
//   - zone 类别：+20 基础分
//   - 每个候选加 ±10 随机噪声
//   - 12% 概率选次优（拟人化）
func (e *DecisionEngine) pickAbility(snap AISnapshot) (Decision, bool) {
	// 找第一个有待选槽位的塔
	for _, t := range snap.Towers {
		if len(t.PendingSlots) == 0 {
			continue
		}
		// 取第一个待选槽位
		slot := t.PendingSlots[0]
		if len(slot.Choices) == 0 {
			continue
		}

		// 对每个候选能力评分
		type scored struct {
			choice AIAbilityChoice
			score  float64
		}
		candidates := make([]scored, 0, len(slot.Choices))
		for _, c := range slot.Choices {
			score := e.scoreAbilityChoice(c)
			candidates = append(candidates, scored{choice: c, score: score})
		}

		// 找最高分和次高分
		best := candidates[0]
		second := candidates[0]
		for _, s := range candidates[1:] {
			if s.score > best.score {
				second = best
				best = s
			} else if s.score > second.score {
				second = s
			}
		}

		// 12% 概率选次优（拟人化）
		chosen := best
		if len(candidates) > 1 && rand.Float64() < 0.12 {
			chosen = second
		}

		return Decision{
			Type:        DecisionChooseAbility,
			Row:         t.Row,
			Col:         t.Col,
			AbilityName: chosen.choice.Name,
			SlotIndex:   slot.SlotIndex,
		}, true
	}
	return Decision{}, false
}

// scoreAbilityChoice 对单个候选能力评分（基于类别偏好 + 个性 + 噪声）。
func (e *DecisionEngine) scoreAbilityChoice(c AIAbilityChoice) float64 {
	var base float64
	switch c.Category {
	case "attack":
		// 攻击类别决定塔身份，无明确偏好，纯随机
		base = 50 + rand.Float64()*20
	case "cc":
		// 防御型 AI 偏好控制
		base = 20 + 30*(1.0-e.personality.Aggression)
	case "damage":
		// 进攻型 AI 偏好伤害
		base = 20 + 30*e.personality.Aggression
	case "buff", "aura":
		base = 20
	case "dot":
		base = 25
	case "zone":
		base = 20
	default:
		base = 15
	}
	// 随机噪声 ±10
	noise := (rand.Float64() - 0.5) * 20
	return base + noise
}

// ── 造塔评分 ──

// pickBuild 选择造塔位置和类型（评分系统）。
//
// 位置评分权重:
//   - 路径距离(40%): 离最近路径点越近越好
//   - 路径覆盖(30%): 射程内能覆盖多少路径点
//   - 塔间距(20%): 与现有塔保持距离以分散覆盖
//   - 随机噪声(10%): 拟人化
func (e *DecisionEngine) pickBuild(snap AISnapshot, availableGold int) Decision {
	// 选性价比最高的可负担塔
	var bestDef AITowerDef
	bestDefScore := -1.0
	for _, d := range snap.TowerDefs {
		if d.Cost > availableGold {
			continue
		}
		score := (d.Damage * d.Range) / float64(d.Cost)
		if score > bestDefScore {
			bestDefScore = score
			bestDef = d
		}
	}

	towerRange := bestDef.Range
	if towerRange <= 0 {
		towerRange = 100 // fallback
	}

	// 对每个可建造格子评分
	type scored struct {
		cell  AICell
		score float64
	}
	candidates := make([]scored, 0, len(snap.BuildCells))

	for _, c := range snap.BuildCells {
		score := e.scoreBuildCell(c, snap, towerRange)
		candidates = append(candidates, scored{cell: c, score: score})
	}

	if len(candidates) == 0 {
		// fallback: 不应该发生，但防御性处理
		c := snap.BuildCells[0]
		return Decision{
			Type: DecisionBuild, Row: c.Row, Col: c.Col,
			TowerKey: bestDef.Key, CenterX: c.X, CenterY: c.Y,
		}
	}

	// 找最高分和次高分
	best := candidates[0]
	second := candidates[0]
	for _, s := range candidates[1:] {
		if s.score > best.score {
			second = best
			best = s
		} else if s.score > second.score || second.cell == best.cell {
			second = s
		}
	}

	chosen := best.cell
	// 拟人化：偶尔选次优位置（已在上层 12% 处理，这里用最优）

	return Decision{
		Type:     DecisionBuild,
		Row:      chosen.Row,
		Col:      chosen.Col,
		TowerKey: bestDef.Key,
		CenterX:  chosen.X,
		CenterY:  chosen.Y,
	}
}

// scoreBuildCell 对单个建造格子评分。
//
// 评分维度（权重调整后更智能）:
//   - 路径覆盖(30%): 射程内能覆盖多少路径点 → DPS 投射效率
//   - 路径距离(20%): 离最近路径点越近越好
//   - 塔间距(15%): 与现有塔保持距离以分散覆盖
//   - 冗余惩罚(15%): 已有塔覆盖同区域则降分
//   - 协同加成(10%): 附近有 CC 塔则 DPS 塔额外加分
//   - 随机噪声(10%): 拟人化
func (e *DecisionEngine) scoreBuildCell(c AICell, snap AISnapshot, towerRange float64) float64 {
	// ── 1. 路径距离分(20%): 离最近路径点越近越好 ──
	pathDistScore := 0.0
	if len(snap.PathPoints) > 0 {
		minDist := math.MaxFloat64
		for _, p := range snap.PathPoints {
			d := math.Hypot(c.X-p.X, c.Y-p.Y)
			if d < minDist {
				minDist = d
			}
		}
		pathDistScore = math.Max(0, 1.0-minDist/500.0)
	} else {
		cx, cy := snap.MapCenterX, snap.MapCenterY
		if cx == 0 && cy == 0 {
			cx, cy = 600, 270
		}
		dist := math.Hypot(c.X-cx, c.Y-cy)
		pathDistScore = math.Max(0, 1.0-dist/400.0)
	}

	// ── 2. 路径覆盖分(30%): 射程内能覆盖多少路径点（DPS 投射效率）──
	coverageScore := 0.0
	if len(snap.PathPoints) > 0 {
		covered := 0
		for _, p := range snap.PathPoints {
			if math.Hypot(c.X-p.X, c.Y-p.Y) <= towerRange {
				covered++
			}
		}
		coverageScore = math.Min(1.0, float64(covered)/10.0)
	}

	// ── 3. 塔间距分(15%): 与现有塔保持距离 ──
	spreadScore := 1.0
	if len(snap.Towers) > 0 {
		minTowerDist := math.MaxFloat64
		for _, t := range snap.Towers {
			dr := float64(c.Row - t.Row)
			dc := float64(c.Col - t.Col)
			d := math.Sqrt(dr*dr + dc*dc)
			if d < minTowerDist {
				minTowerDist = d
			}
		}
		spreadScore = math.Min(1.0, minTowerDist/5.0)
	}

	// ── 4. 冗余惩罚(15%): 已有塔覆盖相同路径区域则降分 ──
	redundancyPenalty := 0.0
	if len(snap.PathPoints) > 0 && len(snap.Towers) > 0 {
		// 统计候选位置射程内的路径点中，已被现有塔覆盖的比例
		coveredByCandidate := 0
		coveredByBoth := 0
		for _, p := range snap.PathPoints {
			if math.Hypot(c.X-p.X, c.Y-p.Y) <= towerRange {
				coveredByCandidate++
				for _, t := range snap.Towers {
					if math.Hypot(float64(t.Col)*60+30-p.X, float64(t.Row)*60+30-p.Y) <= t.Range {
						coveredByBoth++
						break
					}
				}
			}
		}
		if coveredByCandidate > 0 {
			redundancyPenalty = float64(coveredByBoth) / float64(coveredByCandidate)
		}
	}
	redundancyScore := 1.0 - redundancyPenalty // 冗余越高分越低

	// ── 5. 协同加成(10%): 附近有 CC 塔则 DPS 位置更优 ──
	synergyScore := 0.0
	for _, t := range snap.Towers {
		dr := float64(c.Row - t.Row)
		dc := float64(c.Col - t.Col)
		gridDist := math.Sqrt(dr*dr + dc*dc)
		if gridDist <= 4 { // 4 格内视为"附近"
			for _, ab := range t.Abilities {
				if ccAbilities[ab] {
					// 附近有 CC 塔 → 新塔 DPS 效率更高（减速后敌人在射程内停留更久）
					synergyScore = 1.0
					break
				}
			}
		}
		if synergyScore > 0 {
			break
		}
	}

	// ── 6. 随机噪声(10%) ──
	noiseScore := rand.Float64()

	return pathDistScore*0.20 + coverageScore*0.30 + spreadScore*0.15 +
		redundancyScore*0.15 + synergyScore*0.10 + noiseScore*0.10
}

// ── 升级评分 ──

// pickUpgrade 选最佳 ROI 的塔升级（评分系统）。
//
// 评分维度:
//   - 击杀效率(40%): kills / (strength+1)，低效率塔升级收益更大
//   - 位置重要性(30%): 靠近路径的塔更值得投资
//   - 强度提升空间(30%): 强度低的塔升级边际收益更高
func (e *DecisionEngine) pickUpgrade(snap AISnapshot) Decision {
	type scored struct {
		tower AITower
		score float64
	}

	candidates := make([]scored, 0, len(snap.Towers))
	for _, t := range snap.Towers {
		score := e.scoreUpgradeTower(t, snap)
		candidates = append(candidates, scored{tower: t, score: score})
	}

	// 找最高分
	best := candidates[0]
	for _, s := range candidates[1:] {
		if s.score > best.score {
			best = s
		}
	}

	return Decision{
		Type: DecisionUpgrade,
		Row:  best.tower.Row,
		Col:  best.tower.Col,
	}
}

// scoreUpgradeTower 对单座塔评分。
//
// 评分维度（优化后更注重 ROI）：
//   - ROI 估算(25%): (预期伤害提升) / cost
//   - 路径覆盖重要性(25%): 覆盖路径点越多的塔升级价值越高（咽喉要塔）
//   - 收益递减惩罚(20%): 高 strength 塔每次升级边际收益低
//   - 击杀贡献(15%): 有击杀的塔证明位置有效
//   - 能力协同(15%): 有多能力的塔升级收益更高（能力效果受属性缩放）
func (e *DecisionEngine) scoreUpgradeTower(t AITower, snap AISnapshot) float64 {
	// ── 1. ROI 估算(25%): 升级后预期 DPS 提升 ──
	// 伤害公式: Base + Potential * (Strength/100)
	// 升级 +10 strength → DPS 变化 ≈ Potential * 10/100 * AttackSpeed
	// 粗略估算：damage_after ≈ damage * (strength+10) / strength
	str := float64(t.Strength)
	if str <= 0 {
		str = 100
	}
	expectedImprovement := t.Damage * 10.0 / str // 近似 DPS 提升
	cost := float64(e.strengthBuyCost)
	if cost <= 0 {
		cost = 10
	}
	roiScore := math.Min(1.0, expectedImprovement/cost*0.5)

	// ── 2. 路径覆盖重要性(25%): 咽喉塔优先升级 ──
	posScore := 0.0
	if len(snap.PathPoints) > 0 {
		tx := float64(t.Col)*60 + 30
		ty := float64(t.Row)*60 + 30
		covered := 0
		for _, p := range snap.PathPoints {
			if math.Hypot(tx-p.X, ty-p.Y) <= t.Range {
				covered++
			}
		}
		posScore = math.Min(1.0, float64(covered)/10.0)
	} else {
		posScore = 0.5
	}

	// ── 3. 收益递减惩罚(20%): 高 strength 塔升级价值低 ──
	// strength 100 → 1.0，300+ → 0.0
	headroom := math.Max(0, 1.0-float64(t.Strength-100)/200.0)

	// ── 4. 击杀贡献(15%): 有击杀证明位置有效 ──
	killScore := math.Min(1.0, float64(t.Kills)/10.0)

	// ── 5. 能力协同(15%): 能力越多升级收益越高（属性缩放效果更好）──
	abilityScore := math.Min(1.0, float64(len(t.Abilities))/4.0)

	return roiScore*0.25 + posScore*0.25 + headroom*0.20 + killScore*0.15 + abilityScore*0.15
}

// ForceBuild 强制执行造塔决策（被 LLM 战略覆盖调用）。
// 使用全部可用金币选择最佳位置和塔类型。
func (e *DecisionEngine) ForceBuild(snap AISnapshot) Decision {
	return e.pickBuild(snap, snap.Gold)
}

// ForceUpgrade 强制执行升级决策（被 LLM 战略覆盖调用）。
// 选择评分最高的塔进行升级。
func (e *DecisionEngine) ForceUpgrade(snap AISnapshot) Decision {
	return e.pickUpgrade(snap)
}

// ── 卖塔决策 ──

// sellWorthiness 卖塔价值评分阈值。超过此值才考虑卖塔。
const sellWorthiness = 0.6

// pickSell 评估是否有塔值得卖掉（低频决策）。
//
// 评分维度（sell worthiness）：
//   - 零击杀 3+ 波 → 高分（0 贡献）
//   - 低 DPS 效率 → 中分
//   - 无能力（wave 6+ 后）→ 加分
//   - 全部被其他塔覆盖 → 高分（位置冗余）
//
// 安全保底：至少保留 3 座塔（拆到更少会导致防线崩溃）。
func (e *DecisionEngine) pickSell(snap AISnapshot) (Decision, bool) {
	if len(snap.Towers) < 3 {
		return Decision{}, false
	}

	type scored struct {
		tower AITower
		score float64
	}
	var candidates []scored

	for _, t := range snap.Towers {
		score := 0.0

		// ── 零击杀 3+ 波 → 高分 ──
		wavesExisted := snap.Wave - 1 // 粗略估计（建塔早于当前波）
		if wavesExisted < 3 {
			wavesExisted = 3
		}
		if t.Kills == 0 && snap.Wave > 3 {
			score += 0.5
		}

		// ── 低 DPS 效率 ──
		as := t.AttackSpeed
		if as <= 0 {
			as = 1
		}
		dps := t.Damage * as
		towerCost := float64(t.Cost)
		if towerCost <= 0 {
			towerCost = 50
		}
		efficiency := dps / towerCost
		if efficiency < 0.5 {
			score += 0.3
		}

		// ── 无能力（wave 6+ 后仍无能力 → 表现差）──
		if len(t.Abilities) == 0 && snap.Wave > 6 {
			score += 0.2
		}

		// ── 路径覆盖冗余：该塔射程内的路径点全被其他塔覆盖 ──
		if len(snap.PathPoints) > 0 {
			tx := float64(t.Col)*60 + 30
			ty := float64(t.Row)*60 + 30
			coveredByThis := 0
			coveredByOthers := 0
			for _, p := range snap.PathPoints {
				if math.Hypot(tx-p.X, ty-p.Y) <= t.Range {
					coveredByThis++
					for _, other := range snap.Towers {
						if other.Row == t.Row && other.Col == t.Col {
							continue
						}
						ox := float64(other.Col)*60 + 30
						oy := float64(other.Row)*60 + 30
						if math.Hypot(ox-p.X, oy-p.Y) <= other.Range {
							coveredByOthers++
							break
						}
					}
				}
			}
			if coveredByThis > 0 && coveredByOthers == coveredByThis {
				// 该塔覆盖的路径点全被其他塔覆盖 → 位置冗余
				score += 0.3
			}
		}

		_ = wavesExisted // 已在上面使用
		candidates = append(candidates, scored{tower: t, score: score})
	}

	if len(candidates) == 0 {
		return Decision{}, false
	}

	// 找最高分
	best := candidates[0]
	for _, s := range candidates[1:] {
		if s.score > best.score {
			best = s
		}
	}

	if best.score < sellWorthiness {
		return Decision{}, false
	}

	return Decision{
		Type:   DecisionSell,
		Row:    best.tower.Row,
		Col:    best.tower.Col,
		Reason: "这塔没贡献，卖了重建",
	}, true
}

// ── 道具使用决策 ──

// pickItemUse 评估是否应使用道具。
//
// 策略：
//   - 无道具 → 跳过
//   - 对每种道具找最佳目标塔：
//     - Damage 类道具 → 分配给伤害最高的塔（乘法收益最大）
//     - Speed 类道具 → 分配给攻速最低的塔（提升最明显）
//     - Range 类道具 → 分配给射程最低的塔（覆盖改善最大）
//   - 至少有 2 座塔时才使用（避免太早浪费）
func (e *DecisionEngine) pickItemUse(snap AISnapshot) (Decision, bool) {
	if len(snap.Items) == 0 || len(snap.Towers) < 2 {
		return Decision{}, false
	}

	type itemTarget struct {
		itemKind int
		tower    AITower
		score    float64
	}
	var best itemTarget

	for _, it := range snap.Items {
		if it.Count <= 0 {
			continue
		}
		for _, t := range snap.Towers {
			score := e.scoreItemForTower(it, t)
			if score > best.score {
				best = itemTarget{itemKind: it.Kind, tower: t, score: score}
			}
		}
	}

	if best.score <= 0 {
		return Decision{}, false
	}

	return Decision{
		Type:     DecisionUseItem,
		Row:      best.tower.Row,
		Col:      best.tower.Col,
		ItemKind: best.itemKind,
		Reason:   "用个道具加强一下",
	}, true
}

// scoreItemForTower 评估道具对特定塔的价值。
// 核心逻辑：给"受益最大"的塔使用道具。
func (e *DecisionEngine) scoreItemForTower(it AIItem, t AITower) float64 {
	// 基础分：塔的击杀贡献（有贡献的塔值得强化）
	baseScore := math.Min(1.0, float64(t.Kills)/5.0+0.3)

	// 根据道具类别选择最佳目标
	var categoryScore float64
	switch it.Category {
	case "baseDamage", "potentialDamage":
		// Damage 类 → 伤害最高的塔（乘法收益）
		categoryScore = math.Min(1.0, t.Damage/50.0)
	case "baseSpeed", "potentialSpeed":
		// Speed 类 → 攻速最低的塔（提升空间大）
		if t.AttackSpeed > 0 {
			categoryScore = math.Max(0, 1.0-t.AttackSpeed/3.0)
		} else {
			categoryScore = 1.0
		}
	case "baseRange", "potentialRange":
		// Range 类 → 射程最低的塔（覆盖改善大）
		categoryScore = math.Max(0, 1.0-t.Range/200.0)
	default:
		categoryScore = 0.5
	}

	// 能力数量加成：有能力的塔使用道具价值更高
	abilityBonus := math.Min(0.3, float64(len(t.Abilities))*0.1)

	return baseScore*0.4 + categoryScore*0.4 + abilityBonus + rand.Float64()*0.1
}

// ── 付费解锁能力槽决策 ──

// pickUnlockSlot 评估是否应付费解锁某座塔的能力槽。
//
// 策略：
//   - 只在余钱充足时解锁（gold > unlockCost * 1.5）
//   - 优先解锁表现好的塔（高击杀、好位置）
//   - 高 urgency 时不花钱解锁（钱应该用于造塔/升级）
func (e *DecisionEngine) pickUnlockSlot(snap AISnapshot, advice StrategicAdvice) (Decision, bool) {
	// 紧急状况下不花钱解锁（优先造塔/升级）
	if advice.Urgency > 0.7 {
		return Decision{}, false
	}

	type candidate struct {
		tower AITower
		score float64
	}
	var candidates []candidate

	for _, t := range snap.Towers {
		if !t.CanUnlock || t.UnlockCost <= 0 {
			continue
		}
		// 余钱保证：解锁后至少还剩 50% 的金币（留给造塔/升级）
		if snap.Gold < int(float64(t.UnlockCost)*1.5) {
			continue
		}
		// 评分：击杀贡献 + 位置重要性
		killScore := math.Min(1.0, float64(t.Kills)/8.0)
		abilityRoom := 1.0 - float64(len(t.Abilities))/6.0 // 能力越少，解锁价值越高
		if abilityRoom < 0 {
			abilityRoom = 0
		}
		score := killScore*0.5 + abilityRoom*0.5
		candidates = append(candidates, candidate{tower: t, score: score})
	}

	if len(candidates) == 0 {
		return Decision{}, false
	}

	// 找最高分
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	// 最低门槛：评分太低不值得花钱
	if best.score < 0.3 {
		return Decision{}, false
	}

	return Decision{
		Type:   DecisionUnlockSlot,
		Row:    best.tower.Row,
		Col:    best.tower.Col,
		Reason: "解锁新能力",
	}, true
}
