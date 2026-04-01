// anomaly.go — 运行时异常检测器。
// 每帧检查游戏状态，识别敌人卡住、金币异常、FPS 下降等问题。
package autoplay

import (
	"fmt"
	"math"
)

// AnomalySeverity 异常严重程度。
type AnomalySeverity int

const (
	SeverityLow      AnomalySeverity = iota // 低
	SeverityMedium                          // 中
	SeverityHigh                            // 高
	SeverityCritical                        // 严重
)

// String 返回严重程度的字符串表示。
func (s AnomalySeverity) String() string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Anomaly 运行时异常记录。
type Anomaly struct {
	Tick     int             `json:"tick"`
	Type     string          `json:"type"`
	Detail   string          `json:"detail"`
	Severity AnomalySeverity `json:"severity"`
}

// AnomalyDetector 运行时异常检测器。
type AnomalyDetector struct {
	prevEnemyPos  map[int][2]float64 // enemyID -> (x,y)
	stuckCounters map[int]int        // enemyID -> 连续静止帧数
	prevGold      int
	prevLives     int
	prevKills     int
	slowFrames    int       // 连续慢帧计数
	anomalies     []Anomaly // 历史异常列表
	initialized   bool

	// 操作审计
	pendingBuildCheck *buildAudit     // 上一帧建塔操作待验证
	pendingUpgrCheck  *upgradeAudit   // 上一帧升级操作待验证
	prevWaveHP        map[int]float64 // wave -> 该波敌人平均 MaxHP（波次 HP 递增检查）
	wardenBoundsFlag  bool            // 已报告过战灵越界
	towerDamageAccum  map[string]int  // towerKey -> 存在帧数（DPS=0 检测）
	poolHighWater     int             // 敌人池历史最高计数

	// 去重标记（避免同类异常每 tick 重复输出）
	goldNegReported      bool         // 金币为负已报告（回正时重置）
	poolHighReported     bool         // 敌人池高占用已报告（降下来后重置）
	projOverflowReported bool         // 弹射物溢出已报告（降下来后重置）
	waveHPReportedWaves  map[int]bool // 已报告过 HP 回退的波次

	// 战灵活动范围追踪
	wardenMinX, wardenMaxX float64 // 历史访问过的 X 范围
	wardenMinY, wardenMaxY float64 // 历史访问过的 Y 范围
	wardenSamples          int     // 采样次数
	wardenCoverageReported bool    // 已报告过覆盖不足

	// ── 新增：回归检测状态 ──

	// wave_cleared_stall: 波次清除计数停滞
	waveClearedStallReported bool

	// archetype_monoculture: 原型多样性
	seenArchetypes           map[string]bool
	archetypeMonoReported    bool

	// enemy_hp_uniform: HP 一致性
	hpUniformReported map[int]bool // wave -> reported

	// ability_silent: 能力沉默
	towerAbilityTicks    map[string]int  // towerKey -> 战斗帧数（HasTarget=true 的帧）
	abilitySilentChecked bool            // 已做过检查

	// warden_origin_stuck: 战灵原点停留
	wardenReadyTick       int  // wardenReady 变 true 的 tick
	wardenOriginReported  bool

	// projectile_orphan_burst: 弹射物堆积无命中
	projBurstFrames  int // 连续"弹射物多但无击杀"帧数
	projBurstKillRef int // 进入监控时的击杀数

	// speed_floor_violation: 减速越界
	speedFloorReported map[int]bool // enemyID -> reported

	// nan_in_enemy: NaN 检测
	nanReported map[int]bool // enemyID -> reported

	// economy_stall: 经济断档
	econStallTicks    int  // 连续买不起帧数
	econStallMinCost  int  // 最便宜塔缓存
	econStallReported bool

	// boss_too_weak: Boss 存活时间过短
	activeBosses map[int]int // bossEnemyID -> spawnTick

	// interact_mode_stuck: 交互模式停滞
	imodeStuckMode  int // 当前追踪的模式
	imodeStuckTick  int // 进入该模式的 tick
	imodeStuckReported map[int]bool // mode -> reported

	// warden_teleport: 战灵帧间跳跃
	wardenPrevX, wardenPrevY float64
	wardenPosInit            bool
	wardenTeleportCount      int // 累计跳跃次数（防抖）

	// enemy_born_with_hit: 新生敌人带 hit 状态
	enemyFirstSeen map[int]bool // enemyID -> 已首次见过
}

type buildAudit struct {
	tick     int
	row, col int
	towerKey string
}

type upgradeAudit struct {
	tick     int
	row, col int
	prevStr  int
}

// stuckThreshold 判定敌人卡住的帧数阈值。
const stuckThreshold = 60

// NewAnomalyDetector 创建异常检测器。
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		prevEnemyPos:        make(map[int][2]float64),
		stuckCounters:       make(map[int]int),
		prevWaveHP:          make(map[int]float64),
		towerDamageAccum:    make(map[string]int),
		waveHPReportedWaves: make(map[int]bool),
		// 新增
		seenArchetypes:    make(map[string]bool),
		hpUniformReported: make(map[int]bool),
		towerAbilityTicks: make(map[string]int),
		speedFloorReported:  make(map[int]bool),
		nanReported:         make(map[int]bool),
		imodeStuckReported:  make(map[int]bool),
		enemyFirstSeen:      make(map[int]bool),
		activeBosses:        make(map[int]int),
	}
}

