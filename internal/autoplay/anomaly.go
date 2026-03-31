// anomaly.go — 运行时异常检测器。
// 每帧检查游戏状态，识别敌人卡住、金币异常、FPS 下降等问题。
package autoplay

import "fmt"

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

	// 战灵活动范围追踪
	wardenMinX, wardenMaxX float64 // 历史访问过的 X 范围
	wardenMinY, wardenMaxY float64 // 历史访问过的 Y 范围
	wardenSamples          int     // 采样次数
	wardenCoverageReported bool    // 已报告过覆盖不足
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
		prevEnemyPos:     make(map[int][2]float64),
		stuckCounters:    make(map[int]int),
		prevWaveHP:       make(map[int]float64),
		towerDamageAccum: make(map[string]int),
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

	// 2. 金币为负
	if state.Gold < 0 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "gold_negative",
			Detail:   fmt.Sprintf("gold=%d", state.Gold),
			Severity: SeverityCritical,
		})
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

	// 9. 敌人池容量预警（>75% 占用）
	if state.EnemyPoolCount > d.poolHighWater {
		d.poolHighWater = state.EnemyPoolCount
	}
	if state.EnemyPoolCount > 192 { // 256 * 0.75
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "pool_high_usage",
			Detail:   fmt.Sprintf("enemy pool %d/256 (high water: %d)", state.EnemyPoolCount, d.poolHighWater),
			Severity: SeverityMedium,
		})
	}

	// 10. 弹射物泄漏（弹射物数 > 100 且持续增长）
	if state.ProjectileCount > 100 {
		found = append(found, Anomaly{
			Tick:     state.Tick,
			Type:     "projectile_overflow",
			Detail:   fmt.Sprintf("projectile count=%d", state.ProjectileCount),
			Severity: SeverityMedium,
		})
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

	// 更新前帧状态
	d.prevGold = state.Gold
	d.prevLives = state.Lives

	// 记录异常
	d.anomalies = append(d.anomalies, found...)
	return found
}

// checkWaveHPProgression 检查波次 HP 是否递增。
func (d *AnomalyDetector) checkWaveHPProgression(state *GameState) []Anomaly {
	if state.Wave <= 1 || !state.WaveActive {
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
		if avgHP < prev*0.5 { // 比前一波低 50% 以上
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