// Check 执行一帧的异常检测，返回本帧新发现的异常。
func (d *AnomalyDetector) Check(state *GameState, updateMs float64) []Anomaly {
	var found []Anomaly

	if !d.initialized {
		d.prevGold = state.Gold
		d.prevLives = state.Lives
		d.initialized = true
		return nil
	}

	// 1. 敌人卡住检测
	found = append(found, d.checkEnemyStuck(state)...)

	// 2. 金币为负（首次报告后不再重复，回正时重置）
	if state.Gold < 0 {
		if !d.goldNegReported {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "gold_negative",
				Detail:   fmt.Sprintf("gold=%d", state.Gold),
				Severity: SeverityCritical,
			})
			d.goldNegReported = true
		}
	} else {
		d.goldNegReported = false
	}

	// 3. 金币突增（>500/帧）
	goldDelta := state.Gold - d.prevGold
	if goldDelta > 500 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "gold_spike",
			Detail:   fmt.Sprintf("gold changed %d->%d (delta=%d)", d.prevGold, state.Gold, goldDelta),
			Severity: SeverityHigh,
		})
	}

	// 4. 生命骤降（>5/帧）
	livesDelta := d.prevLives - state.Lives
	if livesDelta > 5 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "lives_drop",
			Detail:   fmt.Sprintf("lives dropped %d->%d (delta=%d)", d.prevLives, state.Lives, livesDelta),
			Severity: SeverityMedium,
		})
	}

	// 5. FPS 下降（连续 10 帧 >50ms）
	if updateMs > 50 {
		d.slowFrames++
		if d.slowFrames >= 10 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "fps_drop",
				Detail:   fmt.Sprintf("update took %.1fms for %d consecutive frames", updateMs, d.slowFrames),
				Severity: SeverityMedium,
			})
			d.slowFrames = 0 // 重置避免重复报告
		}
	} else {
		d.slowFrames = 0
	}

	// 6. 塔在非建造位置（tower_orphan）— 需要更多上下文，暂通过 BuildCells 间接检查

	// 7. 死敌行走（HP<=0 仍 Active 且非 Dying）
	for _, e := range state.Enemies {
		if e.Active && !e.Dying && e.HP <= 0 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "dead_enemy_walking",
				Detail:   fmt.Sprintf("enemy#%d archetype=%s HP=%.1f active=true dying=false", e.ID, e.Archetype, e.HP),
				Severity: SeverityCritical,
			})
		}
	}

	// 8. 战灵越界检测（战灵超出地图像素范围）
	if state.WardenReady && !d.wardenBoundsFlag && state.MapPixelW > 0 {
		margin := 50.0 // 允许 50px 容差
		if state.WardenX < -margin || state.WardenX > state.MapPixelW+margin ||
			state.WardenY < -margin || state.WardenY > state.MapPixelH+margin {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "warden_out_of_bounds",
				Detail:   fmt.Sprintf("warden at (%.1f,%.1f), map bounds (%.0f,%.0f)", state.WardenX, state.WardenY, state.MapPixelW, state.MapPixelH),
				Severity: SeverityHigh,
			})
			d.wardenBoundsFlag = true // 只报一次
		}
	}

	// 8b. 战灵活动范围覆盖检测
	//     地图比一屏(1200x540)大时，战灵应该能到达全域。
	//     长时间只停留在原始一屏范围 = 活动范围被限制的 bug。
	found = append(found, d.checkWardenCoverage(state)...)

	// 9. 敌人池容量预警（>75% 占用，首次报告后不再重复，降到 75% 以下时重置）
	if state.EnemyPoolCount > d.poolHighWater {
		d.poolHighWater = state.EnemyPoolCount
	}
	if state.EnemyPoolCount > 192 { // 256 * 0.75
		if !d.poolHighReported {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "pool_high_usage",
				Detail:   fmt.Sprintf("enemy pool %d/256 (high water: %d)", state.EnemyPoolCount, d.poolHighWater),
				Severity: SeverityMedium,
			})
			d.poolHighReported = true
		}
	} else {
		d.poolHighReported = false
	}

	// 10. 弹射物泄漏（弹射物数 > 100，首次报告后不再重复，降下来后重置）
	if state.ProjectileCount > 100 {
		if !d.projOverflowReported {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "projectile_overflow",
				Detail:   fmt.Sprintf("projectile count=%d", state.ProjectileCount),
				Severity: SeverityMedium,
			})
			d.projOverflowReported = true
		}
	} else {
		d.projOverflowReported = false
	}

	// 11. 波次 HP 非递增检测（新波的平均 HP 比前一波低）
	found = append(found, d.checkWaveHPProgression(state)...)

	// 12. 操作结果验证
	found = append(found, d.checkBuildResult(state)...)
	found = append(found, d.checkUpgradeResult(state)...)

	// 13. 击杀守恒（击杀+存活+泄漏 随时间稳定）
	if state.TotalKills > 0 && d.prevKills > 0 {
		killDelta := state.TotalKills - d.prevKills
		if killDelta < 0 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "kill_count_decrease",
				Detail:   fmt.Sprintf("kills went from %d to %d", d.prevKills, state.TotalKills),
				Severity: SeverityCritical,
			})
		}
	}
	d.prevKills = state.TotalKills

	// ── 14. NaN/Inf 检测（最优先，数值污染会导致其他检测误报）──
	found = append(found, d.checkNaN(state)...)

	// ── 15. 减速越界检测 ──
	found = append(found, d.checkSpeedFloor(state)...)

	// ── 16. 战灵原点停留检测 ──
	found = append(found, d.checkWardenOrigin(state)...)

	// ── 17. 波次清除计数停滞检测 ──
	found = append(found, d.checkWaveClearedStall(state)...)

	// ── 18. 敌人原型多样性检测 ──
	found = append(found, d.checkArchetypeMonoculture(state)...)

	// ── 19. 同波敌人 HP 一致性检测 ──
	found = append(found, d.checkEnemyHPUniform(state)...)

	// ── 20. 弹射物堆积无命中检测 ──
	found = append(found, d.checkProjectileOrphan(state)...)

	// ── 21. 能力沉默检测 ──
	found = append(found, d.checkAbilitySilent(state)...)

	// ── 22. 交互模式停滞检测 ──
	found = append(found, d.checkInteractModeStuck(state)...)

	// ── 23. 战灵帧间跳跃检测 ──
	found = append(found, d.checkWardenTeleport(state)...)

	// ── 24. 新生敌人带 hit 状态检测 ──
	found = append(found, d.checkEnemyBornWithHit(state)...)

	// ── 25. 经济断档检测 ──
	found = append(found, d.checkEconomyStall(state)...)

	// ── 26. Boss 存活过短检测 ──
	found = append(found, d.checkBossTooWeak(state)...)

	// 更新前帧状态
	d.prevGold = state.Gold
	d.prevLives = state.Lives

	// 记录异常
	d.anomalies = append(d.anomalies, found...)
	return found
}

// checkWaveHPProgression 检查波次 HP 是否递增（每波只报告一次）。
func (d *AnomalyDetector) checkWaveHPProgression(state *GameState) []Anomaly {
	if state.Wave <= 1 || !state.WaveActive {
		return nil
	}

	// 已对本波报告过，跳过
	if d.waveHPReportedWaves[state.Wave] {
		return nil
	}

	// 计算当前波次敌人平均 MaxHP
	var totalHP float64
	var count int
	for _, e := range state.Enemies {
		if e.Active && !e.Dying {
			totalHP += e.MaxHP
			count++
		}
	}
	if count == 0 {
		return nil
	}
	avgHP := totalHP / float64(count)

	// 与前一波比较
	if prev, ok := d.prevWaveHP[state.Wave-1]; ok && avgHP > 0 && prev > 0 {
		if avgHP < prev*0.15 { // 比前一波低 85% 以上（考虑原型 hpScale 0.5~2.85 的方差）
			d.waveHPReportedWaves[state.Wave] = true
			return []Anomaly{{
				Tick:     state.Tick,
				Type:     "wave_hp_regression",
				Detail:   fmt.Sprintf("wave %d avgHP=%.0f < wave %d avgHP=%.0f", state.Wave, avgHP, state.Wave-1, prev),
				Severity: SeverityHigh,
			}}
		}
	}
	d.prevWaveHP[state.Wave] = avgHP
	return nil
}

// RecordBuildAction 记录建塔操作，下帧验证结果。
func (d *AnomalyDetector) RecordBuildAction(tick, row, col int, towerKey string) {
	d.pendingBuildCheck = &buildAudit{tick: tick, row: row, col: col, towerKey: towerKey}
}

// RecordUpgradeAction 记录升级操作，下帧验证结果。
func (d *AnomalyDetector) RecordUpgradeAction(tick, row, col, prevStrength int) {
	d.pendingUpgrCheck = &upgradeAudit{tick: tick, row: row, col: col, prevStr: prevStrength}
}

// checkBuildResult 验证建塔操作是否生效。
func (d *AnomalyDetector) checkBuildResult(state *GameState) []Anomaly {
	if d.pendingBuildCheck == nil {
		return nil
	}
	audit := d.pendingBuildCheck
	d.pendingBuildCheck = nil

	// 检查塔是否出现在预期位置
	for _, t := range state.Towers {
		if t.Row == audit.row && t.Col == audit.col && t.Key == audit.towerKey {
			return nil // 成功
		}
	}
	// 可能是金币不足导致的正常失败，降级为 LOW
	return []Anomaly{{
		Tick:     state.Tick,
		Type:     "build_silent_fail",
		Detail:   fmt.Sprintf("build %s at (%d,%d) tick=%d not found in towers", audit.towerKey, audit.row, audit.col, audit.tick),
		Severity: SeverityLow,
	}}
}

// checkUpgradeResult 验证升级操作是否生效。
func (d *AnomalyDetector) checkUpgradeResult(state *GameState) []Anomaly {
	if d.pendingUpgrCheck == nil {
		return nil
	}
	audit := d.pendingUpgrCheck
	d.pendingUpgrCheck = nil

	for _, t := range state.Towers {
		if t.Row == audit.row && t.Col == audit.col {
			if t.Strength <= audit.prevStr {
				return []Anomaly{{
					Tick:     state.Tick,
					Type:     "upgrade_no_effect",
					Detail:   fmt.Sprintf("tower at (%d,%d) strength unchanged after upgrade: %d", audit.row, audit.col, t.Strength),
					Severity: SeverityHigh,
				}}
			}
			return nil // 升级生效
		}
	}
	return nil
}

// checkEnemyStuck 检查敌人是否卡住（位置连续 60 帧不变，非 Dying）。
func (d *AnomalyDetector) checkEnemyStuck(state *GameState) []Anomaly {
	var found []Anomaly
	activeIDs := make(map[int]bool)

	for _, e := range state.Enemies {
		if !e.Active || e.Dying {
			continue
		}
		activeIDs[e.ID] = true

		prev, exists := d.prevEnemyPos[e.ID]
		if !exists {
			d.prevEnemyPos[e.ID] = [2]float64{e.X, e.Y}
			d.stuckCounters[e.ID] = 0
			continue
		}

		// 位置几乎不变（<0.1 像素）且速度非零（非 stunned/rooted）
		dx := e.X - prev[0]
		dy := e.Y - prev[1]
		if dx*dx+dy*dy < 0.01 && e.Speed > 0 {
			d.stuckCounters[e.ID]++
			if d.stuckCounters[e.ID] == stuckThreshold {
				found = append(found, Anomaly{
					Tick:     state.Tick,
					Type:     "enemy_stuck",
					Detail:   fmt.Sprintf("enemy#%d at (%.1f,%.1f) not moving for %d ticks", e.ID, e.X, e.Y, stuckThreshold),
					Severity: SeverityHigh,
				})
			}
		} else {
			d.stuckCounters[e.ID] = 0
		}
		d.prevEnemyPos[e.ID] = [2]float64{e.X, e.Y}
	}

	// 清理已消失的敌人
	for id := range d.prevEnemyPos {
		if !activeIDs[id] {
			delete(d.prevEnemyPos, id)
			delete(d.stuckCounters, id)
		}
	}

	return found
}

// Anomalies 返回所有已发现的异常。
func (d *AnomalyDetector) Anomalies() []Anomaly {
	return d.anomalies
}

// checkWardenCoverage 检测战灵是否被困在原始一屏范围内。
//
// 原理：
//   - 持续记录战灵位置的 min/max 包围盒
//   - 如果地图宽度 > 1200（一屏宽），但战灵 X 从未超过 1200 → 活动范围被限制
//   - 如果地图高度 > 540（一屏高），但战灵 Y 从未超过 540 → 同理
//   - 在足够多采样后（600帧 ≈ 10秒游戏时间）才做判定，避免早期误报
//
// 常量参考：ScreenWidth=1200, ScreenHeight=540
func (d *AnomalyDetector) checkWardenCoverage(state *GameState) []Anomaly {
	if !state.WardenReady || d.wardenCoverageReported {
		return nil
	}

	const (
		screenW       = 1200.0
		screenH       = 540.0
		sampleMinTick = 600 // 至少 10 秒（600 帧 @ 60fps）
	)

	// 忽略 (0,0) 初始位置
	if state.WardenX == 0 && state.WardenY == 0 {
		return nil
	}

	// 更新包围盒
	if d.wardenSamples == 0 {
		d.wardenMinX = state.WardenX
		d.wardenMaxX = state.WardenX
		d.wardenMinY = state.WardenY
		d.wardenMaxY = state.WardenY
	} else {
		if state.WardenX < d.wardenMinX {
			d.wardenMinX = state.WardenX
		}
		if state.WardenX > d.wardenMaxX {
			d.wardenMaxX = state.WardenX
		}
		if state.WardenY < d.wardenMinY {
			d.wardenMinY = state.WardenY
		}
		if state.WardenY > d.wardenMaxY {
			d.wardenMaxY = state.WardenY
		}
	}
	d.wardenSamples++

	// 采样不足，暂不判定
	if d.wardenSamples < sampleMinTick {
		return nil
	}

	// 只检查地图比一屏大的维度
	mapExtendsX := state.MapPixelW > screenW+50 // 地图宽度显著超出一屏
	mapExtendsY := state.MapPixelH > screenH+50

	if !mapExtendsX && !mapExtendsY {
		return nil // 地图就一屏大小，不需要检查
	}

	var found []Anomaly
	visitedW := d.wardenMaxX - d.wardenMinX
	visitedH := d.wardenMaxY - d.wardenMinY

	// 地图比一屏宽，但战灵水平活动范围 < 一屏宽的 80%
	if mapExtendsX && visitedW < screenW*0.8 {
		found = append(found, Anomaly{
			Tick: state.Tick,
			Type: "warden_range_limited_x",
			Detail: fmt.Sprintf(
				"map width=%.0f (>%.0f) but warden X range=[%.0f,%.0f] (span=%.0f), never reached beyond first screen",
				state.MapPixelW, screenW, d.wardenMinX, d.wardenMaxX, visitedW),
			Severity: SeverityHigh,
		})
	}

	// 地图比一屏高，但战灵垂直活动范围 < 一屏高的 80%
	if mapExtendsY && visitedH < screenH*0.8 {
		found = append(found, Anomaly{
			Tick: state.Tick,
			Type: "warden_range_limited_y",
			Detail: fmt.Sprintf(
				"map height=%.0f (>%.0f) but warden Y range=[%.0f,%.0f] (span=%.0f), never reached beyond first screen",
				state.MapPixelH, screenH, d.wardenMinY, d.wardenMaxY, visitedH),
			Severity: SeverityHigh,
		})
	}

	if len(found) > 0 {
		d.wardenCoverageReported = true // 只报一次
	}
	return found
}

// ════════════════════════════════════════════════════════════
// 回归检测规则（从已修复 Bug 推导）
// ════════════════════════════════════════════════════════════

// checkNaN 检测敌人属性 NaN/Inf（数值污染，通常是伤害管线除零）。
func (d *AnomalyDetector) checkNaN(state *GameState) []Anomaly {
	var found []Anomaly
	for _, e := range state.Enemies {
		if !e.Active {
			continue
		}
		if d.nanReported[e.ID] {
			continue
		}
		if math.IsNaN(e.HP) || math.IsInf(e.HP, 0) ||
			math.IsNaN(e.MaxHP) || math.IsInf(e.MaxHP, 0) ||
			math.IsNaN(e.Speed) || math.IsInf(e.Speed, 0) ||
			math.IsNaN(e.X) || math.IsInf(e.X, 0) ||
			math.IsNaN(e.Y) || math.IsInf(e.Y, 0) {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "nan_in_enemy",
				Detail:   fmt.Sprintf("enemy#%d archetype=%s HP=%.2f MaxHP=%.2f Speed=%.2f pos=(%.1f,%.1f)", e.ID, e.Archetype, e.HP, e.MaxHP, e.Speed, e.X, e.Y),
				Severity: SeverityCritical,
			})
			d.nanReported[e.ID] = true
		}
	}
	return found
}

// checkSpeedFloor 检测减速突破下限（非 stunned/rooted 敌人 Speed <= 0）。
// 对应 Bug: 减速系统 MinSpeedRatio=0.2 修复前 speed 可降至 0。
func (d *AnomalyDetector) checkSpeedFloor(state *GameState) []Anomaly {
	var found []Anomaly
	for _, e := range state.Enemies {
		if !e.Active || e.Dying || e.IsStunned || e.IsRooted {
			continue
		}
		if d.speedFloorReported[e.ID] {
			continue
		}
		if e.Speed <= 0 {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "speed_floor_violation",
				Detail:   fmt.Sprintf("enemy#%d archetype=%s speed=%.4f (should be >0 when not stunned/rooted)", e.ID, e.Archetype, e.Speed),
				Severity: SeverityCritical,
			})
			d.speedFloorReported[e.ID] = true
		}
	}
	// 清理已消失的敌人
	for id := range d.speedFloorReported {
		alive := false
		for _, e := range state.Enemies {
			if e.ID == id && e.Active {
				alive = true
				break
			}
		}
		if !alive {
			delete(d.speedFloorReported, id)
		}
	}
	return found
}

// checkWardenOrigin 检测战灵停留在原点 (0,0)（初始化失败）。
// 对应 Bug: 战灵出生闪现，所有战灵初始位置从 (600,270) 改为 (0,0)。
func (d *AnomalyDetector) checkWardenOrigin(state *GameState) []Anomaly {
	if !state.WardenReady || d.wardenOriginReported {
		return nil
	}

	// 记录 wardenReady 变 true 的时刻
	if d.wardenReadyTick == 0 {
		d.wardenReadyTick = state.Tick
		return nil
	}

	const graceFrames = 120 // 2 秒容忍期（等待 teleport 到轨道位置）

	if state.Tick-d.wardenReadyTick < graceFrames {
		return nil
	}

	// 在 (0,0) 附近 5px 范围内 → 卡在原点
	if state.WardenX*state.WardenX+state.WardenY*state.WardenY < 25 {
		d.wardenOriginReported = true
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "warden_origin_stuck",
			Detail:   fmt.Sprintf("warden still at (%.1f,%.1f) %d frames after ready", state.WardenX, state.WardenY, state.Tick-d.wardenReadyTick),
			Severity: SeverityCritical,
		}}
	}
	return nil
}

// checkWaveClearedStall 检测 wavesCleared 计数停滞（事件链断裂）。
// 对应 Bug: 手动开波 EvtWaveCleared 不触发，wavesCleared 永远 0。
func (d *AnomalyDetector) checkWaveClearedStall(state *GameState) []Anomaly {
	if d.waveClearedStallReported {
		return nil
	}

	// wave >= 3 但 wavesCleared 仍为 0 → 波次清除事件从未触发
	// 注意: wave 1 的 cleared 可能在 wave 2 开始时才计入，所以等到 wave >= 3
	if state.Wave >= 3 && state.WavesCleared == 0 {
		d.waveClearedStallReported = true
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "wave_cleared_stall",
			Detail:   fmt.Sprintf("wave=%d but wavesCleared=0 (onWaveTransition may not be firing)", state.Wave),
			Severity: SeverityHigh,
		}}
	}
	return nil
}

// checkArchetypeMonoculture 检测敌人原型多样性过低。
// 对应 Bug: 敌人原型全部 "normal"，Spawn 签名不接受原型信息。
func (d *AnomalyDetector) checkArchetypeMonoculture(state *GameState) []Anomaly {
	if d.archetypeMonoReported {
		return nil
	}

	// 持续收集所有见到的原型
	for _, e := range state.Enemies {
		if e.Active && e.Archetype != "" {
			d.seenArchetypes[e.Archetype] = true
		}
	}

	// wave >= 5 后检查（前几波可能只有基础原型）
	if state.Wave < 5 {
		return nil
	}

	// 只见到 1 种原型 → 原型系统可能失效
	if len(d.seenArchetypes) <= 1 {
		archs := ""
		for k := range d.seenArchetypes {
			archs = k
		}
		d.archetypeMonoReported = true
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "archetype_monoculture",
			Detail:   fmt.Sprintf("wave=%d but only %d archetype(s) seen: [%s]", state.Wave, len(d.seenArchetypes), archs),
			Severity: SeverityHigh,
		}}
	}
	return nil
}

// checkEnemyHPUniform 检测同波不同原型敌人 HP 完全一致（缩放未生效）。
// 对应 Bug: pool.go 硬编码 archetype="normal"，缩放倍率无效。
func (d *AnomalyDetector) checkEnemyHPUniform(state *GameState) []Anomaly {
	if !state.WaveActive || d.hpUniformReported[state.Wave] {
		return nil
	}

	// 收集本波活跃敌人的 (archetype, MaxHP) 对
	type pair struct {
		arch string
		hp   float64
	}
	var pairs []pair
	archSet := make(map[string]bool)
	for _, e := range state.Enemies {
		if e.Active && !e.Dying && e.MaxHP > 0 {
			pairs = append(pairs, pair{e.Archetype, e.MaxHP})
			archSet[e.Archetype] = true
		}
	}

	// 需要 >=5 个敌人且 >=2 种原型才有意义
	if len(pairs) < 5 || len(archSet) < 2 {
		return nil
	}

	// 检查所有 MaxHP 是否完全一致
	firstHP := pairs[0].hp
	allSame := true
	for _, p := range pairs[1:] {
		if math.Abs(p.hp-firstHP) > 0.01 {
			allSame = false
			break
		}
	}

	if allSame {
		d.hpUniformReported[state.Wave] = true
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "enemy_hp_uniform",
			Detail:   fmt.Sprintf("wave=%d: %d enemies with %d archetypes all have MaxHP=%.0f (hpScale not applied?)", state.Wave, len(pairs), len(archSet), firstHP),
			Severity: SeverityMedium,
		}}
	}
	return nil
}

// checkProjectileOrphan 检测弹射物堆积但无命中（追踪失效）。
// 对应 Bug: fire-and-forget 弹道，敌人拐弯后子弹全飞偏。
func (d *AnomalyDetector) checkProjectileOrphan(state *GameState) []Anomaly {
	const (
		projThreshold  = 20  // 弹射物数量阈值
		frameThreshold = 120 // 连续帧阈值（2 秒）
	)

	if state.ProjectileCount > projThreshold {
		if d.projBurstFrames == 0 {
			d.projBurstKillRef = state.TotalKills
		}
		d.projBurstFrames++

		if d.projBurstFrames >= frameThreshold {
			killDelta := state.TotalKills - d.projBurstKillRef
			if killDelta == 0 {
				d.projBurstFrames = 0 // 重置，允许再次触发
				return []Anomaly{{
					Tick:     state.Tick,
					Type:     "projectile_orphan_burst",
					Detail:   fmt.Sprintf("projectiles=%d for %d frames with 0 kills (tracking may be broken)", state.ProjectileCount, frameThreshold),
					Severity: SeverityHigh,
				}}
			}
			// 有击杀，重置
			d.projBurstFrames = 0
		}
	} else {
		d.projBurstFrames = 0
	}
	return nil
}

// checkAbilitySilent 检测塔能力长时间未触发（能力系统失效）。
// 对应 Bug: TowerAbilJSON json tag 从 "name" 改为 "type"、bounce 管线缺失等。
// 在游戏中期（wave >= 5）检查一次：有 ability 的塔战斗了足够久但遥测无记录。
func (d *AnomalyDetector) checkAbilitySilent(state *GameState) []Anomaly {
	if d.abilitySilentChecked {
		return nil
	}

	// 追踪有能力的塔的战斗帧数
	for _, t := range state.Towers {
		if len(t.Abilities) > 0 && t.HasTarget {
			d.towerAbilityTicks[t.Key]++
		}
	}

	// wave < 5 不检查（太早，能力可能尚未解锁足够多）
	if state.Wave < 5 {
		return nil
	}

	// 汇总所有在场塔拥有的能力
	declaredAbils := make(map[string]bool)
	for _, t := range state.Towers {
		// 只看战斗时间足够长的塔（300 帧 ≈ 5 秒瞄准时间）
		if d.towerAbilityTicks[t.Key] < 300 {
			continue
		}
		for _, ab := range t.Abilities {
			declaredAbils[ab] = true
		}
	}

	if len(declaredAbils) == 0 {
		return nil
	}

	// 遥测中已触发的能力（检查 ability + buff_type + cc 维度）
	triggered := make(map[string]bool)
	for k := range state.Telemetry.AbilityTriggered {
		triggered[k] = true
	}
	for k := range state.Telemetry.BuffTypesApplied {
		triggered[k] = true
	}
	for k := range state.Telemetry.CCApplied {
		triggered[k] = true
	}

	// 能力 → 遥测关键词映射（能力触发应在某个遥测维度有记录）
	abilToTelemetry := map[string][]string{
		"bounce":      {"bounce"},
		"splash":      {"splash"},
		"burn":        {"burn"},
		"bleed":       {"bleed"},
		"stun":        {"stun"},
		"freeze":      {"slow"},
		"slow":        {"slow"},
		"shield":      {"shield"},
		"hunter":      {"hunter"},
		"shieldBreak": {"shieldBreak", "shield_break"},
	}

	var found []Anomaly
	for ab := range declaredAbils {
		keywords, mapped := abilToTelemetry[ab]
		if !mapped {
			continue // 未映射的能力跳过
		}
		anyTriggered := false
		for _, kw := range keywords {
			if triggered[kw] {
				anyTriggered = true
				break
			}
		}
		if !anyTriggered {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "ability_silent",
				Detail:   fmt.Sprintf("ability '%s' declared on towers but never seen in telemetry (wave=%d)", ab, state.Wave),
				Severity: SeverityHigh,
			})
		}
	}

	if len(found) > 0 {
		d.abilitySilentChecked = true // 只报一次
	}
	return found
}

// checkInteractModeStuck 检测交互模式停滞（HUD 面板无法关闭）。
// 对应 Bug: 炮塔 HUD 点击外部无法关闭。
// 非 idle(0) 模式连续 600 帧（10 秒）不变 → 可能卡住。
// 排除 paused(7) 模式（暂停是正常的长时间状态）。
func (d *AnomalyDetector) checkInteractModeStuck(state *GameState) []Anomaly {
	const stuckFrames = 600 // 10 秒 @60fps

	mode := state.InteractMode

	// 暂停模式不检查
	if mode == 7 {
		d.imodeStuckMode = mode
		d.imodeStuckTick = state.Tick
		return nil
	}

	if mode != d.imodeStuckMode {
		// 模式变化，重置计时
		d.imodeStuckMode = mode
		d.imodeStuckTick = state.Tick
		return nil
	}

	// idle 模式不算停滞
	if mode == 0 {
		return nil
	}

	// 已报过此模式
	if d.imodeStuckReported[mode] {
		return nil
	}

	if state.Tick-d.imodeStuckTick >= stuckFrames {
		d.imodeStuckReported[mode] = true
		name := "unknown"
		names := map[int]string{
			1: "buildMenu", 2: "buildPlace", 3: "towerSel",
			4: "spawnMenu", 5: "spawnPlace", 8: "wardenSelect",
		}
		if n, ok := names[mode]; ok {
			name = n
		}
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "interact_mode_stuck",
			Detail:   fmt.Sprintf("mode=%d(%s) unchanged for %d frames", mode, name, stuckFrames),
			Severity: SeverityHigh,
		}}
	}
	return nil
}

// checkWardenTeleport 检测战灵帧间位置跳跃（移动中瞬移）。
// 对应 Bug: 战灵移动跳跃，高速下位置不连续。
// 帧间距离 > 50px（正常最大速度 ~3px/帧）→ 瞬移。
func (d *AnomalyDetector) checkWardenTeleport(state *GameState) []Anomaly {
	if !state.WardenReady {
		return nil
	}

	// 忽略 (0,0) — 初始未就位
	if state.WardenX == 0 && state.WardenY == 0 {
		return nil
	}

	if !d.wardenPosInit {
		d.wardenPrevX = state.WardenX
		d.wardenPrevY = state.WardenY
		d.wardenPosInit = true
		return nil
	}

	dx := state.WardenX - d.wardenPrevX
	dy := state.WardenY - d.wardenPrevY
	dist := math.Sqrt(dx*dx + dy*dy)

	d.wardenPrevX = state.WardenX
	d.wardenPrevY = state.WardenY

	const teleportThreshold = 50.0 // 正常移动不会一帧超过 50px

	if dist > teleportThreshold {
		d.wardenTeleportCount++
		// 容忍首次（可能是初始 teleport 到轨道位置）
		if d.wardenTeleportCount <= 1 {
			return nil
		}
		return []Anomaly{{
			Tick:     state.Tick,
			Type:     "warden_teleport",
			Detail:   fmt.Sprintf("warden moved %.1fpx in 1 frame (%.1f,%.1f)->(%.1f,%.1f)", dist, state.WardenX-dx, state.WardenY-dy, state.WardenX, state.WardenY),
			Severity: SeverityMedium,
		}}
	}
	return nil
}

// checkEnemyBornWithHit 检测新生敌人立即处于 hit 状态。
// 对应 Bug: 怪物出现就播 hit 帧并一直保持。
// 首次见到某敌人时 IsHit=true → 出生即带 HitFlash（不应该）。
func (d *AnomalyDetector) checkEnemyBornWithHit(state *GameState) []Anomaly {
	var found []Anomaly
	activeIDs := make(map[int]bool)

	for _, e := range state.Enemies {
		if !e.Active || e.Dying {
			continue
		}
		activeIDs[e.ID] = true

		if d.enemyFirstSeen[e.ID] {
			continue // 非首帧
		}
		d.enemyFirstSeen[e.ID] = true

		if e.IsHit {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "enemy_born_with_hit",
				Detail:   fmt.Sprintf("enemy#%d archetype=%s spawned with HitFlash active (IsHit=true)", e.ID, e.Archetype),
				Severity: SeverityHigh,
			})
		}
	}

	// 清理已消失的敌人（ID 可能被 pool 复用）
	for id := range d.enemyFirstSeen {
		if !activeIDs[id] {
			delete(d.enemyFirstSeen, id)
		}
	}
	return found
}

// checkBossTooWeak 检测 Boss 存活时间过短（<5 秒 = 300 帧）。
// 对应手动测试文档 §3.3: "Boss 和小怪没区别"。
func (d *AnomalyDetector) checkBossTooWeak(state *GameState) []Anomaly {
	const weakThreshold = 300 // 5 秒 @60fps

	var found []Anomaly
	activeIDs := make(map[int]bool)

	// 追踪新出现的 Boss
	for _, e := range state.Enemies {
		if !e.Active || !e.Boss {
			continue
		}
		activeIDs[e.ID] = true
		if _, tracked := d.activeBosses[e.ID]; !tracked {
			d.activeBosses[e.ID] = state.Tick
		}
	}

	// 检测消失的 Boss
	for id, spawnTick := range d.activeBosses {
		if activeIDs[id] {
			continue
		}
		aliveTicks := state.Tick - spawnTick
		if aliveTicks < weakThreshold {
			found = append(found, Anomaly{
				Tick:     state.Tick,
				Type:     "boss_too_weak",
				Detail:   fmt.Sprintf("boss (spawned tick=%d) died after %.1fs (%d ticks), threshold=5s", spawnTick, float64(aliveTicks)/60.0, aliveTicks),
				Severity: SeverityMedium,
			})
		}
		delete(d.activeBosses, id)
	}
	return found
}

// checkEconomyStall 检测经济断档（连续 20s 买不起最便宜的塔且有空位）。
// 对应方案文档 §2.1: "连续 N 秒攒不出下一座塔"。
func (d *AnomalyDetector) checkEconomyStall(state *GameState) []Anomaly {
	if d.econStallReported {
		return nil
	}

	// 缓存最便宜塔价格
	if d.econStallMinCost == 0 && len(state.TowerDefs) > 0 {
		d.econStallMinCost = state.TowerDefs[0].Cost
		for _, td := range state.TowerDefs[1:] {
			if td.Cost < d.econStallMinCost {
				d.econStallMinCost = td.Cost
			}
		}
	}
	if d.econStallMinCost == 0 {
		return nil
	}

	const stallThreshold = 1200 // 20 秒 @60fps

	// 有空地但买不起
	if len(state.BuildCells) > 0 && state.Gold < d.econStallMinCost && state.Wave >= 2 {
		d.econStallTicks++
		if d.econStallTicks >= stallThreshold {
			d.econStallReported = true
			return []Anomaly{{
				Tick:     state.Tick,
				Type:     "economy_stall",
				Detail:   fmt.Sprintf("wave=%d gold=%d < minCost=%d for %ds (economy may be too tight)", state.Wave, state.Gold, d.econStallMinCost, stallThreshold/60),
				Severity: SeverityMedium,
			}}
		}
	} else {
		d.econStallTicks = 0
	}
	return nil
}

// RecordPanic 记录 panic 恢复异常。
func (d *AnomalyDetector) RecordPanic(tick int, detail string) {
	a := Anomaly{
		Tick:     tick,
		Type:     "panic_recovered",
		Detail:   detail,
		Severity: SeverityCritical,
	}
	d.anomalies = append(d.anomalies, a)
}
